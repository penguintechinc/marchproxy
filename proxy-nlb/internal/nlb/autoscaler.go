package nlb

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	scaleOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_scale_operations_total",
			Help: "Total number of scale operations",
		},
		[]string{"protocol", "direction"},
	)

	currentReplicas = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nlb_current_replicas",
			Help: "Current number of replicas per protocol",
		},
		[]string{"protocol"},
	)

	scaleDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_scale_decisions_total",
			Help: "Total number of scaling decisions made",
		},
		[]string{"protocol", "decision"},
	)
)

// ScalingMetrics holds metrics used for scaling decisions
type ScalingMetrics struct {
	CPUUtilization    float64
	MemoryUtilization float64
	ConnectionCount   int
	RequestRate       float64
	ErrorRate         float64
	Timestamp         time.Time
}

// ScalingPolicy defines autoscaling behavior
type ScalingPolicy struct {
	Protocol             Protocol
	MinReplicas          int
	MaxReplicas          int
	TargetCPU            float64 // Target CPU utilization (0-100)
	TargetMemory         float64 // Target memory utilization (0-100)
	TargetConnPerReplica int     // Target connections per replica
	ScaleUpThreshold     float64 // Scale up when metric > threshold
	ScaleDownThreshold   float64 // Scale down when metric < threshold
	ScaleUpCooldown      time.Duration
	ScaleDownCooldown    time.Duration
	EvaluationPeriods    int // Number of periods before scaling
}

// DefaultScalingPolicy returns default scaling policy
func DefaultScalingPolicy(protocol Protocol) *ScalingPolicy {
	return &ScalingPolicy{
		Protocol:             protocol,
		MinReplicas:          1,
		MaxReplicas:          10,
		TargetCPU:            70.0,
		TargetMemory:         80.0,
		TargetConnPerReplica: 1000,
		ScaleUpThreshold:     0.8,
		ScaleDownThreshold:   0.3,
		ScaleUpCooldown:      3 * time.Minute,
		ScaleDownCooldown:    5 * time.Minute,
		EvaluationPeriods:    3,
	}
}

// ScalingHistory tracks recent scaling operations
type ScalingHistory struct {
	Timestamp time.Time
	Protocol  Protocol
	Action    string // "scale_up" or "scale_down"
	FromCount int
	ToCount   int
	Reason    string
}

// Autoscaler manages automatic scaling of module containers
type Autoscaler struct {
	policies        map[Protocol]*ScalingPolicy
	metricsHistory  map[Protocol][]*ScalingMetrics
	scalingHistory  []*ScalingHistory
	lastScaleTime   map[Protocol]time.Time
	router          *Router
	scalingEnabled  bool
	evaluationInterval time.Duration
	maxHistorySize  int
	mu              sync.RWMutex
	logger          *logrus.Logger
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// NewAutoscaler creates a new autoscaler
func NewAutoscaler(router *Router, logger *logrus.Logger) *Autoscaler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Autoscaler{
		policies:           make(map[Protocol]*ScalingPolicy),
		metricsHistory:     make(map[Protocol][]*ScalingMetrics),
		scalingHistory:     make([]*ScalingHistory, 0),
		lastScaleTime:      make(map[Protocol]time.Time),
		router:             router,
		scalingEnabled:     false,
		evaluationInterval: 30 * time.Second,
		maxHistorySize:     100,
		logger:             logger,
		ctx:                ctx,
		cancel:             cancel,
	}
}

// SetPolicy sets scaling policy for a protocol
func (as *Autoscaler) SetPolicy(policy *ScalingPolicy) error {
	if policy == nil {
		return errors.New("policy cannot be nil")
	}

	if policy.MinReplicas < 0 || policy.MaxReplicas < policy.MinReplicas {
		return errors.New("invalid replica count configuration")
	}

	as.mu.Lock()
	defer as.mu.Unlock()

	as.policies[policy.Protocol] = policy

	as.logger.WithFields(logrus.Fields{
		"protocol":     policy.Protocol.String(),
		"min_replicas": policy.MinReplicas,
		"max_replicas": policy.MaxReplicas,
		"target_cpu":   policy.TargetCPU,
	}).Info("Scaling policy configured")

	return nil
}

// GetPolicy returns scaling policy for a protocol
func (as *Autoscaler) GetPolicy(protocol Protocol) *ScalingPolicy {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.policies[protocol]
}

// RecordMetrics records metrics for scaling decisions
func (as *Autoscaler) RecordMetrics(protocol Protocol, metrics *ScalingMetrics) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.metricsHistory[protocol] == nil {
		as.metricsHistory[protocol] = make([]*ScalingMetrics, 0)
	}

	metrics.Timestamp = time.Now()
	as.metricsHistory[protocol] = append(as.metricsHistory[protocol], metrics)

	// Trim history
	if len(as.metricsHistory[protocol]) > as.maxHistorySize {
		as.metricsHistory[protocol] = as.metricsHistory[protocol][1:]
	}
}

// Start starts the autoscaler evaluation loop
func (as *Autoscaler) Start() error {
	as.mu.Lock()
	if as.scalingEnabled {
		as.mu.Unlock()
		return errors.New("autoscaler already running")
	}
	as.scalingEnabled = true
	as.mu.Unlock()

	as.wg.Add(1)
	go as.evaluationLoop()

	as.logger.Info("Autoscaler started")
	return nil
}

// Stop stops the autoscaler
func (as *Autoscaler) Stop() {
	as.mu.Lock()
	if !as.scalingEnabled {
		as.mu.Unlock()
		return
	}
	as.scalingEnabled = false
	as.mu.Unlock()

	as.cancel()
	as.wg.Wait()

	as.logger.Info("Autoscaler stopped")
}

// evaluationLoop periodically evaluates scaling decisions
func (as *Autoscaler) evaluationLoop() {
	defer as.wg.Done()

	ticker := time.NewTicker(as.evaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.evaluate()
		}
	}
}

// evaluate evaluates scaling decisions for all protocols
func (as *Autoscaler) evaluate() {
	as.mu.RLock()
	protocols := make([]Protocol, 0, len(as.policies))
	for protocol := range as.policies {
		protocols = append(protocols, protocol)
	}
	as.mu.RUnlock()

	for _, protocol := range protocols {
		if decision := as.evaluateProtocol(protocol); decision != "none" {
			as.executeScaling(protocol, decision)
		}
	}
}

// evaluateProtocol evaluates scaling decision for a specific protocol
func (as *Autoscaler) evaluateProtocol(protocol Protocol) string {
	as.mu.RLock()
	policy, exists := as.policies[protocol]
	if !exists {
		as.mu.RUnlock()
		return "none"
	}

	metrics := as.metricsHistory[protocol]
	lastScaleTime := as.lastScaleTime[protocol]
	as.mu.RUnlock()

	if len(metrics) < policy.EvaluationPeriods {
		scaleDecisions.WithLabelValues(protocol.String(), "insufficient_data").Inc()
		return "none"
	}

	// Get current replica count
	modules := as.router.GetModules(protocol)
	currentCount := len(modules)

	// Calculate average metrics over evaluation periods
	recentMetrics := metrics[len(metrics)-policy.EvaluationPeriods:]
	avgCPU := 0.0
	avgMemory := 0.0
	avgConnPerReplica := 0.0

	for _, m := range recentMetrics {
		avgCPU += m.CPUUtilization
		avgMemory += m.MemoryUtilization
		if currentCount > 0 {
			avgConnPerReplica += float64(m.ConnectionCount) / float64(currentCount)
		}
	}

	avgCPU /= float64(len(recentMetrics))
	avgMemory /= float64(len(recentMetrics))
	avgConnPerReplica /= float64(len(recentMetrics))

	// Check cooldown periods
	now := time.Now()
	timeSinceLastScale := now.Sub(lastScaleTime)

	// Scale up decision
	cpuPressure := avgCPU / policy.TargetCPU
	memPressure := avgMemory / policy.TargetMemory
	connPressure := avgConnPerReplica / float64(policy.TargetConnPerReplica)

	maxPressure := max64(cpuPressure, max64(memPressure, connPressure))

	if maxPressure >= policy.ScaleUpThreshold && currentCount < policy.MaxReplicas {
		if timeSinceLastScale >= policy.ScaleUpCooldown {
			scaleDecisions.WithLabelValues(protocol.String(), "scale_up").Inc()
			as.logger.WithFields(logrus.Fields{
				"protocol":      protocol.String(),
				"current_count": currentCount,
				"cpu_pressure":  cpuPressure,
				"mem_pressure":  memPressure,
				"conn_pressure": connPressure,
			}).Info("Scale up decision")
			return "scale_up"
		}
	}

	// Scale down decision
	if maxPressure <= policy.ScaleDownThreshold && currentCount > policy.MinReplicas {
		if timeSinceLastScale >= policy.ScaleDownCooldown {
			scaleDecisions.WithLabelValues(protocol.String(), "scale_down").Inc()
			as.logger.WithFields(logrus.Fields{
				"protocol":      protocol.String(),
				"current_count": currentCount,
				"cpu_pressure":  cpuPressure,
				"mem_pressure":  memPressure,
				"conn_pressure": connPressure,
			}).Info("Scale down decision")
			return "scale_down"
		}
	}

	scaleDecisions.WithLabelValues(protocol.String(), "none").Inc()
	return "none"
}

// executeScaling executes scaling operation
func (as *Autoscaler) executeScaling(protocol Protocol, action string) {
	modules := as.router.GetModules(protocol)
	currentCount := len(modules)

	var targetCount int
	if action == "scale_up" {
		targetCount = currentCount + 1
		scaleOperations.WithLabelValues(protocol.String(), "up").Inc()
	} else if action == "scale_down" {
		targetCount = currentCount - 1
		scaleOperations.WithLabelValues(protocol.String(), "down").Inc()
	} else {
		return
	}

	// Record scaling operation
	as.mu.Lock()
	as.lastScaleTime[protocol] = time.Now()
	as.scalingHistory = append(as.scalingHistory, &ScalingHistory{
		Timestamp: time.Now(),
		Protocol:  protocol,
		Action:    action,
		FromCount: currentCount,
		ToCount:   targetCount,
		Reason:    "autoscaling",
	})
	as.mu.Unlock()

	currentReplicas.WithLabelValues(protocol.String()).Set(float64(targetCount))

	as.logger.WithFields(logrus.Fields{
		"protocol": protocol.String(),
		"action":   action,
		"from":     currentCount,
		"to":       targetCount,
	}).Info("Scaling operation executed")

	// TODO: Integrate with container orchestrator (Docker, Kubernetes) to actually scale
}

// GetStats returns autoscaler statistics
func (as *Autoscaler) GetStats() map[string]interface{} {
	as.mu.RLock()
	defer as.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["enabled"] = as.scalingEnabled
	stats["evaluation_interval"] = as.evaluationInterval.String()

	policyStats := make(map[string]interface{})
	for protocol, policy := range as.policies {
		policyStats[protocol.String()] = map[string]interface{}{
			"min_replicas":      policy.MinReplicas,
			"max_replicas":      policy.MaxReplicas,
			"target_cpu":        policy.TargetCPU,
			"target_memory":     policy.TargetMemory,
			"scale_up_cooldown": policy.ScaleUpCooldown.String(),
		}
	}
	stats["policies"] = policyStats

	// Recent scaling history
	recentHistory := make([]map[string]interface{}, 0)
	historyCount := min(10, len(as.scalingHistory))
	for i := len(as.scalingHistory) - historyCount; i < len(as.scalingHistory); i++ {
		h := as.scalingHistory[i]
		recentHistory = append(recentHistory, map[string]interface{}{
			"timestamp": h.Timestamp,
			"protocol":  h.Protocol.String(),
			"action":    h.Action,
			"from":      h.FromCount,
			"to":        h.ToCount,
			"reason":    h.Reason,
		})
	}
	stats["recent_scaling_history"] = recentHistory

	return stats
}

// Helper function for float64 max
func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
