// Package metrics provides Prometheus instrumentation for the AILB service.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metric collectors for the AILB service.
type Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	ActiveRequests   prometheus.Gauge
	TokensProcessed  *prometheus.CounterVec
	ProviderErrors   *prometheus.CounterVec
}

// New creates and registers all Prometheus metrics.
func New() *Metrics {
	m := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "ailb",
				Name:      "requests_total",
				Help:      "Total number of requests by provider, model, and status.",
			},
			[]string{"provider", "model", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "ailb",
				Name:      "request_duration_seconds",
				Help:      "Request latency distribution by provider.",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
			},
			[]string{"provider", "model"},
		),
		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "ailb",
				Name:      "active_requests",
				Help:      "Number of currently in-flight requests.",
			},
		),
		TokensProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "ailb",
				Name:      "tokens_processed_total",
				Help:      "Total tokens processed by direction (input/output) and provider.",
			},
			[]string{"provider", "direction"},
		),
		ProviderErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "ailb",
				Name:      "provider_errors_total",
				Help:      "Total provider errors by provider and error type.",
			},
			[]string{"provider", "error_type"},
		),
	}

	prometheus.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.ActiveRequests,
		m.TokensProcessed,
		m.ProviderErrors,
	)

	return m
}

// Handler returns the Prometheus metrics HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordRequest records a completed request.
func (m *Metrics) RecordRequest(provider, model, status string, durationSec float64, inputTokens, outputTokens int) {
	m.RequestsTotal.WithLabelValues(provider, model, status).Inc()
	m.RequestDuration.WithLabelValues(provider, model).Observe(durationSec)

	if inputTokens > 0 {
		m.TokensProcessed.WithLabelValues(provider, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		m.TokensProcessed.WithLabelValues(provider, "output").Add(float64(outputTokens))
	}
}

// RecordError records a provider error.
func (m *Metrics) RecordError(provider, errorType string) {
	m.ProviderErrors.WithLabelValues(provider, errorType).Inc()
}
