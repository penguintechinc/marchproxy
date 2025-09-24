// Package test provides load testing for MarchProxy dual proxy architecture
package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadTestConfig holds configuration for load tests
type LoadTestConfig struct {
	BaseURL         string
	Duration        time.Duration
	Concurrency     int
	RequestsPerSec  int
	TimeoutPerReq   time.Duration
	KeepAlive       bool
	TLSInsecure     bool
}

// LoadTestResults holds the results of a load test
type LoadTestResults struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedRequests   int64
	AverageLatency   time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	RequestsPerSec   float64
	ThroughputMBps   float64
	ErrorRate        float64
	StatusCodes      map[int]int64
	Errors           map[string]int64
	LatencyHistogram []time.Duration
}

// LoadTester manages load testing execution
type LoadTester struct {
	config  LoadTestConfig
	client  *http.Client
	results *LoadTestResults
	mutex   sync.RWMutex
}

// NewLoadTester creates a new load tester
func NewLoadTester(config LoadTestConfig) *LoadTester {
	transport := &http.Transport{
		MaxIdleConns:        config.Concurrency * 2,
		MaxIdleConnsPerHost: config.Concurrency,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   !config.KeepAlive,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.TLSInsecure,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.TimeoutPerReq,
	}

	return &LoadTester{
		config: config,
		client: client,
		results: &LoadTestResults{
			StatusCodes: make(map[int]int64),
			Errors:      make(map[string]int64),
		},
	}
}

// ExecuteLoadTest runs a load test with the given configuration
func (lt *LoadTester) ExecuteLoadTest(ctx context.Context, endpoint string, method string, body []byte) *LoadTestResults {
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(ctx, lt.config.Duration)
	defer cancel()

	// Create worker channels
	requestChan := make(chan struct{}, lt.config.RequestsPerSec)
	resultChan := make(chan *RequestResult, lt.config.Concurrency*10)

	// Start result collector
	var wg sync.WaitGroup
	wg.Add(1)
	go lt.collectResults(resultChan, &wg)

	// Start rate limiter if needed
	if lt.config.RequestsPerSec > 0 {
		wg.Add(1)
		go lt.rateLimiter(ctx, requestChan, &wg)
	}

	// Start workers
	workerWg := sync.WaitGroup{}
	for i := 0; i < lt.config.Concurrency; i++ {
		workerWg.Add(1)
		go lt.worker(ctx, requestChan, resultChan, endpoint, method, body, &workerWg)
	}

	// Wait for workers to complete
	workerWg.Wait()
	close(resultChan)

	// Wait for result collection to complete
	wg.Wait()

	// Calculate final metrics
	lt.calculateFinalMetrics(time.Since(startTime))

	return lt.results
}

// RequestResult holds the result of a single request
type RequestResult struct {
	StatusCode   int
	Latency      time.Duration
	BytesRead    int64
	Error        error
	Timestamp    time.Time
}

// worker executes individual requests
func (lt *LoadTester) worker(ctx context.Context, requestChan <-chan struct{}, resultChan chan<- *RequestResult, endpoint, method string, body []byte, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-requestChan:
			if !ok {
				return
			}

			result := lt.executeRequest(ctx, endpoint, method, body)
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// executeRequest performs a single HTTP request
func (lt *LoadTester) executeRequest(ctx context.Context, endpoint, method string, body []byte) *RequestResult {
	start := time.Now()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, lt.config.BaseURL+endpoint, bodyReader)
	if err != nil {
		return &RequestResult{
			Error:     err,
			Latency:   time.Since(start),
			Timestamp: start,
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := lt.client.Do(req)
	if err != nil {
		return &RequestResult{
			Error:     err,
			Latency:   time.Since(start),
			Timestamp: start,
		}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	return &RequestResult{
		StatusCode: resp.StatusCode,
		Latency:    time.Since(start),
		BytesRead:  int64(len(bodyBytes)),
		Timestamp:  start,
	}
}

// rateLimiter controls the rate of requests
func (lt *LoadTester) rateLimiter(ctx context.Context, requestChan chan<- struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(requestChan)

	interval := time.Second / time.Duration(lt.config.RequestsPerSec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case requestChan <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// collectResults processes and aggregates request results
func (lt *LoadTester) collectResults(resultChan <-chan *RequestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for result := range resultChan {
		lt.mutex.Lock()

		atomic.AddInt64(&lt.results.TotalRequests, 1)

		if result.Error != nil {
			atomic.AddInt64(&lt.results.FailedRequests, 1)
			lt.results.Errors[result.Error.Error()]++
		} else {
			atomic.AddInt64(&lt.results.SuccessfulReqs, 1)
			lt.results.StatusCodes[result.StatusCode]++
		}

		// Store latency for percentile calculations
		lt.results.LatencyHistogram = append(lt.results.LatencyHistogram, result.Latency)

		lt.mutex.Unlock()
	}
}

// calculateFinalMetrics computes final test metrics
func (lt *LoadTester) calculateFinalMetrics(totalDuration time.Duration) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	if len(lt.results.LatencyHistogram) == 0 {
		return
	}

	// Sort latencies for percentile calculations
	latencies := make([]time.Duration, len(lt.results.LatencyHistogram))
	copy(latencies, lt.results.LatencyHistogram)

	// Simple bubble sort for demonstration (would use sort.Slice in production)
	for i := 0; i < len(latencies); i++ {
		for j := 0; j < len(latencies)-1-i; j++ {
			if latencies[j] > latencies[j+1] {
				latencies[j], latencies[j+1] = latencies[j+1], latencies[j]
			}
		}
	}

	// Calculate metrics
	lt.results.MinLatency = latencies[0]
	lt.results.MaxLatency = latencies[len(latencies)-1]

	// Calculate average
	var totalLatency time.Duration
	for _, latency := range latencies {
		totalLatency += latency
	}
	lt.results.AverageLatency = totalLatency / time.Duration(len(latencies))

	// Calculate percentiles
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)
	if p95Index < len(latencies) {
		lt.results.P95Latency = latencies[p95Index]
	}
	if p99Index < len(latencies) {
		lt.results.P99Latency = latencies[p99Index]
	}

	// Calculate throughput
	lt.results.RequestsPerSec = float64(lt.results.TotalRequests) / totalDuration.Seconds()
	lt.results.ErrorRate = float64(lt.results.FailedRequests) / float64(lt.results.TotalRequests) * 100

	// Calculate data throughput (simplified)
	var totalBytes int64
	for _, latency := range lt.results.LatencyHistogram {
		totalBytes += 1024 // Assume average response size
	}
	lt.results.ThroughputMBps = float64(totalBytes) / (1024 * 1024) / totalDuration.Seconds()
}

// PrintResults displays load test results
func (lt *LoadTester) PrintResults() {
	results := lt.results

	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Total Requests: %d\n", results.TotalRequests)
	fmt.Printf("Successful: %d\n", results.SuccessfulReqs)
	fmt.Printf("Failed: %d\n", results.FailedRequests)
	fmt.Printf("Error Rate: %.2f%%\n", results.ErrorRate)
	fmt.Printf("Requests/sec: %.2f\n", results.RequestsPerSec)
	fmt.Printf("Throughput: %.2f MB/s\n", results.ThroughputMBps)
	fmt.Printf("\n=== Latency ===\n")
	fmt.Printf("Average: %v\n", results.AverageLatency)
	fmt.Printf("Min: %v\n", results.MinLatency)
	fmt.Printf("Max: %v\n", results.MaxLatency)
	fmt.Printf("95th percentile: %v\n", results.P95Latency)
	fmt.Printf("99th percentile: %v\n", results.P99Latency)

	fmt.Printf("\n=== Status Codes ===\n")
	for code, count := range results.StatusCodes {
		fmt.Printf("%d: %d\n", code, count)
	}

	if len(results.Errors) > 0 {
		fmt.Printf("\n=== Errors ===\n")
		for err, count := range results.Errors {
			fmt.Printf("%s: %d\n", err, count)
		}
	}
}

// Standard load test configurations
var (
	LightLoadConfig = LoadTestConfig{
		Duration:       30 * time.Second,
		Concurrency:    10,
		RequestsPerSec: 50,
		TimeoutPerReq:  5 * time.Second,
		KeepAlive:      true,
		TLSInsecure:    true,
	}

	MediumLoadConfig = LoadTestConfig{
		Duration:       60 * time.Second,
		Concurrency:    50,
		RequestsPerSec: 200,
		TimeoutPerReq:  5 * time.Second,
		KeepAlive:      true,
		TLSInsecure:    true,
	}

	HeavyLoadConfig = LoadTestConfig{
		Duration:       120 * time.Second,
		Concurrency:    100,
		RequestsPerSec: 500,
		TimeoutPerReq:  10 * time.Second,
		KeepAlive:      true,
		TLSInsecure:    true,
	}

	StressTestConfig = LoadTestConfig{
		Duration:       300 * time.Second,
		Concurrency:    200,
		RequestsPerSec: 1000,
		TimeoutPerReq:  15 * time.Second,
		KeepAlive:      true,
		TLSInsecure:    true,
	}
)

// TestManagerLoadLight tests manager under light load
func TestManagerLoadLight(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LightLoadConfig
	config.BaseURL = fmt.Sprintf("http://localhost:%s", ManagerPort)

	tester := NewLoadTester(config)
	results := tester.ExecuteLoadTest(context.Background(), "/healthz", "GET", nil)

	tester.PrintResults()

	// Assert basic performance criteria
	assert.Greater(t, results.RequestsPerSec, float64(10), "Should handle at least 10 requests per second")
	assert.Less(t, results.ErrorRate, float64(5), "Error rate should be less than 5%")
	assert.Less(t, results.AverageLatency, 100*time.Millisecond, "Average latency should be under 100ms")
}

// TestProxyEgressLoadMedium tests egress proxy under medium load
func TestProxyEgressLoadMedium(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := MediumLoadConfig
	config.BaseURL = fmt.Sprintf("http://localhost:%s", AdminPort)

	tester := NewLoadTester(config)
	results := tester.ExecuteLoadTest(context.Background(), "/healthz", "GET", nil)

	tester.PrintResults()

	// Assert performance criteria
	assert.Greater(t, results.RequestsPerSec, float64(50), "Should handle at least 50 requests per second")
	assert.Less(t, results.ErrorRate, float64(2), "Error rate should be less than 2%")
	assert.Less(t, results.AverageLatency, 50*time.Millisecond, "Average latency should be under 50ms")
	assert.Less(t, results.P95Latency, 200*time.Millisecond, "95th percentile should be under 200ms")
}

// TestProxyIngressLoadHeavy tests ingress proxy under heavy load
func TestProxyIngressLoadHeavy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := HeavyLoadConfig
	config.BaseURL = fmt.Sprintf("http://localhost:%s", ProxyIngressPort)

	tester := NewLoadTester(config)
	results := tester.ExecuteLoadTest(context.Background(), "/health", "GET", nil)

	tester.PrintResults()

	// Assert performance criteria for ingress proxy
	assert.Greater(t, results.RequestsPerSec, float64(100), "Should handle at least 100 requests per second")
	assert.Less(t, results.ErrorRate, float64(10), "Error rate should be less than 10%") // More lenient for ingress
	assert.Less(t, results.P99Latency, 500*time.Millisecond, "99th percentile should be under 500ms")
}

// TestDualProxyStress tests both proxies under stress
func TestDualProxyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Test both proxies simultaneously
	var wg sync.WaitGroup
	var egressResults, ingressResults *LoadTestResults

	// Test egress proxy
	wg.Add(1)
	go func() {
		defer wg.Done()
		config := StressTestConfig
		config.BaseURL = fmt.Sprintf("http://localhost:%s", AdminPort)
		config.Duration = 60 * time.Second // Shorter for stress test

		tester := NewLoadTester(config)
		egressResults = tester.ExecuteLoadTest(context.Background(), "/metrics", "GET", nil)
		t.Logf("Egress Proxy Stress Test Results:")
		tester.PrintResults()
	}()

	// Test ingress proxy
	wg.Add(1)
	go func() {
		defer wg.Done()
		config := StressTestConfig
		config.BaseURL = fmt.Sprintf("http://localhost:%s", ProxyIngressPort)
		config.Duration = 60 * time.Second // Shorter for stress test

		tester := NewLoadTester(config)
		ingressResults = tester.ExecuteLoadTest(context.Background(), "/health", "GET", nil)
		t.Logf("Ingress Proxy Stress Test Results:")
		tester.PrintResults()
	}()

	wg.Wait()

	// Verify both proxies handled the stress reasonably
	if egressResults != nil {
		assert.Less(t, egressResults.ErrorRate, float64(15), "Egress proxy error rate should be manageable under stress")
		assert.Greater(t, egressResults.RequestsPerSec, float64(50), "Egress proxy should maintain minimum throughput")
	}

	if ingressResults != nil {
		assert.Less(t, ingressResults.ErrorRate, float64(20), "Ingress proxy error rate should be manageable under stress")
		assert.Greater(t, ingressResults.RequestsPerSec, float64(30), "Ingress proxy should maintain minimum throughput")
	}
}

// TestManagerAPILoad tests manager API endpoints under load
func TestManagerAPILoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API load test in short mode")
	}

	config := MediumLoadConfig
	config.BaseURL = fmt.Sprintf("http://localhost:%s", ManagerPort)
	config.Duration = 30 * time.Second

	// Test different endpoints
	endpoints := []string{"/healthz", "/metrics", "/api/license-status"}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("endpoint_%s", endpoint), func(t *testing.T) {
			tester := NewLoadTester(config)
			results := tester.ExecuteLoadTest(context.Background(), endpoint, "GET", nil)

			t.Logf("Load test results for %s:", endpoint)
			tester.PrintResults()

			// Basic performance assertions
			assert.Greater(t, results.TotalRequests, int64(100), "Should complete significant number of requests")
			assert.Less(t, results.ErrorRate, float64(50), "Should handle most requests successfully")
		})
	}
}

// TestConcurrentUsers simulates realistic user load patterns
func TestConcurrentUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent user test in short mode")
	}

	// Simulate realistic user behavior with mixed requests
	userScenarios := []struct {
		name     string
		baseURL  string
		endpoint string
		method   string
		body     []byte
		weight   int // Relative frequency
	}{
		{"health_check", fmt.Sprintf("http://localhost:%s", ManagerPort), "/healthz", "GET", nil, 30},
		{"metrics", fmt.Sprintf("http://localhost:%s", AdminPort), "/metrics", "GET", nil, 10},
		{"proxy_health", fmt.Sprintf("http://localhost:%s", AdminPort), "/healthz", "GET", nil, 40},
		{"license_status", fmt.Sprintf("http://localhost:%s", ManagerPort), "/api/license-status", "GET", nil, 20},
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, scenario := range userScenarios {
		// Start goroutines based on weight
		for i := 0; i < scenario.weight; i++ {
			wg.Add(1)
			go func(s struct {
				name     string
				baseURL  string
				endpoint string
				method   string
				body     []byte
				weight   int
			}) {
				defer wg.Done()

				config := LoadTestConfig{
					BaseURL:        s.baseURL,
					Duration:       45 * time.Second,
					Concurrency:    5,
					RequestsPerSec: 10,
					TimeoutPerReq:  3 * time.Second,
					KeepAlive:      true,
					TLSInsecure:    true,
				}

				tester := NewLoadTester(config)
				results := tester.ExecuteLoadTest(ctx, s.endpoint, s.method, s.body)

				t.Logf("Scenario %s results: %d requests, %.2f%% errors, %.2f req/s",
					s.name, results.TotalRequests, results.ErrorRate, results.RequestsPerSec)

			}(scenario)
		}
	}

	wg.Wait()
	t.Log("Concurrent user simulation completed")
}

// BenchmarkProxyThroughput benchmarks maximum proxy throughput
func BenchmarkProxyThroughput(b *testing.B) {
	config := LoadTestConfig{
		BaseURL:        fmt.Sprintf("http://localhost:%s", AdminPort),
		Duration:       time.Duration(b.N) * time.Millisecond,
		Concurrency:    50,
		RequestsPerSec: 0, // No rate limiting for max throughput
		TimeoutPerReq:  1 * time.Second,
		KeepAlive:      true,
		TLSInsecure:    true,
	}

	tester := NewLoadTester(config)

	b.ResetTimer()
	results := tester.ExecuteLoadTest(context.Background(), "/healthz", "GET", nil)

	b.ReportMetric(float64(results.RequestsPerSec), "req/s")
	b.ReportMetric(float64(results.AverageLatency.Nanoseconds())/1000000, "avg_latency_ms")
	b.ReportMetric(results.ErrorRate, "error_rate_%")
}

// BenchmarkManagerAPIPerformance benchmarks manager API performance
func BenchmarkManagerAPIPerformance(b *testing.B) {
	endpoints := []string{"/healthz", "/metrics"}

	for _, endpoint := range endpoints {
		b.Run(endpoint, func(b *testing.B) {
			config := LoadTestConfig{
				BaseURL:        fmt.Sprintf("http://localhost:%s", ManagerPort),
				Duration:       time.Duration(b.N) * time.Millisecond,
				Concurrency:    25,
				RequestsPerSec: 0,
				TimeoutPerReq:  2 * time.Second,
				KeepAlive:      true,
				TLSInsecure:    true,
			}

			tester := NewLoadTester(config)

			b.ResetTimer()
			results := tester.ExecuteLoadTest(context.Background(), endpoint, "GET", nil)

			b.ReportMetric(float64(results.RequestsPerSec), "req/s")
			b.ReportMetric(float64(results.AverageLatency.Nanoseconds())/1000000, "avg_latency_ms")
		})
	}
}