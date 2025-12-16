package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Collector collects metrics from Envoy admin API
type Collector struct {
	adminAddr  string
	httpClient *http.Client
	logger     *logrus.Logger

	// Cached metrics
	mu             sync.RWMutex
	cachedMetrics  *Metrics
	lastCollection time.Time
	cacheTimeout   time.Duration
}

// Metrics contains performance and traffic metrics
type Metrics struct {
	Timestamp         int64                   `json:"timestamp"`
	TotalConnections  int64                   `json:"total_connections"`
	ActiveConnections int64                   `json:"active_connections"`
	TotalRequests     int64                   `json:"total_requests"`
	RequestsPerSecond int64                   `json:"requests_per_second"`
	Latency           LatencyMetrics          `json:"latency"`
	StatusCodes       map[string]int64        `json:"status_codes"`
	Routes            map[string]RouteMetrics `json:"routes"`
}

// LatencyMetrics contains latency percentiles
type LatencyMetrics struct {
	P50Ms  float64 `json:"p50_ms"`
	P90Ms  float64 `json:"p90_ms"`
	P95Ms  float64 `json:"p95_ms"`
	P99Ms  float64 `json:"p99_ms"`
	AvgMs  float64 `json:"avg_ms"`
}

// RouteMetrics contains per-route metrics
type RouteMetrics struct {
	Requests      int64   `json:"requests"`
	Errors        int64   `json:"errors"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

// NewCollector creates a new metrics collector
func NewCollector(adminAddr string, logger *logrus.Logger) *Collector {
	if logger == nil {
		logger = logrus.New()
	}

	return &Collector{
		adminAddr: adminAddr,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:       logger,
		cacheTimeout: 5 * time.Second,
	}
}

// GetMetrics retrieves current metrics from Envoy
func (c *Collector) GetMetrics() (*Metrics, error) {
	c.mu.RLock()
	if c.cachedMetrics != nil && time.Since(c.lastCollection) < c.cacheTimeout {
		metrics := c.cachedMetrics
		c.mu.RUnlock()
		return metrics, nil
	}
	c.mu.RUnlock()

	// Collect fresh metrics
	metrics, err := c.collectMetrics()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.cachedMetrics = metrics
	c.lastCollection = time.Now()
	c.mu.Unlock()

	return metrics, nil
}

// collectMetrics fetches metrics from Envoy admin API
func (c *Collector) collectMetrics() (*Metrics, error) {
	c.logger.Debug("Collecting metrics from Envoy admin API")

	// Get stats from Envoy admin endpoint
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s/stats?format=json", c.adminAddr))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("admin API returned %d: %s", resp.StatusCode, body)
	}

	var envoyStats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&envoyStats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Parse Envoy stats into our metrics format
	metrics := &Metrics{
		Timestamp:    time.Now().Unix(),
		StatusCodes:  make(map[string]int64),
		Routes:       make(map[string]RouteMetrics),
	}

	// Extract key metrics (simplified - real implementation would parse envoy stats)
	metrics.TotalConnections = c.getStatInt(envoyStats, "downstream_cx_total", 0)
	metrics.ActiveConnections = c.getStatInt(envoyStats, "downstream_cx_active", 0)
	metrics.TotalRequests = c.getStatInt(envoyStats, "downstream_rq_total", 0)

	// Calculate RPS from delta
	if c.cachedMetrics != nil {
		timeDelta := float64(metrics.Timestamp - c.cachedMetrics.Timestamp)
		if timeDelta > 0 {
			reqDelta := metrics.TotalRequests - c.cachedMetrics.TotalRequests
			metrics.RequestsPerSecond = int64(float64(reqDelta) / timeDelta)
		}
	}

	// Extract latency metrics (simplified)
	metrics.Latency = LatencyMetrics{
		P50Ms: c.getStatFloat(envoyStats, "downstream_rq_time.p50", 0),
		P90Ms: c.getStatFloat(envoyStats, "downstream_rq_time.p90", 0),
		P95Ms: c.getStatFloat(envoyStats, "downstream_rq_time.p95", 0),
		P99Ms: c.getStatFloat(envoyStats, "downstream_rq_time.p99", 0),
		AvgMs: c.getStatFloat(envoyStats, "downstream_rq_time.avg", 0),
	}

	// Extract status code distribution
	statusCodes := []string{"200", "201", "204", "301", "302", "400", "401", "403", "404", "500", "502", "503"}
	for _, code := range statusCodes {
		metrics.StatusCodes[code] = c.getStatInt(envoyStats, fmt.Sprintf("downstream_rq_%s", code), 0)
	}

	return metrics, nil
}

// getStatInt safely extracts an int64 stat value
func (c *Collector) getStatInt(stats map[string]interface{}, key string, defaultVal int64) int64 {
	if val, ok := stats[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		}
	}
	return defaultVal
}

// getStatFloat safely extracts a float64 stat value
func (c *Collector) getStatFloat(stats map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := stats[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return defaultVal
}

// Reset resets cached metrics
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cachedMetrics = nil
	c.lastCollection = time.Time{}
}
