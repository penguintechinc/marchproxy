package qos

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	qosBytesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_qos_bytes_processed_total",
			Help: "Total bytes processed by QoS",
		},
		[]string{"priority"},
	)

	qosPacketsDropped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_qos_packets_dropped_total",
			Help: "Total packets dropped by QoS",
		},
		[]string{"priority", "reason"},
	)

	qosQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "marchproxy_qos_queue_depth",
			Help: "Current QoS queue depth",
		},
		[]string{"priority"},
	)
)

// Priority levels
const (
	PriorityP0 = iota // Highest priority
	PriorityP1
	PriorityP2
	PriorityP3 // Lowest priority (Best effort)
)

// TrafficShaper implements QoS traffic shaping with priority queues
type TrafficShaper struct {
	mu sync.RWMutex

	// Token buckets for rate limiting
	buckets map[int]*TokenBucket

	// Priority queues
	queues map[int]*PriorityQueue

	// DSCP marker
	dscpMarker *DSCPMarker

	// Configuration
	defaultBandwidth int64
	burstSize        int64
	queueDepth       int

	// Statistics
	stats *Stats

	logger *logrus.Logger
}

// Stats holds QoS statistics
type Stats struct {
	mu sync.RWMutex

	BytesProcessed   map[int]uint64
	PacketsProcessed map[int]uint64
	PacketsDropped   map[int]uint64
	QueueDepth       map[int]int
}

// NewTrafficShaper creates a new traffic shaper
func NewTrafficShaper(defaultBandwidth, burstSize int64, queueDepth int, dscpMapping map[string]uint8, logger *logrus.Logger) *TrafficShaper {
	ts := &TrafficShaper{
		buckets:          make(map[int]*TokenBucket),
		queues:           make(map[int]*PriorityQueue),
		defaultBandwidth: defaultBandwidth,
		burstSize:        burstSize,
		queueDepth:       queueDepth,
		logger:           logger,
		stats: &Stats{
			BytesProcessed:   make(map[int]uint64),
			PacketsProcessed: make(map[int]uint64),
			PacketsDropped:   make(map[int]uint64),
			QueueDepth:       make(map[int]int),
		},
	}

	// Initialize token buckets for each priority
	for i := PriorityP0; i <= PriorityP3; i++ {
		// Higher priority gets more bandwidth
		priorityMultiplier := float64(4 - i)
		bandwidth := int64(float64(defaultBandwidth) * priorityMultiplier / 10.0)
		ts.buckets[i] = NewTokenBucket(bandwidth, burstSize)
	}

	// Initialize priority queues
	for i := PriorityP0; i <= PriorityP3; i++ {
		ts.queues[i] = NewPriorityQueue(queueDepth, i)
	}

	// Initialize DSCP marker
	ts.dscpMarker = NewDSCPMarker(dscpMapping)

	logger.WithFields(logrus.Fields{
		"default_bandwidth": defaultBandwidth,
		"burst_size":        burstSize,
		"queue_depth":       queueDepth,
	}).Info("QoS traffic shaper initialized")

	return ts
}

// Shape processes a packet through QoS
func (ts *TrafficShaper) Shape(packet *Packet) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	priority := packet.Priority
	size := int64(packet.Size)

	// Check if we have tokens for this packet
	bucket := ts.buckets[priority]
	if !bucket.TryConsume(size) {
		// No tokens available, enqueue if possible
		queue := ts.queues[priority]
		if err := queue.Enqueue(packet); err != nil {
			// Queue full, drop packet
			ts.recordDrop(priority, "queue_full")
			return fmt.Errorf("packet dropped: queue full")
		}
		return nil
	}

	// Mark DSCP
	if err := ts.dscpMarker.Mark(packet); err != nil {
		ts.logger.WithError(err).Warn("Failed to mark DSCP")
	}

	// Record stats
	ts.recordProcessed(priority, size)

	return nil
}

// ProcessQueues processes pending packets from queues
func (ts *TrafficShaper) ProcessQueues() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	processed := 0

	// Process queues in priority order
	for priority := PriorityP0; priority <= PriorityP3; priority++ {
		queue := ts.queues[priority]
		bucket := ts.buckets[priority]

		for {
			packet := queue.Peek()
			if packet == nil {
				break
			}

			size := int64(packet.Size)
			if !bucket.TryConsume(size) {
				break
			}

			// Dequeue and process
			queue.Dequeue()

			// Mark DSCP
			if err := ts.dscpMarker.Mark(packet); err != nil {
				ts.logger.WithError(err).Warn("Failed to mark DSCP")
			}

			ts.recordProcessed(priority, size)
			processed++
		}
	}

	return processed
}

// UpdateBandwidth updates bandwidth allocation for a priority
func (ts *TrafficShaper) UpdateBandwidth(priority int, bandwidth int64) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	bucket, ok := ts.buckets[priority]
	if !ok {
		return fmt.Errorf("invalid priority: %d", priority)
	}

	bucket.SetRate(bandwidth)

	ts.logger.WithFields(logrus.Fields{
		"priority":  priority,
		"bandwidth": bandwidth,
	}).Info("Updated bandwidth allocation")

	return nil
}

// GetStats returns current statistics
func (ts *TrafficShaper) GetStats() map[string]interface{} {
	ts.stats.mu.RLock()
	defer ts.stats.mu.RUnlock()

	stats := make(map[string]interface{})

	for priority := PriorityP0; priority <= PriorityP3; priority++ {
		priorityName := fmt.Sprintf("P%d", priority)
		stats[priorityName] = map[string]interface{}{
			"bytes_processed":   ts.stats.BytesProcessed[priority],
			"packets_processed": ts.stats.PacketsProcessed[priority],
			"packets_dropped":   ts.stats.PacketsDropped[priority],
			"queue_depth":       ts.queues[priority].Depth(),
		}
	}

	return stats
}

// recordProcessed records a processed packet
func (ts *TrafficShaper) recordProcessed(priority int, size int64) {
	ts.stats.mu.Lock()
	defer ts.stats.mu.Unlock()

	ts.stats.BytesProcessed[priority] += uint64(size)
	ts.stats.PacketsProcessed[priority]++

	priorityName := fmt.Sprintf("P%d", priority)
	qosBytesProcessed.WithLabelValues(priorityName).Add(float64(size))
}

// recordDrop records a dropped packet
func (ts *TrafficShaper) recordDrop(priority int, reason string) {
	ts.stats.mu.Lock()
	defer ts.stats.mu.Unlock()

	ts.stats.PacketsDropped[priority]++

	priorityName := fmt.Sprintf("P%d", priority)
	qosPacketsDropped.WithLabelValues(priorityName, reason).Inc()
}

// Start starts the traffic shaper background processing
func (ts *TrafficShaper) Start() {
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			ts.ProcessQueues()
		}
	}()

	ts.logger.Info("QoS traffic shaper started")
}

// Packet represents a network packet with QoS metadata
type Packet struct {
	Data     []byte
	Size     int
	Priority int
	DSCP     uint8
	SrcIP    string
	DstIP    string
	SrcPort  uint16
	DstPort  uint16
	Protocol string
}
