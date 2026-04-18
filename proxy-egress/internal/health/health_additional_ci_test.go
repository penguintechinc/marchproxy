//go:build ci

package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

// TestAddServiceAndRemove tests service registration and removal
func TestAddServiceAndRemove(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:       30 * time.Second,
		Timeout:             5 * time.Second,
		HealthyThreshold:    2,
		UnhealthyThreshold:  3,
		EnabledChecks:       []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	service := &manager.Service{
		IPFQDN: "localhost:8080",
	}

	checker.AddService(service)
	health := checker.GetServiceHealth(service)
	if health == nil {
		t.Fatal("expected service health to exist after AddService")
	}
	if health.Status != StatusUnknown {
		t.Errorf("expected initial status StatusUnknown, got %s", health.Status)
	}

	checker.RemoveService(service)
	health = checker.GetServiceHealth(service)
	if health != nil {
		t.Error("expected service health to be nil after RemoveService")
	}
}

// TestGetAllServiceHealthMultiple tests retrieving multiple services
func TestGetAllServiceHealthMultiple(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:      30 * time.Second,
		Timeout:            5 * time.Second,
		EnabledChecks:      []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	service1 := &manager.Service{IPFQDN: "localhost:8080"}
	service2 := &manager.Service{IPFQDN: "localhost:8081"}
	service3 := &manager.Service{IPFQDN: "localhost:8082"}

	checker.AddService(service1)
	checker.AddService(service2)
	checker.AddService(service3)

	all := checker.GetAllServiceHealth()
	if len(all) != 3 {
		t.Errorf("expected 3 services, got %d", len(all))
	}

	if _, exists := all["localhost:8080"]; !exists {
		t.Error("expected service1 in all services")
	}
	if _, exists := all["localhost:8081"]; !exists {
		t.Error("expected service2 in all services")
	}
	if _, exists := all["localhost:8082"]; !exists {
		t.Error("expected service3 in all services")
	}
}

// TestStartStop tests health checker start/stop lifecycle
func TestStartStop(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:      30 * time.Second,
		Timeout:            5 * time.Second,
		EnabledChecks:      []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	err := checker.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Starting twice should error
	err = checker.Start()
	if err == nil {
		t.Error("expected Start to error when already running")
	}

	err = checker.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Stop should succeed even if not running
	err = checker.Stop()
	if err != nil {
		t.Fatalf("Stop failed on second call: %v", err)
	}
}

// TestHTTPHealthCheckRequest tests HTTP health check with successful response
func TestHTTPHealthCheckRequest(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Extract host:port from server URL
	listener := server.Listener
	hostPort := listener.Addr().String()

	check := NewHTTPHealthCheck("http", HTTPExpected{
		StatusCodes:     []int{200, 201, 202, 204},
		MaxResponseTime: 5 * time.Second,
	})

	service := &manager.Service{IPFQDN: hostPort}

	result := check.Check(context.Background(), service)
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s, error: %v", result.Status, result.Error)
	}
}

// TestHTTPHealthCheckInvalidStatus tests HTTP check with unexpected status
func TestHTTPHealthCheckInvalidStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	hostPort := server.Listener.Addr().String()

	check := NewHTTPHealthCheck("http", HTTPExpected{
		StatusCodes:     []int{200, 201},
		MaxResponseTime: 5 * time.Second,
	})

	service := &manager.Service{IPFQDN: hostPort}
	result := check.Check(context.Background(), service)
	if result.Status != StatusUnhealthy {
		t.Errorf("expected StatusUnhealthy for 500, got %s", result.Status)
	}
}

// TestHTTPHealthCheckSlowResponse tests response time threshold
func TestHTTPHealthCheckSlowResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hostPort := server.Listener.Addr().String()

	check := NewHTTPHealthCheck("http", HTTPExpected{
		StatusCodes:     []int{200},
		MaxResponseTime: 100 * time.Millisecond, // Lower than sleep
	})

	service := &manager.Service{IPFQDN: hostPort}
	result := check.Check(context.Background(), service)
	if result.Status != StatusDegraded {
		t.Errorf("expected StatusDegraded for slow response, got %s", result.Status)
	}
}

// TestTCPHealthCheckSuccess tests TCP connectivity check
func TestTCPHealthCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hostPort := server.Listener.Addr().String()

	check := NewTCPHealthCheck("tcp")
	service := &manager.Service{IPFQDN: hostPort}

	result := check.Check(context.Background(), service)
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s, error: %v", result.Status, result.Error)
	}
}

// TestTCPHealthCheckFailure tests TCP check to unreachable host
func TestTCPHealthCheckFailure(t *testing.T) {
	check := NewTCPHealthCheck("tcp")
	// Use an address that will not be reachable
	service := &manager.Service{IPFQDN: "127.0.0.1:1"}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := check.Check(ctx, service)
	if result.Status != StatusUnhealthy {
		t.Errorf("expected StatusUnhealthy, got %s", result.Status)
	}
	if result.Error == nil {
		t.Error("expected error for failed TCP connection")
	}
}

// TestProcessHealthCheckEnabled tests process health check
func TestProcessHealthCheck(t *testing.T) {
	check := NewProcessHealthCheck("process")
	if !check.Enabled() {
		t.Error("expected process check to be enabled")
	}
	if check.Name() != "process" {
		t.Errorf("expected name 'process', got %s", check.Name())
	}

	service := &manager.Service{IPFQDN: "localhost:8080"}
	result := check.Check(context.Background(), service)
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", result.Status)
	}
}

// TestMemoryHealthCheck tests memory health check
func TestMemoryHealthCheck(t *testing.T) {
	check := NewMemoryHealthCheck("memory", 1024*1024*1024)
	if !check.Enabled() {
		t.Error("expected memory check to be enabled")
	}
	if check.Name() != "memory" {
		t.Errorf("expected name 'memory', got %s", check.Name())
	}

	service := &manager.Service{IPFQDN: "localhost:8080"}
	result := check.Check(context.Background(), service)
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", result.Status)
	}
}

// TestDiskHealthCheck tests disk health check
func TestDiskHealthCheck(t *testing.T) {
	check := NewDiskHealthCheck("disk", "/tmp", 1024*1024*1024)
	if !check.Enabled() {
		t.Error("expected disk check to be enabled")
	}
	if check.Name() != "disk" {
		t.Errorf("expected name 'disk', got %s", check.Name())
	}

	service := &manager.Service{IPFQDN: "localhost:8080"}
	result := check.Check(context.Background(), service)
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", result.Status)
	}
}

// TestReadinessProbe tests readiness probe execution
func TestReadinessProbe(t *testing.T) {
	config := ProbeConfig{
		Type:    ProbeTypeHTTP,
		Path:    "/ready",
		Port:    8080,
		Timeout: 5 * time.Second,
	}
	probe := NewReadinessProbe("readiness", config)
	if probe.Name() != "readiness" {
		t.Errorf("expected probe name 'readiness', got %s", probe.Name())
	}
	retrievedConfig := probe.GetConfig()
	if retrievedConfig.Type != config.Type {
		t.Error("expected GetConfig to return the same config type")
	}

	result := probe.Execute(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", result.Status)
	}
}

// TestLivenessProbe tests liveness probe execution
func TestLivenessProbe(t *testing.T) {
	config := ProbeConfig{
		Type:    ProbeTypeTCP,
		Path:    "",
		Port:    8080,
		Timeout: 5 * time.Second,
	}
	probe := NewLivenessProbe("liveness", config)
	if probe.Name() != "liveness" {
		t.Errorf("expected probe name 'liveness', got %s", probe.Name())
	}

	result := probe.Execute(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", result.Status)
	}
}

// TestExecuteProbeNotFound tests probe execution with non-existent probe
func TestExecuteProbeNotFound(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
		EnableProbes:  true,
		ProbeEndpoints: map[string]ProbeConfig{
			"ready": {
				Type:    ProbeTypeHTTP,
				Path:    "/ready",
				Port:    8080,
				Timeout: 5 * time.Second,
			},
		},
	}
	checker := NewHealthChecker(cfg)

	result := checker.ExecuteProbe("nonexistent")
	if result.Status != StatusUnknown {
		t.Errorf("expected StatusUnknown, got %s", result.Status)
	}
	if result.Error == nil {
		t.Error("expected error for non-existent probe")
	}
}

// TestDetermineHealthStatusTransitions tests status transitions
func TestDetermineHealthStatusTransitions(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:       30 * time.Second,
		Timeout:             5 * time.Second,
		HealthyThreshold:    2,
		UnhealthyThreshold:  3,
		EnabledChecks:       []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	// Create service health
	sh := &ServiceHealth{
		Service:               &manager.Service{IPFQDN: "localhost:8080"},
		Status:                StatusUnknown,
		ConsecutiveSuccesses:  0,
		ConsecutiveFailures:   0,
	}

	// Single success should result in degraded (below threshold)
	sh.ConsecutiveSuccesses = 1
	sh.ConsecutiveFailures = 0
	result := &CheckResult{Status: StatusHealthy}
	newStatus := checker.determineHealthStatus(sh, result)
	if newStatus != StatusDegraded && newStatus != StatusUnknown {
		t.Errorf("unexpected status after 1 success: %s", newStatus)
	}

	// Two successes should be healthy
	sh.ConsecutiveSuccesses = 2
	newStatus = checker.determineHealthStatus(sh, result)
	if newStatus != StatusHealthy {
		t.Errorf("expected StatusHealthy after 2 successes, got %s", newStatus)
	}

	// Failure should stay degraded if already healthy
	sh.Status = StatusHealthy
	sh.ConsecutiveSuccesses = 0
	sh.ConsecutiveFailures = 1
	result = &CheckResult{Status: StatusUnhealthy}
	newStatus = checker.determineHealthStatus(sh, result)
	if newStatus != StatusDegraded {
		t.Errorf("expected StatusDegraded on first failure from healthy, got %s", newStatus)
	}

	// Three failures should be unhealthy
	sh.ConsecutiveFailures = 3
	newStatus = checker.determineHealthStatus(sh, result)
	if newStatus != StatusUnhealthy {
		t.Errorf("expected StatusUnhealthy after 3 failures, got %s", newStatus)
	}
}

// TestGetSystemHealthAllHealthy tests system status when all services healthy
func TestGetSystemHealthAllHealthy(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:  30 * time.Second,
		Timeout:        5 * time.Second,
		EnableProbes:   false,
		EnabledChecks:  []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	service1 := &manager.Service{IPFQDN: "localhost:8080"}
	service2 := &manager.Service{IPFQDN: "localhost:8081"}

	checker.AddService(service1)
	checker.AddService(service2)

	// Manually set statuses to healthy
	all := checker.GetAllServiceHealth()
	for _, sh := range all {
		sh.Status = StatusHealthy
	}

	sysHealth := checker.GetSystemHealth()
	if sysHealth.Status != StatusHealthy {
		t.Errorf("expected system StatusHealthy when all services healthy, got %s", sysHealth.Status)
	}
}

// TestGetSystemHealthSomeDegraded tests system status with mixed health
func TestGetSystemHealthSomeDegraded(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:  30 * time.Second,
		Timeout:        5 * time.Second,
		EnableProbes:   false,
		EnabledChecks:  []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	service1 := &manager.Service{IPFQDN: "localhost:8080"}
	service2 := &manager.Service{IPFQDN: "localhost:8081"}

	checker.AddService(service1)
	checker.AddService(service2)

	all := checker.GetAllServiceHealth()
	services := make([]*ServiceHealth, 0, len(all))
	for _, sh := range all {
		services = append(services, sh)
	}

	if len(services) >= 1 {
		services[0].Status = StatusHealthy
	}
	if len(services) >= 2 {
		services[1].Status = StatusDegraded
	}

	sysHealth := checker.GetSystemHealth()
	if sysHealth.Status != StatusDegraded {
		t.Errorf("expected system StatusDegraded with mixed health, got %s", sysHealth.Status)
	}
}

// TestGetSystemHealthAllUnhealthy tests system status when all services unhealthy
func TestGetSystemHealthAllUnhealthy(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:  30 * time.Second,
		Timeout:        5 * time.Second,
		EnableProbes:   false,
		EnabledChecks:  []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	service1 := &manager.Service{IPFQDN: "localhost:8080"}
	service2 := &manager.Service{IPFQDN: "localhost:8081"}

	checker.AddService(service1)
	checker.AddService(service2)

	all := checker.GetAllServiceHealth()
	for _, sh := range all {
		sh.Status = StatusUnhealthy
	}

	sysHealth := checker.GetSystemHealth()
	if sysHealth.Status != StatusUnhealthy {
		t.Errorf("expected system StatusUnhealthy when all services unhealthy, got %s", sysHealth.Status)
	}
}

// TestGetMetricsSummary tests metrics reporting
func TestGetMetricsSummary(t *testing.T) {
	cfg := HealthConfig{
		CheckInterval:  30 * time.Second,
		Timeout:        5 * time.Second,
		EnabledChecks:  []string{"tcp"},
	}
	checker := NewHealthChecker(cfg)

	summary := checker.getMetricsSummary()
	if summary.TotalChecks != 0 {
		t.Errorf("expected TotalChecks=0 initially, got %d", summary.TotalChecks)
	}
	if summary.CheckFailures != 0 {
		t.Errorf("expected CheckFailures=0 initially, got %d", summary.CheckFailures)
	}
}

// TestHTTPHealthCheckConfigure tests HTTP check configuration
func TestHTTPHealthCheckConfigure(t *testing.T) {
	check := NewHTTPHealthCheck("http", HTTPExpected{
		StatusCodes:     []int{200},
		MaxResponseTime: 5 * time.Second,
	})

	config := map[string]interface{}{
		"path":   "/custom",
		"method": "POST",
	}
	err := check.Configure(config)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
}

// TestTCPHealthCheckConfigure tests TCP check configuration
func TestTCPHealthCheckConfigure(t *testing.T) {
	check := NewTCPHealthCheck("tcp")
	err := check.Configure(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
}

// TestProbeTypesConstants tests probe type constants
func TestProbeTypesConstants(t *testing.T) {
	tests := []struct {
		name      string
		probeType ProbeType
		expected  string
	}{
		{"HTTP", ProbeTypeHTTP, "http"},
		{"TCP", ProbeTypeTCP, "tcp"},
		{"Exec", ProbeTypeExec, "exec"},
	}

	for _, tt := range tests {
		if string(tt.probeType) != tt.expected {
			t.Errorf("probe type %s: expected %q, got %q", tt.name, tt.expected, tt.probeType)
		}
	}
}

// TestHealthMetricsRecording tests metrics recording
func TestHealthMetricsRecording(t *testing.T) {
	metrics := &HealthMetrics{}

	metrics.recordCheck(100 * time.Millisecond)
	if metrics.TotalChecks != 1 {
		t.Errorf("expected TotalChecks=1, got %d", metrics.TotalChecks)
	}

	metrics.recordCheckFailure()
	if metrics.CheckFailures != 1 {
		t.Errorf("expected CheckFailures=1, got %d", metrics.CheckFailures)
	}

	metrics.recordStatusChange()
	if metrics.StatusChanges != 1 {
		t.Errorf("expected StatusChanges=1, got %d", metrics.StatusChanges)
	}
}
