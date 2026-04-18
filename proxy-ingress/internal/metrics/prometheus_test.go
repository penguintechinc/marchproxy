//go:build ci

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPrometheusMetrics(t *testing.T) {
	config := MetricsConfig{
		Namespace:        "test_ingress",
		CollectionInterval: 10 * time.Second,
	}

	pm := NewPrometheusMetrics(config)
	if pm == nil {
		t.Fatal("expected non-nil PrometheusMetrics")
	}
	if pm.registry == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestNewPrometheusMetricsWithDefaults(t *testing.T) {
	config := MetricsConfig{}
	pm := NewPrometheusMetrics(config)

	if pm == nil {
		t.Fatal("expected non-nil PrometheusMetrics")
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	if config.Namespace != "marchproxy_ingress" {
		t.Errorf("expected Namespace 'marchproxy_ingress', got %q", config.Namespace)
	}
	if config.CollectionInterval != 15*time.Second {
		t.Errorf("expected CollectionInterval 15s, got %v", config.CollectionInterval)
	}
	if !config.ExposeGoMetrics {
		t.Error("expected ExposeGoMetrics to be true")
	}
	if !config.ExposeProcessMetrics {
		t.Error("expected ExposeProcessMetrics to be true")
	}
}

func TestPrometheusMetricsInitialization(t *testing.T) {
	config := MetricsConfig{
		Namespace:           "test_metric",
		HistogramBuckets:    []float64{0.1, 0.5, 1.0, 2.5, 5.0},
		CollectionInterval:  10 * time.Second,
		ExposeGoMetrics:     true,
		ExposeProcessMetrics: true,
	}

	pm := NewPrometheusMetrics(config)

	// Verify metrics are initialized
	if pm.requestsTotal == nil {
		t.Error("expected requestsTotal to be initialized")
	}
	if pm.requestDuration == nil {
		t.Error("expected requestDuration to be initialized")
	}
	if pm.virtualHostRequests == nil {
		t.Error("expected virtualHostRequests to be initialized")
	}
}

func TestRecordRequest(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordRequest("GET", "/api/test", "200", "backend1", "vhost1")
	// Should not panic
}

func TestRecordRequestDuration(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordRequestDuration("POST", "/api/data", "backend1", "vhost1", 250*time.Millisecond)
	// Should not panic
}

func TestRecordRequestSize(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordRequestSize("PUT", "/api/resource", "vhost1", 5120)
	// Should not panic
}

func TestRecordResponseSize(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordResponseSize("DELETE", "/api/resource", "204", "vhost1", 0)
	// Should not panic
}

func TestSetActiveConnections(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetActiveConnections(42)
	// Should not panic
}

func TestRecordUpstreamRequest(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordUpstreamRequest("backend-pool-1", "200")
	pm.RecordUpstreamRequest("backend-pool-1", "500")
	// Should not panic
}

func TestRecordUpstreamDuration(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordUpstreamDuration("backend-pool-1", 150*time.Millisecond)
	pm.RecordUpstreamDuration("backend-pool-2", 300*time.Millisecond)
	// Should not panic
}

func TestRecordUpstreamError(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordUpstreamError("backend-pool-1", "connection_timeout")
	pm.RecordUpstreamError("backend-pool-1", "request_rejected")
	// Should not panic
}

func TestSetBackendHealth(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetBackendHealth("backend1", "10.0.0.1", "8080", true)
	pm.SetBackendHealth("backend2", "10.0.0.2", "8080", false)
	pm.SetBackendHealth("backend3", "10.0.0.3", "8080", true)
	// Should not panic
}

func TestRecordAuthAttempt(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordAuthAttempt("oauth", "success")
	pm.RecordAuthAttempt("jwt", "failure")
	pm.RecordAuthAttempt("basic", "success")
	// Should not panic
}

func TestSetGoroutines(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetGoroutines(128)
	// Should not panic
}

func TestSetMemoryUsage(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetMemoryUsage(536870912) // 512 MB
	// Should not panic
}

func TestSetCPUUsage(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetCPUUsage(45.5)
	pm.SetCPUUsage(0.0)
	pm.SetCPUUsage(100.0)
	// Should not panic
}

func TestSetOpenFiles(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetOpenFiles(256)
	// Should not panic
}

func TestRecordVirtualHostRequest(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordVirtualHostRequest("vhost1", "backend1", "success")
	pm.RecordVirtualHostRequest("vhost2", "backend2", "failure")
	// Should not panic
}

func TestRecordPathRoutingRequest(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordPathRoutingRequest("vhost1", "/api/*", "backend1")
	pm.RecordPathRoutingRequest("vhost1", "/static/*", "cdn-backend")
	// Should not panic
}

func TestSetSSLCertificateExpiry(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	expiry := time.Now().AddDate(1, 0, 0)
	pm.SetSSLCertificateExpiry("vhost1", "Let's Encrypt", "example.com", expiry)
	// Should not panic
}

func TestRecordReverseProxyRequest(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordReverseProxyRequest("api.example.com", "backend-api", "success")
	pm.RecordReverseProxyRequest("static.example.com", "cdn", "success")
	// Should not panic
}

func TestRecordMTLSHandshake(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordMTLSHandshake("TLSv1.3", "TLS_AES_256_GCM_SHA384", "success")
	pm.RecordMTLSHandshake("TLSv1.2", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", "failure")
	// Should not panic
}

func TestRecordMTLSAuthentication(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.RecordMTLSAuthentication("client.example.com", "Engineering", "success")
	pm.RecordMTLSAuthentication("invalid.example.com", "Unknown", "failure")
	// Should not panic
}

func TestSetTLSCertificateInfo(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	pm.SetTLSCertificateInfo("server", "Let's Encrypt", "example.com", "12345678", 1.0)
	pm.SetTLSCertificateInfo("client", "Internal CA", "client.example.com", "87654321", 1.0)
	// Should not panic
}

func TestAddCustomMetric(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())

	customCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "custom_test_counter",
		Help: "Test custom counter",
	})

	pm.AddCustomMetric("test_counter", customCounter)
	// Should not panic
}

func TestGetRegistry(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	registry := pm.GetRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry != pm.registry {
		t.Error("expected returned registry to match internal registry")
	}
}

func TestNewMetricsCollector(t *testing.T) {
	config := DefaultMetricsConfig()
	mc := NewMetricsCollector(config)

	if mc == nil {
		t.Fatal("expected non-nil MetricsCollector")
	}
	if mc.prometheus == nil {
		t.Fatal("expected non-nil prometheus")
	}
	if !mc.enabled {
		t.Error("expected enabled to be true")
	}
}

func TestMetricsCollectorWithGoMetrics(t *testing.T) {
	config := MetricsConfig{
		Namespace:           "test",
		ExposeGoMetrics:     true,
		ExposeProcessMetrics: true,
		CollectionInterval:   10 * time.Second,
	}
	mc := NewMetricsCollector(config)

	if mc == nil {
		t.Fatal("expected non-nil MetricsCollector")
	}
}

func TestMetricsCollectorGetPrometheus(t *testing.T) {
	mc := NewMetricsCollector(DefaultMetricsConfig())
	pm := mc.GetPrometheus()

	if pm == nil {
		t.Fatal("expected non-nil PrometheusMetrics")
	}
}

func TestStartCollection(t *testing.T) {
	mc := NewMetricsCollector(DefaultMetricsConfig())
	mc.StartCollection()
	// Should not panic - collection runs in background
}

func TestCollectMetrics(t *testing.T) {
	mc := NewMetricsCollector(DefaultMetricsConfig())
	mc.collectMetrics()
	// Should not panic
}

func TestStartServerShutdown(t *testing.T) {
	mc := NewMetricsCollector(DefaultMetricsConfig())

	go func() {
		// Don't actually start, would block
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Shutdown without starting should be safe
	err := mc.StopServer(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewSystemCollector(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	sc := NewSystemCollector(pm)

	if sc == nil {
		t.Fatal("expected non-nil SystemCollector")
	}
}

func TestSystemCollectorMethods(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	sc := NewSystemCollector(pm)

	if sc.Name() != "system" {
		t.Errorf("expected Name 'system', got %q", sc.Name())
	}

	if !sc.Enabled() {
		t.Error("expected Enabled to be true")
	}

	err := sc.Collect()
	if err != nil {
		t.Errorf("unexpected error from Collect: %v", err)
	}
}

func TestNewIngressCollector(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	ic := NewIngressCollector(pm)

	if ic == nil {
		t.Fatal("expected non-nil IngressCollector")
	}
	if ic.vhosts == nil {
		t.Fatal("expected non-nil vhosts map")
	}
	if ic.backends == nil {
		t.Fatal("expected non-nil backends map")
	}
}

func TestIngressCollectorMethods(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	ic := NewIngressCollector(pm)

	if ic.Name() != "ingress" {
		t.Errorf("expected Name 'ingress', got %q", ic.Name())
	}

	if !ic.Enabled() {
		t.Error("expected Enabled to be true")
	}

	err := ic.Collect()
	if err != nil {
		t.Errorf("unexpected error from Collect: %v", err)
	}
}

func TestIngressCollectorUpdateVirtualHostStats(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	ic := NewIngressCollector(pm)

	stats := &VirtualHostStats{
		Requests:      100,
		Responses:     98,
		Errors:        2,
		BytesSent:     50000,
		BytesReceived: 10000,
		AverageLatency: 50 * time.Millisecond,
		SSLEnabled:     true,
	}

	ic.UpdateVirtualHostStats("vhost1", stats)
	// Should not panic
}

func TestIngressCollectorUpdateBackendStats(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	ic := NewIngressCollector(pm)

	stats := &BackendStats{
		Requests:         500,
		Errors:           10,
		AverageLatency:   100 * time.Millisecond,
		ActiveConnections: 25,
		Healthy:           true,
	}

	ic.UpdateBackendStats("backend1", stats)
	// Should not panic
}

func TestMetricsConfigDefaults(t *testing.T) {
	config := MetricsConfig{}

	pm := NewPrometheusMetrics(config)
	if pm == nil {
		t.Fatal("expected non-nil PrometheusMetrics")
	}
}

func TestMetricsConfigWithCustomBuckets(t *testing.T) {
	customBuckets := []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0}
	config := MetricsConfig{
		Namespace:        "custom",
		HistogramBuckets: customBuckets,
	}

	pm := NewPrometheusMetrics(config)
	if pm == nil {
		t.Fatal("expected non-nil PrometheusMetrics")
	}
}

func TestNewMetricsMiddleware(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	mm := NewMetricsMiddleware(pm)

	if mm == nil {
		t.Fatal("expected non-nil MetricsMiddleware")
	}
	if !mm.enabled {
		t.Error("expected enabled to be true")
	}
}

func TestMetricsMiddlewareRecordHTTPMetrics(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	mm := NewMetricsMiddleware(pm)

	mm.RecordHTTPMetrics("GET", "/api/test", "200", "backend1", "vhost1", 100*time.Millisecond, 1024, 2048)
	// Should not panic
}

func TestMetricsMiddlewareDisable(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	mm := NewMetricsMiddleware(pm)

	mm.Disable()
	if mm.enabled {
		t.Error("expected enabled to be false after Disable()")
	}

	// Recording while disabled should not panic
	mm.RecordHTTPMetrics("GET", "/api/test", "200", "backend1", "vhost1", 100*time.Millisecond, 1024, 2048)
}

func TestMetricsMiddlewareEnable(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())
	mm := NewMetricsMiddleware(pm)

	mm.Disable()
	mm.Enable()

	if !mm.enabled {
		t.Error("expected enabled to be true after Enable()")
	}
}

func TestMetricsMultipleRecords(t *testing.T) {
	pm := NewPrometheusMetrics(DefaultMetricsConfig())

	// Record multiple metrics
	for i := 0; i < 10; i++ {
		pm.RecordRequest("GET", "/api/test", "200", "backend1", "vhost1")
		pm.RecordRequestDuration("GET", "/api/test", "backend1", "vhost1", time.Duration(i*100)*time.Millisecond)
		pm.SetActiveConnections(i * 10)
	}
	// Should not panic
}

func TestVirtualHostStats(t *testing.T) {
	stats := &VirtualHostStats{
		Requests:      1000,
		Responses:     998,
		Errors:        2,
		BytesSent:     500000,
		BytesReceived: 100000,
		AverageLatency: 75 * time.Millisecond,
		CertExpiry:     time.Now().AddDate(1, 0, 0),
		SSLEnabled:     true,
	}

	if stats.Requests != 1000 {
		t.Errorf("expected Requests 1000, got %d", stats.Requests)
	}
	if !stats.SSLEnabled {
		t.Error("expected SSLEnabled to be true")
	}
}

func TestBackendStats(t *testing.T) {
	stats := &BackendStats{
		Requests:         5000,
		Errors:           50,
		AverageLatency:   150 * time.Millisecond,
		ActiveConnections: 100,
		Healthy:           true,
	}

	if stats.Requests != 5000 {
		t.Errorf("expected Requests 5000, got %d", stats.Requests)
	}
	if stats.ActiveConnections != 100 {
		t.Errorf("expected ActiveConnections 100, got %d", stats.ActiveConnections)
	}
}
