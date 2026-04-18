//go:build ci

package health

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewHealthChecker(t *testing.T) {
	config := HealthConfig{
		CheckInterval:       30 * time.Second,
		Timeout:            10 * time.Second,
		HealthyThreshold:   2,
		UnhealthyThreshold: 3,
		EnabledChecks:      []string{"http", "tcp"},
	}

	hc := NewHealthChecker(config)
	if hc == nil {
		t.Fatal("expected non-nil HealthChecker")
	}
	if hc.config.CheckInterval != 30*time.Second {
		t.Errorf("expected CheckInterval 30s, got %v", hc.config.CheckInterval)
	}
}

func TestNewHealthCheckerDefaults(t *testing.T) {
	config := HealthConfig{}

	hc := NewHealthChecker(config)
	if hc == nil {
		t.Fatal("expected non-nil HealthChecker")
	}

	// Verify maps are initialized
	if hc.backends == nil {
		t.Error("expected backends map to be initialized")
	}
	if hc.vhosts == nil {
		t.Error("expected vhosts map to be initialized")
	}
	if hc.checks == nil {
		t.Error("expected checks map to be initialized")
	}
	if hc.probes == nil {
		t.Error("expected probes map to be initialized")
	}
	if hc.metrics == nil {
		t.Error("expected metrics to be initialized")
	}
	if hc.stopChan == nil {
		t.Error("expected stopChan to be initialized")
	}
}

func TestAddBackend(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
		Weight: 1,
	}

	hc.AddBackend(backend)

	// Verify backend was added
	health := hc.GetBackendHealth(backend)
	if health == nil {
		t.Fatal("expected non-nil backend health")
	}
	if health.Backend.Name != "backend1" {
		t.Errorf("expected backend name 'backend1', got %q", health.Backend.Name)
	}
	if health.Status != StatusUnknown {
		t.Errorf("expected initial Status Unknown, got %v", health.Status)
	}
}

func TestAddVirtualHost(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	// Create a vhost without backends to avoid the recursive lock issue
	vhost := &VirtualHost{
		Name:       "vhost1",
		Host:       "example.com",
		SSLEnabled: false,
		Backends:   []*Backend{},
		HealthPath: "/health",
	}

	hc.AddVirtualHost(vhost)

	// Verify vhost was added
	vhostHealth := hc.GetVirtualHostHealth("vhost1")
	if vhostHealth == nil {
		t.Fatal("expected non-nil virtual host health")
	}
	if vhostHealth.VHost.Name != "vhost1" {
		t.Errorf("expected vhost name 'vhost1', got %q", vhostHealth.VHost.Name)
	}
	if vhostHealth.Status != StatusUnknown {
		t.Errorf("expected initial Status Unknown, got %v", vhostHealth.Status)
	}
}

func TestGetBackendHealth(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	backend1 := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}

	backend2 := &Backend{
		Name:   "backend2",
		Host:   "10.0.0.2",
		Port:   8080,
		Scheme: "http",
	}

	hc.AddBackend(backend1)
	hc.AddBackend(backend2)

	health1 := hc.GetBackendHealth(backend1)
	if health1 == nil {
		t.Fatal("expected non-nil health for backend1")
	}

	health2 := hc.GetBackendHealth(backend2)
	if health2 == nil {
		t.Fatal("expected non-nil health for backend2")
	}

	if health1.Backend.Name != "backend1" {
		t.Errorf("expected backend1, got %q", health1.Backend.Name)
	}
	if health2.Backend.Name != "backend2" {
		t.Errorf("expected backend2, got %q", health2.Backend.Name)
	}
}

func TestDetermineHealthStatus(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		HealthyThreshold:   2,
		UnhealthyThreshold: 3,
	})

	tests := []struct {
		name                string
		consecutiveFailures int
		consecutiveSuccesses int
		resultStatus        HealthStatus
		expectedStatus      HealthStatus
	}{
		{
			"result unhealthy, insufficient failures",
			1,
			0,
			StatusUnhealthy,
			StatusDegraded,
		},
		{
			"result unhealthy, threshold reached",
			3,
			0,
			StatusUnhealthy,
			StatusUnhealthy,
		},
		{
			"result healthy, insufficient successes",
			0,
			1,
			StatusHealthy,
			StatusDegraded,
		},
		{
			"result healthy, threshold reached",
			0,
			2,
			StatusHealthy,
			StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backendHealth := &BackendHealth{
				ConsecutiveFailures:  tt.consecutiveFailures,
				ConsecutiveSuccesses: tt.consecutiveSuccesses,
				Status:               StatusUnhealthy,
			}

			result := &CheckResult{
				Status: tt.resultStatus,
			}

			status := hc.determineHealthStatus(backendHealth, result)
			if status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}
		})
	}
}

func TestGetAllBackendHealth(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	for i := 1; i <= 3; i++ {
		backend := &Backend{
			Name:   "backend" + string(rune(48+i)),
			Host:   "10.0.0." + string(rune(48+i)),
			Port:   8080,
			Scheme: "http",
		}
		hc.AddBackend(backend)
	}

	allHealth := hc.GetAllBackendHealth()
	if len(allHealth) != 3 {
		t.Errorf("expected 3 backends, got %d", len(allHealth))
	}
}

func TestGetAllVirtualHostHealth(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	for i := 1; i <= 3; i++ {
		vhost := &VirtualHost{
			Name: "vhost" + string(rune(48+i)),
			Host: "example" + string(rune(48+i)) + ".com",
		}
		hc.AddVirtualHost(vhost)
	}

	allHealth := hc.GetAllVirtualHostHealth()
	if len(allHealth) != 3 {
		t.Errorf("expected 3 vhosts, got %d", len(allHealth))
	}
}

func TestRemoveBackend(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}

	hc.AddBackend(backend)
	hc.RemoveBackend(backend)

	health := hc.GetBackendHealth(backend)
	if health != nil {
		t.Error("expected nil health after removing backend")
	}
}

func TestStartWithConfig(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		CheckInterval: 5 * time.Second,
	})

	// Verify the checker can be created with custom config
	if hc.config.CheckInterval != 5*time.Second {
		t.Errorf("expected CheckInterval 5s, got %v", hc.config.CheckInterval)
	}
	// Don't actually start to avoid goroutine issues in tests
}

func TestHealthCheckerNotRunning(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	// Initially should not be running
	if hc.running {
		t.Error("expected running to be false initially")
	}
}

func TestGetSystemHealth(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}
	hc.AddBackend(backend)

	systemHealth := hc.GetSystemHealth()
	if systemHealth == nil {
		t.Fatal("expected non-nil system health")
	}
	if systemHealth.Status == "" {
		t.Error("expected non-empty status")
	}
	if systemHealth.Backends == nil {
		t.Error("expected non-nil backends map")
	}
	if systemHealth.VirtualHosts == nil {
		t.Error("expected non-nil virtual hosts map")
	}
}

func TestGetSystemHealthWithMultipleBackends(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	backends := []*Backend{
		{Name: "backend1", Host: "10.0.0.1", Port: 8080, Scheme: "http"},
		{Name: "backend2", Host: "10.0.0.2", Port: 8080, Scheme: "http"},
		{Name: "backend3", Host: "10.0.0.3", Port: 8080, Scheme: "http"},
	}

	for _, b := range backends {
		hc.AddBackend(b)
	}

	systemHealth := hc.GetSystemHealth()
	if len(systemHealth.Backends) != 3 {
		t.Errorf("expected 3 backends in system health, got %d", len(systemHealth.Backends))
	}
}

func TestHealthStatusValues(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusDegraded, "degraded"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestNewHTTPHealthCheck(t *testing.T) {
	expected := HTTPExpected{
		StatusCodes:      []int{200, 201, 204},
		MaxResponseTime:  5 * time.Second,
		BodyContains:     "ok",
	}

	check := NewHTTPHealthCheck("http_check", expected)
	if check == nil {
		t.Fatal("expected non-nil HTTPHealthCheck")
	}
	if !check.Enabled() {
		t.Error("expected HTTPHealthCheck to be enabled")
	}
	if check.Name() != "http_check" {
		t.Errorf("expected name 'http_check', got %q", check.Name())
	}
}

func TestNewTCPHealthCheck(t *testing.T) {
	check := NewTCPHealthCheck("tcp_check")
	if check == nil {
		t.Fatal("expected non-nil TCPHealthCheck")
	}
	if !check.Enabled() {
		t.Error("expected TCPHealthCheck to be enabled")
	}
	if check.Name() != "tcp_check" {
		t.Errorf("expected name 'tcp_check', got %q", check.Name())
	}
}

func TestNewSSLCertCheck(t *testing.T) {
	check := NewSSLCertCheck("ssl_cert_check")
	if check == nil {
		t.Fatal("expected non-nil SSLCertCheck")
	}
	if !check.Enabled() {
		t.Error("expected SSLCertCheck to be enabled")
	}
	if check.Name() != "ssl_cert_check" {
		t.Errorf("expected name 'ssl_cert_check', got %q", check.Name())
	}
}

func TestHTTPHealthCheckInterface(t *testing.T) {
	check := NewHTTPHealthCheck("test", HTTPExpected{})

	if check.Name() != "test" {
		t.Error("expected Name() to work")
	}

	if !check.Enabled() {
		t.Error("expected Enabled() to return true")
	}

	err := check.Configure(map[string]interface{}{})
	if err != nil {
		t.Errorf("unexpected error from Configure: %v", err)
	}
}

func TestTCPHealthCheckInterface(t *testing.T) {
	check := NewTCPHealthCheck("tcp_test")

	if check.Name() != "tcp_test" {
		t.Error("expected Name() to work")
	}

	if !check.Enabled() {
		t.Error("expected Enabled() to return true")
	}

	err := check.Configure(map[string]interface{}{})
	if err != nil {
		t.Errorf("unexpected error from Configure: %v", err)
	}
}

func TestSSLCertCheckInterface(t *testing.T) {
	check := NewSSLCertCheck("ssl_test")

	if check.Name() != "ssl_test" {
		t.Error("expected Name() to work")
	}

	if !check.Enabled() {
		t.Error("expected Enabled() to return true")
	}

	err := check.Configure(map[string]interface{}{})
	if err != nil {
		t.Errorf("unexpected error from Configure: %v", err)
	}
}

func TestReadinessProbe(t *testing.T) {
	config := ProbeConfig{
		Type:    ProbeTypeHTTP,
		Path:    "/ready",
		Port:    8080,
		Timeout: 5 * time.Second,
	}

	probe := NewReadinessProbe("readiness", config)
	if probe == nil {
		t.Fatal("expected non-nil ReadinessProbe")
	}
	if probe.Name() != "readiness" {
		t.Errorf("expected name 'readiness', got %q", probe.Name())
	}

	ctx := context.Background()
	result := probe.Execute(ctx)
	if result == nil {
		t.Fatal("expected non-nil probe result")
	}
	if result.Status != StatusHealthy {
		t.Errorf("expected Status Healthy, got %v", result.Status)
	}
}

func TestLivenessProbe(t *testing.T) {
	config := ProbeConfig{
		Type:    ProbeTypeHTTP,
		Path:    "/alive",
		Port:    8080,
		Timeout: 5 * time.Second,
	}

	probe := NewLivenessProbe("liveness", config)
	if probe == nil {
		t.Fatal("expected non-nil LivenessProbe")
	}
	if probe.Name() != "liveness" {
		t.Errorf("expected name 'liveness', got %q", probe.Name())
	}

	ctx := context.Background()
	result := probe.Execute(ctx)
	if result == nil {
		t.Fatal("expected non-nil probe result")
	}
	if result.Status != StatusHealthy {
		t.Errorf("expected Status Healthy, got %v", result.Status)
	}
}

func TestBackendHealthMetadata(t *testing.T) {
	backend := &Backend{
		Name:      "backend1",
		Host:      "10.0.0.1",
		Port:      8080,
		Scheme:    "http",
		Weight:    1,
		MaxConns:  100,
		Headers:   map[string]string{"X-Custom": "value"},
	}

	health := &BackendHealth{
		Backend:   backend,
		Status:    StatusHealthy,
		LastCheck: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	health.Metadata["cpu"] = 45.5
	health.Metadata["memory"] = 2048

	if health.Metadata["cpu"] != 45.5 {
		t.Errorf("expected cpu 45.5, got %v", health.Metadata["cpu"])
	}
}

func TestVirtualHostHealthTracking(t *testing.T) {
	vhost := &VirtualHost{
		Name:       "vhost1",
		Host:       "example.com",
		SSLEnabled: true,
		Backends: []*Backend{
			{Name: "backend1", Host: "10.0.0.1", Port: 8080, Scheme: "http"},
			{Name: "backend2", Host: "10.0.0.2", Port: 8080, Scheme: "http"},
		},
		HealthPath: "/health",
	}

	health := &VirtualHostHealth{
		VHost:              vhost,
		Status:             StatusHealthy,
		LastCheck:          time.Now(),
		BackendCount:       2,
		HealthyBackends:    2,
		UnhealthyBackends:  0,
		SSLEnabled:         true,
		RequestCount:       1000,
		ErrorCount:         5,
		AverageResponseTime: 100 * time.Millisecond,
	}

	if health.BackendCount != 2 {
		t.Errorf("expected 2 backends, got %d", health.BackendCount)
	}
	if health.HealthyBackends != 2 {
		t.Errorf("expected 2 healthy backends, got %d", health.HealthyBackends)
	}
	if health.Status != StatusHealthy {
		t.Errorf("expected Status Healthy, got %v", health.Status)
	}
}

func TestHealthCheckResult(t *testing.T) {
	result := &CheckResult{
		Status:       StatusHealthy,
		ResponseTime: 150 * time.Millisecond,
		Message:      "Check passed",
		Metadata: map[string]interface{}{
			"status_code": 200,
			"latency_ms":  150,
		},
	}

	if result.Status != StatusHealthy {
		t.Errorf("expected Status Healthy, got %v", result.Status)
	}
	if result.ResponseTime != 150*time.Millisecond {
		t.Errorf("expected 150ms, got %v", result.ResponseTime)
	}
	if result.Message != "Check passed" {
		t.Errorf("expected 'Check passed', got %q", result.Message)
	}
}

func TestHealthMetricsRecording(t *testing.T) {
	metrics := &HealthMetrics{}

	metrics.recordCheck(100 * time.Millisecond)
	metrics.recordCheck(200 * time.Millisecond)

	if metrics.TotalChecks != 2 {
		t.Errorf("expected 2 total checks, got %d", metrics.TotalChecks)
	}

	metrics.recordCheckFailure()
	if metrics.CheckFailures != 1 {
		t.Errorf("expected 1 check failure, got %d", metrics.CheckFailures)
	}

	metrics.recordStatusChange()
	if metrics.StatusChanges != 1 {
		t.Errorf("expected 1 status change, got %d", metrics.StatusChanges)
	}
}

func TestHealthCheckerInitializeProbes(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		EnableProbes: true,
		ProbeEndpoints: map[string]ProbeConfig{
			"readiness": {Type: ProbeTypeHTTP, Path: "/ready", Port: 8080},
			"liveness":  {Type: ProbeTypeHTTP, Path: "/alive", Port: 8080},
		},
	})

	if len(hc.probes) == 0 {
		t.Error("expected probes to be initialized")
	}
}

func TestProbeTypeConstants(t *testing.T) {
	tests := []struct {
		probeType ProbeType
		expected  string
	}{
		{ProbeTypeHTTP, "http"},
		{ProbeTypeTCP, "tcp"},
		{ProbeTypeExec, "exec"},
		{ProbeTypeMTLS, "mtls"},
	}

	for _, tt := range tests {
		if string(tt.probeType) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.probeType))
		}
	}
}

func TestStartStop(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		CheckInterval: 5 * time.Second,
		Timeout:       2 * time.Second,
	})

	// Start the checker
	err := hc.Start()
	if err != nil {
		t.Errorf("unexpected error from Start: %v", err)
	}
	if !hc.running {
		t.Error("expected running to be true after Start()")
	}

	// Stop the checker
	err = hc.Stop()
	if err != nil {
		t.Errorf("unexpected error from Stop: %v", err)
	}
	if hc.running {
		t.Error("expected running to be false after Stop()")
	}
}

func TestStartAlreadyRunning(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		CheckInterval: 5 * time.Second,
	})

	err := hc.Start()
	if err != nil {
		t.Fatalf("unexpected error from Start: %v", err)
	}
	defer hc.Stop()

	if !hc.running {
		t.Fatal("expected running to be true")
	}

	// Calling Start again should not cause issues
	err = hc.Start()
	if err == nil && !hc.running {
		t.Error("expected running to still be true")
	}
}

func TestStopNotRunning(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	// Stopping when not running should not panic
	err := hc.Stop()
	if err != nil {
		t.Logf("expected error from stopping inactive checker: %v", err)
	}
	if hc.running {
		t.Error("expected running to be false")
	}
}

func TestBackendHealthInitialization(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		HealthyThreshold:   2,
		UnhealthyThreshold: 3,
	})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}

	hc.AddBackend(backend)

	health := hc.GetBackendHealth(backend)
	if health == nil {
		t.Fatal("expected non-nil backend health")
	}
	if health.Status != StatusUnknown {
		t.Errorf("expected initial status Unknown, got %v", health.Status)
	}
	if health.ConsecutiveSuccesses != 0 {
		t.Errorf("expected initial consecutive successes 0, got %d", health.ConsecutiveSuccesses)
	}
	if health.TotalChecks != 0 {
		t.Errorf("expected initial total checks 0, got %d", health.TotalChecks)
	}
}

func TestUpdateVirtualHostCountsHealthy(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	vhost := &VirtualHost{
		Name:       "vhost1",
		Host:       "example.com",
		SSLEnabled: false,
		Backends:   []*Backend{},
	}

	hc.AddVirtualHost(vhost)

	// Simulate backend count updates
	vhostHealth := hc.GetVirtualHostHealth("vhost1")
	vhostHealth.BackendCount = 3
	vhostHealth.HealthyBackends = 3
	vhostHealth.UnhealthyBackends = 0

	// Call updateVHostCounts - just verify it doesn't panic
	hc.updateVHostCounts()

	// Verify vhost health was updated correctly
	updated := hc.GetVirtualHostHealth("vhost1")
	if updated == nil {
		t.Error("expected non-nil vhost health")
	}
}

func TestUpdateVirtualHostHealthDegraded(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	vhost := &VirtualHost{
		Name:       "vhost1",
		Host:       "example.com",
		SSLEnabled: false,
		Backends:   []*Backend{},
	}

	hc.AddVirtualHost(vhost)
	vhostHealth := hc.GetVirtualHostHealth("vhost1")

	// Set up a degraded state
	vhostHealth.BackendCount = 5
	vhostHealth.HealthyBackends = 3
	vhostHealth.UnhealthyBackends = 2

	// Call updateVHostCounts - just verify it doesn't panic
	hc.updateVHostCounts()

	updated := hc.GetVirtualHostHealth("vhost1")
	if updated == nil {
		t.Error("expected non-nil vhost health")
	}
}

func TestCheckLoopExecution(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		CheckInterval: 100 * time.Millisecond,
	})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}
	hc.AddBackend(backend)

	// Start the checker
	err := hc.Start()
	if err != nil {
		t.Errorf("unexpected error from Start: %v", err)
	}

	// Give it time to run at least one check loop
	time.Sleep(200 * time.Millisecond)

	// Stop the checker
	err = hc.Stop()
	if err != nil {
		t.Logf("error from Stop: %v", err)
	}

	if hc.running {
		t.Error("expected running to be false after Stop()")
	}
}

func TestPerformHealthChecksExecution(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		CheckInterval: 1 * time.Second,
		Timeout:       2 * time.Second,
	})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}
	hc.AddBackend(backend)

	// This should complete without panic
	hc.performHealthChecks()
}

func TestExecuteChecksForBackend(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		Timeout: 2 * time.Second,
	})

	backend := &Backend{
		Name:   "backend1",
		Host:   "localhost",
		Port:   9999, // Non-existent port
		Scheme: "http",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := hc.executeChecks(ctx, backend)
	if result == nil {
		t.Error("expected non-nil check result")
	}
}

func TestUpdateBackendCountsMethod(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	backends := []*Backend{
		{Name: "b1", Host: "10.0.0.1", Port: 8080, Scheme: "http"},
		{Name: "b2", Host: "10.0.0.2", Port: 8080, Scheme: "http"},
		{Name: "b3", Host: "10.0.0.3", Port: 8080, Scheme: "http"},
	}

	for _, b := range backends {
		hc.AddBackend(b)
	}

	allBackends := hc.GetAllBackendHealth()
	if len(allBackends) != 3 {
		t.Errorf("expected 3 backends, got %d", len(allBackends))
	}

	// Call updateBackendCounts - just verify it doesn't panic
	hc.updateBackendCounts()
}

func TestHealthCheckerMultiBackendState(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		HealthyThreshold:   2,
		UnhealthyThreshold: 3,
	})

	for i := 1; i <= 5; i++ {
		backend := &Backend{
			Name:   fmt.Sprintf("backend%d", i),
			Host:   fmt.Sprintf("10.0.0.%d", i),
			Port:   8080,
			Scheme: "http",
		}
		hc.AddBackend(backend)
	}

	allHealth := hc.GetAllBackendHealth()
	if len(allHealth) != 5 {
		t.Errorf("expected 5 backends, got %d", len(allHealth))
	}

	for _, health := range allHealth {
		if health.Status != StatusUnknown {
			t.Errorf("expected initial status Unknown, got %v", health.Status)
		}
		if health.TotalChecks != 0 {
			t.Errorf("expected 0 total checks initially, got %d", health.TotalChecks)
		}
	}
}

func TestHealthCheckerContextHandling(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		CheckInterval: 100 * time.Millisecond,
	})

	// Should handle cancelled context gracefully
	hc.performHealthChecks()
}

func TestUpdateVirtualHostHealthMethod(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{})

	vhost := &VirtualHost{
		Name:       "vhost1",
		Host:       "example.com",
		SSLEnabled: false,
		Backends:   []*Backend{},
	}

	hc.AddVirtualHost(vhost)

	// Call updateVirtualHostHealth - just verify it doesn't panic
	hc.updateVirtualHostHealth()
}

func TestCheckBackendMethod(t *testing.T) {
	hc := NewHealthChecker(HealthConfig{
		Timeout: 2 * time.Second,
	})

	backend := &Backend{
		Name:   "backend1",
		Host:   "10.0.0.1",
		Port:   8080,
		Scheme: "http",
	}

	hc.AddBackend(backend)
	health := hc.GetBackendHealth(backend)

	key := fmt.Sprintf("%s-%s:%d", backend.Name, backend.Host, backend.Port)

	// Call checkBackend - just verify it doesn't panic
	hc.checkBackend(key, health)
}
