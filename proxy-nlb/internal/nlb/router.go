package nlb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	routedConnections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_routed_connections_total",
			Help: "Total number of connections routed by protocol",
		},
		[]string{"protocol", "module"},
	)

	routingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_routing_errors_total",
			Help: "Total number of routing errors",
		},
		[]string{"protocol", "error_type"},
	)

	activeConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nlb_active_connections",
			Help: "Number of active connections per module",
		},
		[]string{"protocol", "module"},
	)
)

// ModuleEndpoint represents a backend module container
type ModuleEndpoint struct {
	Name         string
	Protocol     Protocol
	Address      string
	GRPCPort     int
	Healthy      bool
	ActiveConns  int
	MaxConns     int
	Version      string // For blue/green deployments
	Weight       int    // For weighted routing
	LastHealthy  time.Time
	mu           sync.RWMutex
}

// IsHealthy checks if the endpoint is healthy
func (m *ModuleEndpoint) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Healthy
}

// SetHealthy sets the health status
func (m *ModuleEndpoint) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Healthy = healthy
	if healthy {
		m.LastHealthy = time.Now()
	}
}

// IncrementConns increments active connection count
func (m *ModuleEndpoint) IncrementConns() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ActiveConns >= m.MaxConns {
		return errors.New("max connections reached")
	}
	m.ActiveConns++
	activeConnections.WithLabelValues(m.Protocol.String(), m.Name).Set(float64(m.ActiveConns))
	return nil
}

// DecrementConns decrements active connection count
func (m *ModuleEndpoint) DecrementConns() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ActiveConns > 0 {
		m.ActiveConns--
	}
	activeConnections.WithLabelValues(m.Protocol.String(), m.Name).Set(float64(m.ActiveConns))
}

// GetActiveConns returns current active connections
func (m *ModuleEndpoint) GetActiveConns() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ActiveConns
}

// Router handles traffic routing to appropriate module containers
type Router struct {
	endpoints map[Protocol][]*ModuleEndpoint
	mu        sync.RWMutex
	logger    *logrus.Logger
	inspector *ProtocolInspector
}

// NewRouter creates a new traffic router
func NewRouter(logger *logrus.Logger) *Router {
	return &Router{
		endpoints: make(map[Protocol][]*ModuleEndpoint),
		logger:    logger,
		inspector: NewProtocolInspector(),
	}
}

// RegisterModule registers a module endpoint for a specific protocol
func (r *Router) RegisterModule(module *ModuleEndpoint) error {
	if module == nil {
		return errors.New("module cannot be nil")
	}

	if module.Protocol == ProtocolUnknown {
		return errors.New("invalid protocol")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.endpoints[module.Protocol] == nil {
		r.endpoints[module.Protocol] = make([]*ModuleEndpoint, 0)
	}

	// Check for duplicate
	for _, existing := range r.endpoints[module.Protocol] {
		if existing.Name == module.Name {
			return fmt.Errorf("module %s already registered for protocol %s", module.Name, module.Protocol)
		}
	}

	r.endpoints[module.Protocol] = append(r.endpoints[module.Protocol], module)

	r.logger.WithFields(logrus.Fields{
		"module":   module.Name,
		"protocol": module.Protocol.String(),
		"address":  module.Address,
		"port":     module.GRPCPort,
	}).Info("Module registered")

	return nil
}

// UnregisterModule removes a module endpoint
func (r *Router) UnregisterModule(protocol Protocol, moduleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	modules, exists := r.endpoints[protocol]
	if !exists {
		return fmt.Errorf("no modules registered for protocol %s", protocol)
	}

	for i, module := range modules {
		if module.Name == moduleName {
			// Remove from slice
			r.endpoints[protocol] = append(modules[:i], modules[i+1:]...)
			r.logger.WithFields(logrus.Fields{
				"module":   moduleName,
				"protocol": protocol.String(),
			}).Info("Module unregistered")
			return nil
		}
	}

	return fmt.Errorf("module %s not found for protocol %s", moduleName, protocol)
}

// RouteConnection routes a connection to the appropriate module
func (r *Router) RouteConnection(ctx context.Context, data []byte) (*ModuleEndpoint, error) {
	// Detect protocol
	protocol, err := r.inspector.InspectProtocol(data)
	if err != nil {
		routingErrors.WithLabelValues("unknown", "detection_error").Inc()
		return nil, fmt.Errorf("protocol detection failed: %w", err)
	}

	if protocol == ProtocolUnknown {
		routingErrors.WithLabelValues("unknown", "unknown_protocol").Inc()
		return nil, errors.New("unknown protocol")
	}

	// Get available modules for protocol
	module, err := r.selectModule(protocol)
	if err != nil {
		routingErrors.WithLabelValues(protocol.String(), "no_module").Inc()
		return nil, err
	}

	// Increment connection count
	if err := module.IncrementConns(); err != nil {
		routingErrors.WithLabelValues(protocol.String(), "max_connections").Inc()
		return nil, fmt.Errorf("module capacity exceeded: %w", err)
	}

	routedConnections.WithLabelValues(protocol.String(), module.Name).Inc()

	r.logger.WithFields(logrus.Fields{
		"protocol": protocol.String(),
		"module":   module.Name,
		"address":  module.Address,
	}).Debug("Connection routed")

	return module, nil
}

// selectModule selects the best module for the protocol using least connections algorithm
func (r *Router) selectModule(protocol Protocol) (*ModuleEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules, exists := r.endpoints[protocol]
	if !exists || len(modules) == 0 {
		return nil, fmt.Errorf("no modules available for protocol %s", protocol)
	}

	// Filter healthy modules
	var healthyModules []*ModuleEndpoint
	for _, module := range modules {
		if module.IsHealthy() {
			healthyModules = append(healthyModules, module)
		}
	}

	if len(healthyModules) == 0 {
		return nil, fmt.Errorf("no healthy modules available for protocol %s", protocol)
	}

	// Select module with least connections
	var selected *ModuleEndpoint
	minConns := int(^uint(0) >> 1) // Max int

	for _, module := range healthyModules {
		conns := module.GetActiveConns()
		if conns < minConns {
			minConns = conns
			selected = module
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("failed to select module for protocol %s", protocol)
	}

	return selected, nil
}

// GetModules returns all registered modules for a protocol
func (r *Router) GetModules(protocol Protocol) []*ModuleEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := r.endpoints[protocol]
	result := make([]*ModuleEndpoint, len(modules))
	copy(result, modules)
	return result
}

// GetAllModules returns all registered modules
func (r *Router) GetAllModules() map[Protocol][]*ModuleEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[Protocol][]*ModuleEndpoint)
	for protocol, modules := range r.endpoints {
		result[protocol] = make([]*ModuleEndpoint, len(modules))
		copy(result[protocol], modules)
	}
	return result
}

// GetStats returns routing statistics
func (r *Router) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})
	protocolStats := make(map[string]interface{})

	for protocol, modules := range r.endpoints {
		moduleStats := make([]map[string]interface{}, 0)
		totalConns := 0
		healthyCount := 0

		for _, module := range modules {
			conns := module.GetActiveConns()
			totalConns += conns

			if module.IsHealthy() {
				healthyCount++
			}

			moduleStats = append(moduleStats, map[string]interface{}{
				"name":         module.Name,
				"address":      module.Address,
				"healthy":      module.IsHealthy(),
				"active_conns": conns,
				"max_conns":    module.MaxConns,
				"version":      module.Version,
				"weight":       module.Weight,
			})
		}

		protocolStats[protocol.String()] = map[string]interface{}{
			"total_modules":  len(modules),
			"healthy_modules": healthyCount,
			"total_connections": totalConns,
			"modules":        moduleStats,
		}
	}

	stats["protocols"] = protocolStats
	stats["total_protocols"] = len(r.endpoints)

	return stats
}
