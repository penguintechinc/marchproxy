//go:build ci

package logging

import (
	"fmt"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	config := LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		Structured: true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v, want nil", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.config.Level != "info" {
		t.Errorf("expected Level 'info', got %q", logger.config.Level)
	}
}

func TestDefaultLogConfig(t *testing.T) {
	cfg := DefaultLogConfig()
	if cfg.Level != "info" {
		t.Errorf("expected Level 'info', got %q", cfg.Level)
	}
	if cfg.Format != "text" {
		t.Errorf("expected Format 'text', got %q", cfg.Format)
	}
	if cfg.Output != "stdout" {
		t.Errorf("expected Output 'stdout', got %q", cfg.Output)
	}
	service, ok := cfg.Fields["service"].(string)
	if !ok || service != "marchproxy-ingress" {
		t.Errorf("expected service field 'marchproxy-ingress', got %v", service)
	}
}

func TestLogMTLSAuth(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := MTLSLogEntry{
		Timestamp:    time.Now(),
		Level:        "info",
		Message:      "mTLS auth test",
		ClientCN:     "test-client.example.com",
		ClientOU:     "Engineering",
		ClientSerial: "12345",
		ServerName:   "server.example.com",
		TLSVersion:   "TLSv1.3",
		CipherSuite:  "TLS_AES_256_GCM_SHA384",
		Result:       "success",
		VirtualHost:  "vhost1",
		Backend:      "backend1",
		RequestID:    "req-123",
		RemoteAddr:   "192.168.1.100",
	}

	// Should not panic
	logger.LogMTLSAuth(entry)
}

func TestLogMTLSAuthWithError(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := MTLSLogEntry{
		Timestamp:    time.Now(),
		Level:        "warn",
		Message:      "mTLS auth failed",
		ClientCN:     "test-client.example.com",
		Result:       "failure",
		Error:        "certificate validation failed",
		VirtualHost:  "vhost1",
		Backend:      "backend1",
		RequestID:    "req-456",
		RemoteAddr:   "192.168.1.101",
	}

	// Should not panic
	logger.LogMTLSAuth(entry)
}

func TestLogRequest(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := RequestLogEntry{
		Timestamp:       time.Now(),
		Level:           "info",
		Message:         "Request processed",
		Method:          "GET",
		URL:             "http://example.com/api/v1/test",
		Path:            "/api/v1/test",
		StatusCode:      200,
		ResponseTime:    time.Millisecond * 100,
		RequestSize:     1024,
		ResponseSize:    2048,
		UserAgent:       "Mozilla/5.0",
		Referer:         "http://example.com",
		XForwardedFor:   "192.168.1.100",
		VirtualHost:     "vhost1",
		Backend:         "backend1",
		BackendEndpoint: "10.0.0.1:8080",
		RequestID:       "req-789",
		RemoteAddr:      "192.168.1.100",
	}

	// Should not panic
	logger.LogRequest(entry)
}

func TestLogRequestWithError(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := RequestLogEntry{
		Timestamp:    time.Now(),
		Message:      "Request failed",
		Method:       "POST",
		URL:          "http://example.com/api/v1/data",
		Path:         "/api/v1/data",
		StatusCode:   500,
		ResponseTime: time.Millisecond * 250,
		VirtualHost:  "vhost1",
		Backend:      "backend1",
		RequestID:    "req-error",
		RemoteAddr:   "192.168.1.102",
		Error:        "connection refused",
	}

	// Should not panic
	logger.LogRequest(entry)
}

func TestLogRequestWith4xxStatus(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := RequestLogEntry{
		Timestamp:    time.Now(),
		Message:      "Client error",
		Method:       "GET",
		StatusCode:   404,
		ResponseTime: time.Millisecond * 50,
		VirtualHost:  "vhost1",
		Backend:      "backend1",
		RequestID:    "req-404",
		RemoteAddr:   "192.168.1.103",
	}

	// Should not panic
	logger.LogRequest(entry)
}

func TestLogRequestWithHeaders(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := RequestLogEntry{
		Timestamp:    time.Now(),
		Message:      "Request with headers",
		Method:       "GET",
		StatusCode:   200,
		ResponseTime: time.Millisecond * 75,
		VirtualHost:  "vhost1",
		Backend:      "backend1",
		RequestID:    "req-hdr",
		RemoteAddr:   "192.168.1.104",
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token",
		},
	}

	// Should not panic
	logger.LogRequest(entry)
}

func TestLogHealth(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := HealthLogEntry{
		Timestamp:       time.Now(),
		Level:           "debug",
		Message:         "Health check passed",
		CheckType:       "http",
		Target:          "backend1:8080",
		Status:          "healthy",
		ResponseTime:    time.Millisecond * 50,
		VirtualHost:     "vhost1",
		Backend:         "backend1",
		BackendEndpoint: "10.0.0.1:8080",
	}

	// Should not panic
	logger.LogHealth(entry)
}

func TestLogHealthDegraded(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := HealthLogEntry{
		Timestamp:    time.Now(),
		Message:      "Health check degraded",
		CheckType:    "tcp",
		Target:       "backend2:8080",
		Status:       "degraded",
		ResponseTime: time.Millisecond * 150,
		VirtualHost:  "vhost2",
		Backend:      "backend2",
		Error:        "slow response",
	}

	// Should not panic
	logger.LogHealth(entry)
}

func TestLogHealthUnhealthy(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := HealthLogEntry{
		Timestamp:    time.Now(),
		Message:      "Health check failed",
		CheckType:    "http",
		Target:       "backend3:8080",
		Status:       "unhealthy",
		ResponseTime: time.Millisecond * 500,
		VirtualHost:  "vhost3",
		Backend:      "backend3",
		Error:        "connection timeout",
	}

	// Should not panic
	logger.LogHealth(entry)
}

func TestLogHealthWithMetadata(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entry := HealthLogEntry{
		Timestamp:    time.Now(),
		Message:      "Health check with metadata",
		CheckType:    "ssl_cert",
		Target:       "backend4:443",
		Status:       "healthy",
		ResponseTime: time.Millisecond * 100,
		Metadata: map[string]interface{}{
			"cert_expiry": "2025-12-31",
			"cert_days":   "365",
		},
	}

	// Should not panic
	logger.LogHealth(entry)
}

func TestLogConfigUpdate(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	fields := map[string]interface{}{
		"config_path": "/app/configs/new.yaml",
		"reload_time": time.Now(),
	}

	// Should not panic
	logger.LogConfigUpdate("Configuration reloaded", fields)
}

func TestLogCertificateEvent(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	certInfo := map[string]interface{}{
		"cert_path":  "/app/certs/server.crt",
		"issuer":     "Let's Encrypt",
		"subject":    "example.com",
		"expiry":     "2025-12-31",
		"thumbprint": "abc123",
	}

	// Should not panic
	logger.LogCertificateEvent("Certificate renewed", certInfo)
}

func TestLogLoadBalancer(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Should not panic
	logger.LogLoadBalancer("Request routed", "backend-pool-1", "round_robin", "10.0.0.5:8080")
}

func TestLogCircuitBreaker(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Should not panic
	logger.LogCircuitBreaker("Circuit breaker tripped", "backend-pool-2", "open", 0.75)
}

func TestLogRateLimit(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Should not panic
	logger.LogRateLimit("Rate limit exceeded", "192.168.1.200", "quota exceeded", 1000)
}

func TestLogError(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	testErr := fmt.Errorf("test error")
	fields := map[string]interface{}{
		"request_id": "req-error-123",
		"retry":      3,
	}

	// Should not panic
	logger.LogError(testErr, "test_context", fields)
}

func TestLogStartup(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Should not panic
	logger.LogStartup("1.0.0", "2025-01-15T10:30:00Z")
}

func TestLogShutdown(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Should not panic
	logger.LogShutdown("graceful shutdown requested")
}

func TestMTLSLogEntry_AllFields(t *testing.T) {
	entry := MTLSLogEntry{
		Timestamp:    time.Now(),
		Level:        "info",
		Message:      "Full entry",
		ClientCN:     "client",
		ClientOU:     "unit",
		ClientSerial: "serial",
		ServerName:   "server",
		TLSVersion:   "1.3",
		CipherSuite:  "suite",
		Result:       "success",
		Error:        "none",
		VirtualHost:  "vhost",
		Backend:      "backend",
		RequestID:    "req",
		RemoteAddr:   "addr",
	}

	if entry.ClientCN != "client" {
		t.Errorf("expected ClientCN 'client', got %q", entry.ClientCN)
	}
}

func TestRequestLogEntry_AllFields(t *testing.T) {
	entry := RequestLogEntry{
		Timestamp:       time.Now(),
		Level:           "info",
		Message:         "Full request",
		Method:          "GET",
		URL:             "http://test.com",
		Path:            "/path",
		StatusCode:      200,
		ResponseTime:    100 * time.Millisecond,
		RequestSize:     1024,
		ResponseSize:    2048,
		UserAgent:       "agent",
		Referer:         "ref",
		XForwardedFor:   "xff",
		VirtualHost:     "vhost",
		Backend:         "backend",
		BackendEndpoint: "endpoint",
		RequestID:       "req",
		RemoteAddr:      "addr",
		Headers:         map[string]string{"key": "value"},
		Error:           "none",
	}

	if entry.Method != "GET" {
		t.Errorf("expected Method 'GET', got %q", entry.Method)
	}
}

func TestHealthLogEntry_AllFields(t *testing.T) {
	entry := HealthLogEntry{
		Timestamp:       time.Now(),
		Level:           "info",
		Message:         "Full health",
		CheckType:       "http",
		Target:          "target",
		Status:          "healthy",
		ResponseTime:    50 * time.Millisecond,
		Error:           "none",
		VirtualHost:     "vhost",
		Backend:         "backend",
		BackendEndpoint: "endpoint",
		Metadata:        map[string]interface{}{"key": "value"},
	}

	if entry.Status != "healthy" {
		t.Errorf("expected Status 'healthy', got %q", entry.Status)
	}
}

func TestLogConfigWithDifferentFields(t *testing.T) {
	config := LogConfig{
		Level:       "debug",
		Format:      "json",
		Output:      "file",
		File:        "/var/log/app.log",
		MaxSize:     100,
		MaxAge:      30,
		MaxBackups:  5,
		Compress:    true,
		Structured:  true,
		Fields:      map[string]interface{}{"env": "test"},
		SyslogAddr:  "localhost:514",
		SyslogNet:   "udp",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if logger.config.Level != "debug" {
		t.Errorf("expected Level 'debug', got %q", logger.config.Level)
	}
	if logger.config.File != "/var/log/app.log" {
		t.Errorf("expected File '/var/log/app.log', got %q", logger.config.File)
	}
	if !logger.config.Compress {
		t.Error("expected Compress to be true")
	}
}

func TestLogRequestStatusCodeRanges(t *testing.T) {
	logger, err := NewLogger(DefaultLogConfig())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	tests := []struct {
		name       string
		statusCode int
	}{
		{"2xx success", 200},
		{"2xx created", 201},
		{"2xx accepted", 202},
		{"2xx no content", 204},
		{"3xx redirect", 301},
		{"4xx bad request", 400},
		{"4xx not found", 404},
		{"5xx server error", 500},
		{"5xx service unavailable", 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := RequestLogEntry{
				Timestamp:    time.Now(),
				Message:      "Test",
				StatusCode:   tt.statusCode,
				ResponseTime: time.Millisecond * 50,
				VirtualHost:  "vhost",
				Backend:      "backend",
				RequestID:    "req",
				RemoteAddr:   "addr",
			}
			// Should not panic
			logger.LogRequest(entry)
		})
	}
}
