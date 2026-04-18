//go:build ci

package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockHealthChecker implements HealthChecker for testing
type MockHealthChecker struct {
	healthy bool
	status  map[string]interface{}
}

func (m *MockHealthChecker) IsHealthy() bool {
	return m.healthy
}

func (m *MockHealthChecker) GetStatus() map[string]interface{} {
	if m.status == nil {
		return map[string]interface{}{"status": "ok"}
	}
	return m.status
}

// TestNewMonitor tests Monitor creation
func TestNewMonitor(t *testing.T) {
	monitor := NewMonitor(9090)

	if monitor == nil {
		t.Error("NewMonitor returned nil")
	}
	if monitor.registry == nil {
		t.Error("registry not initialized")
	}
	if monitor.server == nil {
		t.Error("HTTP server not initialized")
	}
}

// TestMonitorMetricsInitialization tests that all metrics are initialized
func TestMonitorMetricsInitialization(t *testing.T) {
	monitor := NewMonitor(9091)

	if monitor.metrics.activeConnections == nil {
		t.Error("activeConnections metric not initialized")
	}
	if monitor.metrics.totalConnections == nil {
		t.Error("totalConnections metric not initialized")
	}
	if monitor.metrics.cpuUsage == nil {
		t.Error("cpuUsage metric not initialized")
	}
	if monitor.metrics.memoryUsage == nil {
		t.Error("memoryUsage metric not initialized")
	}
	if monitor.metrics.goroutines == nil {
		t.Error("goroutines metric not initialized")
	}
	if monitor.metrics.authAttempts == nil {
		t.Error("authAttempts metric not initialized")
	}
	if monitor.metrics.requestLatency == nil {
		t.Error("requestLatency metric not initialized")
	}
	if monitor.metrics.licenseStatus == nil {
		t.Error("licenseStatus metric not initialized")
	}
}

// TestSetHealthChecker tests setting a health checker
func TestSetHealthChecker(t *testing.T) {
	monitor := NewMonitor(9092)
	hc := &MockHealthChecker{healthy: true}

	monitor.SetHealthChecker(hc)

	if monitor.healthz != hc {
		t.Error("health checker not set correctly")
	}
}

// TestHealthzHandlerHealthy tests /healthz endpoint when healthy
func TestHealthzHandlerHealthy(t *testing.T) {
	monitor := NewMonitor(9093)
	hc := &MockHealthChecker{healthy: true}
	monitor.SetHealthChecker(hc)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	monitor.healthzHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("expected 'OK', got %q", w.Body.String())
	}
}

// TestHealthzHandlerUnhealthy tests /healthz endpoint when unhealthy
func TestHealthzHandlerUnhealthy(t *testing.T) {
	monitor := NewMonitor(9094)
	hc := &MockHealthChecker{healthy: false}
	monitor.SetHealthChecker(hc)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	monitor.healthzHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

// TestHealthzHandlerNoHealthChecker tests /healthz when no checker is set
func TestHealthzHandlerNoHealthChecker(t *testing.T) {
	monitor := NewMonitor(9095)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	monitor.healthzHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

// TestStatusHandler tests /status endpoint
func TestStatusHandler(t *testing.T) {
	monitor := NewMonitor(9096)
	hc := &MockHealthChecker{
		healthy: true,
		status: map[string]interface{}{
			"components": "ok",
			"uptime":     "1h",
		},
	}
	monitor.SetHealthChecker(hc)

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	monitor.statusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type application/json")
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["healthy"] != true {
		t.Error("expected healthy to be true")
	}
	if response["version"] == "" {
		t.Error("expected version to be set")
	}
}

// TestStatusHandlerWithoutHealthChecker tests /status without health checker
func TestStatusHandlerWithoutHealthChecker(t *testing.T) {
	monitor := NewMonitor(9097)

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	monitor.statusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["healthy"] != false {
		t.Error("expected healthy to be false")
	}
}

// TestRecordConnection tests recording a connection
func TestRecordConnection(t *testing.T) {
	monitor := NewMonitor(9098)

	monitor.RecordConnection("tcp", "192.168.1.1", "10.0.0.1")

	// Verify metric was recorded (no error means success)
	// We can't directly inspect Prometheus metrics, so just ensure no panic
}

// TestRecordConnectionClosed tests recording connection closure
func TestRecordConnectionClosed(t *testing.T) {
	monitor := NewMonitor(9099)

	duration := 5 * time.Second
	monitor.RecordConnectionClosed("tcp", "192.168.1.1", "10.0.0.1", duration)

	// Verify metric was recorded
}

// TestRecordBytesTransferred tests recording bytes transferred
func TestRecordBytesTransferred(t *testing.T) {
	monitor := NewMonitor(9100)

	monitor.RecordBytesTransferred("inbound", "tcp", 1024)
	monitor.RecordBytesTransferred("outbound", "tcp", 2048)

	// Verify metrics were recorded
}

// TestRecordAuthAttempt tests recording authentication attempts
func TestRecordAuthAttempt(t *testing.T) {
	monitor := NewMonitor(9101)

	monitor.RecordAuthAttempt("oauth2", "success")
	monitor.RecordAuthAttempt("oauth2", "failure")

	// Verify metrics were recorded
}

// TestRecordAuthAttemptVariations tests various auth attempt results
func TestRecordAuthAttemptVariations(t *testing.T) {
	monitor := NewMonitor(9102)

	tests := []struct {
		authType string
		result   string
	}{
		{"oauth2", "success"},
		{"oauth2", "invalid_token"},
		{"jwt", "success"},
		{"jwt", "expired"},
		{"mfa", "verified"},
		{"mfa", "failed"},
	}

	for _, tt := range tests {
		monitor.RecordAuthAttempt(tt.authType, tt.result)
	}

	// All should complete without error
}

// TestRecordRequestLatency tests recording request latency
func TestRecordRequestLatency(t *testing.T) {
	monitor := NewMonitor(9103)

	latencies := []time.Duration{
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, latency := range latencies {
		monitor.RecordRequestLatency("http", "/api/v1/test", latency)
	}

	// Verify metrics were recorded
}

// TestUpdateThroughput tests updating throughput metric
func TestUpdateThroughput(t *testing.T) {
	monitor := NewMonitor(9104)

	monitor.UpdateThroughput("inbound", 1024.5)
	monitor.UpdateThroughput("outbound", 2048.75)

	// Verify metrics were recorded
}

// TestUpdateEBPFRules tests updating eBPF rules count
func TestUpdateEBPFRules(t *testing.T) {
	monitor := NewMonitor(9105)

	monitor.UpdateEBPFRules(10)
	monitor.UpdateEBPFRules(25)

	// Verify metric was updated
}

// TestRecordEBPFPacketFiltered tests recording eBPF packet filtering
func TestRecordEBPFPacketFiltered(t *testing.T) {
	monitor := NewMonitor(9106)

	monitor.RecordEBPFPacketFiltered("drop", "rule-1")
	monitor.RecordEBPFPacketFiltered("allow", "rule-2")
	monitor.RecordEBPFPacketFiltered("drop", "rule-1")

	// Verify metrics were recorded
}

// TestUpdateLicenseStatus tests updating license status metric
func TestUpdateLicenseStatus(t *testing.T) {
	monitor := NewMonitor(9107)

	expiryTime := time.Now().Add(30 * 24 * time.Hour)
	monitor.UpdateLicenseStatus(true, expiryTime)

	// Verify metric was updated

	monitor.UpdateLicenseStatus(false, time.Time{})
	// Verify metric was updated
}

// TestUpdateLicenseStatusExpired tests expired license scenario
func TestUpdateLicenseStatusExpired(t *testing.T) {
	monitor := NewMonitor(9108)

	expiredTime := time.Now().Add(-24 * time.Hour)
	monitor.UpdateLicenseStatus(false, expiredTime)

	// Verify metric was updated to reflect expiration
}

// TestUpdateConfigVersion tests updating config version metric
func TestUpdateConfigVersion(t *testing.T) {
	monitor := NewMonitor(9109)

	monitor.UpdateConfigVersion("cluster-1", "hash-abc123")
	monitor.UpdateConfigVersion("cluster-2", "hash-def456")

	// Verify metric was updated
}

// TestMonitorMultipleMetrics tests recording multiple metrics in sequence
func TestMonitorMultipleMetrics(t *testing.T) {
	monitor := NewMonitor(9110)
	hc := &MockHealthChecker{healthy: true}
	monitor.SetHealthChecker(hc)

	// Record various metrics
	monitor.RecordConnection("tcp", "192.168.1.1", "10.0.0.1")
	monitor.RecordBytesTransferred("inbound", "tcp", 1024)
	monitor.RecordAuthAttempt("oauth2", "success")
	monitor.RecordRequestLatency("http", "/api/v1/test", 100*time.Millisecond)
	monitor.UpdateThroughput("inbound", 1024.5)
	monitor.UpdateEBPFRules(5)
	monitor.UpdateLicenseStatus(true, time.Now().Add(30*24*time.Hour))

	// Verify health check still works
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	monitor.healthzHandler(w, req)

	if w.Code != http.StatusOK {
		t.Error("health check failed after recording metrics")
	}
}

// TestShutdown tests monitor shutdown (basic test - can't fully test without Start)
func TestShutdown(t *testing.T) {
	monitor := NewMonitor(9111)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Shutdown without starting should not error
	err := monitor.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// TestCollectSystemMetrics tests system metrics collection
func TestCollectSystemMetrics(t *testing.T) {
	monitor := NewMonitor(9112)

	// Call collectSystemMetrics once
	go func() {
		monitor.collectSystemMetrics()
	}()

	// Give it a moment to run
	time.Sleep(100 * time.Millisecond)
}

// TestMonitorConcurrency tests concurrent metric recording
func TestMonitorConcurrency(t *testing.T) {
	monitor := NewMonitor(9113)

	// Run concurrent metric recording
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				monitor.RecordConnection("tcp", "192.168.1.1", "10.0.0.1")
				monitor.RecordBytesTransferred("inbound", "tcp", 1024)
				monitor.RecordAuthAttempt("oauth2", "success")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without panic or race condition
}

// TestMonitorMutexProtection tests that monitor uses proper mutex protection
func TestMonitorMutexProtection(t *testing.T) {
	monitor := NewMonitor(9114)
	hc1 := &MockHealthChecker{healthy: true}
	hc2 := &MockHealthChecker{healthy: false}

	done := make(chan bool, 5)

	// Concurrent health checker changes
	for i := 0; i < 5; i++ {
		go func(id int) {
			if id%2 == 0 {
				monitor.SetHealthChecker(hc1)
			} else {
				monitor.SetHealthChecker(hc2)
			}
			done <- true
		}(i)
	}

	// Concurrent health check reads
	for i := 0; i < 5; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/healthz", nil)
			w := httptest.NewRecorder()
			monitor.healthzHandler(w, req)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestStatusHandlerJSONFormat tests that status response is valid JSON
func TestStatusHandlerJSONFormat(t *testing.T) {
	monitor := NewMonitor(9115)
	hc := &MockHealthChecker{healthy: true}
	monitor.SetHealthChecker(hc)

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	monitor.statusHandler(w, req)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Verify expected fields exist
	expectedFields := []string{"timestamp", "healthy", "version", "goroutines", "memory_alloc"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("expected field %q in status response", field)
		}
	}
}

// TestRecordConnectionMultipleEndpoints tests recording connections to different endpoints
func TestRecordConnectionMultipleEndpoints(t *testing.T) {
	monitor := NewMonitor(9116)

	endpoints := []struct {
		protocol string
		source   string
		dest     string
	}{
		{"tcp", "192.168.1.1", "10.0.0.1"},
		{"tcp", "192.168.1.2", "10.0.0.2"},
		{"udp", "192.168.1.1", "10.0.0.1"},
		{"http2", "192.168.1.3", "10.0.0.3"},
	}

	for _, ep := range endpoints {
		monitor.RecordConnection(ep.protocol, ep.source, ep.dest)
	}

	// All should complete without error
}

// TestRecordBytesTransferredVariousDirections tests bytes transfer in various directions
func TestRecordBytesTransferredVariousDirections(t *testing.T) {
	monitor := NewMonitor(9117)

	directions := []string{"inbound", "outbound", "forward", "reverse"}
	protocols := []string{"tcp", "udp", "http", "grpc"}

	for _, dir := range directions {
		for _, proto := range protocols {
			monitor.RecordBytesTransferred(dir, proto, 1024)
		}
	}

	// All should complete without error
}

// TestUpdateMultipleConfigs tests updating multiple config versions
func TestUpdateMultipleConfigs(t *testing.T) {
	monitor := NewMonitor(9118)

	configs := []struct {
		clusterID   string
		versionHash string
	}{
		{"cluster-1", "hash1"},
		{"cluster-1", "hash2"}, // Update same cluster
		{"cluster-2", "hash1"},
		{"cluster-3", "hash3"},
	}

	for _, cfg := range configs {
		monitor.UpdateConfigVersion(cfg.clusterID, cfg.versionHash)
	}

	// All should complete without error
}

// TestNewMonitorWithDifferentPorts tests creating monitors with different ports
func TestNewMonitorWithDifferentPorts(t *testing.T) {
	ports := []int{8000, 8001, 8002, 9000, 9090, 9091}

	for _, port := range ports {
		monitor := NewMonitor(port)
		if monitor == nil {
			t.Errorf("failed to create monitor for port %d", port)
		}
	}
}

// TestHealthCheckerInterface tests the HealthChecker interface contract
func TestHealthCheckerInterface(t *testing.T) {
	hc := &MockHealthChecker{
		healthy: true,
		status: map[string]interface{}{
			"db":    "connected",
			"cache": "ok",
		},
	}

	if !hc.IsHealthy() {
		t.Error("expected IsHealthy to return true")
	}

	status := hc.GetStatus()
	if status["db"] != "connected" {
		t.Error("expected db status to be connected")
	}
}
