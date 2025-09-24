package killkrill

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// LogrusToKillKrill converts a logrus entry to KillKrill format
func LogrusToKillKrill(entry *logrus.Entry) LogEntry {
	hostname, _ := os.Hostname()

	// Convert logrus level to string
	level := strings.ToLower(entry.Level.String())

	// Extract fields for labels
	labels := make(map[string]interface{})
	tags := make([]string, 0)

	for key, value := range entry.Data {
		switch key {
		case "trace_id", "span_id", "transaction_id":
			// These will be set directly on the LogEntry
			continue
		case "tags":
			if tagSlice, ok := value.([]string); ok {
				tags = tagSlice
			}
		default:
			labels[key] = value
		}
	}

	logEntry := LogEntry{
		Timestamp:   entry.Time.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		LogLevel:    level,
		Message:     entry.Message,
		ServiceName: "marchproxy-proxy",
		Hostname:    hostname,
		ECSVersion:  "8.0",
		Labels:      labels,
		Tags:        tags,
	}

	// Set trace information if available
	if traceID, ok := entry.Data["trace_id"].(string); ok {
		logEntry.TraceID = traceID
	}
	if spanID, ok := entry.Data["span_id"].(string); ok {
		logEntry.SpanID = spanID
	}
	if txnID, ok := entry.Data["transaction_id"].(string); ok {
		logEntry.TransactionID = txnID
	}

	return logEntry
}

// PrometheusToKillKrill converts Prometheus metrics to KillKrill format
func PrometheusToKillKrill(metricFamily *dto.MetricFamily) []MetricEntry {
	var entries []MetricEntry

	metricName := metricFamily.GetName()
	help := metricFamily.GetHelp()

	for _, metric := range metricFamily.GetMetric() {
		labels := make(map[string]string)
		for _, labelPair := range metric.GetLabel() {
			labels[labelPair.GetName()] = labelPair.GetValue()
		}

		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00")
		if metric.GetTimestampMs() > 0 {
			timestamp = time.Unix(0, metric.GetTimestampMs()*1000000).UTC().Format("2006-01-02T15:04:05.000Z07:00")
		}

		switch metricFamily.GetType() {
		case dto.MetricType_COUNTER:
			entries = append(entries, MetricEntry{
				Name:      metricName,
				Type:      "counter",
				Value:     metric.GetCounter().GetValue(),
				Labels:    labels,
				Timestamp: timestamp,
				Help:      help,
			})

		case dto.MetricType_GAUGE:
			entries = append(entries, MetricEntry{
				Name:      metricName,
				Type:      "gauge",
				Value:     metric.GetGauge().GetValue(),
				Labels:    labels,
				Timestamp: timestamp,
				Help:      help,
			})

		case dto.MetricType_HISTOGRAM:
			histogram := metric.GetHistogram()

			// Send histogram buckets
			for _, bucket := range histogram.GetBucket() {
				bucketLabels := make(map[string]string)
				for k, v := range labels {
					bucketLabels[k] = v
				}
				bucketLabels["le"] = fmt.Sprintf("%g", bucket.GetUpperBound())

				entries = append(entries, MetricEntry{
					Name:      metricName + "_bucket",
					Type:      "counter",
					Value:     float64(bucket.GetCumulativeCount()),
					Labels:    bucketLabels,
					Timestamp: timestamp,
					Help:      help + " (bucket)",
				})
			}

			// Send histogram count
			countLabels := make(map[string]string)
			for k, v := range labels {
				countLabels[k] = v
			}
			entries = append(entries, MetricEntry{
				Name:      metricName + "_count",
				Type:      "counter",
				Value:     float64(histogram.GetSampleCount()),
				Labels:    countLabels,
				Timestamp: timestamp,
				Help:      help + " (count)",
			})

			// Send histogram sum
			sumLabels := make(map[string]string)
			for k, v := range labels {
				sumLabels[k] = v
			}
			entries = append(entries, MetricEntry{
				Name:      metricName + "_sum",
				Type:      "counter",
				Value:     histogram.GetSampleSum(),
				Labels:    sumLabels,
				Timestamp: timestamp,
				Help:      help + " (sum)",
			})

		case dto.MetricType_SUMMARY:
			summary := metric.GetSummary()

			// Send summary quantiles
			for _, quantile := range summary.GetQuantile() {
				quantileLabels := make(map[string]string)
				for k, v := range labels {
					quantileLabels[k] = v
				}
				quantileLabels["quantile"] = fmt.Sprintf("%g", quantile.GetQuantile())

				entries = append(entries, MetricEntry{
					Name:      metricName,
					Type:      "gauge",
					Value:     quantile.GetValue(),
					Labels:    quantileLabels,
					Timestamp: timestamp,
					Help:      help + " (quantile)",
				})
			}

			// Send summary count
			countLabels := make(map[string]string)
			for k, v := range labels {
				countLabels[k] = v
			}
			entries = append(entries, MetricEntry{
				Name:      metricName + "_count",
				Type:      "counter",
				Value:     float64(summary.GetSampleCount()),
				Labels:    countLabels,
				Timestamp: timestamp,
				Help:      help + " (count)",
			})

			// Send summary sum
			sumLabels := make(map[string]string)
			for k, v := range labels {
				sumLabels[k] = v
			}
			entries = append(entries, MetricEntry{
				Name:      metricName + "_sum",
				Type:      "counter",
				Value:     summary.GetSampleSum(),
				Labels:    sumLabels,
				Timestamp: timestamp,
				Help:      help + " (sum)",
			})
		}
	}

	return entries
}

// DirectMetricEntry creates a KillKrill metric entry directly
func DirectMetricEntry(name, metricType string, value float64, labels map[string]string, help string) MetricEntry {
	return MetricEntry{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		Help:      help,
	}
}

// GatherMetricsFromRegistry converts all metrics from a Prometheus registry
func GatherMetricsFromRegistry(registry *prometheus.Registry) ([]MetricEntry, error) {
	var allEntries []MetricEntry

	metricFamilies, err := registry.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}

	for _, mf := range metricFamilies {
		entries := PrometheusToKillKrill(mf)
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}