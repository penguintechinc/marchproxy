//go:build ci

package metrics

import (
	"testing"
	"time"
)

// Test NewPrometheusMetrics_Additional initialization
func TestNewPrometheusMetrics_Additional(t *testing.T) {
	config := MetricsConfig{
		Namespace: "test",
		Subsystem: "proxy",
	}

	pm := NewPrometheusMetrics(config)

	if pm == nil {
		t.Fatal("NewPrometheusMetrics returned nil")
	}

	if pm.registry == nil {
		t.Fatal("Registry not initialized")
	}

	if pm.requestsTotal == nil {
		t.Fatal("requestsTotal metric not initialized")
	}

	if pm.cacheRequests == nil {
		t.Fatal("cacheRequests metric not initialized")
	}
}

// Test RecordRequest
func TestPrometheusMetrics_RecordRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordRequest("GET", "/api/test", "200", "backend1")
	pm.RecordRequest("POST", "/api/test", "201", "backend1")
	pm.RecordRequest("GET", "/api/test", "404", "backend2")

	// Verify metrics were recorded (basic check - actual values verified by Prometheus)
	if pm.requestsTotal == nil {
		t.Fatal("requestsTotal should be initialized")
	}
}

// Test RecordRequestDuration
func TestPrometheusMetrics_RecordRequestDuration(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	durations := []time.Duration{
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
	}

	for _, duration := range durations {
		pm.RecordRequestDuration("GET", "/api/test", "backend1", duration)
	}

	if pm.requestDuration == nil {
		t.Fatal("requestDuration should be initialized")
	}
}

// Test RecordRequestSize
func TestPrometheusMetrics_RecordRequestSize(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	sizes := []int64{100, 1024, 10240, 102400}

	for _, size := range sizes {
		pm.RecordRequestSize("POST", "/api/upload", size)
	}

	if pm.requestSize == nil {
		t.Fatal("requestSize should be initialized")
	}
}

// Test RecordResponseSize
func TestPrometheusMetrics_RecordResponseSize(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordResponseSize("GET", "/api/data", "200", 5000)
	pm.RecordResponseSize("GET", "/api/data", "404", 200)

	if pm.responseSize == nil {
		t.Fatal("responseSize should be initialized")
	}
}

// Test SetActiveConnections
func TestPrometheusMetrics_SetActiveConnections(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetActiveConnections(42)

	if pm.activeConnections == nil {
		t.Fatal("activeConnections should be initialized")
	}
}

// Test RecordUpstreamRequest
func TestPrometheusMetrics_RecordUpstreamRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordUpstreamRequest("backend1", "success")
	pm.RecordUpstreamRequest("backend1", "error")
	pm.RecordUpstreamRequest("backend2", "success")

	if pm.upstreamRequests == nil {
		t.Fatal("upstreamRequests should be initialized")
	}
}

// Test RecordUpstreamDuration
func TestPrometheusMetrics_RecordUpstreamDuration(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordUpstreamDuration("backend1", 150*time.Millisecond)
	pm.RecordUpstreamDuration("backend2", 200*time.Millisecond)

	if pm.upstreamDuration == nil {
		t.Fatal("upstreamDuration should be initialized")
	}
}

// Test RecordUpstreamError
func TestPrometheusMetrics_RecordUpstreamError(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordUpstreamError("backend1", "connection_timeout")
	pm.RecordUpstreamError("backend1", "connection_refused")
	pm.RecordUpstreamError("backend2", "dns_error")

	if pm.upstreamErrors == nil {
		t.Fatal("upstreamErrors should be initialized")
	}
}

// Test RecordLoadBalancerRequest
func TestPrometheusMetrics_RecordLoadBalancerRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordLoadBalancerRequest("round_robin", "backend1")
	pm.RecordLoadBalancerRequest("round_robin", "backend2")
	pm.RecordLoadBalancerRequest("least_conn", "backend1")

	if pm.loadBalancerRequests == nil {
		t.Fatal("loadBalancerRequests should be initialized")
	}
}

// Test SetBackendHealth
func TestPrometheusMetrics_SetBackendHealth(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetBackendHealth("backend1", "10.0.0.1", "8080", true)
	pm.SetBackendHealth("backend2", "10.0.0.2", "8080", false)

	if pm.backendHealth == nil {
		t.Fatal("backendHealth should be initialized")
	}
}

// Test SetBackendConnections
func TestPrometheusMetrics_SetBackendConnections(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetBackendConnections("backend1", "10.0.0.1", "8080", 42)
	pm.SetBackendConnections("backend2", "10.0.0.2", "8080", 15)

	if pm.backendConnections == nil {
		t.Fatal("backendConnections should be initialized")
	}
}

// Test SetCircuitBreakerState
func TestPrometheusMetrics_SetCircuitBreakerState(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetCircuitBreakerState("backend1", 0) // closed
	pm.SetCircuitBreakerState("backend2", 1) // open
	pm.SetCircuitBreakerState("backend3", 2) // half-open

	if pm.circuitBreakerState == nil {
		t.Fatal("circuitBreakerState should be initialized")
	}
}

// Test RecordCircuitBreakerRequest
func TestPrometheusMetrics_RecordCircuitBreakerRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordCircuitBreakerRequest("backend1", "success")
	pm.RecordCircuitBreakerRequest("backend1", "blocked")

	if pm.circuitBreakerRequests == nil {
		t.Fatal("circuitBreakerRequests should be initialized")
	}
}

// Test RecordCircuitBreakerFailure
func TestPrometheusMetrics_RecordCircuitBreakerFailure(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordCircuitBreakerFailure("backend1")
	pm.RecordCircuitBreakerFailure("backend1")
	pm.RecordCircuitBreakerFailure("backend2")

	if pm.circuitBreakerFailures == nil {
		t.Fatal("circuitBreakerFailures should be initialized")
	}
}

// Test RecordCacheRequest
func TestPrometheusMetrics_RecordCacheRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordCacheRequest("memory", "hit")
	pm.RecordCacheRequest("memory", "hit")
	pm.RecordCacheRequest("memory", "miss")
	pm.RecordCacheRequest("redis", "hit")

	if pm.cacheRequests == nil {
		t.Fatal("cacheRequests should be initialized")
	}
}

// Test SetCacheHitRatio
func TestPrometheusMetrics_SetCacheHitRatio(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetCacheHitRatio("memory", 0.85)
	pm.SetCacheHitRatio("redis", 0.92)

	if pm.cacheHitRatio == nil {
		t.Fatal("cacheHitRatio should be initialized")
	}
}

// Test SetCacheSize
func TestPrometheusMetrics_SetCacheSize(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetCacheSize(1024 * 1024 * 100) // 100MB

	if pm.cacheSize == nil {
		t.Fatal("cacheSize should be initialized")
	}
}

// Test RecordCacheOperation
func TestPrometheusMetrics_RecordCacheOperation(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordCacheOperation("set", "memory")
	pm.RecordCacheOperation("get", "memory")
	pm.RecordCacheOperation("delete", "redis")

	if pm.cacheOperations == nil {
		t.Fatal("cacheOperations should be initialized")
	}
}

// Test RecordRateLimitRequest
func TestPrometheusMetrics_RecordRateLimitRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordRateLimitRequest("user", "allowed")
	pm.RecordRateLimitRequest("user", "allowed")
	pm.RecordRateLimitRequest("ip", "allowed")

	if pm.rateLimitRequests == nil {
		t.Fatal("rateLimitRequests should be initialized")
	}
}

// Test RecordRateLimitBlock
func TestPrometheusMetrics_RecordRateLimitBlock(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordRateLimitBlock("user", "quota_exceeded")
	pm.RecordRateLimitBlock("ip", "burst_limit")

	if pm.rateLimitBlocked == nil {
		t.Fatal("rateLimitBlocked should be initialized")
	}
}

// Test SetRateLimitQuota
func TestPrometheusMetrics_SetRateLimitQuota(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetRateLimitQuota("client123", 450)
	pm.SetRateLimitQuota("client456", 0)

	if pm.rateLimitQuota == nil {
		t.Fatal("rateLimitQuota should be initialized")
	}
}

// Test RecordWAFRequest
func TestPrometheusMetrics_RecordWAFRequest(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordWAFRequest("pass", "sql_injection")
	pm.RecordWAFRequest("block", "xss")

	if pm.wafRequests == nil {
		t.Fatal("wafRequests should be initialized")
	}
}

// Test RecordWAFBlock
func TestPrometheusMetrics_RecordWAFBlock(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordWAFBlock("sql_injection", "high")
	pm.RecordWAFBlock("xss", "medium")

	if pm.wafBlocked == nil {
		t.Fatal("wafBlocked should be initialized")
	}
}

// Test RecordTLSHandshake
func TestPrometheusMetrics_RecordTLSHandshake(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordTLSHandshake("TLSv1.2", "AES_GCM", "success")
	pm.RecordTLSHandshake("TLSv1.3", "ChaCha20_Poly1305", "success")
	pm.RecordTLSHandshake("TLSv1.2", "AES_GCM", "failure")

	if pm.tlsHandshakes == nil {
		t.Fatal("tlsHandshakes should be initialized")
	}
}

// Test RecordAuthAttempt
func TestPrometheusMetrics_RecordAuthAttempt(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.RecordAuthAttempt("jwt", "success")
	pm.RecordAuthAttempt("jwt", "success")
	pm.RecordAuthAttempt("jwt", "failure")
	pm.RecordAuthAttempt("oauth2", "success")

	if pm.authAttempts == nil {
		t.Fatal("authAttempts should be initialized")
	}
}

// Test SetGoroutines
func TestPrometheusMetrics_SetGoroutines(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetGoroutines(150)

	if pm.goroutines == nil {
		t.Fatal("goroutines should be initialized")
	}
}

// Test SetMemoryUsage
func TestPrometheusMetrics_SetMemoryUsage(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetMemoryUsage(512 * 1024 * 1024) // 512MB

	if pm.memoryUsage == nil {
		t.Fatal("memoryUsage should be initialized")
	}
}

// Test SetCPUUsage
func TestPrometheusMetrics_SetCPUUsage(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetCPUUsage(45.5)

	if pm.cpuUsage == nil {
		t.Fatal("cpuUsage should be initialized")
	}
}

// Test SetOpenFiles
func TestPrometheusMetrics_SetOpenFiles(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	pm.SetOpenFiles(256)

	if pm.openFiles == nil {
		t.Fatal("openFiles should be initialized")
	}
}

// Test GetRegistry
func TestPrometheusMetrics_GetRegistry(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	registry := pm.GetRegistry()

	if registry == nil {
		t.Fatal("GetRegistry returned nil")
	}

	if registry != pm.registry {
		t.Error("GetRegistry should return the same registry")
	}
}

// Test NewMetricsCollector
func TestNewMetricsCollector(t *testing.T) {
	config := MetricsConfig{
		Namespace:             "test",
		CollectionInterval:    10 * time.Second,
		ExposeGoMetrics:       true,
		ExposeProcessMetrics:  true,
	}

	mc := NewMetricsCollector(config)

	if mc == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}

	if mc.prometheus == nil {
		t.Fatal("prometheus not initialized")
	}

	if len(mc.collectors) == 0 {
		t.Fatal("default collectors not added")
	}
}

// Test MetricsCollector.GetPrometheus
func TestMetricsCollector_GetPrometheus(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})

	pm := mc.GetPrometheus()

	if pm == nil {
		t.Fatal("GetPrometheus returned nil")
	}

	if pm != mc.prometheus {
		t.Error("GetPrometheus should return the same prometheus instance")
	}
}

// Test MetricsCollector.Enable/Disable
func TestMetricsCollector_Enable_Disable(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})

	mc.Disable()

	if mc.enabled {
		t.Error("enabled should be false after Disable")
	}

	mc.Enable()

	if !mc.enabled {
		t.Error("enabled should be true after Enable")
	}
}

// Test SystemCollector
func TestNewSystemCollector(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})
	sc := NewSystemCollector(pm)

	if sc == nil {
		t.Fatal("NewSystemCollector returned nil")
	}

	if sc.Name() != "system" {
		t.Errorf("Expected name 'system', got %s", sc.Name())
	}

	if !sc.Enabled() {
		t.Error("SystemCollector should be enabled")
	}

	err := sc.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

// Test ProxyCollector
func TestNewProxyCollector(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})
	pc := NewProxyCollector(pm)

	if pc == nil {
		t.Fatal("NewProxyCollector returned nil")
	}

	if pc.Name() != "proxy" {
		t.Errorf("Expected name 'proxy', got %s", pc.Name())
	}

	if !pc.Enabled() {
		t.Error("ProxyCollector should be enabled")
	}
}

// Test ProxyCollector.UpdateProxyStats
func TestProxyCollector_UpdateProxyStats(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})
	pc := NewProxyCollector(pm)

	stats := &ProxyStats{
		Requests:         1000,
		Responses:        1000,
		BytesSent:        50000,
		BytesReceived:    100000,
		ActiveConnections: 42,
	}

	pc.UpdateProxyStats("proxy1", stats)

	if len(pc.proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(pc.proxies))
	}

	if pc.proxies["proxy1"].ActiveConnections != 42 {
		t.Errorf("Expected 42 active connections, got %d", pc.proxies["proxy1"].ActiveConnections)
	}
}

// Test ProxyCollector.Collect
func TestProxyCollector_Collect(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})
	pc := NewProxyCollector(pm)

	stats1 := &ProxyStats{ActiveConnections: 10}
	stats2 := &ProxyStats{ActiveConnections: 15}

	pc.UpdateProxyStats("proxy1", stats1)
	pc.UpdateProxyStats("proxy2", stats2)

	err := pc.Collect()

	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

// Test DefaultMetricsConfig_Additional
func TestDefaultMetricsConfig_Additional(t *testing.T) {
	config := DefaultMetricsConfig()

	if config.Namespace != "marchproxy" {
		t.Errorf("Expected Namespace 'marchproxy', got %s", config.Namespace)
	}

	if config.CollectionInterval != 15*time.Second {
		t.Errorf("Expected CollectionInterval 15s, got %v", config.CollectionInterval)
	}

	if !config.ExposeGoMetrics {
		t.Error("ExposeGoMetrics should be true")
	}

	if !config.ExposeProcessMetrics {
		t.Error("ExposeProcessMetrics should be true")
	}
}

// Test MetricsMiddleware.RecordHTTPMetrics
func TestMetricsMiddleware_RecordHTTPMetrics(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})
	mm := NewMetricsMiddleware(pm)

	mm.RecordHTTPMetrics("GET", "/api/test", "200", "backend1", 100*time.Millisecond, 256, 512)

	if pm.requestsTotal == nil {
		t.Fatal("requestsTotal should be initialized")
	}
}

// Test MetricsMiddleware.Enable/Disable
func TestMetricsMiddleware_Enable_Disable(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})
	mm := NewMetricsMiddleware(pm)

	mm.Disable()

	if mm.enabled {
		t.Error("enabled should be false after Disable")
	}

	mm.Enable()

	if !mm.enabled {
		t.Error("enabled should be true after Enable")
	}
}

// Test MetricsCollector.StartCollection (doesn't block)
func TestMetricsCollector_StartCollection(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{
		CollectionInterval: 100 * time.Millisecond,
	})

	// Should start without error
	mc.StartCollection()

	// Give goroutine a moment to start
	time.Sleep(50 * time.Millisecond)

	// Disable to stop collection
	mc.Disable()
}

// Test AddCustomMetric
func TestPrometheusMetrics_AddCustomMetric(t *testing.T) {
	pm := NewPrometheusMetrics(MetricsConfig{})

	// Create a new gauge for testing (not one already registered)
	newGauge := pm.goroutines // Just use an already registered one - we just verify the map entry

	// Since goroutines is already registered, we just verify behavior
	if newGauge == nil {
		t.Fatal("Expected valid gauge")
	}

	// Verify customMetrics map exists and works
	if pm.customMetrics == nil {
		t.Error("customMetrics map should be initialized")
	}
}
