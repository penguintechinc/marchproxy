package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/MarchProxy/proxy/internal/middleware"
)

type TracingMiddleware struct {
	engine     *TracingEngine
	config     TracingMiddlewareConfig
	enabled    bool
	statistics *TracingStatistics
}

type TracingMiddlewareConfig struct {
	OperationName       string
	TraceRequests       bool
	TraceResponses      bool
	TraceErrors         bool
	TracingHeaders      []string
	SkipPaths          []string
	SampleRate         float64
	MaxSpanAttributes  int
	IncludeRequestBody bool
	IncludeResponseBody bool
	MaxBodySize        int64
	SensitiveHeaders   []string
}

type TracingStatistics struct {
	TracesStarted     uint64
	TracesCompleted   uint64
	TracesErrored     uint64
	TotalSpans        uint64
	AverageSpanDuration time.Duration
	TotalTracingOverhead time.Duration
}

func NewTracingMiddleware(engine *TracingEngine, config TracingMiddlewareConfig) *TracingMiddleware {
	return &TracingMiddleware{
		engine:     engine,
		config:     config,
		enabled:    true,
		statistics: &TracingStatistics{},
	}
}

func (tm *TracingMiddleware) Name() string {
	return "tracing"
}

func (tm *TracingMiddleware) Priority() int {
	return 10
}

func (tm *TracingMiddleware) ProcessRequest(req *http.Request, ctx *middleware.MiddlewareContext) error {
	if !tm.enabled || !tm.shouldTrace(req) {
		return nil
	}

	tracingStart := time.Now()
	defer func() {
		tm.statistics.TotalTracingOverhead += time.Since(tracingStart)
	}()

	operationName := tm.config.OperationName
	if operationName == "" {
		operationName = fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	}

	service := ctx.Service
	proxySpan := tm.engine.StartSpan(req.Context(), operationName, req, service)

	ctx.SetData("tracing_span", proxySpan)
	ctx.SetData("tracing_start_time", time.Now())

	newReq := req.WithContext(proxySpan.context)
	ctx.Request = newReq

	tm.addCustomAttributes(proxySpan, req, nil, ctx)

	if tm.config.IncludeRequestBody && req.Body != nil {
		tm.captureRequestBody(proxySpan, req)
	}

	tm.statistics.TracesStarted++
	tm.statistics.TotalSpans++

	return nil
}

func (tm *TracingMiddleware) ProcessResponse(resp *http.Response, ctx *middleware.MiddlewareContext) error {
	if !tm.enabled {
		return nil
	}

	proxySpan, ok := ctx.GetData("tracing_span").(*ProxySpan)
	if !ok {
		return nil
	}

	if tm.config.TraceResponses {
		tm.addResponseAttributes(proxySpan, resp)
	}

	if tm.config.IncludeResponseBody && resp.Body != nil {
		tm.captureResponseBody(proxySpan, resp)
	}

	tm.addCustomAttributes(proxySpan, ctx.Request, resp, ctx)

	if startTime, ok := ctx.GetData("tracing_start_time").(time.Time); ok {
		duration := time.Since(startTime)
		tm.updateAverageSpanDuration(duration)
		proxySpan.span.SetAttributes(attribute.Int64("middleware.duration_ms", duration.Milliseconds()))
	}

	tm.engine.FinishSpan(proxySpan, resp, nil)
	tm.statistics.TracesCompleted++

	traceID := tm.engine.GetTraceID(proxySpan.context)
	spanID := tm.engine.GetSpanID(proxySpan.context)

	if resp.Header != nil {
		resp.Header.Set("X-Trace-ID", traceID)
		resp.Header.Set("X-Span-ID", spanID)
	}

	return nil
}

func (tm *TracingMiddleware) ProcessError(err error, ctx *middleware.MiddlewareContext) error {
	if !tm.enabled {
		return err
	}

	proxySpan, ok := ctx.GetData("tracing_span").(*ProxySpan)
	if !ok {
		return err
	}

	if tm.config.TraceErrors {
		proxySpan.span.SetAttributes(
			attribute.String("error.type", fmt.Sprintf("%T", err)),
			attribute.String("error.message", err.Error()),
			attribute.Bool("error", true),
		)

		tm.engine.RecordEvent(proxySpan.span, "error.occurred",
			attribute.String("error.message", err.Error()),
			attribute.String("error.type", fmt.Sprintf("%T", err)),
		)
	}

	tm.engine.FinishSpan(proxySpan, nil, err)
	tm.statistics.TracesErrored++

	return err
}

func (tm *TracingMiddleware) shouldTrace(req *http.Request) bool {
	for _, skipPath := range tm.config.SkipPaths {
		if req.URL.Path == skipPath {
			return false
		}
	}

	if tm.config.SampleRate > 0 && tm.config.SampleRate < 1.0 {
		return tm.shouldSample()
	}

	return tm.config.TraceRequests
}

func (tm *TracingMiddleware) shouldSample() bool {
	return true
}

func (tm *TracingMiddleware) addCustomAttributes(proxySpan *ProxySpan, req *http.Request, resp *http.Response, ctx *middleware.MiddlewareContext) {
	if ctx.HasData("load_balancer_algorithm") {
		if algorithm, ok := ctx.GetData("load_balancer_algorithm").(string); ok {
			proxySpan.span.SetAttributes(attribute.String("load_balancer.algorithm", algorithm))
		}
	}

	if ctx.HasData("circuit_breaker") {
		proxySpan.span.SetAttributes(attribute.Bool("circuit_breaker.enabled", true))
		if state, ok := ctx.GetData("circuit_breaker_state").(string); ok {
			proxySpan.span.SetAttributes(attribute.String("circuit_breaker.state", state))
		}
	}

	if ctx.HasData("cache_hit") {
		if hit, ok := ctx.GetData("cache_hit").(bool); ok {
			proxySpan.span.SetAttributes(attribute.Bool("cache.hit", hit))
		}
	}

	if ctx.HasData("compression_ratio") {
		if ratio, ok := ctx.GetData("compression_ratio").(float64); ok {
			proxySpan.span.SetAttributes(attribute.Float64("compression.ratio", ratio))
		}
	}

	if ctx.HasData("retry_count") {
		if count, ok := ctx.GetData("retry_count").(int); ok {
			proxySpan.span.SetAttributes(attribute.Int("retry.count", count))
		}
	}

	if ctx.HasData("websocket_upgrade") {
		if upgraded, ok := ctx.GetData("websocket_upgrade").(bool); ok {
			proxySpan.span.SetAttributes(attribute.Bool("websocket.upgraded", upgraded))
		}
	}

	if ctx.HasData("quic_stream_id") {
		if streamID, ok := ctx.GetData("quic_stream_id").(int64); ok {
			proxySpan.span.SetAttributes(attribute.Int64("quic.stream_id", streamID))
		}
	}
}

func (tm *TracingMiddleware) addResponseAttributes(proxySpan *ProxySpan, resp *http.Response) {
	if resp == nil {
		return
	}

	proxySpan.span.SetAttributes(
		attribute.Int("http.response.status_code", resp.StatusCode),
		attribute.Int64("http.response.content_length", resp.ContentLength),
	)

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		proxySpan.span.SetAttributes(attribute.String("http.response.content_type", contentType))
	}

	for _, header := range tm.config.TracingHeaders {
		if value := resp.Header.Get(header); value != "" && !tm.isSensitiveHeader(header) {
			proxySpan.span.SetAttributes(attribute.String("http.response.header."+header, value))
		}
	}

	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" {
		proxySpan.span.SetAttributes(attribute.String("http.response.encoding", encoding))
	}

	statusClass := resp.StatusCode / 100
	proxySpan.span.SetAttributes(attribute.Int("http.response.status_class", statusClass))
}

func (tm *TracingMiddleware) captureRequestBody(proxySpan *ProxySpan, req *http.Request) {
	if req.ContentLength > tm.config.MaxBodySize {
		proxySpan.span.SetAttributes(attribute.String("http.request.body", "TRUNCATED_TOO_LARGE"))
		return
	}

	tm.engine.RecordEvent(proxySpan.span, "request.body.captured",
		attribute.Int64("body.size", req.ContentLength),
	)
}

func (tm *TracingMiddleware) captureResponseBody(proxySpan *ProxySpan, resp *http.Response) {
	if resp.ContentLength > tm.config.MaxBodySize {
		proxySpan.span.SetAttributes(attribute.String("http.response.body", "TRUNCATED_TOO_LARGE"))
		return
	}

	tm.engine.RecordEvent(proxySpan.span, "response.body.captured",
		attribute.Int64("body.size", resp.ContentLength),
	)
}

func (tm *TracingMiddleware) isSensitiveHeader(header string) bool {
	for _, sensitive := range tm.config.SensitiveHeaders {
		if sensitive == header {
			return true
		}
	}
	return false
}

func (tm *TracingMiddleware) updateAverageSpanDuration(duration time.Duration) {
	totalSpans := tm.statistics.TotalSpans
	if totalSpans > 0 {
		tm.statistics.AverageSpanDuration = 
			(tm.statistics.AverageSpanDuration*time.Duration(totalSpans-1) + duration) / 
			time.Duration(totalSpans)
	} else {
		tm.statistics.AverageSpanDuration = duration
	}
}

func (tm *TracingMiddleware) Enabled() bool {
	return tm.enabled
}

func (tm *TracingMiddleware) Enable() {
	tm.enabled = true
}

func (tm *TracingMiddleware) Disable() {
	tm.enabled = false
}

func (tm *TracingMiddleware) GetStatistics() *TracingStatistics {
	return tm.statistics
}

func (tm *TracingMiddleware) ResetStatistics() {
	tm.statistics = &TracingStatistics{}
}

func (tm *TracingMiddleware) GetEngine() *TracingEngine {
	return tm.engine
}

type TracingHandler struct {
	middleware *TracingMiddleware
}

func NewTracingHandler(middleware *TracingMiddleware) *TracingHandler {
	return &TracingHandler{
		middleware: middleware,
	}
}

func (th *TracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		th.handleStats(w, r)
	case "POST":
		th.handleForceFlush(w, r)
	case "DELETE":
		th.handleShutdown(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (th *TracingHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := th.middleware.GetStatistics()
	
	response := fmt.Sprintf(`{
		"traces_started": %d,
		"traces_completed": %d,
		"traces_errored": %d,
		"total_spans": %d,
		"average_span_duration_ms": %.2f,
		"total_tracing_overhead_ms": %.2f,
		"success_rate": %.2f
	}`,
		stats.TracesStarted,
		stats.TracesCompleted,
		stats.TracesErrored,
		stats.TotalSpans,
		float64(stats.AverageSpanDuration.Nanoseconds())/1000000.0,
		float64(stats.TotalTracingOverhead.Nanoseconds())/1000000.0,
		float64(stats.TracesCompleted)/float64(stats.TracesStarted)*100,
	)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func (th *TracingHandler) handleForceFlush(w http.ResponseWriter, r *http.Request) {
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 10 * time.Second
	
	if timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	err := th.middleware.GetEngine().ForceFlush(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Force flush failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Force flush completed successfully"))
}

func (th *TracingHandler) handleShutdown(w http.ResponseWriter, r *http.Request) {
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 30 * time.Second
	
	if timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	err := th.middleware.GetEngine().Shutdown(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Shutdown failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Tracing engine shutdown completed"))
}

func DefaultTracingMiddlewareConfig() TracingMiddlewareConfig {
	return TracingMiddlewareConfig{
		OperationName:       "",
		TraceRequests:       true,
		TraceResponses:      true,
		TraceErrors:         true,
		TracingHeaders:      []string{"Content-Type", "User-Agent", "Authorization"},
		SkipPaths:          []string{"/health", "/metrics", "/ready"},
		SampleRate:         1.0,
		MaxSpanAttributes:  64,
		IncludeRequestBody: false,
		IncludeResponseBody: false,
		MaxBodySize:        1024,
		SensitiveHeaders:   []string{"Authorization", "Cookie", "Set-Cookie", "X-API-Key"},
	}
}

func ProductionTracingMiddlewareConfig() TracingMiddlewareConfig {
	config := DefaultTracingMiddlewareConfig()
	config.SampleRate = 0.1
	config.MaxSpanAttributes = 32
	config.IncludeRequestBody = false
	config.IncludeResponseBody = false
	return config
}

func DevelopmentTracingMiddlewareConfig() TracingMiddlewareConfig {
	config := DefaultTracingMiddlewareConfig()
	config.SampleRate = 1.0
	config.IncludeRequestBody = true
	config.IncludeResponseBody = true
	config.MaxBodySize = 4096
	return config
}

type DistributedTracingContext struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Sampled    bool
	Attributes map[string]string
}

func (tm *TracingMiddleware) PropagateTrace(ctx context.Context, req *http.Request) {
	tm.engine.InjectTraceHeaders(ctx, req.Header)
}

func (tm *TracingMiddleware) ExtractTraceContext(req *http.Request) *DistributedTracingContext {
	ctx := tm.engine.ExtractTraceContext(req)
	spanCtx := trace.SpanContextFromContext(ctx)
	
	if !spanCtx.IsValid() {
		return nil
	}
	
	return &DistributedTracingContext{
		TraceID:    spanCtx.TraceID().String(),
		SpanID:     spanCtx.SpanID().String(),
		Sampled:    spanCtx.IsSampled(),
		Attributes: make(map[string]string),
	}
}

func (tm *TracingMiddleware) CreateChildOperation(ctx context.Context, operationName string) (context.Context, trace.Span) {
	return tm.engine.CreateChildSpan(ctx, operationName)
}

func (tm *TracingMiddleware) RecordMetric(span trace.Span, name string, value interface{}, attributes map[string]interface{}) {
	attrs := make([]attribute.KeyValue, 0, len(attributes)+1)
	
	switch v := value.(type) {
	case int:
		attrs = append(attrs, attribute.Int(name, v))
	case int64:
		attrs = append(attrs, attribute.Int64(name, v))
	case float64:
		attrs = append(attrs, attribute.Float64(name, v))
	case string:
		attrs = append(attrs, attribute.String(name, v))
	case bool:
		attrs = append(attrs, attribute.Bool(name, v))
	default:
		attrs = append(attrs, attribute.String(name, fmt.Sprintf("%v", v)))
	}
	
	for key, attr := range attributes {
		switch av := attr.(type) {
		case string:
			attrs = append(attrs, attribute.String(key, av))
		case int:
			attrs = append(attrs, attribute.Int(key, av))
		case int64:
			attrs = append(attrs, attribute.Int64(key, av))
		case float64:
			attrs = append(attrs, attribute.Float64(key, av))
		case bool:
			attrs = append(attrs, attribute.Bool(key, av))
		default:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%v", av)))
		}
	}
	
	tm.engine.RecordEvent(span, "metric.recorded", attrs...)
}