package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusMetrics struct {
	registry *prometheus.Registry

	// Request metrics
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	requestSize        *prometheus.HistogramVec
	responseSize       *prometheus.HistogramVec

	// Ingress-specific metrics
	virtualHostRequests    *prometheus.CounterVec
	pathRoutingRequests    *prometheus.CounterVec
	sslCertificateExpiry   *prometheus.GaugeVec
	reverseProxyRequests   *prometheus.CounterVec

	// Proxy metrics
	activeConnections  prometheus.Gauge
	upstreamRequests   *prometheus.CounterVec
	upstreamDuration   *prometheus.HistogramVec
	upstreamErrors     *prometheus.CounterVec

	// Load balancing metrics
	loadBalancerRequests *prometheus.CounterVec
	backendHealth       *prometheus.GaugeVec
	backendConnections  *prometheus.GaugeVec

	// Circuit breaker metrics
	circuitBreakerState    *prometheus.GaugeVec
	circuitBreakerRequests *prometheus.CounterVec
	circuitBreakerFailures *prometheus.CounterVec

	// Cache metrics
	cacheRequests      *prometheus.CounterVec
	cacheHitRatio      *prometheus.GaugeVec
	cacheSize          prometheus.Gauge
	cacheOperations    *prometheus.CounterVec

	// Rate limiting metrics
	rateLimitRequests  *prometheus.CounterVec
	rateLimitBlocked   *prometheus.CounterVec
	rateLimitQuota     *prometheus.GaugeVec

	// mTLS and Security metrics
	mtlsHandshakes     *prometheus.CounterVec
	mtlsAuthentication *prometheus.CounterVec
	tlsCertificateInfo *prometheus.GaugeVec
	wafRequests        *prometheus.CounterVec
	wafBlocked         *prometheus.CounterVec
	authAttempts       *prometheus.CounterVec

	// System metrics (initialized for future system metrics collection)
	goroutines         prometheus.Gauge  //nolint:unused
	memoryUsage        prometheus.Gauge  //nolint:unused
	cpuUsage           prometheus.Gauge  //nolint:unused
	openFiles          prometheus.Gauge  //nolint:unused

	// Custom metrics
	customMetrics      map[string]prometheus.Collector
	mutex              sync.RWMutex
}

type MetricsConfig struct {
	Namespace       string
	Subsystem       string
	EnabledMetrics  []string
	CustomLabels    map[string]string
	HistogramBuckets []float64
	CollectionInterval time.Duration
	ExposeGoMetrics    bool
	ExposeProcessMetrics bool
}

type MetricsCollector struct {
	prometheus *PrometheusMetrics
	config     MetricsConfig
	collectors []Collector
	server     *http.Server
	enabled    bool
	mutex      sync.RWMutex
}

type Collector interface {
	Collect() error
	Name() string
	Enabled() bool
}

type SystemCollector struct {
	goroutines *prometheus.GaugeFunc
	memory     *prometheus.GaugeFunc
	cpu        *prometheus.GaugeFunc
	files      *prometheus.GaugeFunc
}

type IngressCollector struct {
	metrics   *PrometheusMetrics
	vhosts    map[string]*VirtualHostStats
	backends  map[string]*BackendStats
	mutex     sync.RWMutex
}

type VirtualHostStats struct {
	Requests         uint64
	Responses        uint64
	Errors           uint64
	BytesSent        uint64
	BytesReceived    uint64
	AverageLatency   time.Duration
	CertExpiry       time.Time
	SSLEnabled       bool
}

type BackendStats struct {
	Requests         uint64
	Errors           uint64
	AverageLatency   time.Duration
	ActiveConnections int64
	Healthy          bool
}

func NewPrometheusMetrics(config MetricsConfig) *PrometheusMetrics {
	registry := prometheus.NewRegistry()

	if config.Namespace == "" {
		config.Namespace = "marchproxy_ingress"
	}

	if len(config.HistogramBuckets) == 0 {
		config.HistogramBuckets = prometheus.DefBuckets
	}

	pm := &PrometheusMetrics{
		registry:      registry,
		customMetrics: make(map[string]prometheus.Collector),
	}

	pm.initializeMetrics(config)
	pm.registerMetrics()

	return pm
}

func (pm *PrometheusMetrics) initializeMetrics(config MetricsConfig) {
	// Request metrics
	pm.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code", "backend", "vhost"},
	)

	pm.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   config.HistogramBuckets,
		},
		[]string{"method", "path", "backend", "vhost"},
	)

	pm.requestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "path", "vhost"},
	)

	pm.responseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "path", "status_code", "vhost"},
	)

	// Ingress-specific metrics
	pm.virtualHostRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "ingress",
			Name:      "vhost_requests_total",
			Help:      "Total requests per virtual host",
		},
		[]string{"vhost", "backend", "status"},
	)

	pm.pathRoutingRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "ingress",
			Name:      "path_routing_requests_total",
			Help:      "Total requests routed by path",
		},
		[]string{"vhost", "path_pattern", "backend"},
	)

	pm.sslCertificateExpiry = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "ingress",
			Name:      "ssl_certificate_expiry_timestamp",
			Help:      "SSL certificate expiry timestamp",
		},
		[]string{"vhost", "issuer", "subject"},
	)

	pm.reverseProxyRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "ingress",
			Name:      "reverse_proxy_requests_total",
			Help:      "Total reverse proxy requests",
		},
		[]string{"source_vhost", "target_backend", "result"},
	)

	// Proxy metrics
	pm.activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "proxy",
			Name:      "active_connections",
			Help:      "Number of active proxy connections",
		},
	)

	pm.upstreamRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "upstream",
			Name:      "requests_total",
			Help:      "Total upstream requests",
		},
		[]string{"backend", "status"},
	)

	pm.upstreamDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: "upstream",
			Name:      "request_duration_seconds",
			Help:      "Upstream request duration in seconds",
			Buckets:   config.HistogramBuckets,
		},
		[]string{"backend"},
	)

	pm.upstreamErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "upstream",
			Name:      "errors_total",
			Help:      "Total upstream errors",
		},
		[]string{"backend", "error_type"},
	)

	// Load balancing metrics
	pm.loadBalancerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "loadbalancer",
			Name:      "requests_total",
			Help:      "Total load balancer requests",
		},
		[]string{"algorithm", "backend"},
	)

	pm.backendHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "backend",
			Name:      "health_status",
			Help:      "Backend health status (1=healthy, 0=unhealthy)",
		},
		[]string{"backend", "host", "port"},
	)

	pm.backendConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "backend",
			Name:      "active_connections",
			Help:      "Number of active connections to backend",
		},
		[]string{"backend", "host", "port"},
	)

	// Circuit breaker metrics
	pm.circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "circuitbreaker",
			Name:      "state",
			Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"backend"},
	)

	pm.circuitBreakerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "circuitbreaker",
			Name:      "requests_total",
			Help:      "Total circuit breaker requests",
		},
		[]string{"backend", "result"},
	)

	pm.circuitBreakerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "circuitbreaker",
			Name:      "failures_total",
			Help:      "Total circuit breaker failures",
		},
		[]string{"backend"},
	)

	// Cache metrics
	pm.cacheRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "cache",
			Name:      "requests_total",
			Help:      "Total cache requests",
		},
		[]string{"store", "result"},
	)

	pm.cacheHitRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "cache",
			Name:      "hit_ratio",
			Help:      "Cache hit ratio",
		},
		[]string{"store"},
	)

	pm.cacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "cache",
			Name:      "size_bytes",
			Help:      "Cache size in bytes",
		},
	)

	pm.cacheOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "cache",
			Name:      "operations_total",
			Help:      "Total cache operations",
		},
		[]string{"operation", "store"},
	)

	// Rate limiting metrics
	pm.rateLimitRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "ratelimit",
			Name:      "requests_total",
			Help:      "Total rate limit requests",
		},
		[]string{"client_type", "result"},
	)

	pm.rateLimitBlocked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "ratelimit",
			Name:      "blocked_total",
			Help:      "Total rate limit blocks",
		},
		[]string{"client_type", "reason"},
	)

	pm.rateLimitQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "ratelimit",
			Name:      "quota_remaining",
			Help:      "Remaining rate limit quota",
		},
		[]string{"client_id"},
	)

	// mTLS and Security metrics
	pm.mtlsHandshakes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "mtls",
			Name:      "handshakes_total",
			Help:      "Total mTLS handshakes",
		},
		[]string{"version", "cipher", "result"},
	)

	pm.mtlsAuthentication = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "mtls",
			Name:      "authentication_total",
			Help:      "Total mTLS authentication attempts",
		},
		[]string{"client_cn", "client_ou", "result"},
	)

	pm.tlsCertificateInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "tls",
			Name:      "certificate_info",
			Help:      "TLS certificate information",
		},
		[]string{"type", "issuer", "subject", "serial"},
	)

	pm.wafRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "waf",
			Name:      "requests_total",
			Help:      "Total WAF requests",
		},
		[]string{"action", "rule_category"},
	)

	pm.wafBlocked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "waf",
			Name:      "blocked_total",
			Help:      "Total WAF blocks",
		},
		[]string{"rule_category", "severity"},
	)

	pm.authAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: "auth",
			Name:      "attempts_total",
			Help:      "Total authentication attempts",
		},
		[]string{"method", "result"},
	)

	// System metrics
	pm.goroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "system",
			Name:      "goroutines",
			Help:      "Number of goroutines",
		},
	)

	pm.memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "system",
			Name:      "memory_bytes",
			Help:      "Memory usage in bytes",
		},
	)

	pm.cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "system",
			Name:      "cpu_usage_percent",
			Help:      "CPU usage percentage",
		},
	)

	pm.openFiles = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: "system",
			Name:      "open_files",
			Help:      "Number of open files",
		},
	)
}

func (pm *PrometheusMetrics) registerMetrics() {
	pm.registry.MustRegister(
		pm.requestsTotal,
		pm.requestDuration,
		pm.requestSize,
		pm.responseSize,
		pm.virtualHostRequests,
		pm.pathRoutingRequests,
		pm.sslCertificateExpiry,
		pm.reverseProxyRequests,
		pm.activeConnections,
		pm.upstreamRequests,
		pm.upstreamDuration,
		pm.upstreamErrors,
		pm.loadBalancerRequests,
		pm.backendHealth,
		pm.backendConnections,
		pm.circuitBreakerState,
		pm.circuitBreakerRequests,
		pm.circuitBreakerFailures,
		pm.cacheRequests,
		pm.cacheHitRatio,
		pm.cacheSize,
		pm.cacheOperations,
		pm.rateLimitRequests,
		pm.rateLimitBlocked,
		pm.rateLimitQuota,
		pm.mtlsHandshakes,
		pm.mtlsAuthentication,
		pm.tlsCertificateInfo,
		pm.wafRequests,
		pm.wafBlocked,
		pm.authAttempts,
		pm.goroutines,
		pm.memoryUsage,
		pm.cpuUsage,
		pm.openFiles,
	)
}

// Ingress-specific metrics methods
func (pm *PrometheusMetrics) RecordVirtualHostRequest(vhost, backend, status string) {
	pm.virtualHostRequests.WithLabelValues(vhost, backend, status).Inc()
}

func (pm *PrometheusMetrics) RecordPathRoutingRequest(vhost, pathPattern, backend string) {
	pm.pathRoutingRequests.WithLabelValues(vhost, pathPattern, backend).Inc()
}

func (pm *PrometheusMetrics) SetSSLCertificateExpiry(vhost, issuer, subject string, expiry time.Time) {
	pm.sslCertificateExpiry.WithLabelValues(vhost, issuer, subject).Set(float64(expiry.Unix()))
}

func (pm *PrometheusMetrics) RecordReverseProxyRequest(sourceVhost, targetBackend, result string) {
	pm.reverseProxyRequests.WithLabelValues(sourceVhost, targetBackend, result).Inc()
}

// mTLS metrics methods
func (pm *PrometheusMetrics) RecordMTLSHandshake(version, cipher, result string) {
	pm.mtlsHandshakes.WithLabelValues(version, cipher, result).Inc()
}

func (pm *PrometheusMetrics) RecordMTLSAuthentication(clientCN, clientOU, result string) {
	pm.mtlsAuthentication.WithLabelValues(clientCN, clientOU, result).Inc()
}

func (pm *PrometheusMetrics) SetTLSCertificateInfo(certType, issuer, subject, serial string, value float64) {
	pm.tlsCertificateInfo.WithLabelValues(certType, issuer, subject, serial).Set(value)
}

// Standard metrics methods (simplified versions of egress implementation)
func (pm *PrometheusMetrics) RecordRequest(method, path, statusCode, backend, vhost string) {
	pm.requestsTotal.WithLabelValues(method, path, statusCode, backend, vhost).Inc()
}

func (pm *PrometheusMetrics) RecordRequestDuration(method, path, backend, vhost string, duration time.Duration) {
	pm.requestDuration.WithLabelValues(method, path, backend, vhost).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordRequestSize(method, path, vhost string, size int64) {
	pm.requestSize.WithLabelValues(method, path, vhost).Observe(float64(size))
}

func (pm *PrometheusMetrics) RecordResponseSize(method, path, statusCode, vhost string, size int64) {
	pm.responseSize.WithLabelValues(method, path, statusCode, vhost).Observe(float64(size))
}

func (pm *PrometheusMetrics) SetActiveConnections(count int) {
	pm.activeConnections.Set(float64(count))
}

func (pm *PrometheusMetrics) RecordUpstreamRequest(backend, status string) {
	pm.upstreamRequests.WithLabelValues(backend, status).Inc()
}

func (pm *PrometheusMetrics) RecordUpstreamDuration(backend string, duration time.Duration) {
	pm.upstreamDuration.WithLabelValues(backend).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordUpstreamError(backend, errorType string) {
	pm.upstreamErrors.WithLabelValues(backend, errorType).Inc()
}

func (pm *PrometheusMetrics) SetBackendHealth(backend, host, port string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	pm.backendHealth.WithLabelValues(backend, host, port).Set(value)
}

func (pm *PrometheusMetrics) RecordAuthAttempt(method, result string) {
	pm.authAttempts.WithLabelValues(method, result).Inc()
}

func (pm *PrometheusMetrics) SetGoroutines(count int) {
	pm.goroutines.Set(float64(count))
}

func (pm *PrometheusMetrics) SetMemoryUsage(bytes int64) {
	pm.memoryUsage.Set(float64(bytes))
}

func (pm *PrometheusMetrics) SetCPUUsage(percent float64) {
	pm.cpuUsage.Set(percent)
}

func (pm *PrometheusMetrics) SetOpenFiles(count int) {
	pm.openFiles.Set(float64(count))
}

func (pm *PrometheusMetrics) AddCustomMetric(name string, collector prometheus.Collector) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.customMetrics[name] = collector
	pm.registry.MustRegister(collector)
}

func (pm *PrometheusMetrics) GetRegistry() *prometheus.Registry {
	return pm.registry
}

func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	mc := &MetricsCollector{
		prometheus: NewPrometheusMetrics(config),
		config:     config,
		collectors: make([]Collector, 0),
		enabled:    true,
	}

	if config.ExposeGoMetrics {
		mc.prometheus.registry.MustRegister(prometheus.NewGoCollector())
	}

	if config.ExposeProcessMetrics {
		mc.prometheus.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	mc.addDefaultCollectors()
	return mc
}

func (mc *MetricsCollector) addDefaultCollectors() {
	systemCollector := NewSystemCollector(mc.prometheus)
	mc.collectors = append(mc.collectors, systemCollector)

	ingressCollector := NewIngressCollector(mc.prometheus)
	mc.collectors = append(mc.collectors, ingressCollector)
}

func (mc *MetricsCollector) StartCollection() {
	if mc.config.CollectionInterval == 0 {
		mc.config.CollectionInterval = 15 * time.Second
	}

	ticker := time.NewTicker(mc.config.CollectionInterval)
	go func() {
		for range ticker.C {
			mc.collectMetrics()
		}
	}()
}

func (mc *MetricsCollector) collectMetrics() {
	if !mc.enabled {
		return
	}

	for _, collector := range mc.collectors {
		if collector.Enabled() {
			collector.Collect()
		}
	}
}

func (mc *MetricsCollector) StartServer(addr string) error {
	handler := promhttp.HandlerFor(mc.prometheus.registry, promhttp.HandlerOpts{})

	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mc.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return mc.server.ListenAndServe()
}

func (mc *MetricsCollector) StopServer(ctx context.Context) error {
	if mc.server != nil {
		return mc.server.Shutdown(ctx)
	}
	return nil
}

func (mc *MetricsCollector) GetPrometheus() *PrometheusMetrics {
	return mc.prometheus
}

func NewSystemCollector(metrics *PrometheusMetrics) *SystemCollector {
	return &SystemCollector{}
}

func (sc *SystemCollector) Collect() error {
	return nil
}

func (sc *SystemCollector) Name() string {
	return "system"
}

func (sc *SystemCollector) Enabled() bool {
	return true
}

func NewIngressCollector(metrics *PrometheusMetrics) *IngressCollector {
	return &IngressCollector{
		metrics:  metrics,
		vhosts:   make(map[string]*VirtualHostStats),
		backends: make(map[string]*BackendStats),
	}
}

func (ic *IngressCollector) Collect() error {
	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	totalConnections := int64(0)
	for _, stats := range ic.backends {
		totalConnections += stats.ActiveConnections
	}

	ic.metrics.SetActiveConnections(int(totalConnections))
	return nil
}

func (ic *IngressCollector) Name() string {
	return "ingress"
}

func (ic *IngressCollector) Enabled() bool {
	return true
}

func (ic *IngressCollector) UpdateVirtualHostStats(vhost string, stats *VirtualHostStats) {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()
	ic.vhosts[vhost] = stats
}

func (ic *IngressCollector) UpdateBackendStats(backend string, stats *BackendStats) {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()
	ic.backends[backend] = stats
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Namespace:          "marchproxy_ingress",
		CollectionInterval: 15 * time.Second,
		HistogramBuckets:   prometheus.DefBuckets,
		ExposeGoMetrics:    true,
		ExposeProcessMetrics: true,
		EnabledMetrics:     []string{"all"},
	}
}

type MetricsMiddleware struct {
	metrics *PrometheusMetrics
	enabled bool
}

func NewMetricsMiddleware(metrics *PrometheusMetrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
		enabled: true,
	}
}

func (mm *MetricsMiddleware) RecordHTTPMetrics(method, path, statusCode, backend, vhost string, duration time.Duration, requestSize, responseSize int64) {
	if !mm.enabled {
		return
	}

	mm.metrics.RecordRequest(method, path, statusCode, backend, vhost)
	mm.metrics.RecordRequestDuration(method, path, backend, vhost, duration)
	mm.metrics.RecordRequestSize(method, path, vhost, requestSize)
	mm.metrics.RecordResponseSize(method, path, statusCode, vhost, responseSize)
}

func (mm *MetricsMiddleware) Enable() {
	mm.enabled = true
}

func (mm *MetricsMiddleware) Disable() {
	mm.enabled = false
}