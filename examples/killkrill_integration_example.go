// Example demonstrating KillKrill integration with MarchProxy
//go:build example

package main

import (
	"log"
	"time"

	"marchproxy-egress/internal/killkrill"
	"marchproxy-egress/internal/logging"
	"marchproxy-egress/internal/metrics"
)

func main() {
	// Example KillKrill configuration
	killKrillConfig := killkrill.Config{
		LogEndpoint:     "https://killkrill.example.com/api/v1/logs",
		MetricsEndpoint: "https://killkrill.example.com/api/v1/metrics",
		APIKey:          "your-api-key-here",
		SourceName:      "marchproxy-example",
		Application:     "proxy",
		Enabled:         true,
		BatchSize:       10, // Small batch for demo
		FlushInterval:   5 * time.Second,
		Timeout:         10 * time.Second,
		UseHTTP3:        true,
		TLSInsecure:     false,
	}

	// Create KillKrill client
	client, err := killkrill.NewClient(killKrillConfig)
	if err != nil {
		log.Fatalf("Failed to create KillKrill client: %v", err)
	}
	defer client.Close()

	// Create logger with KillKrill integration
	logger, err := logging.NewLoggerWithKillKrill("info", "", &killKrillConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create metrics collector with KillKrill integration
	metricsConfig := metrics.MetricsConfig{
		Namespace:        "marchproxy_example",
		CollectionInterval: 10 * time.Second,
		ExposeGoMetrics:   true,
		ExposeProcessMetrics: true,
		KillKrillConfig:  &killKrillConfig,
	}
	metricsCollector := metrics.NewMetricsCollector(metricsConfig)
	defer metricsCollector.Close()

	// Start metrics collection
	metricsCollector.StartCollection()

	// Example logging
	logger.Info("Starting MarchProxy with KillKrill integration",
		"version", "1.0.0",
		"killkrill_enabled", killKrillConfig.Enabled,
		"source_name", killKrillConfig.SourceName)

	// Example metrics
	prometheus := metricsCollector.GetPrometheus()
	prometheus.RecordRequest("GET", "/api/health", "200", "backend-1")
	prometheus.RecordRequestDuration("GET", "/api/health", "backend-1", 50*time.Millisecond)
	prometheus.SetActiveConnections(5)

	// Example direct KillKrill usage
	client.SendLog(killkrill.LogEntry{
		LogLevel:    "info",
		Message:     "Direct KillKrill log example",
		ServiceName: "marchproxy-example",
		Labels: map[string]interface{}{
			"component": "example",
			"action":    "demo",
		},
		Tags: []string{"example", "demo"},
	})

	client.SendMetric(killkrill.MetricEntry{
		Name:   "example_counter",
		Type:   "counter",
		Value:  1,
		Labels: map[string]string{"component": "example"},
		Help:   "Example counter metric",
	})

	// Let the client flush logs and metrics
	log.Println("Waiting for flush...")
	time.Sleep(10 * time.Second)

	log.Println("KillKrill integration example completed!")
}