package multicloud

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	routingDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_multicloud_routing_decisions_total",
			Help: "Total routing decisions made",
		},
		[]string{"algorithm", "backend"},
	)

	backendSelections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_multicloud_backend_selections_total",
			Help: "Total backend selections",
		},
		[]string{"backend", "cloud", "region"},
	)
)

// Router implements intelligent multi-cloud routing
type Router struct {
	mu sync.RWMutex

	backends   []*Backend
	algorithm  RoutingAlgorithm
	monitor    *HealthMonitor
	costAnalyzer *CostAnalyzer

	logger *logrus.Logger
}

// Backend represents a backend server
type Backend struct {
	Name     string
	URL      string
	Weight   int
	Priority int
	Cloud    string
	Region   string
	Cost     float64
	Healthy  bool
	Latency  int64 // microseconds
	Connections int
}

// NewRouter creates a new multi-cloud router
func NewRouter(algorithm string, backends []*Backend, logger *logrus.Logger) (*Router, error) {
	var algo RoutingAlgorithm

	switch algorithm {
	case "latency":
		algo = &LatencyBasedAlgorithm{}
	case "cost":
		algo = &CostBasedAlgorithm{}
	case "geo":
		algo = &GeoProximityAlgorithm{}
	case "roundrobin":
		algo = &RoundRobinAlgorithm{}
	case "leastconn":
		algo = &LeastConnectionAlgorithm{}
	default:
		return nil, fmt.Errorf("unknown routing algorithm: %s", algorithm)
	}

	router := &Router{
		backends:  backends,
		algorithm: algo,
		logger:    logger,
	}

	// Initialize health monitor
	router.monitor = NewHealthMonitor(backends, logger)

	// Initialize cost analyzer
	router.costAnalyzer = NewCostAnalyzer(backends, logger)

	logger.WithFields(logrus.Fields{
		"algorithm": algorithm,
		"backends":  len(backends),
	}).Info("Multi-cloud router initialized")

	return router, nil
}

// Route selects the best backend for a request
func (r *Router) Route(request *Request) (*Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter healthy backends
	healthyBackends := r.getHealthyBackends()
	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	// Apply routing algorithm
	backend := r.algorithm.Select(healthyBackends, request)
	if backend == nil {
		return nil, fmt.Errorf("no suitable backend found")
	}

	// Record metrics
	routingDecisions.WithLabelValues(r.algorithm.Name(), backend.Name).Inc()
	backendSelections.WithLabelValues(backend.Name, backend.Cloud, backend.Region).Inc()

	r.logger.WithFields(logrus.Fields{
		"algorithm": r.algorithm.Name(),
		"backend":   backend.Name,
		"cloud":     backend.Cloud,
		"region":    backend.Region,
	}).Debug("Routed request to backend")

	return backend, nil
}

// getHealthyBackends returns only healthy backends
func (r *Router) getHealthyBackends() []*Backend {
	var healthy []*Backend
	for _, backend := range r.backends {
		if backend.Healthy {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

// UpdateBackendHealth updates the health status of a backend
func (r *Router) UpdateBackendHealth(name string, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, backend := range r.backends {
		if backend.Name == name {
			backend.Healthy = healthy
			r.logger.WithFields(logrus.Fields{
				"backend": name,
				"healthy": healthy,
			}).Info("Backend health updated")
			break
		}
	}
}

// UpdateBackendLatency updates the latency of a backend
func (r *Router) UpdateBackendLatency(name string, latency int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, backend := range r.backends {
		if backend.Name == name {
			backend.Latency = latency
			break
		}
	}
}

// IncrementConnections increments the connection count for a backend
func (r *Router) IncrementConnections(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, backend := range r.backends {
		if backend.Name == name {
			backend.Connections++
			break
		}
	}
}

// DecrementConnections decrements the connection count for a backend
func (r *Router) DecrementConnections(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, backend := range r.backends {
		if backend.Name == name {
			if backend.Connections > 0 {
				backend.Connections--
			}
			break
		}
	}
}

// Start starts the router's background tasks
func (r *Router) Start() error {
	// Start health monitoring
	if err := r.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}

	// Start cost analysis
	r.costAnalyzer.Start()

	r.logger.Info("Multi-cloud router started")
	return nil
}

// Stop stops the router
func (r *Router) Stop() {
	if r.monitor != nil {
		r.monitor.Stop()
	}
	if r.costAnalyzer != nil {
		r.costAnalyzer.Stop()
	}
	r.logger.Info("Multi-cloud router stopped")
}

// GetStats returns router statistics
func (r *Router) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"algorithm":      r.algorithm.Name(),
		"total_backends": len(r.backends),
		"healthy_backends": len(r.getHealthyBackends()),
	}

	backends := make([]map[string]interface{}, 0, len(r.backends))
	for _, b := range r.backends {
		backends = append(backends, map[string]interface{}{
			"name":        b.Name,
			"cloud":       b.Cloud,
			"region":      b.Region,
			"healthy":     b.Healthy,
			"latency":     b.Latency,
			"connections": b.Connections,
		})
	}
	stats["backends"] = backends

	return stats
}

// Request represents a routing request
type Request struct {
	SourceIP   string
	DestIP     string
	Protocol   string
	SourceLat  float64
	SourceLong float64
}
