//go:build ci

package tracing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"marchproxy-egress/internal/manager"
	"marchproxy-egress/internal/middleware"
)

// MockService creates a test service
func mockService() *manager.Service {
	return &manager.Service{
		Name:    "test-service",
		Host:    "localhost",
		Port:    8080,
		Scheme:  "http",
		Healthy: true,
	}
}

// TestNewTracingEngine tests the creation of a tracing engine
func TestNewTracingEngine(t *testing.T) {
	tests := []struct {
		name      string
		config    TracingConfig
		expectErr bool
	}{
		{
			name: "valid stdout exporter",
			config: TracingConfig{
				ServiceName:   "test",
				ServiceVersion: "1.0.0",
				ExporterType:  ExporterStdout,
				SamplingRate:  0.5,
			},
			expectErr: false,
		},
		{
			name: "valid console exporter",
			config: TracingConfig{
				ServiceName:   "test",
				ServiceVersion: "1.0.0",
				ExporterType:  ExporterConsole,
				SamplingRate:  0.5,
			},
			expectErr: false,
		},
		{
			name: "default exporter fallback",
			config: TracingConfig{
				ServiceName:   "test",
				ServiceVersion: "1.0.0",
				ExporterType:  ExporterType("invalid"),
				SamplingRate:  0.5,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewTracingEngine(tt.config)
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error %v, got %v", tt.expectErr, err)
			}
			if engine != nil && !tt.expectErr {
				if err := engine.Shutdown(context.Background()); err != nil {
					t.Errorf("Shutdown failed: %v", err)
				}
			}
		})
	}
}

// TestTracingEngineSampling tests sampling rate logic
func TestTracingEngineSampling(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
	}{
		{"never sample", 0.0},
		{"always sample", 1.0},
		{"ratio sample", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TracingConfig{
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
				ExporterType:   ExporterStdout,
				SamplingRate:   tt.samplingRate,
			}
			engine, err := NewTracingEngine(config)
			if err != nil {
				t.Fatalf("NewTracingEngine failed: %v", err)
			}

			// Verify sampler is configured
			if engine.sampler == nil {
				t.Error("sampler should not be nil")
			}

			if err := engine.Shutdown(context.Background()); err != nil {
				t.Errorf("Shutdown failed: %v", err)
			}
		})
	}
}

// TestStartSpan tests span creation
func TestStartSpan(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	service := mockService()

	span := engine.StartSpan(context.Background(), "test-op", req, service)

	if span == nil {
		t.Error("StartSpan returned nil")
	}
	if span.span == nil {
		t.Error("ProxySpan.span should not be nil")
	}
	if span.startTime.IsZero() {
		t.Error("ProxySpan.startTime should be set")
	}
	if span.service != service {
		t.Error("ProxySpan.service mismatch")
	}
}

// TestFinishSpan tests span completion with success
func TestFinishSpan(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	span := engine.StartSpan(context.Background(), "test-op", req, mockService())

	resp := &http.Response{
		StatusCode:  200,
		ContentLength: 1024,
		Header:      make(http.Header),
	}

	engine.FinishSpan(span, resp, nil)

	// Verify span was completed (no panic, span ended)
	if span.response != resp {
		t.Error("response not set correctly")
	}
}

// TestFinishSpanWithError tests span completion with error
func TestFinishSpanWithError(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	span := engine.StartSpan(context.Background(), "test-op", req, mockService())

	testErr := fmt.Errorf("test error")
	engine.FinishSpan(span, nil, testErr)

	// Verify span was completed with error
	if span.response != nil {
		t.Error("response should remain nil")
	}
}

// TestExtractClientIP tests client IP extraction
func TestExtractClientIP(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
	})
	defer engine.Shutdown(context.Background())

	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1"},
			expected: "192.168.1.1",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "10.0.0.1"},
			expected: "10.0.0.1",
		},
		{
			name:     "no special headers",
			headers:  map[string]string{},
			expected: "127.0.0.1:12345", // httptest default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			ip := engine.extractClientIP(req)
			if tt.headers["X-Forwarded-For"] != "" && ip != tt.headers["X-Forwarded-For"] {
				t.Errorf("expected %s, got %s", tt.headers["X-Forwarded-For"], ip)
			}
			if tt.headers["X-Real-IP"] != "" && ip != tt.headers["X-Real-IP"] {
				t.Errorf("expected %s, got %s", tt.headers["X-Real-IP"], ip)
			}
		})
	}
}

// TestIsSensitiveHeader tests sensitive header detection
func TestIsSensitiveHeader(t *testing.T) {
	config := TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		HeaderCapture: HeaderCaptureConfig{
			SensitiveHeaders: []string{"Authorization", "Cookie"},
		},
	}
	engine, _ := NewTracingEngine(config)
	defer engine.Shutdown(context.Background())

	tests := []struct {
		header   string
		expected bool
	}{
		{"Authorization", true},
		{"Cookie", true},
		{"Content-Type", false},
		{"User-Agent", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := engine.isSensitiveHeader(tt.header)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestTruncateHeaderValue tests header value truncation
func TestTruncateHeaderValue(t *testing.T) {
	config := TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		HeaderCapture: HeaderCaptureConfig{
			MaxHeaderLength: 10,
		},
	}
	engine, _ := NewTracingEngine(config)
	defer engine.Shutdown(context.Background())

	tests := []struct {
		value    string
		expected string
	}{
		{"short", "short"},
		{"this_is_long_text", "this_is_lo..."},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := engine.truncateHeaderValue(tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetTraceID tests trace ID retrieval
func TestGetTraceID(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	span := engine.StartSpan(context.Background(), "test", req, mockService())

	traceID := engine.GetTraceID(span.context)
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
}

// TestGetSpanID tests span ID retrieval
func TestGetSpanID(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	span := engine.StartSpan(context.Background(), "test", req, mockService())

	spanID := engine.GetSpanID(span.context)
	if spanID == "" {
		t.Error("expected non-empty span ID")
	}
}

// TestIsTracing tests tracing state detection
func TestIsTracing(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	span := engine.StartSpan(context.Background(), "test", req, mockService())

	if !engine.IsTracing(span.context) {
		t.Error("expected IsTracing to return true")
	}

	if engine.IsTracing(context.Background()) {
		t.Error("expected IsTracing to return false for empty context")
	}
}

// TestInjectTraceHeaders tests header injection
func TestInjectTraceHeaders(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	span := engine.StartSpan(context.Background(), "test", req, mockService())

	headers := make(http.Header)
	engine.InjectTraceHeaders(span.context, headers)

	if len(headers) == 0 {
		t.Error("expected headers to be injected")
	}
}

// TestCreateChildSpan tests child span creation
func TestCreateChildSpan(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	parentSpan := engine.StartSpan(context.Background(), "parent", req, mockService())

	ctx, childSpan := engine.CreateChildSpan(parentSpan.context, "child")

	if ctx == nil {
		t.Error("CreateChildSpan returned nil context")
	}
	if childSpan == nil {
		t.Error("CreateChildSpan returned nil span")
	}
}

// TestAddSpanAttribute tests attribute addition with various types
func TestAddSpanAttribute(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	span := engine.StartSpan(context.Background(), "test", req, mockService())

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "key1", "value1"},
		{"int", "key2", 42},
		{"int64", "key3", int64(9999)},
		{"float64", "key4", 3.14},
		{"bool", "key5", true},
		{"other", "key6", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine.AddSpanAttribute(span.span, tt.key, tt.value)
			// No error expected
		})
	}
}

// TestDefaultTracingConfig tests default configuration
func TestDefaultTracingConfig(t *testing.T) {
	config := DefaultTracingConfig()

	if config.ServiceName != "marchproxy" {
		t.Errorf("expected service name marchproxy, got %s", config.ServiceName)
	}
	if config.Environment != "production" {
		t.Errorf("expected environment production, got %s", config.Environment)
	}
	if config.SamplingRate != 0.1 {
		t.Errorf("expected sampling rate 0.1, got %f", config.SamplingRate)
	}
}

// TestDevelopmentTracingConfig tests development configuration
func TestDevelopmentTracingConfig(t *testing.T) {
	config := DevelopmentTracingConfig()

	if config.Environment != "development" {
		t.Errorf("expected environment development, got %s", config.Environment)
	}
	if config.SamplingRate != 1.0 {
		t.Errorf("expected sampling rate 1.0, got %f", config.SamplingRate)
	}
}

// TestProductionTracingConfig tests production configuration
func TestProductionTracingConfig(t *testing.T) {
	config := ProductionTracingConfig()

	if config.Environment != "production" {
		t.Errorf("expected environment production, got %s", config.Environment)
	}
	if config.SamplingRate != 0.05 {
		t.Errorf("expected sampling rate 0.05, got %f", config.SamplingRate)
	}
}

// TestUserIDAttributeExtractor tests user ID extraction
func TestUserIDAttributeExtractor(t *testing.T) {
	tests := []struct {
		name       string
		headerVal  string
		queryVal   string
		expectNil  bool
	}{
		{"from header", "user123", "", false},
		{"from query", "", "user456", false},
		{"header priority", "user123", "user456", false},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			if tt.headerVal != "" {
				req.Header.Set("X-User-ID", tt.headerVal)
			}
			if tt.queryVal != "" {
				req.URL.RawQuery = "user_id=" + tt.queryVal
			}

			attrs := UserIDAttributeExtractor(req, nil, nil)
			if tt.expectNil && len(attrs) > 0 {
				t.Error("expected nil attrs")
			}
			if !tt.expectNil && len(attrs) == 0 {
				t.Error("expected non-nil attrs")
			}
		})
	}
}

// TestAPIVersionAttributeExtractor tests API version extraction
func TestAPIVersionAttributeExtractor(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com?version=v2", nil)
	req.Header.Set("API-Version", "v1")

	attrs := APIVersionAttributeExtractor(req, nil, nil)
	if len(attrs) == 0 {
		t.Error("expected non-nil attrs")
	}
}

// TestTenantAttributeExtractor tests tenant extraction
func TestTenantAttributeExtractor(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Tenant-ID", "tenant123")

	attrs := TenantAttributeExtractor(req, nil, nil)
	if len(attrs) == 0 {
		t.Error("expected non-nil attrs")
	}
}

// TestCacheAttributeExtractor tests cache status extraction
func TestCacheAttributeExtractor(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("X-Cache", "HIT")
	resp.Header.Set("Age", "300")

	attrs := CacheAttributeExtractor(nil, resp, nil)
	if len(attrs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(attrs))
	}
}

// TestRecordEvent tests event recording
func TestRecordEvent(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	span := engine.StartSpan(context.Background(), "test", req, mockService())

	attrs := []attribute.KeyValue{
		attribute.String("key", "value"),
	}
	engine.RecordEvent(span.span, "test-event", attrs...)
	// No error expected
}

// TestNewTracingMiddleware tests middleware creation
func TestNewTracingMiddleware(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
	})
	defer engine.Shutdown(context.Background())

	config := DefaultTracingMiddlewareConfig()
	tm := NewTracingMiddleware(engine, config)

	if tm == nil {
		t.Error("NewTracingMiddleware returned nil")
	}
	if tm.Name() != "tracing" {
		t.Errorf("expected name tracing, got %s", tm.Name())
	}
	if tm.Priority() != 10 {
		t.Errorf("expected priority 10, got %d", tm.Priority())
	}
}

// TestTracingMiddlewareEnabled tests enable/disable functionality
func TestTracingMiddlewareEnabled(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
	})
	defer engine.Shutdown(context.Background())

	tm := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())

	if !tm.Enabled() {
		t.Error("expected middleware to be enabled by default")
	}

	tm.Disable()
	if tm.Enabled() {
		t.Error("expected middleware to be disabled")
	}

	tm.Enable()
	if !tm.Enabled() {
		t.Error("expected middleware to be enabled")
	}
}

// TestTracingMiddlewareProcessRequest tests request processing
func TestTracingMiddlewareProcessRequest(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	tm := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	mwCtx := &middleware.MiddlewareContext{
		Request: req,
		Service: mockService(),
	}

	err := tm.ProcessRequest(req, mwCtx)
	if err != nil {
		t.Errorf("ProcessRequest failed: %v", err)
	}

	_, hasSpan := mwCtx.GetData("tracing_span").(*ProxySpan)
	if !hasSpan {
		t.Error("expected tracing_span in context")
	}
}

// TestTracingMiddlewareProcessResponse tests response processing
func TestTracingMiddlewareProcessResponse(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	tm := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	mwCtx := &middleware.MiddlewareContext{
		Request: req,
		Service: mockService(),
	}

	tm.ProcessRequest(req, mwCtx)

	resp := &http.Response{
		StatusCode:    200,
		ContentLength: 1024,
		Header:        make(http.Header),
	}

	err := tm.ProcessResponse(resp, mwCtx)
	if err != nil {
		t.Errorf("ProcessResponse failed: %v", err)
	}

	if resp.Header.Get("X-Trace-ID") == "" {
		t.Error("expected X-Trace-ID header")
	}
	if resp.Header.Get("X-Span-ID") == "" {
		t.Error("expected X-Span-ID header")
	}
}

// TestTracingMiddlewareProcessError tests error handling
func TestTracingMiddlewareProcessError(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	tm := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	mwCtx := &middleware.MiddlewareContext{
		Request: req,
		Service: mockService(),
	}

	tm.ProcessRequest(req, mwCtx)

	testErr := fmt.Errorf("test error")
	err := tm.ProcessError(testErr, mwCtx)

	if err != testErr {
		t.Error("expected error to be returned")
	}

	stats := tm.GetStatistics()
	if stats.TracesErrored == 0 {
		t.Error("expected TracesErrored to be incremented")
	}
}

// TestTracingMiddlewareSkipPaths tests path skipping
func TestTracingMiddlewareSkipPaths(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
	})
	defer engine.Shutdown(context.Background())

	config := DefaultTracingMiddlewareConfig()
	config.SkipPaths = []string{"/health", "/metrics"}
	tm := NewTracingMiddleware(engine, config)

	healthReq := httptest.NewRequest("GET", "http://example.com/health", nil)
	if tm.shouldTrace(healthReq) {
		t.Error("expected health path to be skipped")
	}

	normalReq := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	if !tm.shouldTrace(normalReq) {
		t.Error("expected normal path to be traced")
	}
}

// TestTracingStatistics tests statistics tracking
func TestTracingStatistics(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	tm := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	mwCtx := &middleware.MiddlewareContext{
		Request: req,
		Service: mockService(),
	}

	tm.ProcessRequest(req, mwCtx)
	resp := &http.Response{
		StatusCode:    200,
		ContentLength: 1024,
		Header:        make(http.Header),
	}
	tm.ProcessResponse(resp, mwCtx)

	stats := tm.GetStatistics()
	if stats.TracesStarted == 0 {
		t.Error("expected TracesStarted to be incremented")
	}
	if stats.TracesCompleted == 0 {
		t.Error("expected TracesCompleted to be incremented")
	}

	tm.ResetStatistics()
	stats = tm.GetStatistics()
	if stats.TracesStarted != 0 {
		t.Error("expected statistics to be reset")
	}
}

// TestDistributedTracingContext tests context extraction
func TestDistributedTracingContext(t *testing.T) {
	engine, _ := NewTracingEngine(TracingConfig{
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	})
	defer engine.Shutdown(context.Background())

	tm := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())

	req := httptest.NewRequest("GET", "http://example.com", nil)
	dtc := tm.ExtractTraceContext(req)

	// Context may be nil if no trace headers present
	if dtc != nil {
		if dtc.TraceID == "" {
			t.Error("expected non-empty TraceID")
		}
	}
}
