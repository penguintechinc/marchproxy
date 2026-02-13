package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestNewPrometheusMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{})
	if metrics == nil {
		t.Fatal("Expected metrics to be created, got nil")
	}

	if metrics.registry == nil {
		t.Fatal("Expected registry to be initialized")
	}

	// Test that all metric vectors are initialized
	if metrics.requestsTotal == nil {
		t.Error("Expected requestsTotal to be initialized")
	}
	if metrics.requestDuration == nil {
		t.Error("Expected requestDuration to be initialized")
	}
	if metrics.upstreamRequests == nil {
		t.Error("Expected upstreamRequests to be initialized")
	}
}

func TestRecordRequest(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test recording requests
	metrics.RecordRequest("GET", "/api/test", "200", "backend1")
	metrics.RecordRequest("POST", "/api/users", "404", "backend2")
	metrics.RecordRequest("GET", "/api/test", "200", "backend1")

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the requests_total metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_http_requests_total" {
			found = true
			if len(mf.Metric) < 2 {
				t.Error("Expected at least 2 metric entries")
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_http_requests_total metric")
	}
}

func TestRecordRequestDuration(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test recording durations
	metrics.RecordRequestDuration("GET", "/api/test", "backend1", time.Millisecond*100)
	metrics.RecordRequestDuration("POST", "/api/users", "backend2", time.Millisecond*200)

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the request_duration metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_http_request_duration_seconds" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_http_request_duration_seconds metric")
	}
}

func TestSetActiveConnections(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test setting active connections
	metrics.SetActiveConnections(42)

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the active_connections metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_proxy_active_connections" {
			found = true
			if len(mf.Metric) > 0 && *mf.Metric[0].Gauge.Value != 42 {
				t.Errorf("Expected active connections to be 42, got %f", *mf.Metric[0].Gauge.Value)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_proxy_active_connections metric")
	}
}

func TestRecordUpstreamRequest(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test recording upstream requests
	metrics.RecordUpstreamRequest("backend1", "success")
	metrics.RecordUpstreamRequest("backend2", "error")

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the upstream_requests metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_upstream_requests_total" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_upstream_requests_total metric")
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test circuit breaker metrics
	metrics.SetCircuitBreakerState("service1", 0) // closed
	metrics.SetCircuitBreakerState("service2", 1) // open
	metrics.RecordCircuitBreakerRequest("service1", "success")
	metrics.RecordCircuitBreakerFailure("service1")

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check circuit breaker metrics exist
	var foundState, foundRequests, foundFailures bool
	for _, mf := range metricFamilies {
		switch *mf.Name {
		case "proxy_circuitbreaker_state":
			foundState = true
		case "proxy_circuitbreaker_requests_total":
			foundRequests = true
		case "proxy_circuitbreaker_failures_total":
			foundFailures = true
		}
	}

	if !foundState {
		t.Error("Expected to find proxy_circuitbreaker_state metric")
	}
	if !foundRequests {
		t.Error("Expected to find proxy_circuitbreaker_requests_total metric")
	}
	if !foundFailures {
		t.Error("Expected to find proxy_circuitbreaker_failures_total metric")
	}
}

func TestCacheMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test cache metrics
	metrics.RecordCacheRequest("memory", "hit")
	metrics.RecordCacheRequest("memory", "miss")
	metrics.SetCacheHitRatio("memory", 0.75)
	metrics.SetCacheSize(1024)
	metrics.RecordCacheOperation("set", "memory")

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check cache metrics exist
	var foundRequests, foundHitRatio, foundSize, foundOperations bool
	for _, mf := range metricFamilies {
		switch *mf.Name {
		case "proxy_cache_requests_total":
			foundRequests = true
		case "proxy_cache_hit_ratio":
			foundHitRatio = true
		case "proxy_cache_size_bytes":
			foundSize = true
		case "proxy_cache_operations_total":
			foundOperations = true
		}
	}

	if !foundRequests {
		t.Error("Expected to find proxy_cache_requests_total metric")
	}
	if !foundHitRatio {
		t.Error("Expected to find proxy_cache_hit_ratio metric")
	}
	if !foundSize {
		t.Error("Expected to find proxy_cache_size_bytes metric")
	}
	if !foundOperations {
		t.Error("Expected to find proxy_cache_operations_total metric")
	}
}

func TestRateLimitMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Test rate limit metrics
	metrics.RecordRateLimitRequest("client1", "allowed")
	metrics.RecordRateLimitBlock("client1", "quota_exceeded")
	metrics.SetRateLimitQuota("client1", 1000)

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check rate limit metrics exist
	var foundRequests, foundBlocked, foundQuota bool
	for _, mf := range metricFamilies {
		switch *mf.Name {
		case "proxy_ratelimit_requests_total":
			foundRequests = true
		case "proxy_ratelimit_blocked_total":
			foundBlocked = true
		case "proxy_ratelimit_quota_remaining":
			foundQuota = true
		}
	}

	if !foundRequests {
		t.Error("Expected to find proxy_ratelimit_requests_total metric")
	}
	if !foundBlocked {
		t.Error("Expected to find proxy_ratelimit_blocked_total metric")
	}
	if !foundQuota {
		t.Error("Expected to find proxy_ratelimit_quota_remaining metric")
	}
}

func TestHTTPHandler(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Create test server with promhttp handler
	handler := promhttp.HandlerFor(metrics.GetRegistry(), promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request to metrics endpoint
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain content type, got %s", contentType)
	}
}

func TestMetricsCollector(t *testing.T) {
	config := MetricsConfig{
		Namespace:            "proxy",
		CollectionInterval:   time.Second,
		ExposeGoMetrics:      false,
		ExposeProcessMetrics: false,
	}

	collector := NewMetricsCollector(config)
	if collector == nil {
		t.Fatal("Expected collector to be created, got nil")
	}

	// Test that prometheus metrics are accessible
	pm := collector.GetPrometheus()
	if pm == nil {
		t.Error("Expected prometheus metrics to be accessible")
	}

	// Test enable/disable
	collector.Disable()
	collector.Enable()

	// Close collector
	err := collector.Close()
	if err != nil {
		t.Errorf("Failed to close collector: %v", err)
	}
}

func TestMetricsCollectorServer(t *testing.T) {
	config := MetricsConfig{
		Namespace:            "proxy_server_test",
		CollectionInterval:   time.Second,
		ExposeGoMetrics:      false,
		ExposeProcessMetrics: false,
	}

	collector := NewMetricsCollector(config)
	if collector == nil {
		t.Fatal("Expected collector to be created, got nil")
	}

	// Start server in background with error reporting
	serverErr := make(chan error, 1)
	go func() {
		err := collector.StartServer(":0")
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Check for startup errors
	select {
	case err := <-serverErr:
		if err != nil {
			t.Skipf("Server failed to start (expected in some CI environments): %v", err)
		}
	default:
		// Server is running, stop it gracefully
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	collector.StopServer(ctx)
}

func TestAddCustomMetric(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	// Create custom counter
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "custom_test_counter",
		Help: "A test counter",
	})

	// Register custom metric
	metrics.AddCustomMetric("custom_test", counter)

	// Increment custom metric
	counter.Inc()

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the custom metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "custom_test_counter" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find custom_test_counter metric")
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	if config.Namespace != "marchproxy" {
		t.Errorf("Expected namespace 'marchproxy', got %s", config.Namespace)
	}

	if config.CollectionInterval != 15*time.Second {
		t.Errorf("Expected collection interval 15s, got %v", config.CollectionInterval)
	}

	if !config.ExposeGoMetrics {
		t.Error("Expected ExposeGoMetrics to be true")
	}

	if !config.ExposeProcessMetrics {
		t.Error("Expected ExposeProcessMetrics to be true")
	}
}

func TestMetricsMiddleware(t *testing.T) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})
	middleware := NewMetricsMiddleware(metrics)

	// Record HTTP metrics
	middleware.RecordHTTPMetrics("GET", "/api/test", "200", "backend1", time.Millisecond*100, 1024, 2048)

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check that request metrics were recorded
	var foundRequests bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_http_requests_total" {
			foundRequests = true
			break
		}
	}

	if !foundRequests {
		t.Error("Expected to find proxy_http_requests_total metric from middleware")
	}

	// Test disable/enable
	middleware.Disable()
	middleware.Enable()
}

func BenchmarkRecordRequest(b *testing.B) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRequest("GET", "/api/test", "200", "backend1")
	}
}

func BenchmarkRecordRequestDuration(b *testing.B) {
	metrics := NewPrometheusMetrics(MetricsConfig{Namespace: "proxy"})
	duration := time.Millisecond * 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRequestDuration("GET", "/api/test", "backend1", duration)
	}
}
