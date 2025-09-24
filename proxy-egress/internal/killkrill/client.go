// Package killkrill provides client functionality for sending logs and metrics to KillKrill
package killkrill

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// Config holds KillKrill client configuration
type Config struct {
	LogEndpoint     string        `json:"log_endpoint"`
	MetricsEndpoint string        `json:"metrics_endpoint"`
	APIKey          string        `json:"api_key"`
	SourceName      string        `json:"source_name"`
	Application     string        `json:"application"`
	Enabled         bool          `json:"enabled"`
	BatchSize       int           `json:"batch_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	Timeout         time.Duration `json:"timeout"`
	UseHTTP3        bool          `json:"use_http3"`
	TLSInsecure     bool          `json:"tls_insecure"`
}

// LogEntry represents a single log entry for KillKrill
type LogEntry struct {
	Timestamp     string                 `json:"timestamp"`
	LogLevel      string                 `json:"log_level"`
	Message       string                 `json:"message"`
	ServiceName   string                 `json:"service_name"`
	Hostname      string                 `json:"hostname"`
	LoggerName    string                 `json:"logger_name,omitempty"`
	ThreadName    string                 `json:"thread_name,omitempty"`
	ECSVersion    string                 `json:"ecs_version"`
	Labels        map[string]interface{} `json:"labels,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	TransactionID string                 `json:"transaction_id,omitempty"`
}

// LogBatch represents a batch of logs to send to KillKrill
type LogBatch struct {
	Source      string     `json:"source"`
	Application string     `json:"application"`
	Logs        []LogEntry `json:"logs"`
}

// MetricEntry represents a single metric for KillKrill
type MetricEntry struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // counter, gauge, histogram
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Help      string                 `json:"help,omitempty"`
}

// MetricsBatch represents a batch of metrics to send to KillKrill
type MetricsBatch struct {
	Source  string        `json:"source"`
	Metrics []MetricEntry `json:"metrics"`
}

// Client is the KillKrill client
type Client struct {
	config      Config
	httpClient  *http.Client
	http3Client *http.Client
	logBuffer   []LogEntry
	metricBuffer []MetricEntry
	logMutex    sync.Mutex
	metricMutex sync.Mutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewClient creates a new KillKrill client
func NewClient(config Config) (*Client, error) {
	if !config.Enabled {
		return &Client{config: config, stopCh: make(chan struct{})}, nil
	}

	// Set defaults
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 10 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Application == "" {
		config.Application = "marchproxy-proxy"
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSInsecure,
	}

	// Create HTTP/1.1 client
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	var http3Client *http.Client
	if config.UseHTTP3 {
		// Create HTTP/3 client
		http3Transport := &http3.Transport{
			TLSClientConfig: tlsConfig,
			QUICConfig: &quic.Config{
				HandshakeIdleTimeout: 5 * time.Second,
				MaxIdleTimeout:       30 * time.Second,
			},
		}
		http3Client = &http.Client{
			Timeout:   config.Timeout,
			Transport: http3Transport,
		}
	}

	client := &Client{
		config:       config,
		httpClient:   httpClient,
		http3Client:  http3Client,
		logBuffer:    make([]LogEntry, 0, config.BatchSize),
		metricBuffer: make([]MetricEntry, 0, config.BatchSize),
		stopCh:       make(chan struct{}),
	}

	// Start flush goroutine
	client.wg.Add(1)
	go client.flushLoop()

	return client, nil
}

// Close shuts down the client
func (c *Client) Close() error {
	if !c.config.Enabled {
		return nil
	}

	close(c.stopCh)
	c.wg.Wait()

	// Flush any remaining logs and metrics
	c.flushLogs()
	c.flushMetrics()

	return nil
}

// SendLog adds a log entry to the buffer
func (c *Client) SendLog(entry LogEntry) {
	if !c.config.Enabled {
		return
	}

	c.logMutex.Lock()
	defer c.logMutex.Unlock()

	// Set defaults
	if entry.ECSVersion == "" {
		entry.ECSVersion = "8.0"
	}
	if entry.ServiceName == "" {
		entry.ServiceName = "marchproxy-proxy"
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00")
	}

	c.logBuffer = append(c.logBuffer, entry)

	// Flush if buffer is full
	if len(c.logBuffer) >= c.config.BatchSize {
		c.flushLogsLocked()
	}
}

// SendMetric adds a metric entry to the buffer
func (c *Client) SendMetric(entry MetricEntry) {
	if !c.config.Enabled {
		return
	}

	c.metricMutex.Lock()
	defer c.metricMutex.Unlock()

	// Set defaults
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00")
	}

	c.metricBuffer = append(c.metricBuffer, entry)

	// Flush if buffer is full
	if len(c.metricBuffer) >= c.config.BatchSize {
		c.flushMetricsLocked()
	}
}

// flushLoop runs the periodic flush
func (c *Client) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flushLogs()
			c.flushMetrics()
		case <-c.stopCh:
			return
		}
	}
}

// flushLogs flushes the log buffer
func (c *Client) flushLogs() {
	c.logMutex.Lock()
	defer c.logMutex.Unlock()
	c.flushLogsLocked()
}

// flushLogsLocked flushes logs (must hold logMutex)
func (c *Client) flushLogsLocked() {
	if len(c.logBuffer) == 0 {
		return
	}

	batch := LogBatch{
		Source:      c.config.SourceName,
		Application: c.config.Application,
		Logs:        make([]LogEntry, len(c.logBuffer)),
	}
	copy(batch.Logs, c.logBuffer)
	c.logBuffer = c.logBuffer[:0] // Clear buffer

	go func() {
		if err := c.sendLogBatch(batch); err != nil {
			// TODO: Add proper error handling/logging
			fmt.Printf("Failed to send log batch to KillKrill: %v\n", err)
		}
	}()
}

// flushMetrics flushes the metrics buffer
func (c *Client) flushMetrics() {
	c.metricMutex.Lock()
	defer c.metricMutex.Unlock()
	c.flushMetricsLocked()
}

// flushMetricsLocked flushes metrics (must hold metricMutex)
func (c *Client) flushMetricsLocked() {
	if len(c.metricBuffer) == 0 {
		return
	}

	batch := MetricsBatch{
		Source:  c.config.SourceName,
		Metrics: make([]MetricEntry, len(c.metricBuffer)),
	}
	copy(batch.Metrics, c.metricBuffer)
	c.metricBuffer = c.metricBuffer[:0] // Clear buffer

	go func() {
		if err := c.sendMetricBatch(batch); err != nil {
			// TODO: Add proper error handling/logging
			fmt.Printf("Failed to send metric batch to KillKrill: %v\n", err)
		}
	}()
}

// sendLogBatch sends a log batch to KillKrill
func (c *Client) sendLogBatch(batch LogBatch) error {
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal log batch: %w", err)
	}

	return c.sendRequest("POST", c.config.LogEndpoint, body)
}

// sendMetricBatch sends a metric batch to KillKrill
func (c *Client) sendMetricBatch(batch MetricsBatch) error {
	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal metric batch: %w", err)
	}

	return c.sendRequest("POST", c.config.MetricsEndpoint, body)
}

// sendRequest sends a request to KillKrill
func (c *Client) sendRequest(method, url string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.config.APIKey)

	// Choose client based on HTTP version preference
	client := c.httpClient
	if c.config.UseHTTP3 && c.http3Client != nil {
		client = c.http3Client
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("KillKrill responded with status %d", resp.StatusCode)
	}

	return nil
}