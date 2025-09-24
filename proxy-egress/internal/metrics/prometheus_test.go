package metrics

import (
	"testing"
	"time"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPrometheusMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics()
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

func TestIncrementRequestsTotal(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Test incrementing requests
	metrics.IncrementRequestsTotal("GET", "200")
	metrics.IncrementRequestsTotal("POST", "404")
	metrics.IncrementRequestsTotal("GET", "200")

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the requests_total metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_requests_total" {
			found = true
			if len(mf.Metric) < 2 {
				t.Error("Expected at least 2 metric entries")
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_requests_total metric")
	}
}

func TestRecordRequestDuration(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Test recording durations
	metrics.RecordRequestDuration("GET", time.Millisecond*100)
	metrics.RecordRequestDuration("POST", time.Millisecond*200)

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the request_duration metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "proxy_request_duration_seconds" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_request_duration_seconds metric")
	}
}

func TestSetActiveConnections(t *testing.T) {
	metrics := NewPrometheusMetrics()

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
		if *mf.Name == "proxy_active_connections" {
			found = true
			if len(mf.Metric) > 0 && *mf.Metric[0].Gauge.Value != 42 {
				t.Errorf("Expected active connections to be 42, got %f", *mf.Metric[0].Gauge.Value)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find proxy_active_connections metric")
	}
}

func TestIncrementUpstreamRequests(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Test incrementing upstream requests
	metrics.IncrementUpstreamRequests("backend1", "success")
	metrics.IncrementUpstreamRequests("backend2", "error")

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
	metrics := NewPrometheusMetrics()

	// Test circuit breaker metrics
	metrics.SetCircuitBreakerState("service1", 0) // closed
	metrics.SetCircuitBreakerState("service2", 1) // open
	metrics.IncrementCircuitBreakerRequests("service1", "success")
	metrics.IncrementCircuitBreakerFailures("service1")

	// Gather metrics to verify
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Check circuit breaker metrics exist
	var foundState, foundRequests, foundFailures bool
	for _, mf := range metricFamilies {
		switch *mf.Name {
		case "proxy_circuit_breaker_state":
			foundState = true
		case "proxy_circuit_breaker_requests_total":
			foundRequests = true
		case "proxy_circuit_breaker_failures_total":
			foundFailures = true
		}
	}

	if !foundState {
		t.Error("Expected to find proxy_circuit_breaker_state metric")
	}
	if !foundRequests {
		t.Error("Expected to find proxy_circuit_breaker_requests_total metric")
	}
	if !foundFailures {
		t.Error("Expected to find proxy_circuit_breaker_failures_total metric")
	}
}

func TestCacheMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Test cache metrics
	metrics.IncrementCacheRequests("hit")
	metrics.IncrementCacheRequests("miss")
	metrics.SetCacheHitRatio("default", 0.75)
	metrics.SetCacheSize(1024)
	metrics.IncrementCacheOperations("set")

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
	metrics := NewPrometheusMetrics()

	// Test rate limit metrics
	metrics.IncrementRateLimitRequests("client1", "allowed")
	metrics.IncrementRateLimitBlocked("client1", "quota_exceeded")
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
		case "proxy_rate_limit_requests_total":
			foundRequests = true
		case "proxy_rate_limit_blocked_total":
			foundBlocked = true
		case "proxy_rate_limit_quota":
			foundQuota = true
		}
	}

	if !foundRequests {
		t.Error("Expected to find proxy_rate_limit_requests_total metric")
	}
	if !foundBlocked {
		t.Error("Expected to find proxy_rate_limit_blocked_total metric")
	}
	if !foundQuota {
		t.Error("Expected to find proxy_rate_limit_quota metric")
	}
}

func TestHTTPHandler(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Create test server
	handler := metrics.Handler()
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

func TestStartMetricsServer(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Start metrics server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := metrics.StartServer(ctx, ":0") // Use port 0 for random available port
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Metrics server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop server
	cancel()

	// Give server time to stop
	time.Sleep(100 * time.Millisecond)
}

func TestRegisterCustomMetric(t *testing.T) {
	metrics := NewPrometheusMetrics()

	// Create custom counter
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "custom_test_counter",
		Help: "A test counter",
	})

	// Register custom metric
	err := metrics.RegisterCustomMetric(counter)
	if err != nil {
		t.Errorf("Failed to register custom metric: %v", err)
	}

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

func BenchmarkIncrementRequestsTotal(b *testing.B) {
	metrics := NewPrometheusMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncrementRequestsTotal("GET", "200")
	}
}

func BenchmarkRecordRequestDuration(b *testing.B) {
	metrics := NewPrometheusMetrics()
	duration := time.Millisecond * 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRequestDuration("GET", duration)
	}
}