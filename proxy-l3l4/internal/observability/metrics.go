package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Connection metrics
	ActiveConnections    prometheus.Gauge
	TotalConnections     prometheus.Counter
	ConnectionDuration   prometheus.Histogram
	ConnectionErrors     prometheus.Counter

	// Traffic metrics
	BytesSent     prometheus.Counter
	BytesReceived prometheus.Counter
	PacketsSent   prometheus.Counter
	PacketsReceived prometheus.Counter

	// Routing metrics
	RoutingDecisions prometheus.CounterVec
	BackendLatency   prometheus.HistogramVec

	// QoS metrics
	QoSBytesProcessed prometheus.CounterVec
	QoSPacketsDropped prometheus.CounterVec
	QoSQueueDepth     prometheus.GaugeVec

	// NUMA metrics
	NumaWorkers      prometheus.Gauge
	NumaNodesActive  prometheus.Gauge

	// Acceleration metrics
	XDPPacketsProcessed prometheus.Counter
	AFXDPPacketsProcessed prometheus.Counter
}

// NewMetrics creates and registers Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_connections",
				Help:      "Number of active connections",
			},
		),
		TotalConnections: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "total_connections_total",
				Help:      "Total number of connections",
			},
		),
		ConnectionDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "connection_duration_seconds",
				Help:      "Connection duration in seconds",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
			},
		),
		ConnectionErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "connection_errors_total",
				Help:      "Total connection errors",
			},
		),
		BytesSent: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bytes_sent_total",
				Help:      "Total bytes sent",
			},
		),
		BytesReceived: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bytes_received_total",
				Help:      "Total bytes received",
			},
		),
		PacketsSent: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "packets_sent_total",
				Help:      "Total packets sent",
			},
		),
		PacketsReceived: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "packets_received_total",
				Help:      "Total packets received",
			},
		),
		RoutingDecisions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "routing_decisions_total",
				Help:      "Total routing decisions",
			},
			[]string{"algorithm", "backend"},
		),
		BackendLatency: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "backend_latency_seconds",
				Help:      "Backend latency in seconds",
				Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 12),
			},
			[]string{"backend", "cloud", "region"},
		),
		QoSBytesProcessed: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "qos_bytes_processed_total",
				Help:      "Total bytes processed by QoS",
			},
			[]string{"priority"},
		),
		QoSPacketsDropped: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "qos_packets_dropped_total",
				Help:      "Total packets dropped by QoS",
			},
			[]string{"priority", "reason"},
		),
		QoSQueueDepth: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "qos_queue_depth",
				Help:      "Current QoS queue depth",
			},
			[]string{"priority"},
		),
		NumaWorkers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "numa_workers",
				Help:      "Number of NUMA-aware workers",
			},
		),
		NumaNodesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "numa_nodes_active",
				Help:      "Number of active NUMA nodes",
			},
		),
		XDPPacketsProcessed: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "xdp_packets_processed_total",
				Help:      "Total packets processed by XDP",
			},
		),
		AFXDPPacketsProcessed: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "afxdp_packets_processed_total",
				Help:      "Total packets processed by AF_XDP",
			},
		),
	}
}
