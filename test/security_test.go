// Package test provides security testing for MarchProxy dual proxy architecture
package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SecurityTestClient provides methods for security testing
type SecurityTestClient struct {
	client *http.Client
}

// NewSecurityTestClient creates a new security test client
func NewSecurityTestClient() *SecurityTestClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // For testing only
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	return &SecurityTestClient{client: client}
}

// makeRequest performs an HTTP request with security testing in mind
func (stc *SecurityTestClient) makeRequest(method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return stc.client.Do(req)
}

// TestSecurityHeaders verifies security headers are properly set
func TestSecurityHeaders(t *testing.T) {
	stc := NewSecurityTestClient()

	services := []struct {
		name string
		url  string
	}{
		{"manager", fmt.Sprintf("http://localhost:%s/healthz", ManagerPort)},
		{"proxy-egress", fmt.Sprintf("http://localhost:%s/healthz", AdminPort)},
	}

	expectedHeaders := map[string][]string{
		"X-Frame-Options":        {"DENY", "SAMEORIGIN"},
		"X-Content-Type-Options": {"nosniff"},
		"X-XSS-Protection":       {"1; mode=block", "0"},
		"Referrer-Policy":        {"strict-origin-when-cross-origin", "no-referrer"},
	}

	for _, service := range services {
		t.Run(service.name, func(t *testing.T) {
			resp, err := stc.makeRequest("GET", service.url, nil, nil)
			if err != nil {
				t.Skipf("Service %s not available: %v", service.name, err)
				return
			}
			defer resp.Body.Close()

			// Check for security headers
			for header, acceptableValues := range expectedHeaders {
				headerValue := resp.Header.Get(header)
				if headerValue != "" {
					found := false
					for _, acceptable := range acceptableValues {
						if headerValue == acceptable {
							found = true
							break
						}
					}
					assert.True(t, found,
						"Service %s: Header %s has value '%s', expected one of %v",
						service.name, header, headerValue, acceptableValues)
				} else {
					t.Logf("Service %s: Missing security header %s", service.name, header)
				}
			}

			// Check that sensitive headers are not exposed
			sensitiveHeaders := []string{
				"Server",
				"X-Powered-By",
				"X-AspNet-Version",
				"X-AspNetMvc-Version",
			}

			for _, header := range sensitiveHeaders {
				headerValue := resp.Header.Get(header)
				if headerValue != "" {
					t.Logf("Service %s: Potentially sensitive header %s: %s", service.name, header, headerValue)
				}
			}
		})
	}
}

// TestHTTPMethodSecurity tests HTTP method security
func TestHTTPMethodSecurity(t *testing.T) {
	stc := NewSecurityTestClient()

	testCases := []struct {
		name           string
		url            string
		method         string
		expectedStatus []int
	}{
		{"manager_options", fmt.Sprintf("http://localhost:%s/healthz", ManagerPort), "OPTIONS", []int{200, 405}},
		{"manager_trace", fmt.Sprintf("http://localhost:%s/healthz", ManagerPort), "TRACE", []int{405, 501}},
		{"manager_connect", fmt.Sprintf("http://localhost:%s/healthz", ManagerPort), "CONNECT", []int{405, 501}},
		{"proxy_options", fmt.Sprintf("http://localhost:%s/healthz", AdminPort), "OPTIONS", []int{200, 405}},
		{"proxy_trace", fmt.Sprintf("http://localhost:%s/healthz", AdminPort), "TRACE", []int{405, 501}},
		{"proxy_patch", fmt.Sprintf("http://localhost:%s/healthz", AdminPort), "PATCH", []int{405, 501}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := stc.makeRequest(tc.method, tc.url, nil, nil)
			if err != nil {
				t.Skipf("Request failed: %v", err)
				return
			}
			defer resp.Body.Close()

			found := false
			for _, expectedStatus := range tc.expectedStatus {
				if resp.StatusCode == expectedStatus {
					found = true
					break
				}
			}
			assert.True(t, found,
				"Method %s should return one of %v, got %d",
				tc.method, tc.expectedStatus, resp.StatusCode)
		})
	}
}

// TestInputValidation tests input validation and injection attacks
func TestInputValidation(t *testing.T) {
	stc := NewSecurityTestClient()

	// SQL injection payloads
	sqlPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"1' UNION SELECT * FROM users--",
		"'; INSERT INTO users VALUES ('hacker'); --",
	}

	// XSS payloads
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"javascript:alert('XSS')",
		"<img src=x onerror=alert('XSS')>",
		"<svg onload=alert('XSS')>",
	}

	// Command injection payloads
	cmdPayloads := []string{
		"; ls -la",
		"| cat /etc/passwd",
		"&& whoami",
		"`id`",
	}

	allPayloads := append(append(sqlPayloads, xssPayloads...), cmdPayloads...)

	testEndpoints := []struct {
		name   string
		url    string
		method string
	}{
		{"manager_api", fmt.Sprintf("http://localhost:%s/api/license-status", ManagerPort), "GET"},
		{"proxy_metrics", fmt.Sprintf("http://localhost:%s/metrics", AdminPort), "GET"},
	}

	for _, endpoint := range testEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			for i, payload := range allPayloads {
				// Test in URL parameters
				testURL := fmt.Sprintf("%s?test=%s", endpoint.url, url.QueryEscape(payload))

				resp, err := stc.makeRequest(endpoint.method, testURL, nil, nil)
				if err != nil {
					continue // Skip if service unavailable
				}

				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				// Check that the payload isn't reflected back unescaped
				if strings.Contains(string(body), payload) && !strings.Contains(string(body), url.QueryEscape(payload)) {
					t.Errorf("Payload %d may be vulnerable to injection: %s", i, payload)
				}

				// Check for common error messages that might indicate injection
				errorIndicators := []string{
					"syntax error",
					"mysql_fetch",
					"ORA-",
					"Microsoft OLE DB",
					"SQLServer JDBC",
				}

				bodyLower := strings.ToLower(string(body))
				for _, indicator := range errorIndicators {
					if strings.Contains(bodyLower, strings.ToLower(indicator)) {
						t.Errorf("Potential SQL injection vulnerability detected with payload: %s", payload)
					}
				}
			}
		})
	}
}

// TestAuthenticationSecurity tests authentication mechanisms
func TestAuthenticationSecurity(t *testing.T) {
	stc := NewSecurityTestClient()

	// Test endpoints that should require authentication
	protectedEndpoints := []string{
		fmt.Sprintf("http://localhost:%s/api/config/default", ManagerPort),
		fmt.Sprintf("http://localhost:%s/api/proxy/register", ManagerPort),
	}

	for _, endpoint := range protectedEndpoints {
		t.Run(fmt.Sprintf("no_auth_%s", endpoint), func(t *testing.T) {
			// Test without authentication
			resp, err := stc.makeRequest("GET", endpoint, nil, nil)
			if err != nil {
				t.Skipf("Endpoint not available: %v", err)
				return
			}
			defer resp.Body.Close()

			// Should return 401 Unauthorized or 403 Forbidden
			assert.Contains(t, []int{401, 403, 404}, resp.StatusCode,
				"Protected endpoint should require authentication")
		})

		t.Run(fmt.Sprintf("weak_auth_%s", endpoint), func(t *testing.T) {
			// Test with weak/invalid authentication
			weakTokens := []string{
				"",
				"weak",
				"admin",
				"password",
				"123456",
				"Bearer invalid",
				"Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin")),
			}

			for _, token := range weakTokens {
				headers := map[string]string{
					"Authorization": token,
					"X-API-Key":     token,
				}

				resp, err := stc.makeRequest("GET", endpoint, headers, nil)
				if err != nil {
					continue
				}
				defer resp.Body.Close()

				// Should still return authentication error
				assert.Contains(t, []int{401, 403, 404}, resp.StatusCode,
					"Weak token '%s' should be rejected", token)
			}
		})
	}
}

// TestRateLimitingSecurity tests rate limiting implementation
func TestRateLimitingSecurity(t *testing.T) {
	stc := NewSecurityTestClient()

	endpoints := []string{
		fmt.Sprintf("http://localhost:%s/healthz", ManagerPort),
		fmt.Sprintf("http://localhost:%s/healthz", AdminPort),
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("rate_limit_%s", endpoint), func(t *testing.T) {
			// Make rapid requests to test rate limiting
			var responses []int

			for i := 0; i < 100; i++ {
				resp, err := stc.makeRequest("GET", endpoint, nil, nil)
				if err != nil {
					continue
				}
				responses = append(responses, resp.StatusCode)
				resp.Body.Close()

				// Small delay to avoid connection issues
				time.Sleep(10 * time.Millisecond)
			}

			if len(responses) > 50 {
				// Check if any rate limiting occurred (429 status code)
				rateLimited := false
				for _, status := range responses {
					if status == 429 {
						rateLimited = true
						break
					}
				}

				if rateLimited {
					t.Logf("Rate limiting detected (good security practice)")
				} else {
					t.Logf("No rate limiting detected - consider implementing rate limiting")
				}
			}
		})
	}
}

// TestTLSConfiguration tests TLS/SSL security
func TestTLSConfiguration(t *testing.T) {
	// Test TLS configuration if HTTPS endpoints are available
	httpsEndpoints := []string{
		"https://localhost:8443/healthz", // Example HTTPS endpoint
	}

	for _, endpoint := range httpsEndpoints {
		t.Run(fmt.Sprintf("tls_%s", endpoint), func(t *testing.T) {
			// Test with secure TLS client
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
					MinVersion:         tls.VersionTLS12,
				},
			}

			secureClient := &http.Client{
				Transport: transport,
				Timeout:   5 * time.Second,
			}

			resp, err := secureClient.Get(endpoint)
			if err != nil {
				t.Skipf("HTTPS endpoint not available or has TLS issues: %v", err)
				return
			}
			defer resp.Body.Close()

			// If we get here, TLS is properly configured
			assert.Equal(t, 200, resp.StatusCode, "HTTPS endpoint should be accessible with proper TLS")
		})
	}
}

// TestDirectoryTraversal tests for directory traversal vulnerabilities
func TestDirectoryTraversal(t *testing.T) {
	stc := NewSecurityTestClient()

	traversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\drivers\\etc\\hosts",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"..%252f..%252f..%252fetc%252fpasswd",
	}

	baseEndpoints := []string{
		fmt.Sprintf("http://localhost:%s", ManagerPort),
		fmt.Sprintf("http://localhost:%s", AdminPort),
	}

	for _, baseURL := range baseEndpoints {
		t.Run(fmt.Sprintf("traversal_%s", baseURL), func(t *testing.T) {
			for _, payload := range traversalPayloads {
				testURL := fmt.Sprintf("%s/%s", baseURL, payload)

				resp, err := stc.makeRequest("GET", testURL, nil, nil)
				if err != nil {
					continue
				}

				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				// Check for signs of successful directory traversal
				suspiciousContent := []string{
					"root:x:0:0:",
					"[boot loader]",
					"# This file was automatically generated",
				}

				for _, content := range suspiciousContent {
					if strings.Contains(string(body), content) {
						t.Errorf("Potential directory traversal vulnerability with payload: %s", payload)
					}
				}
			}
		})
	}
}

// TestHTTPSRedirection tests HTTPS redirection
func TestHTTPSRedirection(t *testing.T) {
	// Test that HTTP traffic is redirected to HTTPS (if configured)
	httpEndpoints := []string{
		fmt.Sprintf("http://localhost:%s/healthz", ManagerPort),
	}

	for _, endpoint := range httpEndpoints {
		t.Run(fmt.Sprintf("https_redirect_%s", endpoint), func(t *testing.T) {
			resp, err := NewSecurityTestClient().makeRequest("GET", endpoint, nil, nil)
			if err != nil {
				t.Skipf("Endpoint not available: %v", err)
				return
			}
			defer resp.Body.Close()

			// Check for HTTPS redirect
			if resp.StatusCode >= 300 && resp.StatusCode < 400 {
				location := resp.Header.Get("Location")
				if strings.HasPrefix(location, "https://") {
					t.Logf("HTTPS redirection properly configured")
				} else {
					t.Logf("Redirection found but not to HTTPS: %s", location)
				}
			} else {
				t.Logf("No HTTPS redirection detected (may be intentional for development)")
			}
		})
	}
}

// TestCORSConfiguration tests CORS security
func TestCORSConfiguration(t *testing.T) {
	stc := NewSecurityTestClient()

	endpoints := []string{
		fmt.Sprintf("http://localhost:%s/healthz", ManagerPort),
		fmt.Sprintf("http://localhost:%s/healthz", AdminPort),
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("cors_%s", endpoint), func(t *testing.T) {
			headers := map[string]string{
				"Origin":                        "https://malicious-site.com",
				"Access-Control-Request-Method": "GET",
			}

			resp, err := stc.makeRequest("OPTIONS", endpoint, headers, nil)
			if err != nil {
				t.Skipf("Endpoint not available: %v", err)
				return
			}
			defer resp.Body.Close()

			// Check CORS headers
			corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if corsOrigin == "*" {
				t.Logf("CORS allows all origins - potential security risk")
			} else if corsOrigin != "" {
				t.Logf("CORS origin restriction: %s", corsOrigin)
			}

			corsMethods := resp.Header.Get("Access-Control-Allow-Methods")
			if strings.Contains(corsMethods, "DELETE") || strings.Contains(corsMethods, "PUT") {
				t.Logf("CORS allows potentially dangerous methods: %s", corsMethods)
			}
		})
	}
}

// TestInformationDisclosure tests for information disclosure
func TestInformationDisclosure(t *testing.T) {
	stc := NewSecurityTestClient()

	// Test various endpoints for information disclosure
	testEndpoints := []string{
		fmt.Sprintf("http://localhost:%s/healthz", ManagerPort),
		fmt.Sprintf("http://localhost:%s/metrics", AdminPort),
		fmt.Sprintf("http://localhost:%s/version", ManagerPort),
		fmt.Sprintf("http://localhost:%s/debug/pprof", AdminPort),
	}

	sensitiveInfo := []string{
		"password",
		"secret",
		"token",
		"key",
		"private",
		"internal",
		"/etc/",
		"/home/",
		"database",
		"connection",
	}

	for _, endpoint := range testEndpoints {
		t.Run(fmt.Sprintf("info_disclosure_%s", endpoint), func(t *testing.T) {
			resp, err := stc.makeRequest("GET", endpoint, nil, nil)
			if err != nil {
				t.Skipf("Endpoint not available: %v", err)
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			bodyStr := strings.ToLower(string(body))

			for _, sensitive := range sensitiveInfo {
				if strings.Contains(bodyStr, sensitive) {
					t.Logf("Potential information disclosure in %s: contains '%s'", endpoint, sensitive)
				}
			}
		})
	}
}

// TestDOSResistance tests basic DOS resistance
func TestDOSResistance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DOS resistance test in short mode")
	}

	stc := NewSecurityTestClient()

	endpoints := []string{
		fmt.Sprintf("http://localhost:%s/healthz", ManagerPort),
		fmt.Sprintf("http://localhost:%s/healthz", AdminPort),
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("dos_resistance_%s", endpoint), func(t *testing.T) {
			// Test with large request body
			largeBody := bytes.Repeat([]byte("A"), 10*1024*1024) // 10MB

			resp, err := stc.makeRequest("POST", endpoint, nil, largeBody)
			if err != nil {
				t.Logf("Large request properly rejected or connection failed: %v", err)
				return
			}
			defer resp.Body.Close()

			// Should reject large requests
			if resp.StatusCode == 413 || resp.StatusCode == 400 {
				t.Logf("Large request properly rejected with status %d", resp.StatusCode)
			} else {
				t.Logf("Large request handling: status %d", resp.StatusCode)
			}

			// Test with many headers
			manyHeaders := make(map[string]string)
			for i := 0; i < 1000; i++ {
				manyHeaders[fmt.Sprintf("X-Test-Header-%d", i)] = "value"
			}

			resp2, err := stc.makeRequest("GET", endpoint, manyHeaders, nil)
			if err != nil {
				t.Logf("Many headers request properly rejected: %v", err)
				return
			}
			defer resp2.Body.Close()

			t.Logf("Many headers request status: %d", resp2.StatusCode)
		})
	}
}

// BenchmarkSecurityOverhead benchmarks security overhead
func BenchmarkSecurityOverhead(b *testing.B) {
	stc := NewSecurityTestClient()
	endpoint := fmt.Sprintf("http://localhost:%s/healthz", AdminPort)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := stc.makeRequest("GET", endpoint, nil, nil)
			if err != nil {
				continue
			}
			resp.Body.Close()
		}
	})
}