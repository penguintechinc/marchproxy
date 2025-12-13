package observability

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer manages distributed tracing
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	logger   *logrus.Logger
}

// NewTracer creates a new OpenTelemetry tracer
func NewTracer(serviceName, endpoint string, sampleRate float64, logger *logrus.Logger) (*Tracer, error) {
	ctx := context.Background()

	// For now, use a no-op exporter since OTLP requires additional configuration
	// In production, this would be configured with actual Jaeger/Tempo endpoints
	_ = endpoint // Placeholder for future OTLP configuration

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with sampling (no exporter for now)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
	)

	// Register as global provider
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(serviceName)

	logger.WithFields(logrus.Fields{
		"service":     serviceName,
		"endpoint":    endpoint,
		"sample_rate": sampleRate,
	}).Info("OpenTelemetry tracer initialized")

	return &Tracer{
		provider: tp,
		tracer:   tracer,
		logger:   logger,
	}, nil
}

// StartSpan starts a new tracing span
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// Shutdown gracefully shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		if err := t.provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer: %w", err)
		}
		t.logger.Info("Tracer shutdown complete")
	}
	return nil
}

// GetTracer returns the underlying OpenTelemetry tracer
func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}

// TraceConnection creates a span for a connection
func (t *Tracer) TraceConnection(ctx context.Context, srcIP, dstIP string) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "proxy.connection")
	span.SetAttributes(
		semconv.NetPeerIPKey.String(srcIP),
		semconv.NetHostIPKey.String(dstIP),
	)
	return ctx, span
}

// TraceRouting creates a span for routing decisions
func (t *Tracer) TraceRouting(ctx context.Context, algorithm, backend string) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "proxy.routing")
	span.SetAttributes(
		semconv.HTTPRouteKey.String(algorithm),
		semconv.PeerServiceKey.String(backend),
	)
	return ctx, span
}
