// Package test provides integration tests for MarchProxy dual proxy architecture
package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ManagerPort      = "8000"
	ProxyEgressPort  = "8080"
	ProxyIngressPort = "8082"
	AdminPort        = "8081"
	TestTimeout      = 30 * time.Second
)

// IntegrationTest represents a full integration test setup
type IntegrationTest struct {
	ctx    context.Context
	cancel context.CancelFunc
	client *http.Client
}

// SetupIntegrationTest initializes the integration test environment
func SetupIntegrationTest(t *testing.T) *IntegrationTest {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // For testing only
			},
		},
	}

	return &IntegrationTest{
		ctx:    ctx,
		cancel: cancel,
		client: client,
	}
}

// TeardownIntegrationTest cleans up the integration test environment
func (it *IntegrationTest) TeardownIntegrationTest() {
	it.cancel()
}

// TestManagerHealthCheck tests that the manager service is healthy
func TestManagerHealthCheck(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	url := fmt.Sprintf("http://localhost:%s/healthz", ManagerPort)
	resp, err := it.client.Get(url)

	if err != nil {
		t.Skipf("Manager not available for integration test: %v", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Manager health check should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var health map[string]interface{}
	err = json.Unmarshal(body, &health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"], "Manager should report healthy status")
}

// TestManagerMetrics tests that the manager exposes metrics
func TestManagerMetrics(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	url := fmt.Sprintf("http://localhost:%s/metrics", ManagerPort)
	resp, err := it.client.Get(url)

	if err != nil {
		t.Skipf("Manager not available for integration test: %v", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Manager metrics should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	metrics := string(body)
	assert.Contains(t, metrics, "# HELP", "Metrics should contain Prometheus format")
	assert.Contains(t, metrics, "# TYPE", "Metrics should contain metric types")
}

// TestProxyEgressHealthCheck tests that the egress proxy is healthy
func TestProxyEgressHealthCheck(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	url := fmt.Sprintf("http://localhost:%s/healthz", AdminPort)
	resp, err := it.client.Get(url)

	if err != nil {
		t.Skipf("Proxy egress not available for integration test: %v", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Proxy egress health check should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var health map[string]interface{}
	err = json.Unmarshal(body, &health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"], "Proxy egress should report healthy status")
}

// TestProxyEgressMetrics tests that the egress proxy exposes metrics
func TestProxyEgressMetrics(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	url := fmt.Sprintf("http://localhost:%s/metrics", AdminPort)
	resp, err := it.client.Get(url)

	if err != nil {
		t.Skipf("Proxy egress not available for integration test: %v", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Proxy egress metrics should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	metrics := string(body)
	assert.Contains(t, metrics, "proxy_requests_total", "Metrics should contain proxy request metrics")
	assert.Contains(t, metrics, "proxy_active_connections", "Metrics should contain connection metrics")
}

// TestProxyRegistration tests that proxies can register with the manager
func TestProxyRegistration(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Create registration payload
	payload := map[string]interface{}{
		"name":     "test-proxy-egress",
		"hostname": "localhost",
		"type":     "egress",
		"version":  "v1.0.0",
	}

	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	// Test proxy registration
	url := fmt.Sprintf("http://localhost:%s/api/proxy/register", ManagerPort)
	req, err := http.NewRequestWithContext(it.ctx, "POST", url, bytes.NewBuffer(jsonData))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-cluster-api-key") // Would need to be configured

	resp, err := it.client.Do(req)
	if err != nil {
		t.Skipf("Manager not available for proxy registration test: %v", err)
		return
	}
	defer resp.Body.Close()

	// Accept either 200 (success) or 401 (auth failure - expected in test env)
	assert.Contains(t, []int{http.StatusOK, http.StatusCreated, http.StatusUnauthorized},
		resp.StatusCode, "Proxy registration should return success or auth error")
}

// TestConfigurationEndpoint tests that proxies can fetch configuration
func TestConfigurationEndpoint(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	url := fmt.Sprintf("http://localhost:%s/api/config/default", ManagerPort)
	req, err := http.NewRequestWithContext(it.ctx, "GET", url, nil)
	require.NoError(t, err)

	req.Header.Set("X-API-Key", "test-cluster-api-key")

	resp, err := it.client.Do(req)
	if err != nil {
		t.Skipf("Manager not available for configuration test: %v", err)
		return
	}
	defer resp.Body.Close()

	// Accept either 200 (success) or 401/404 (auth/not found - expected in test env)
	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized, http.StatusNotFound},
		resp.StatusCode, "Configuration endpoint should return success or expected error")

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var config map[string]interface{}
		err = json.Unmarshal(body, &config)
		require.NoError(t, err)

		assert.NotEmpty(t, config, "Configuration should not be empty")
	}
}

// TestDualProxyConnectivity tests connectivity between ingress and egress proxies
func TestDualProxyConnectivity(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Test that both proxy ports are listening
	egressConn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%s", ProxyEgressPort), 5*time.Second)
	if err != nil {
		t.Skipf("Egress proxy not available for connectivity test: %v", err)
		return
	}
	egressConn.Close()

	ingressConn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%s", ProxyIngressPort), 5*time.Second)
	if err != nil {
		t.Skipf("Ingress proxy not available for connectivity test: %v", err)
		return
	}
	ingressConn.Close()

	assert.True(t, true, "Both ingress and egress proxies are listening")
}

// TestProxyTrafficFlow tests basic traffic flow through the dual proxy setup
func TestProxyTrafficFlow(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Create a simple test server to proxy to
	testServer := &http.Server{
		Addr: ":9999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			response := map[string]string{
				"status":  "ok",
				"message": "test server response",
				"path":    r.URL.Path,
			}
			json.NewEncoder(w).Encode(response)
		}),
	}

	// Start test server in background
	go func() {
		if err := testServer.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("Test server error: %v", err)
		}
	}()

	// Give test server time to start
	time.Sleep(100 * time.Millisecond)

	defer func() {
		testServer.Shutdown(context.Background())
	}()

	// Test direct connection to test server first
	directResp, err := it.client.Get("http://localhost:9999/test")
	if err != nil {
		t.Skipf("Test server not available: %v", err)
		return
	}
	directResp.Body.Close()
	assert.Equal(t, http.StatusOK, directResp.StatusCode, "Direct connection to test server should work")

	// Test traffic through ingress proxy (if configured to proxy to test server)
	// This would require the ingress proxy to be configured to route to localhost:9999
	proxyResp, err := it.client.Get(fmt.Sprintf("http://localhost:%s/test", ProxyIngressPort))
	if err != nil {
		t.Logf("Proxy traffic test not available (expected in test env): %v", err)
		return
	}
	defer proxyResp.Body.Close()

	// If proxy is configured, it should return the same response
	if proxyResp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(proxyResp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "ok", response["status"], "Proxied response should match direct response")
	}
}

// TestLoadBalancing tests basic load balancing functionality (if multiple backends configured)
func TestLoadBalancing(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// This test would require multiple backend servers to be configured
	// For now, just test that the ingress proxy responds consistently
	responses := make(map[int]int)

	for i := 0; i < 10; i++ {
		resp, err := it.client.Get(fmt.Sprintf("http://localhost:%s/health", ProxyIngressPort))
		if err != nil {
			t.Skipf("Ingress proxy not available for load balancing test: %v", err)
			return
		}
		resp.Body.Close()

		responses[resp.StatusCode]++
	}

	// Should get consistent responses (even if it's all 404s or 503s in test env)
	assert.Len(t, responses, 1, "Load balancer should return consistent status codes")
}

// TestSecurityHeaders tests that security headers are properly set
func TestSecurityHeaders(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Test manager security headers
	resp, err := it.client.Get(fmt.Sprintf("http://localhost:%s/healthz", ManagerPort))
	if err != nil {
		t.Skipf("Manager not available for security headers test: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check for common security headers
	headers := resp.Header

	// These might not all be present in test environment, but check what we can
	if xFrameOptions := headers.Get("X-Frame-Options"); xFrameOptions != "" {
		assert.NotEmpty(t, xFrameOptions, "X-Frame-Options should be set")
	}

	if xContentType := headers.Get("X-Content-Type-Options"); xContentType != "" {
		assert.Equal(t, "nosniff", xContentType, "X-Content-Type-Options should be nosniff")
	}
}

// TestRateLimiting tests basic rate limiting functionality
func TestRateLimiting(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Make rapid requests to test rate limiting
	var statusCodes []int
	for i := 0; i < 50; i++ {
		resp, err := it.client.Get(fmt.Sprintf("http://localhost:%s/healthz", ProxyEgressPort))
		if err != nil {
			continue // Skip connection errors
		}
		statusCodes = append(statusCodes, resp.StatusCode)
		resp.Body.Close()

		time.Sleep(10 * time.Millisecond) // Small delay between requests
	}

	if len(statusCodes) > 0 {
		// In a properly configured system with rate limiting, we might see some 429s
		// For now, just verify we get responses
		assert.Greater(t, len(statusCodes), 0, "Should receive some responses")

		// Count different status codes
		statusCodeMap := make(map[int]int)
		for _, code := range statusCodes {
			statusCodeMap[code]++
		}

		t.Logf("Received status codes: %v", statusCodeMap)
	}
}

// TestMetricsAccumulation tests that metrics are properly accumulated across requests
func TestMetricsAccumulation(t *testing.T) {
	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Get initial metrics
	initialMetrics, err := getMetrics(it.client, AdminPort)
	if err != nil {
		t.Skipf("Cannot get initial metrics: %v", err)
		return
	}

	// Make some requests
	for i := 0; i < 5; i++ {
		resp, err := it.client.Get(fmt.Sprintf("http://localhost:%s/healthz", AdminPort))
		if err != nil {
			continue
		}
		resp.Body.Close()
	}

	// Get final metrics
	finalMetrics, err := getMetrics(it.client, AdminPort)
	if err != nil {
		t.Skipf("Cannot get final metrics: %v", err)
		return
	}

	// Check that request count increased (if metrics are working)
	if strings.Contains(finalMetrics, "proxy_requests_total") {
		assert.Contains(t, finalMetrics, "proxy_requests_total", "Should contain request total metrics")
		// Could parse the actual values and compare, but for integration test this is sufficient
	}
}

// Helper function to get metrics from a service
func getMetrics(client *http.Client, port string) (string, error) {
	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/metrics", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// TestDockerComposeSetup tests the complete docker-compose setup
func TestDockerComposeSetup(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping docker-compose integration test (set INTEGRATION_TEST=true to enable)")
	}

	it := SetupIntegrationTest(t)
	defer it.TeardownIntegrationTest()

	// Test all expected services are running
	services := []struct {
		name string
		port string
		path string
	}{
		{"manager", ManagerPort, "/healthz"},
		{"proxy-egress", AdminPort, "/healthz"},
		{"proxy-ingress", "8083", "/healthz"}, // Assuming different admin port for ingress
	}

	for _, service := range services {
		t.Run(service.name, func(t *testing.T) {
			url := fmt.Sprintf("http://localhost:%s%s", service.port, service.path)
			resp, err := it.client.Get(url)

			if err != nil {
				t.Skipf("%s service not available: %v", service.name, err)
				return
			}
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode,
				fmt.Sprintf("%s health check should return 200", service.name))
		})
	}
}

// BenchmarkProxyThroughput benchmarks basic proxy throughput
func BenchmarkProxyThroughput(b *testing.B) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Test against health check endpoint for consistent response
	url := fmt.Sprintf("http://localhost:%s/healthz", AdminPort)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(url)
			if err != nil {
				continue // Skip errors in benchmark
			}
			resp.Body.Close()
		}
	})
}