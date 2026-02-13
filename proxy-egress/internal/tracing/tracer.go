package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"marchproxy-egress/internal/manager"
)

type TracingEngine struct {
	tracer     oteltrace.Tracer
	config     TracingConfig
	provider   *trace.TracerProvider
	propagator propagation.TextMapPropagator
	exporter   trace.SpanExporter
	processor  trace.SpanProcessor
	sampler    trace.Sampler
}

type TracingConfig struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	ExporterType    ExporterType
	JaegerEndpoint  string
	OTLPEndpoint    string
	SamplingRate    float64
	MaxSpansPerTrace int
	ResourceAttributes map[string]string
	HeaderCapture   HeaderCaptureConfig
	CustomAttributes map[string]AttributeExtractor
	SpanProcessors  []SpanProcessorConfig
	BatchConfig     BatchConfig
}

type ExporterType string

const (
	ExporterStdout  ExporterType = "stdout"
	ExporterConsole ExporterType = "console"
)

type HeaderCaptureConfig struct {
	RequestHeaders  []string
	ResponseHeaders []string
	SensitiveHeaders []string
	MaxHeaderLength int
}

type AttributeExtractor func(req *http.Request, resp *http.Response, service *manager.Service) []attribute.KeyValue

type SpanProcessorConfig struct {
	Type       string
	BatchSize  int
	Timeout    time.Duration
	ExportTimeout time.Duration
}

type BatchConfig struct {
	BatchTimeout       time.Duration
	ExportTimeout      time.Duration
	MaxBatchSize       int
	MaxQueueSize       int
	BlockOnQueueFull   bool
}

type ProxySpan struct {
	span     oteltrace.Span
	context  context.Context
	startTime time.Time
	service  *manager.Service
	request  *http.Request
	response *http.Response
	attributes map[string]interface{}
}

type SpanMetrics struct {
	SpanCount        uint64
	ErrorCount       uint64
	TotalDuration    time.Duration
	AverageDuration  time.Duration
	SuccessRate      float64
	P95Duration      time.Duration
	P99Duration      time.Duration
}

func NewTracingEngine(config TracingConfig) (*TracingEngine, error) {
	te := &TracingEngine{
		config: config,
	}

	if err := te.initializeTracer(); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	return te, nil
}

func (te *TracingEngine) initializeTracer() error {
	exporter, err := te.createExporter()
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}
	te.exporter = exporter

	processor := te.createSpanProcessor()
	te.processor = processor

	sampler := te.createSampler()
	te.sampler = sampler

	resource := te.createResource()

	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(processor),
		trace.WithSampler(sampler),
		trace.WithResource(resource),
	)

	te.provider = tp
	otel.SetTracerProvider(tp)

	te.propagator = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(te.propagator)

	te.tracer = tp.Tracer(
		te.config.ServiceName,
		oteltrace.WithInstrumentationVersion(te.config.ServiceVersion),
	)

	return nil
}

func (te *TracingEngine) createExporter() (trace.SpanExporter, error) {
	switch te.config.ExporterType {
	// case ExporterJaeger:  // Deprecated - Jaeger exporter removed, use OTLP exporter instead
	//	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(te.config.JaegerEndpoint)))

	// case ExporterOTLP:  // Temporarily disabled due to genproto conflicts
	//	return otlptracehttp.New(
	//		context.Background(),
	//		otlptracehttp.WithEndpoint(te.config.OTLPEndpoint),
	//	)
	
	case ExporterStdout, ExporterConsole:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	
	default:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
}

func (te *TracingEngine) createSpanProcessor() trace.SpanProcessor {
	if len(te.config.SpanProcessors) > 0 {
		var processors []trace.SpanProcessor
		for _, config := range te.config.SpanProcessors {
			switch config.Type {
			case "batch":
				processors = append(processors, trace.NewBatchSpanProcessor(
					te.exporter,
					trace.WithBatchTimeout(config.Timeout),
					trace.WithExportTimeout(config.ExportTimeout),
					// trace.WithMaxBatchSize(config.BatchSize),
				))
			case "simple":
				processors = append(processors, trace.NewSimpleSpanProcessor(te.exporter))
			}
		}
		
		if len(processors) == 1 {
			return processors[0]
		}
		
		return trace.NewBatchSpanProcessor(te.exporter)
	}

	if te.config.BatchConfig.BatchTimeout > 0 {
		opts := []trace.BatchSpanProcessorOption{
			trace.WithBatchTimeout(te.config.BatchConfig.BatchTimeout),
			trace.WithExportTimeout(te.config.BatchConfig.ExportTimeout),
			trace.WithMaxExportBatchSize(te.config.BatchConfig.MaxBatchSize),
			trace.WithMaxQueueSize(te.config.BatchConfig.MaxQueueSize),
		}
		if te.config.BatchConfig.BlockOnQueueFull {
			opts = append(opts, trace.WithBlocking())
		}
		return trace.NewBatchSpanProcessor(te.exporter, opts...)
	}

	return trace.NewBatchSpanProcessor(te.exporter)
}

func (te *TracingEngine) createSampler() trace.Sampler {
	if te.config.SamplingRate <= 0 {
		return trace.NeverSample()
	}
	if te.config.SamplingRate >= 1.0 {
		return trace.AlwaysSample()
	}
	return trace.TraceIDRatioBased(te.config.SamplingRate)
}

func (te *TracingEngine) createResource() *resource.Resource {
	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(te.config.ServiceName),
		semconv.ServiceVersionKey.String(te.config.ServiceVersion),
		attribute.String("environment", te.config.Environment),
	}

	for key, value := range te.config.ResourceAttributes {
		attributes = append(attributes, attribute.String(key, value))
	}

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		attributes...,
	)
}

func (te *TracingEngine) StartSpan(ctx context.Context, operationName string, req *http.Request, service *manager.Service) *ProxySpan {
	ctx = te.propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))

	spanCtx, span := te.tracer.Start(ctx, operationName,
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		oteltrace.WithTimestamp(time.Now()),
	)

	proxySpan := &ProxySpan{
		span:      span,
		context:   spanCtx,
		startTime: time.Now(),
		service:   service,
		request:   req,
		attributes: make(map[string]interface{}),
	}

	te.setRequestAttributes(proxySpan)
	te.setServiceAttributes(proxySpan)

	return proxySpan
}

func (te *TracingEngine) setRequestAttributes(ps *ProxySpan) {
	req := ps.request
	
	ps.span.SetAttributes(
		semconv.HTTPMethodKey.String(req.Method),
		semconv.HTTPURLKey.String(req.URL.String()),
		semconv.HTTPSchemeKey.String(req.URL.Scheme),
		semconv.HTTPHostKey.String(req.Host),
		semconv.HTTPTargetKey.String(req.URL.Path),
		semconv.HTTPUserAgentKey.String(req.UserAgent()),
		semconv.HTTPRequestContentLengthKey.Int64(req.ContentLength),
	)

	if req.URL.RawQuery != "" {
		ps.span.SetAttributes(attribute.String("http.query", req.URL.RawQuery))
	}

	if req.Referer() != "" {
		ps.span.SetAttributes(attribute.String("http.request.header.referer", req.Referer()))
	}

	clientIP := te.extractClientIP(req)
	if clientIP != "" {
		ps.span.SetAttributes(semconv.HTTPClientIPKey.String(clientIP))
	}

	for _, header := range te.config.HeaderCapture.RequestHeaders {
		if value := req.Header.Get(header); value != "" {
			if !te.isSensitiveHeader(header) {
				truncatedValue := te.truncateHeaderValue(value)
				ps.span.SetAttributes(attribute.String("http.request.header."+header, truncatedValue))
			}
		}
	}
}

func (te *TracingEngine) setServiceAttributes(ps *ProxySpan) {
	if ps.service != nil {
		ps.span.SetAttributes(
			attribute.String("service.name", ps.service.Name),
			attribute.String("service.host", ps.service.Host),
			attribute.Int("service.port", ps.service.Port),
			attribute.String("service.scheme", ps.service.Scheme),
			attribute.Bool("service.healthy", ps.service.Healthy),
		)
	}
}

func (te *TracingEngine) FinishSpan(ps *ProxySpan, resp *http.Response, err error) {
	ps.response = resp

	if resp != nil {
		te.setResponseAttributes(ps)
	}

	if err != nil {
		te.setErrorAttributes(ps, err)
	}

	for extractor := range te.config.CustomAttributes {
		attrs := te.config.CustomAttributes[extractor](ps.request, ps.response, ps.service)
		ps.span.SetAttributes(attrs...)
	}

	duration := time.Since(ps.startTime)
	ps.span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))

	if err != nil {
		ps.span.SetStatus(codes.Error, err.Error())
	} else if resp != nil && resp.StatusCode >= 400 {
		ps.span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
	} else {
		ps.span.SetStatus(codes.Ok, "")
	}

	ps.span.End(oteltrace.WithTimestamp(time.Now()))
}

func (te *TracingEngine) setResponseAttributes(ps *ProxySpan) {
	resp := ps.response
	
	ps.span.SetAttributes(
		semconv.HTTPStatusCodeKey.Int(resp.StatusCode),
		semconv.HTTPResponseContentLengthKey.Int64(resp.ContentLength),
	)

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		ps.span.SetAttributes(attribute.String("http.response.content_type", contentType))
	}

	for _, header := range te.config.HeaderCapture.ResponseHeaders {
		if value := resp.Header.Get(header); value != "" {
			if !te.isSensitiveHeader(header) {
				truncatedValue := te.truncateHeaderValue(value)
				ps.span.SetAttributes(attribute.String("http.response.header."+header, truncatedValue))
			}
		}
	}
}

func (te *TracingEngine) setErrorAttributes(ps *ProxySpan, err error) {
	ps.span.SetAttributes(
		attribute.String("error.type", fmt.Sprintf("%T", err)),
		attribute.String("error.message", err.Error()),
		attribute.Bool("error", true),
	)
}

func (te *TracingEngine) extractClientIP(req *http.Request) string {
	forwarded := req.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	realIP := req.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	return req.RemoteAddr
}

func (te *TracingEngine) isSensitiveHeader(header string) bool {
	for _, sensitive := range te.config.HeaderCapture.SensitiveHeaders {
		if sensitive == header {
			return true
		}
	}
	return false
}

func (te *TracingEngine) truncateHeaderValue(value string) string {
	maxLength := te.config.HeaderCapture.MaxHeaderLength
	if maxLength > 0 && len(value) > maxLength {
		return value[:maxLength] + "..."
	}
	return value
}

func (te *TracingEngine) InjectTraceHeaders(ctx context.Context, headers http.Header) {
	te.propagator.Inject(ctx, propagation.HeaderCarrier(headers))
}

func (te *TracingEngine) ExtractTraceContext(req *http.Request) context.Context {
	return te.propagator.Extract(context.Background(), propagation.HeaderCarrier(req.Header))
}

func (te *TracingEngine) CreateChildSpan(ctx context.Context, operationName string) (context.Context, oteltrace.Span) {
	return te.tracer.Start(ctx, operationName,
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
	)
}

func (te *TracingEngine) RecordEvent(span oteltrace.Span, name string, attributes ...attribute.KeyValue) {
	span.AddEvent(name, oteltrace.WithAttributes(attributes...))
}

func (te *TracingEngine) AddSpanAttribute(span oteltrace.Span, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	default:
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

func (te *TracingEngine) GetTraceID(ctx context.Context) string {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}

func (te *TracingEngine) GetSpanID(ctx context.Context) string {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.SpanID().String()
	}
	return ""
}

func (te *TracingEngine) IsTracing(ctx context.Context) bool {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	return spanCtx.IsValid()
}

func (te *TracingEngine) Shutdown(ctx context.Context) error {
	if te.provider != nil {
		return te.provider.Shutdown(ctx)
	}
	return nil
}

func (te *TracingEngine) ForceFlush(ctx context.Context) error {
	if te.provider != nil {
		return te.provider.ForceFlush(ctx)
	}
	return nil
}

func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		ServiceName:      "marchproxy",
		ServiceVersion:   "1.0.0",
		Environment:      "production",
		ExporterType:     ExporterStdout,
		JaegerEndpoint:   "http://localhost:14268/api/traces",
		OTLPEndpoint:     "http://localhost:4318",
		SamplingRate:     0.1,
		MaxSpansPerTrace: 1000,
		ResourceAttributes: map[string]string{
			"proxy.type": "reverse",
			"proxy.mode": "http",
		},
		HeaderCapture: HeaderCaptureConfig{
			RequestHeaders:   []string{"Authorization", "Content-Type", "Accept", "User-Agent"},
			ResponseHeaders:  []string{"Content-Type", "Content-Length", "Cache-Control"},
			SensitiveHeaders: []string{"Authorization", "Cookie", "Set-Cookie", "X-API-Key"},
			MaxHeaderLength:  256,
		},
		CustomAttributes: make(map[string]AttributeExtractor),
		BatchConfig: BatchConfig{
			BatchTimeout:     5 * time.Second,
			ExportTimeout:    10 * time.Second,
			MaxBatchSize:     512,
			MaxQueueSize:     2048,
			BlockOnQueueFull: false,
		},
	}
}

func DevelopmentTracingConfig() TracingConfig {
	config := DefaultTracingConfig()
	config.Environment = "development"
	config.ExporterType = ExporterStdout
	config.SamplingRate = 1.0
	return config
}

func ProductionTracingConfig() TracingConfig {
	config := DefaultTracingConfig()
	config.Environment = "production"
	config.SamplingRate = 0.05
	config.BatchConfig.MaxBatchSize = 1024
	config.BatchConfig.MaxQueueSize = 4096
	return config
}

func UserIDAttributeExtractor(req *http.Request, resp *http.Response, service *manager.Service) []attribute.KeyValue {
	userID := req.Header.Get("X-User-ID")
	if userID == "" {
		userID = req.URL.Query().Get("user_id")
	}
	
	if userID != "" {
		return []attribute.KeyValue{
			attribute.String("user.id", userID),
		}
	}
	
	return nil
}

func APIVersionAttributeExtractor(req *http.Request, resp *http.Response, service *manager.Service) []attribute.KeyValue {
	version := req.Header.Get("API-Version")
	if version == "" {
		version = req.URL.Query().Get("version")
	}
	
	if version != "" {
		return []attribute.KeyValue{
			attribute.String("api.version", version),
		}
	}
	
	return nil
}

func TenantAttributeExtractor(req *http.Request, resp *http.Response, service *manager.Service) []attribute.KeyValue {
	tenant := req.Header.Get("X-Tenant-ID")
	if tenant == "" {
		tenant = req.URL.Query().Get("tenant")
	}
	
	if tenant != "" {
		return []attribute.KeyValue{
			attribute.String("tenant.id", tenant),
		}
	}
	
	return nil
}

func CacheAttributeExtractor(req *http.Request, resp *http.Response, service *manager.Service) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	
	if resp != nil {
		if cacheStatus := resp.Header.Get("X-Cache"); cacheStatus != "" {
			attrs = append(attrs, attribute.String("cache.status", cacheStatus))
		}
		
		if age := resp.Header.Get("Age"); age != "" {
			if ageInt, err := strconv.Atoi(age); err == nil {
				attrs = append(attrs, attribute.Int("cache.age", ageInt))
			}
		}
	}
	
	return attrs
}