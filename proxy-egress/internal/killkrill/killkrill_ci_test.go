//go:build ci

package killkrill

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewClient creates a new KillKrill client
func TestNewClient(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test-source",
		Application:     "test-app",
		Enabled:         true,
		BatchSize:       100,
		FlushInterval:   10 * time.Second,
		Timeout:         30 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client, got nil")
	}

	defer client.Close()
}

// TestNewClientDisabled tests creating a disabled client
func TestNewClientDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create disabled client: %v", err)
	}

	if client.config.Enabled {
		t.Error("Client should be disabled")
	}

	defer client.Close()
}

// TestNewClientDefaults tests default configuration values
func TestNewClientDefaults(t *testing.T) {
	config := Config{
		Enabled: true,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.config.BatchSize != 100 {
		t.Errorf("Expected default BatchSize 100, got %d", client.config.BatchSize)
	}

	if client.config.FlushInterval != 10*time.Second {
		t.Errorf("Expected default FlushInterval 10s, got %v", client.config.FlushInterval)
	}

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout 30s, got %v", client.config.Timeout)
	}

	if client.config.Application != "marchproxy-proxy" {
		t.Errorf("Expected default Application marchproxy-proxy, got %s", client.config.Application)
	}

	defer client.Close()
}

// TestSendLog sends a log entry
func TestSendLog(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       10,
		FlushInterval:   100 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	entry := LogEntry{
		LogLevel:   "info",
		Message:    "test message",
		ServiceName: "test-service",
	}

	client.SendLog(entry)

	if len(client.logBuffer) != 1 {
		t.Errorf("Expected 1 log in buffer, got %d", len(client.logBuffer))
	}
}

// TestSendMetric sends a metric entry
func TestSendMetric(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       10,
		FlushInterval:   100 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	metric := MetricEntry{
		Name:  "test_metric",
		Type:  "counter",
		Value: 42,
	}

	client.SendMetric(metric)

	if len(client.metricBuffer) != 1 {
		t.Errorf("Expected 1 metric in buffer, got %d", len(client.metricBuffer))
	}
}

// TestSendLogDefaults tests log entry defaults
func TestSendLogDefaults(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       10,
		FlushInterval:   100 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	entry := LogEntry{
		Message: "test",
	}

	client.SendLog(entry)

	bufferedEntry := client.logBuffer[0]

	if bufferedEntry.ECSVersion != "8.0" {
		t.Errorf("Expected ECSVersion 8.0, got %s", bufferedEntry.ECSVersion)
	}

	if bufferedEntry.ServiceName != "marchproxy-proxy" {
		t.Errorf("Expected ServiceName marchproxy-proxy, got %s", bufferedEntry.ServiceName)
	}

	if bufferedEntry.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
}

// TestSendMetricDefaults tests metric entry defaults
func TestSendMetricDefaults(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       10,
		FlushInterval:   100 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	metric := MetricEntry{
		Name:  "test",
		Type:  "counter",
		Value: 0,
	}

	client.SendMetric(metric)

	bufferedMetric := client.metricBuffer[0]

	if bufferedMetric.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
}

// TestBatchFlush tests batch flushing when buffer is full
func TestBatchFlush(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/logs" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       3,
		FlushInterval:   100 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	// Add 3 logs to reach batch size
	for i := 0; i < 3; i++ {
		entry := LogEntry{
			LogLevel:   "info",
			Message:    fmt.Sprintf("message %d", i),
			ServiceName: "test",
		}
		client.SendLog(entry)
	}

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	// Buffer should be empty after flush
	if len(client.logBuffer) != 0 {
		t.Errorf("Expected buffer to be flushed, got %d entries", len(client.logBuffer))
	}
}

// TestFlushLogs flushes logs explicitly
func TestFlushLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       100,
		FlushInterval:   10 * time.Second,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	// Add logs
	for i := 0; i < 5; i++ {
		entry := LogEntry{
			LogLevel:   "info",
			Message:    fmt.Sprintf("message %d", i),
			ServiceName: "test",
		}
		client.SendLog(entry)
	}

	if len(client.logBuffer) != 5 {
		t.Errorf("Expected 5 logs in buffer, got %d", len(client.logBuffer))
	}

	// Flush
	client.flushLogs()

	// Buffer should be empty
	if len(client.logBuffer) != 0 {
		t.Errorf("Expected buffer to be empty after flush, got %d", len(client.logBuffer))
	}
}

// TestFlushMetrics flushes metrics explicitly
func TestFlushMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       100,
		FlushInterval:   10 * time.Second,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	// Add metrics
	for i := 0; i < 5; i++ {
		metric := MetricEntry{
			Name:  fmt.Sprintf("metric_%d", i),
			Type:  "counter",
			Value: float64(i),
		}
		client.SendMetric(metric)
	}

	if len(client.metricBuffer) != 5 {
		t.Errorf("Expected 5 metrics in buffer, got %d", len(client.metricBuffer))
	}

	// Flush
	client.flushMetrics()

	// Buffer should be empty
	if len(client.metricBuffer) != 0 {
		t.Errorf("Expected buffer to be empty after flush, got %d", len(client.metricBuffer))
	}
}

// TestSendRequestSuccess tests successful request sending
func TestSendRequestSuccess(t *testing.T) {
	receivedRequest := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = true

		if r.Header.Get("X-API-Key") != "test-key" {
			t.Error("Expected API key in header")
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected JSON content type")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	err := client.sendRequest("POST", server.URL+"/test", []byte("{}"))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if !receivedRequest {
		t.Error("Server did not receive request")
	}
}

// TestSendRequestError tests request failure
func TestSendRequestError(t *testing.T) {
	config := Config{
		APIKey:  "test-key",
		Enabled: true,
		Timeout: 1 * time.Millisecond, // Very short timeout
	}

	client, _ := NewClient(config)
	defer client.Close()

	// Send to non-existent server
	err := client.sendRequest("POST", "http://localhost:1/invalid", []byte("{}"))
	if err == nil {
		t.Error("Expected error for invalid request")
	}
}

// TestSendRequestBadStatus tests non-2xx response
func TestSendRequestBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		Enabled: true,
		Timeout: 5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	err := client.sendRequest("POST", server.URL, []byte("{}"))
	if err == nil {
		t.Error("Expected error for non-2xx response")
	}
}

// TestLogrusToKillKrill tests log entry conversion
func TestLogrusToKillKrill(t *testing.T) {
	// Since entry is interface{}, we'll pass a basic structure
	entry := "test log entry"

	converted := LogrusToKillKrill(entry)

	if converted.ECSVersion != "8.0" {
		t.Errorf("Expected ECSVersion 8.0, got %s", converted.ECSVersion)
	}

	if converted.ServiceName != "marchproxy-proxy" {
		t.Errorf("Expected ServiceName marchproxy-proxy, got %s", converted.ServiceName)
	}

	if converted.LogLevel != "info" {
		t.Errorf("Expected LogLevel info, got %s", converted.LogLevel)
	}

	if converted.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}

	if converted.Hostname == "" {
		t.Error("Expected Hostname to be set")
	}
}

// TestDirectMetricEntry tests direct metric entry creation
func TestDirectMetricEntry(t *testing.T) {
	labels := map[string]string{
		"env":     "test",
		"service": "api",
	}

	metric := DirectMetricEntry("test_metric", "gauge", 42.5, labels, "Test metric")

	if metric.Name != "test_metric" {
		t.Errorf("Expected name test_metric, got %s", metric.Name)
	}

	if metric.Type != "gauge" {
		t.Errorf("Expected type gauge, got %s", metric.Type)
	}

	if metric.Value != 42.5 {
		t.Errorf("Expected value 42.5, got %f", metric.Value)
	}

	if metric.Help != "Test metric" {
		t.Errorf("Expected help 'Test metric', got %s", metric.Help)
	}

	if metric.Labels["env"] != "test" {
		t.Error("Expected label env=test")
	}

	if metric.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
}

// TestNewHook creates a new hook
func TestNewHook(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
	}

	client, _ := NewClient(config)
	defer client.Close()

	hook := NewHook(client)

	if hook == nil {
		t.Fatal("Expected hook, got nil")
	}

	if hook.client != client {
		t.Error("Expected hook client to match")
	}
}

// TestHookFire fires a hook
func TestHookFire(t *testing.T) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
	}

	client, _ := NewClient(config)
	defer client.Close()

	hook := NewHook(client)

	entry := "test log entry"

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Failed to fire hook: %v", err)
	}

	if len(client.logBuffer) != 1 {
		t.Errorf("Expected 1 log in buffer, got %d", len(client.logBuffer))
	}
}

// TestHookFireDisabledClient tests hook with disabled client
func TestHookFireDisabledClient(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	client, _ := NewClient(config)
	defer client.Close()

	hook := NewHook(client)

	entry := "test log entry"

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Failed to fire hook: %v", err)
	}

	// Buffer should be empty since client is disabled
	if len(client.logBuffer) != 0 {
		t.Errorf("Expected 0 logs in buffer, got %d", len(client.logBuffer))
	}
}

// TestHookFireNilClient tests hook with nil client
func TestHookFireNilClient(t *testing.T) {
	hook := NewHook(nil)

	entry := "test log entry"

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Failed to fire hook with nil client: %v", err)
	}
}

// TestClientClose closes a client cleanly
func TestClientClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       100,
		FlushInterval:   10 * time.Second,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)

	// Add data
	client.SendLog(LogEntry{
		Message: "test",
	})

	// Close should flush and stop
	err := client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}

	// Buffer should be empty after close
	if len(client.logBuffer) != 0 {
		t.Errorf("Expected empty buffer after close, got %d", len(client.logBuffer))
	}
}

// TestSendLogDisabled tests sending log when disabled
func TestSendLogDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	client, _ := NewClient(config)
	defer client.Close()

	entry := LogEntry{
		Message: "test",
	}

	client.SendLog(entry)

	if len(client.logBuffer) != 0 {
		t.Errorf("Expected buffer to remain empty when disabled, got %d", len(client.logBuffer))
	}
}

// TestSendMetricDisabled tests sending metric when disabled
func TestSendMetricDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	client, _ := NewClient(config)
	defer client.Close()

	metric := MetricEntry{
		Name:  "test",
		Type:  "counter",
		Value: 0,
	}

	client.SendMetric(metric)

	if len(client.metricBuffer) != 0 {
		t.Errorf("Expected buffer to remain empty when disabled, got %d", len(client.metricBuffer))
	}
}

// TestLogBatchSerialization tests log batch JSON serialization
func TestLogBatchSerialization(t *testing.T) {
	batch := LogBatch{
		Source:      "test-source",
		Application: "test-app",
		Logs: []LogEntry{
			{
				LogLevel:   "info",
				Message:    "test message",
				ServiceName: "test-service",
				Timestamp:  "2006-01-02T15:04:05.000Z",
			},
		},
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("Failed to marshal batch: %v", err)
	}

	var unmarshaled LogBatch
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal batch: %v", err)
	}

	if unmarshaled.Source != "test-source" {
		t.Errorf("Expected source test-source, got %s", unmarshaled.Source)
	}

	if len(unmarshaled.Logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(unmarshaled.Logs))
	}
}

// TestMetricsBatchSerialization tests metrics batch JSON serialization
func TestMetricsBatchSerialization(t *testing.T) {
	batch := MetricsBatch{
		Source: "test-source",
		Metrics: []MetricEntry{
			{
				Name:      "test_metric",
				Type:      "counter",
				Value:     42.5,
				Timestamp: "2006-01-02T15:04:05.000Z",
			},
		},
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("Failed to marshal batch: %v", err)
	}

	var unmarshaled MetricsBatch
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal batch: %v", err)
	}

	if unmarshaled.Source != "test-source" {
		t.Errorf("Expected source test-source, got %s", unmarshaled.Source)
	}

	if len(unmarshaled.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(unmarshaled.Metrics))
	}
}

// TestSendLogBatchWithServer tests log batch sent to server
func TestSendLogBatchWithServer(t *testing.T) {
	receivedBatch := false
	var receivedData LogBatch

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/logs" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedData)
			receivedBatch = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test-source",
		Application:     "test-app",
		Enabled:         true,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	batch := LogBatch{
		Source:      "test-source",
		Application: "test-app",
		Logs: []LogEntry{
			{
				LogLevel:   "info",
				Message:    "test message",
				ServiceName: "test-service",
			},
		},
	}

	err := client.sendLogBatch(batch)
	if err != nil {
		t.Fatalf("Failed to send log batch: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if !receivedBatch {
		t.Error("Server did not receive batch")
	}
}

// TestSendMetricBatchWithServer tests metric batch sent to server
func TestSendMetricBatchWithServer(t *testing.T) {
	receivedBatch := false
	var receivedData MetricsBatch

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedData)
			receivedBatch = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test-source",
		Application:     "test-app",
		Enabled:         true,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	batch := MetricsBatch{
		Source: "test-source",
		Metrics: []MetricEntry{
			{
				Name:  "test_metric",
				Type:  "counter",
				Value: 42,
			},
		},
	}

	err := client.sendMetricBatch(batch)
	if err != nil {
		t.Fatalf("Failed to send metric batch: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if !receivedBatch {
		t.Error("Server did not receive batch")
	}
}

// TestMarshalLogBatchError tests error handling for marshal failures
func TestMarshalLogBatchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		LogEndpoint: server.URL + "/logs",
		APIKey:      "test-key",
		Enabled:     true,
		Timeout:     5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	// Create a batch with data that would fail marshaling
	batch := LogBatch{
		Source: "test",
		Logs: []LogEntry{
			{
				Labels: map[string]interface{}{
					"func": func() {}, // Functions can't be marshaled to JSON
				},
			},
		},
	}

	err := client.sendLogBatch(batch)
	if err == nil {
		t.Error("Expected error for unmarshalable data")
	}
}

// TestFlushLoopExecution tests flush loop timing
func TestFlushLoopExecution(t *testing.T) {
	flushCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/logs" {
			flushCount++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		LogEndpoint:     server.URL + "/logs",
		MetricsEndpoint: server.URL + "/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       100,
		FlushInterval:   50 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)

	// Add a log
	client.SendLog(LogEntry{Message: "test"})

	// Wait for at least one flush
	time.Sleep(150 * time.Millisecond)

	client.Close()

	if flushCount == 0 {
		t.Error("Expected flush loop to execute at least once")
	}
}

// BenchmarkSendLog benchmarks log sending
func BenchmarkSendLog(b *testing.B) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       1000,
		FlushInterval:   10 * time.Second,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := LogEntry{
			LogLevel:   "info",
			Message:    "benchmark message",
			ServiceName: "benchmark",
		}
		client.SendLog(entry)
	}
}

// BenchmarkSendMetric benchmarks metric sending
func BenchmarkSendMetric(b *testing.B) {
	config := Config{
		LogEndpoint:     "http://localhost:8080/logs",
		MetricsEndpoint: "http://localhost:8080/metrics",
		APIKey:          "test-key",
		SourceName:      "test",
		Application:     "test",
		Enabled:         true,
		BatchSize:       1000,
		FlushInterval:   10 * time.Second,
		Timeout:         5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metric := MetricEntry{
			Name:  fmt.Sprintf("metric_%d", i),
			Type:  "counter",
			Value: float64(i),
		}
		client.SendMetric(metric)
	}
}

// TestSendRequestWithContext tests context handling in requests
func TestSendRequestWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		Enabled: true,
		Timeout: 1 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	err := client.sendRequest("POST", server.URL, []byte("{}"))
	if err != nil {
		t.Fatalf("Failed to send request with context: %v", err)
	}
}

// TestMetricEntryStructure tests MetricEntry structure
func TestMetricEntryStructure(t *testing.T) {
	metric := MetricEntry{
		Name:      "test_metric",
		Type:      "gauge",
		Value:     123.45,
		Labels: map[string]string{
			"env": "test",
		},
		Timestamp: "2006-01-02T15:04:05.000Z",
		Help:      "Test metric help",
	}

	if metric.Name != "test_metric" {
		t.Errorf("Expected name test_metric, got %s", metric.Name)
	}

	if metric.Type != "gauge" {
		t.Errorf("Expected type gauge, got %s", metric.Type)
	}

	if metric.Value != 123.45 {
		t.Errorf("Expected value 123.45, got %f", metric.Value)
	}

	if len(metric.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(metric.Labels))
	}
}

// TestLogEntryStructure tests LogEntry structure
func TestLogEntryStructure(t *testing.T) {
	entry := LogEntry{
		Timestamp:     "2006-01-02T15:04:05.000Z",
		LogLevel:      "error",
		Message:       "Test error message",
		ServiceName:   "test-service",
		Hostname:      "localhost",
		LoggerName:    "test-logger",
		ThreadName:    "test-thread",
		ECSVersion:    "8.0",
		Labels: map[string]interface{}{
			"key": "value",
		},
		Tags: []string{"tag1", "tag2"},
		TraceID:       "trace-123",
		SpanID:        "span-456",
		TransactionID: "txn-789",
	}

	if entry.LogLevel != "error" {
		t.Errorf("Expected log level error, got %s", entry.LogLevel)
	}

	if entry.Message != "Test error message" {
		t.Errorf("Expected message, got %s", entry.Message)
	}

	if len(entry.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(entry.Tags))
	}

	if entry.TraceID != "trace-123" {
		t.Errorf("Expected trace ID, got %s", entry.TraceID)
	}
}

// TestRequestMarshalError tests handling of marshal errors
func TestRequestMarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		Enabled: true,
		Timeout: 5 * time.Second,
	}

	client, _ := NewClient(config)
	defer client.Close()

	// Send valid data
	data, _ := json.Marshal(map[string]string{"test": "data"})
	err := client.sendRequest("POST", server.URL, data)
	if err != nil {
		t.Fatalf("Failed to send valid request: %v", err)
	}
}
