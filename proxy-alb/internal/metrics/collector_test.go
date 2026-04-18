package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/logging"
)

func TestNewCollector(t *testing.T) {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	collector := NewCollector("localhost:9901", logger)
	if collector == nil {
		t.Fatal("collector should not be nil")
	}
	if collector.adminAddr != "localhost:9901" {
		t.Errorf("expected adminAddr 'localhost:9901', got '%s'", collector.adminAddr)
	}
	if collector.logger == nil {
		t.Fatal("logger should be set")
	}
	if collector.cacheTimeout != 5*time.Second {
		t.Errorf("expected cacheTimeout 5s, got %v", collector.cacheTimeout)
	}
}

func TestNewCollectorWithNilLogger(t *testing.T) {
	collector := NewCollector("localhost:9901", nil)
	if collector == nil {
		t.Fatal("collector should not be nil")
	}
	if collector.logger == nil {
		t.Fatal("logger should be created when nil is passed")
	}
}

func TestMetricsStructure(t *testing.T) {
	metrics := &Metrics{
		Timestamp:         1234567890,
		TotalConnections:  100,
		ActiveConnections: 50,
		TotalRequests:     1000,
		RequestsPerSecond: 10,
		Latency: LatencyMetrics{
			P50Ms: 10.5,
			P90Ms: 50.2,
			P95Ms: 75.1,
			P99Ms: 100.0,
			AvgMs: 30.5,
		},
		StatusCodes: map[string]int64{
			"200": 950,
			"500": 50,
		},
		Routes: map[string]RouteMetrics{
			"/api/users": {
				Requests:     500,
				Errors:       10,
				AvgLatencyMs: 25.0,
			},
			"/api/posts": {
				Requests:     450,
				Errors:       5,
				AvgLatencyMs: 35.5,
			},
		},
	}

	if metrics.Timestamp != 1234567890 {
		t.Errorf("expected timestamp 1234567890, got %d", metrics.Timestamp)
	}
	if metrics.TotalRequests != 1000 {
		t.Errorf("expected TotalRequests 1000, got %d", metrics.TotalRequests)
	}
	if metrics.Latency.P99Ms != 100.0 {
		t.Errorf("expected P99Ms 100.0, got %f", metrics.Latency.P99Ms)
	}
	if metrics.StatusCodes["200"] != 950 {
		t.Errorf("expected 200 count 950, got %d", metrics.StatusCodes["200"])
	}
	if len(metrics.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(metrics.Routes))
	}
}

func TestRouteMetrics(t *testing.T) {
	route := RouteMetrics{
		Requests:     100,
		Errors:       5,
		AvgLatencyMs: 25.5,
	}

	if route.Requests != 100 {
		t.Errorf("expected Requests 100, got %d", route.Requests)
	}
	if route.Errors != 5 {
		t.Errorf("expected Errors 5, got %d", route.Errors)
	}
	if route.AvgLatencyMs != 25.5 {
		t.Errorf("expected AvgLatencyMs 25.5, got %f", route.AvgLatencyMs)
	}
}

func TestLatencyMetrics(t *testing.T) {
	latency := LatencyMetrics{
		P50Ms: 10.0,
		P90Ms: 50.0,
		P95Ms: 75.0,
		P99Ms: 100.0,
		AvgMs: 30.0,
	}

	if latency.P50Ms != 10.0 {
		t.Errorf("expected P50Ms 10.0, got %f", latency.P50Ms)
	}
	if latency.P99Ms != 100.0 {
		t.Errorf("expected P99Ms 100.0, got %f", latency.P99Ms)
	}
}

func TestCollectorHTTPClient(t *testing.T) {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	collector := NewCollector("localhost:9901", logger)
	if collector.httpClient == nil {
		t.Fatal("httpClient should be initialized")
	}
	if collector.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", collector.httpClient.Timeout)
	}
}

func TestCollectorCachingDefaults(t *testing.T) {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	collector := NewCollector("localhost:9901", logger)

	if collector.cachedMetrics != nil {
		t.Fatal("initial cachedMetrics should be nil")
	}
	if !collector.lastCollection.IsZero() {
		t.Fatal("initial lastCollection should be zero")
	}
}

func TestMetricsEmpty(t *testing.T) {
	metrics := &Metrics{
		StatusCodes: make(map[string]int64),
		Routes:      make(map[string]RouteMetrics),
	}

	if len(metrics.StatusCodes) != 0 {
		t.Errorf("expected empty StatusCodes, got %d", len(metrics.StatusCodes))
	}
	if len(metrics.Routes) != 0 {
		t.Errorf("expected empty Routes, got %d", len(metrics.Routes))
	}
}

func TestMetricsWithManyRoutes(t *testing.T) {
	metrics := &Metrics{
		Routes: make(map[string]RouteMetrics),
	}

	// Add 10 routes with clear naming
	for i := 1; i <= 10; i++ {
		var route string
		if i < 10 {
			route = "/api/route0" + string(rune('0'+i))
		} else {
			route = "/api/route10"
		}
		metrics.Routes[route] = RouteMetrics{
			Requests:     int64(i * 10),
			Errors:       int64(i),
			AvgLatencyMs: float64(i) * 1.5,
		}
	}

	if len(metrics.Routes) != 10 {
		t.Errorf("expected 10 routes, got %d", len(metrics.Routes))
	}

	// Verify specific route
	if metrics.Routes["/api/route01"].Requests != 10 {
		t.Errorf("expected 10 requests for /api/route01, got %d",
			metrics.Routes["/api/route01"].Requests)
	}
}

func TestCollectorAddrVariations(t *testing.T) {
	addrs := []string{
		"localhost:9901",
		"127.0.0.1:9901",
		"envoy-admin:9901",
		"10.0.0.1:9901",
		"unix:///tmp/envoy.sock",
	}

	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	for _, addr := range addrs {
		collector := NewCollector(addr, logger)
		if collector.adminAddr != addr {
			t.Errorf("expected adminAddr '%s', got '%s'", addr, collector.adminAddr)
		}
	}
}

func TestGetMetrics_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stats" && r.URL.RawQuery == "format=json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"downstream_cx_total": 100,
				"downstream_cx_active": 50,
				"downstream_rq_total": 1000,
				"downstream_rq_time.p50": 10.5,
				"downstream_rq_time.p90": 50.2,
				"downstream_rq_time.p95": 75.3,
				"downstream_rq_time.p99": 99.1,
				"downstream_rq_time.avg": 25.5,
				"downstream_rq_200": 950,
				"downstream_rq_404": 40,
				"downstream_rq_500": 10
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger, _ := logging.NewLogrusAdapter("test")
	collector := NewCollector(server.Listener.Addr().String(), logger)
	metricsData, err := collector.GetMetrics()

	if err != nil {
		t.Fatalf("GetMetrics() unexpected error: %v", err)
	}

	if metricsData == nil {
		t.Fatal("expected non-nil metrics")
	}

	if metricsData.TotalConnections != 100 {
		t.Errorf("expected TotalConnections=100, got %d", metricsData.TotalConnections)
	}

	if metricsData.ActiveConnections != 50 {
		t.Errorf("expected ActiveConnections=50, got %d", metricsData.ActiveConnections)
	}

	if metricsData.TotalRequests != 1000 {
		t.Errorf("expected TotalRequests=1000, got %d", metricsData.TotalRequests)
	}

	if metricsData.Latency.P50Ms != 10.5 {
		t.Errorf("expected P50=10.5, got %f", metricsData.Latency.P50Ms)
	}

	if metricsData.Latency.P90Ms != 50.2 {
		t.Errorf("expected P90=50.2, got %f", metricsData.Latency.P90Ms)
	}

	if metricsData.StatusCodes["200"] != 950 {
		t.Errorf("expected 950 status 200 responses, got %d", metricsData.StatusCodes["200"])
	}
}

func TestGetMetrics_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	logger, _ := logging.NewLogrusAdapter("test")
	collector := NewCollector(server.Listener.Addr().String(), logger)
	_, err := collector.GetMetrics()

	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestGetMetrics_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/stats" && r.URL.RawQuery == "format=json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"downstream_cx_total": 100,
				"downstream_cx_active": 50,
				"downstream_rq_total": 1000
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger, _ := logging.NewLogrusAdapter("test")
	collector := NewCollector(server.Listener.Addr().String(), logger)

	// First call - should fetch
	metricsData1, err := collector.GetMetrics()
	if err != nil {
		t.Fatalf("first GetMetrics() failed: %v", err)
	}

	firstCallCount := callCount

	// Second call within cache window - should use cache
	metricsData2, err := collector.GetMetrics()
	if err != nil {
		t.Fatalf("second GetMetrics() failed: %v", err)
	}

	// Should be the same object (cached)
	if metricsData1 != metricsData2 {
		t.Error("expected cached metrics to be identical")
	}

	// Should not have made additional HTTP calls
	if callCount > firstCallCount {
		t.Errorf("expected 1 HTTP call, but got %d total", callCount)
	}
}

func TestReset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stats" && r.URL.RawQuery == "format=json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"downstream_cx_total": 100,
				"downstream_cx_active": 50,
				"downstream_rq_total": 1000
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger, _ := logging.NewLogrusAdapter("test")
	collector := NewCollector(server.Listener.Addr().String(), logger)

	// Get metrics
	_, err := collector.GetMetrics()
	if err != nil {
		t.Fatalf("GetMetrics() failed: %v", err)
	}

	// Reset cache
	collector.Reset()

	// Get metrics again - should not be cached
	metricsData, err := collector.GetMetrics()
	if err != nil {
		t.Fatalf("GetMetrics() after reset failed: %v", err)
	}

	if metricsData == nil {
		t.Fatal("expected non-nil metrics after reset")
	}
}

func TestGetMetrics_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stats" && r.URL.RawQuery == "format=json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger, _ := logging.NewLogrusAdapter("test")
	collector := NewCollector(server.Listener.Addr().String(), logger)
	_, err := collector.GetMetrics()

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
