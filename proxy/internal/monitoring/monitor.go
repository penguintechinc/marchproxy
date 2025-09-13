// Package monitoring provides health checks and Prometheus metrics
package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Monitor handles health checks and metrics exposition
type Monitor struct {
	server     *http.Server
	registry   *prometheus.Registry
	logger     *logrus.Entry
	healthz    HealthChecker
	
	// Metrics
	metrics struct {
		// Connection metrics
		activeConnections   *prometheus.GaugeVec
		totalConnections    *prometheus.CounterVec
		connectionDuration  *prometheus.HistogramVec
		bytesTransferred    *prometheus.CounterVec
		
		// Authentication metrics
		authAttempts        *prometheus.CounterVec
		authFailures        *prometheus.CounterVec
		
		// Performance metrics
		requestLatency      *prometheus.HistogramVec
		throughput          *prometheus.GaugeVec
		
		// System metrics
		cpuUsage            prometheus.Gauge
		memoryUsage         prometheus.Gauge
		goroutines          prometheus.Gauge
		
		// eBPF metrics
		ebpfRules          prometheus.Gauge
		ebpfPacketsFiltered *prometheus.CounterVec
		
		// License metrics
		licenseStatus      prometheus.Gauge
		licenseExpiry      prometheus.Gauge
		
		// Configuration metrics
		configVersion      *prometheus.GaugeVec
		configUpdateTime   prometheus.Gauge
	}
	
	mutex sync.RWMutex
}

// HealthChecker interface for components that can report health status
type HealthChecker interface {
	IsHealthy() bool
	GetStatus() map[string]interface{}
}

// NewMonitor creates a new monitoring instance
func NewMonitor(port int) *Monitor {
	registry := prometheus.NewRegistry()
	
	m := &Monitor{
		registry: registry,
		logger:   logrus.WithField("component", "monitor"),
	}
	
	// Initialize metrics
	m.initMetrics()
	
	// Register metrics with registry
	m.registerMetrics()
	
	// Create HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/healthz", m.healthzHandler).Methods("GET")
	router.HandleFunc("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP).Methods("GET")
	router.HandleFunc("/status", m.statusHandler).Methods("GET")
	
	m.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return m
}

func (m *Monitor) initMetrics() {
	// Connection metrics
	m.metrics.activeConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "marchproxy_active_connections",
			Help: "Number of currently active connections",
		},
		[]string{"protocol", "source", "destination"},
	)
	
	m.metrics.totalConnections = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_total_connections",
			Help: "Total number of connections handled",
		},
		[]string{"protocol", "status"},
	)
	
	m.metrics.connectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "marchproxy_connection_duration_seconds",
			Help:    "Connection duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"protocol", "source", "destination"},
	)
	
	m.metrics.bytesTransferred = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_bytes_transferred_total",
			Help: "Total bytes transferred",
		},
		[]string{"direction", "protocol"},
	)
	
	// Authentication metrics
	m.metrics.authAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_auth_attempts_total",
			Help: "Total authentication attempts",
		},
		[]string{"auth_type", "result"},
	)
	
	m.metrics.authFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_auth_failures_total",
			Help: "Total authentication failures",
		},
		[]string{"auth_type", "reason"},
	)
	
	// Performance metrics
	m.metrics.requestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "marchproxy_request_latency_seconds",
			Help:    "Request latency in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"protocol", "endpoint"},
	)
	
	m.metrics.throughput = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "marchproxy_throughput_bytes_per_second",
			Help: "Current throughput in bytes per second",
		},
		[]string{"direction"},
	)
	
	// System metrics
	m.metrics.cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_cpu_usage_percent",
			Help: "CPU usage percentage",
		},
	)
	
	m.metrics.memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
	)
	
	m.metrics.goroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_goroutines",
			Help: "Number of goroutines",
		},
	)
	
	// eBPF metrics
	m.metrics.ebpfRules = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_ebpf_rules_loaded",
			Help: "Number of eBPF rules loaded",
		},
	)
	
	m.metrics.ebpfPacketsFiltered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "marchproxy_ebpf_packets_filtered_total",
			Help: "Total packets filtered by eBPF programs",
		},
		[]string{"action", "rule_id"},
	)
	
	// License metrics
	m.metrics.licenseStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_license_status",
			Help: "License status (1 = valid, 0 = invalid/expired)",
		},
	)
	
	m.metrics.licenseExpiry = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_license_expiry_timestamp",
			Help: "License expiry timestamp",
		},
	)
	
	// Configuration metrics
	m.metrics.configVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "marchproxy_config_version",
			Help: "Current configuration version",
		},
		[]string{"cluster_id", "version_hash"},
	)
	
	m.metrics.configUpdateTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "marchproxy_config_last_update_timestamp",
			Help: "Timestamp of last configuration update",
		},
	)
}

func (m *Monitor) registerMetrics() {
	// Register all metrics with the registry
	m.registry.MustRegister(
		m.metrics.activeConnections,
		m.metrics.totalConnections,
		m.metrics.connectionDuration,
		m.metrics.bytesTransferred,
		m.metrics.authAttempts,
		m.metrics.authFailures,
		m.metrics.requestLatency,
		m.metrics.throughput,
		m.metrics.cpuUsage,
		m.metrics.memoryUsage,
		m.metrics.goroutines,
		m.metrics.ebpfRules,
		m.metrics.ebpfPacketsFiltered,
		m.metrics.licenseStatus,
		m.metrics.licenseExpiry,
		m.metrics.configVersion,
		m.metrics.configUpdateTime,
	)
	
	// Register Go runtime metrics
	m.registry.MustRegister(prometheus.NewGoCollector())
	m.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
}

// Start begins the monitoring server
func (m *Monitor) Start() error {
	m.logger.Info("Starting monitoring server", "addr", m.server.Addr)
	
	// Start system metrics collection
	go m.collectSystemMetrics()
	
	// Start HTTP server
	if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start monitoring server: %w", err)
	}
	
	return nil
}

// Shutdown gracefully stops the monitoring server
func (m *Monitor) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down monitoring server")
	return m.server.Shutdown(ctx)
}

// SetHealthChecker sets the health checker component
func (m *Monitor) SetHealthChecker(hc HealthChecker) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.healthz = hc
}

// healthzHandler handles health check requests
func (m *Monitor) healthzHandler(w http.ResponseWriter, r *http.Request) {
	m.mutex.RLock()
	healthz := m.healthz
	m.mutex.RUnlock()
	
	if healthz == nil {
		http.Error(w, "Health checker not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if healthz.IsHealthy() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
	}
}

// statusHandler provides detailed status information
func (m *Monitor) statusHandler(w http.ResponseWriter, r *http.Request) {
	m.mutex.RLock()
	healthz := m.healthz
	m.mutex.RUnlock()
	
	status := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime":    time.Since(time.Now()).String(), // This would be calculated properly
		"version":   "1.0.0",
	}
	
	if healthz != nil {
		status["healthy"] = healthz.IsHealthy()
		status["details"] = healthz.GetStatus()
	} else {
		status["healthy"] = false
		status["details"] = map[string]interface{}{
			"error": "Health checker not initialized",
		}
	}
	
	// Add runtime information
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	status["runtime"] = map[string]interface{}{
		"goroutines":     runtime.NumGoroutine(),
		"memory_alloc":   m1.Alloc,
		"memory_total":   m1.TotalAlloc,
		"memory_sys":     m1.Sys,
		"gc_runs":        m1.NumGC,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// Simple JSON encoding (in production, use proper JSON marshaling)
	fmt.Fprintf(w, `{
		"timestamp": "%s",
		"healthy": %t,
		"version": "%s",
		"goroutines": %d,
		"memory_alloc": %d
	}`, 
		status["timestamp"], 
		status["healthy"], 
		status["version"],
		runtime.NumGoroutine(),
		m1.Alloc,
	)
}

// collectSystemMetrics collects system-level metrics periodically
func (m *Monitor) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// Update goroutine count
		m.metrics.goroutines.Set(float64(runtime.NumGoroutine()))
		
		// Update memory usage
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		m.metrics.memoryUsage.Set(float64(memStats.Alloc))
		
		// CPU usage would require additional system calls or libraries
		// For now, we'll skip this and implement it later with proper CPU monitoring
	}
}

// Metrics access methods for other components

func (m *Monitor) RecordConnection(protocol, source, dest string) {
	m.metrics.activeConnections.WithLabelValues(protocol, source, dest).Inc()
	m.metrics.totalConnections.WithLabelValues(protocol, "established").Inc()
}

func (m *Monitor) RecordConnectionClosed(protocol, source, dest string, duration time.Duration) {
	m.metrics.activeConnections.WithLabelValues(protocol, source, dest).Dec()
	m.metrics.connectionDuration.WithLabelValues(protocol, source, dest).Observe(duration.Seconds())
}

func (m *Monitor) RecordBytesTransferred(direction, protocol string, bytes int64) {
	m.metrics.bytesTransferred.WithLabelValues(direction, protocol).Add(float64(bytes))
}

func (m *Monitor) RecordAuthAttempt(authType, result string) {
	m.metrics.authAttempts.WithLabelValues(authType, result).Inc()
	if result != "success" {
		m.metrics.authFailures.WithLabelValues(authType, result).Inc()
	}
}

func (m *Monitor) RecordRequestLatency(protocol, endpoint string, latency time.Duration) {
	m.metrics.requestLatency.WithLabelValues(protocol, endpoint).Observe(latency.Seconds())
}

func (m *Monitor) UpdateThroughput(direction string, bytesPerSecond float64) {
	m.metrics.throughput.WithLabelValues(direction).Set(bytesPerSecond)
}

func (m *Monitor) UpdateEBPFRules(count int) {
	m.metrics.ebpfRules.Set(float64(count))
}

func (m *Monitor) RecordEBPFPacketFiltered(action, ruleID string) {
	m.metrics.ebpfPacketsFiltered.WithLabelValues(action, ruleID).Inc()
}

func (m *Monitor) UpdateLicenseStatus(valid bool, expiryTime time.Time) {
	if valid {
		m.metrics.licenseStatus.Set(1)
	} else {
		m.metrics.licenseStatus.Set(0)
	}
	
	if !expiryTime.IsZero() {
		m.metrics.licenseExpiry.Set(float64(expiryTime.Unix()))
	}
}

func (m *Monitor) UpdateConfigVersion(clusterID, versionHash string) {
	m.metrics.configVersion.WithLabelValues(clusterID, versionHash).Set(1)
	m.metrics.configUpdateTime.SetToCurrentTime()
}