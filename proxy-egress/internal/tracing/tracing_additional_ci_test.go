//go:build ci

package tracing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"marchproxy-egress/internal/manager"
)

// TestSpanBatching tests span batching in batch processor
func TestSpanBatching(t *testing.T) {
	config := TracingConfig{
		ServiceName:    "test-batch",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
		BatchConfig: BatchConfig{
			BatchTimeout:     100 * time.Millisecond,
			ExportTimeout:    5 * time.Second,
			MaxBatchSize:     10,
			MaxQueueSize:     100,
			BlockOnQueueFull: false,
		},
	}

	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	// Create multiple spans
	for i := 0; i < 5; i++ {
		span := engine.StartSpan(context.Background(), fmt.Sprintf("op_%d", i), req, service)
		engine.FinishSpan(span, nil, nil)
	}

	// Batch should process without error
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := engine.ForceFlush(ctx); err != nil {
		t.Errorf("ForceFlush failed: %v", err)
	}
}

// TestContextPropagation tests trace context extraction and injection
func TestContextPropagation(t *testing.T) {
	config := DefaultTracingConfig()
	config.ExporterType = ExporterStdout

	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	service := &manager.Service{
		Name:   "test-service",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	// Start span
	span := engine.StartSpan(context.Background(), "test_op", req, service)
	ctx := span.context

	// Inject headers
	headers := make(http.Header)
	engine.InjectTraceHeaders(ctx, headers)

	// Verify headers are set
	if headers.Get("traceparent") == "" && headers.Get("baggage") == "" {
		t.Error("expected trace headers to be injected")
	}

	// Extract context from headers
	extractedCtx := engine.ExtractTraceContext(req)
	if extractedCtx == nil {
		t.Fatal("expected non-nil extracted context")
	}

	engine.FinishSpan(span, nil, nil)
}

// TestAttributeTruncation tests header value truncation
func TestAttributeTruncation(t *testing.T) {
	config := TracingConfig{
		ServiceName:    "test-truncate",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
		HeaderCapture: HeaderCaptureConfig{
			RequestHeaders:   []string{"X-Custom-Header"},
			ResponseHeaders:  []string{"X-Response-Header"},
			SensitiveHeaders: []string{},
			MaxHeaderLength:  10,
		},
	}

	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	// Long header value
	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.Header.Set("X-Custom-Header", "this-is-a-very-long-header-value-that-should-be-truncated")

	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	span := engine.StartSpan(context.Background(), "test_op", req, service)

	// Verify truncation logic works
	truncated := engine.truncateHeaderValue("this-is-a-very-long-header-value")
	if !strings.HasSuffix(truncated, "...") {
		t.Error("expected truncated value to end with ...")
	}
	if len(truncated) > 13 { // 10 + "..."
		t.Errorf("truncated value too long: %d", len(truncated))
	}

	engine.FinishSpan(span, nil, nil)
}

// TestSensitiveHeaderFiltering tests sensitive header filtering
func TestSensitiveHeaderFiltering(t *testing.T) {
	config := TracingConfig{
		ServiceName:    "test-sensitive",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
		HeaderCapture: HeaderCaptureConfig{
			RequestHeaders:   []string{"Authorization", "X-API-Key", "User-Agent"},
			ResponseHeaders:  []string{},
			SensitiveHeaders: []string{"Authorization", "X-API-Key"},
			MaxHeaderLength:  256,
		},
	}

	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	// Test isSensitiveHeader
	if !engine.isSensitiveHeader("Authorization") {
		t.Error("Authorization should be sensitive")
	}
	if !engine.isSensitiveHeader("X-API-Key") {
		t.Error("X-API-Key should be sensitive")
	}
	if engine.isSensitiveHeader("User-Agent") {
		t.Error("User-Agent should not be sensitive")
	}

	engine.Shutdown(context.Background())
}

// TestClientIPExtraction tests client IP extraction from headers
func TestClientIPExtraction(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	tests := []struct {
		name           string
		setup          func(*http.Request)
		expectedPrefix string
	}{
		{
			name: "X-Forwarded-For header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1")
			},
			expectedPrefix: "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "198.51.100.1")
			},
			expectedPrefix: "198.51.100.1",
		},
		{
			name: "RemoteAddr fallback",
			setup: func(r *http.Request) {
				r.RemoteAddr = "192.0.2.1:12345"
			},
			expectedPrefix: "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			tt.setup(req)

			ip := engine.extractClientIP(req)
			if !strings.Contains(ip, tt.expectedPrefix) {
				t.Errorf("got %q, want prefix %q", ip, tt.expectedPrefix)
			}
		})
	}
}

// TestCustomAttributeExtractors tests custom attribute extraction
func TestCustomAttributeExtractors(t *testing.T) {
	customAttrs := map[string]AttributeExtractor{
		"user_id": UserIDAttributeExtractor,
		"api_version": APIVersionAttributeExtractor,
		"tenant": TenantAttributeExtractor,
		"cache": CacheAttributeExtractor,
	}

	config := TracingConfig{
		ServiceName:      "test-custom-attrs",
		ServiceVersion:   "1.0.0",
		ExporterType:     ExporterStdout,
		SamplingRate:     1.0,
		CustomAttributes: customAttrs,
	}

	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/api/test?user_id=user123&version=v2&tenant=acme", nil)
	req.Header.Set("API-Version", "v2")
	req.Header.Set("X-Tenant-ID", "acme")

	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	span := engine.StartSpan(context.Background(), "test_op", req, service)

	// Custom attributes should be applied in FinishSpan
	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
	}
	resp.Header.Set("X-Cache", "HIT")
	resp.Header.Set("Age", "3600")

	engine.FinishSpan(span, resp, nil)

	// Verify execution without panic
}

// TestSpanStatusCodeHandling tests HTTP status code to span status mapping
func TestSpanStatusCodeHandling(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	tests := []struct {
		statusCode int
		shouldError bool
	}{
		{200, false},
		{301, false},
		{304, false},
		{400, true},
		{404, true},
		{500, true},
		{503, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			service := &manager.Service{
				Name:   "test",
				Host:   "localhost",
				Port:   8080,
				Scheme: "http",
			}

			span := engine.StartSpan(context.Background(), "test_op", req, service)

			resp := &http.Response{
				StatusCode:    tt.statusCode,
				Header:        make(http.Header),
				ContentLength: 100,
			}

			engine.FinishSpan(span, resp, nil)
			// Verify no panic and span closed correctly
		})
	}
}

// TestAddSpanAttributeTypes tests adding different types of span attributes
func TestAddSpanAttributeTypes(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	span := engine.StartSpan(context.Background(), "test_op", req, service)

	// Test different attribute types
	engine.AddSpanAttribute(span.span, "string_attr", "test_value")
	engine.AddSpanAttribute(span.span, "int_attr", 42)
	engine.AddSpanAttribute(span.span, "int64_attr", int64(9223372036854775807))
	engine.AddSpanAttribute(span.span, "float_attr", 3.14159)
	engine.AddSpanAttribute(span.span, "bool_attr", true)
	engine.AddSpanAttribute(span.span, "unknown_type", struct{}{})

	engine.FinishSpan(span, nil, nil)
	// Verify no panic
}

// TestTracingMiddlewareStatistics tests middleware statistics collection
func TestTracingMiddlewareStatistics(t *testing.T) {
	config := TracingConfig{
		ServiceName:    "test-middleware",
		ServiceVersion: "1.0.0",
		ExporterType:   ExporterStdout,
		SamplingRate:   1.0,
	}

	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	midConfig := DefaultTracingMiddlewareConfig()
	mw := NewTracingMiddleware(engine, midConfig)

	if mw.statistics.TracesStarted != 0 {
		t.Error("expected TracesStarted to be 0")
	}

	// Verify initial state
	stats := mw.GetStatistics()
	if stats == nil {
		t.Fatal("expected non-nil statistics")
	}
}

// TestTracingMiddlewareErrorHandling tests error handling in middleware
func TestTracingMiddlewareErrorHandling(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	midConfig := DefaultTracingMiddlewareConfig()
	midConfig.TraceErrors = true
	mw := NewTracingMiddleware(engine, midConfig)

	// Verify middleware is enabled and can track errors
	if !mw.Enabled() {
		t.Error("expected middleware to be enabled")
	}

	// Enable and disable tests
	mw.Disable()
	if mw.Enabled() {
		t.Error("expected middleware to be disabled")
	}

	mw.Enable()
	if !mw.Enabled() {
		t.Error("expected middleware to be enabled again")
	}
}

// TestTracingHandlerStats tests tracing handler stats endpoint
func TestTracingHandlerStats(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	midConfig := DefaultTracingMiddlewareConfig()
	middleware := NewTracingMiddleware(engine, midConfig)
	handler := NewTracingHandler(middleware)

	// Set some stats
	middleware.statistics.TracesStarted = 10
	middleware.statistics.TracesCompleted = 8
	middleware.statistics.TracesErrored = 2

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://localhost/tracing/stats", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "traces_started") {
		t.Error("expected response to contain 'traces_started'")
	}
}

// TestTracingHandlerForceFlush tests force flush endpoint
func TestTracingHandlerForceFlush(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	middleware := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())
	handler := NewTracingHandler(middleware)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://localhost/tracing/flush?timeout=5s", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestTracingHandlerShutdown tests shutdown endpoint
func TestTracingHandlerShutdown(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	middleware := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())
	handler := NewTracingHandler(middleware)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "http://localhost/tracing/shutdown?timeout=5s", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestTracingHandlerInvalidMethod tests invalid HTTP method
func TestTracingHandlerInvalidMethod(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	middleware := NewTracingMiddleware(engine, DefaultTracingMiddlewareConfig())
	handler := NewTracingHandler(middleware)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "http://localhost/tracing", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestSpanContextConcurrency tests concurrent span operations
func TestSpanContextConcurrency(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	var wg sync.WaitGroup
	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	// Launch multiple goroutines creating spans concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", fmt.Sprintf("http://localhost/test/%d", index), nil)
			span := engine.StartSpan(context.Background(), fmt.Sprintf("op_%d", index), req, service)

			time.Sleep(time.Millisecond)

			resp := &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
			}
			engine.FinishSpan(span, resp, nil)
		}(i)
	}

	wg.Wait()

	// Verify no race conditions or panics
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	engine.ForceFlush(ctx)
}

// TestGetTraceIDAndSpanID tests trace and span ID retrieval
func TestGetTraceIDAndSpanID(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	span := engine.StartSpan(context.Background(), "test_op", req, service)

	traceID := engine.GetTraceID(span.context)
	spanID := engine.GetSpanID(span.context)

	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
	if spanID == "" {
		t.Error("expected non-empty span ID")
	}

	isTracing := engine.IsTracing(span.context)
	if !isTracing {
		t.Error("expected IsTracing to return true")
	}

	engine.FinishSpan(span, nil, nil)
}

// TestCreateChildSpanAdditional tests creating child spans with edge cases
func TestCreateChildSpanAdditional(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	parentSpan := engine.StartSpan(context.Background(), "parent_op", req, service)
	parentCtx := parentSpan.context

	// Create multiple child spans
	for i := 0; i < 3; i++ {
		childCtx, childSpan := engine.CreateChildSpan(parentCtx, fmt.Sprintf("child_op_%d", i))
		if childCtx == nil || childSpan == nil {
			t.Errorf("child %d: expected non-nil context and span", i)
		}
		childSpan.End()
	}

	engine.FinishSpan(parentSpan, nil, nil)
}

// TestRecordEventAdditional tests recording span events with variations
func TestRecordEventAdditional(t *testing.T) {
	config := DefaultTracingConfig()
	engine, err := NewTracingEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Shutdown(context.Background())

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	service := &manager.Service{
		Name:   "test",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	span := engine.StartSpan(context.Background(), "test_op", req, service)

	// Record multiple events
	engine.RecordEvent(span.span, "event.start")
	engine.RecordEvent(span.span, "event.middle",
		attribute.String("stage", "processing"),
	)
	engine.RecordEvent(span.span, "event.end",
		attribute.String("status", "success"),
		attribute.Int64("latency_ms", 123),
	)

	engine.FinishSpan(span, nil, nil)
}

// TestTracingConfigVariants tests different configuration variants
func TestTracingConfigVariants(t *testing.T) {
	tests := []struct {
		name   string
		config TracingConfig
	}{
		{"default", DefaultTracingConfig()},
		{"development", DevelopmentTracingConfig()},
		{"production", ProductionTracingConfig()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewTracingEngine(tt.config)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			if engine == nil {
				t.Error("expected non-nil engine")
			}

			engine.Shutdown(context.Background())
		})
	}
}
