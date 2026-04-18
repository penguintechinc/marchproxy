package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Create metrics once to avoid duplicate registration
var m = createMetricsOnce()

func createMetricsOnce() *metrics.Metrics {
	return metrics.New()
}

func TestMetricsNew(t *testing.T) {
	// Verify metrics object was created (reuse the package-level one)
	if m == nil {
		t.Fatal("expected metrics to be created, got nil")
	}
	if m.RequestsTotal == nil {
		t.Fatal("expected RequestsTotal to be created")
	}
	if m.RequestDuration == nil {
		t.Fatal("expected RequestDuration to be created")
	}
	if m.ActiveRequests == nil {
		t.Fatal("expected ActiveRequests to be created")
	}
	if m.TokensProcessed == nil {
		t.Fatal("expected TokensProcessed to be created")
	}
	if m.ProviderErrors == nil {
		t.Fatal("expected ProviderErrors to be created")
	}
}

func TestRecordRequest(t *testing.T) {
	m.RecordRequest("openai", "gpt-4", "success", 0.5, 100, 50)
	// Should not panic
}

func TestRecordRequestWithZeroTokens(t *testing.T) {
	m.RecordRequest("openai", "gpt-4-test", "success", 0.5, 0, 0)
	// Should not panic
}

func TestRecordError(t *testing.T) {
	m.RecordError("openai-error", "rate_limit")
	m.RecordError("openai-error", "timeout")
	// Should not panic
}

func TestMetricsHandler(t *testing.T) {
	handler := metrics.Handler()
	if handler == nil {
		t.Fatal("expected handler to be created, got nil")
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty response body")
	}
}

func TestRecordRequestMultipleProviders(t *testing.T) {
	m.RecordRequest("openai-multi", "gpt-4", "success", 0.5, 100, 50)
	m.RecordRequest("anthropic-multi", "claude-3", "success", 0.3, 200, 100)
	m.RecordRequest("gemini-multi", "gemini-pro", "error", 0.1, 0, 0)
	// Should not panic
}

func TestRecordRequestMultipleStatuses(t *testing.T) {
	m.RecordRequest("openai-status", "gpt-4", "success", 0.5, 100, 50)
	m.RecordRequest("openai-status", "gpt-4", "error", 0.1, 0, 0)
	m.RecordRequest("openai-status", "gpt-4", "timeout", 2.0, 50, 0)
	// Should not panic
}

func TestRecordRequestInputTokensOnly(t *testing.T) {
	m.RecordRequest("openai-input", "gpt-4", "success", 0.5, 100, 0)
	// Should not panic
}

func TestRecordRequestOutputTokensOnly(t *testing.T) {
	m.RecordRequest("openai-output", "gpt-4", "success", 0.5, 0, 100)
	// Should not panic
}

func TestRecordErrorMultipleTypes(t *testing.T) {
	m.RecordError("openai-multi-error", "rate_limit")
	m.RecordError("openai-multi-error", "timeout")
	m.RecordError("openai-multi-error", "authentication_failed")
	m.RecordError("anthropic-multi-error", "rate_limit")
	// Should not panic
}

func TestMetricsHandlerPrometheusFormat(t *testing.T) {
	m.RecordRequest("openai-prometheus", "gpt-4", "success", 0.5, 100, 50)

	handler := metrics.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	// Check for Prometheus format indicators
	if !strings.Contains(body, "# HELP") {
		t.Error("expected Prometheus HELP comments in response")
	}
	if !strings.Contains(body, "# TYPE") {
		t.Error("expected Prometheus TYPE comments in response")
	}
}

func TestRecordRequestLargeDuration(t *testing.T) {
	m.RecordRequest("openai-large", "gpt-4", "success", 120.5, 5000, 2000)
	// Should not panic
}

func TestActiveRequestsGauge(t *testing.T) {
	m.ActiveRequests.Inc()
	m.ActiveRequests.Inc()
	m.ActiveRequests.Dec()
	// Should not panic
}

func TestMetricsHandlerContentType(t *testing.T) {
	handler := metrics.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("expected Content-Type header to be set")
	}
}

var _ prometheus.Collector = (*prometheus.CounterVec)(nil)
