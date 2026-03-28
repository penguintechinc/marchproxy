package health_test

import (
	"testing"
	"time"

	"marchproxy-egress/internal/health"
)

func TestHealthStatusConstants(t *testing.T) {
	statuses := []health.HealthStatus{
		health.StatusHealthy,
		health.StatusUnhealthy,
		health.StatusDegraded,
		health.StatusUnknown,
	}

	seen := make(map[health.HealthStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Errorf("health status constant must not be empty string")
		}
		if seen[s] {
			t.Errorf("duplicate health status constant value: %q", s)
		}
		seen[s] = true
	}

	if health.StatusHealthy == health.StatusUnhealthy {
		t.Error("StatusHealthy and StatusUnhealthy must be distinct")
	}
	if health.StatusDegraded == health.StatusUnknown {
		t.Error("StatusDegraded and StatusUnknown must be distinct")
	}
}

func TestHealthStatusValues(t *testing.T) {
	if string(health.StatusHealthy) != "healthy" {
		t.Errorf("expected StatusHealthy = %q, got %q", "healthy", health.StatusHealthy)
	}
	if string(health.StatusUnhealthy) != "unhealthy" {
		t.Errorf("expected StatusUnhealthy = %q, got %q", "unhealthy", health.StatusUnhealthy)
	}
	if string(health.StatusDegraded) != "degraded" {
		t.Errorf("expected StatusDegraded = %q, got %q", "degraded", health.StatusDegraded)
	}
	if string(health.StatusUnknown) != "unknown" {
		t.Errorf("expected StatusUnknown = %q, got %q", "unknown", health.StatusUnknown)
	}
}

func TestNewHealthCheckerNotNil(t *testing.T) {
	cfg := health.HealthConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
	}
	checker := health.NewHealthChecker(cfg)
	if checker == nil {
		t.Fatal("expected non-nil checker from NewHealthChecker")
	}
}

func TestNewHealthCheckerDefaults(t *testing.T) {
	// Empty config — constructor applies defaults
	cfg := health.HealthConfig{}
	checker := health.NewHealthChecker(cfg)
	if checker == nil {
		t.Fatal("expected non-nil checker with zero config")
	}
}

func TestNewHealthCheckerWithFullConfig(t *testing.T) {
	cfg := health.HealthConfig{
		CheckInterval:       15 * time.Second,
		Timeout:             3 * time.Second,
		HealthyThreshold:    2,
		UnhealthyThreshold:  3,
		EnabledChecks:       []string{"http", "tcp"},
		HTTPEndpoint:        "/health",
		HTTPPort:            8080,
		EnableProbes:        false,
		NotificationURL:     "",
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
	}
	checker := health.NewHealthChecker(cfg)
	if checker == nil {
		t.Fatal("expected non-nil checker with full config")
	}
}

func TestGetAllServiceHealthEmpty(t *testing.T) {
	cfg := health.HealthConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
	}
	checker := health.NewHealthChecker(cfg)

	all := checker.GetAllServiceHealth()
	if all == nil {
		t.Fatal("GetAllServiceHealth should return non-nil map")
	}
	if len(all) != 0 {
		t.Errorf("expected empty map for new checker, got %d entries", len(all))
	}
}

func TestGetSystemHealthNoServices(t *testing.T) {
	cfg := health.HealthConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
	}
	checker := health.NewHealthChecker(cfg)

	sys := checker.GetSystemHealth()
	if sys == nil {
		t.Fatal("GetSystemHealth should return non-nil result")
	}
	// With no services, status should be unknown
	if sys.Status != health.StatusUnknown {
		t.Errorf("expected StatusUnknown for empty checker, got %q", sys.Status)
	}
}

func TestExecuteProbeNotFound(t *testing.T) {
	cfg := health.HealthConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
	}
	checker := health.NewHealthChecker(cfg)

	result := checker.ExecuteProbe("nonexistent")
	if result == nil {
		t.Fatal("ExecuteProbe should return non-nil result even for missing probe")
	}
	if result.Status != health.StatusUnknown {
		t.Errorf("expected StatusUnknown for missing probe, got %q", result.Status)
	}
	if result.Error == nil {
		t.Error("expected non-nil error for missing probe")
	}
}

func TestHealthCheckerStartStop(t *testing.T) {
	cfg := health.HealthConfig{
		CheckInterval: 100 * time.Millisecond,
		Timeout:       50 * time.Millisecond,
	}
	checker := health.NewHealthChecker(cfg)

	if err := checker.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Starting again should fail
	if err := checker.Start(); err == nil {
		t.Error("expected error on double Start()")
	}

	if err := checker.Stop(); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}

func TestProbeTypeConstants(t *testing.T) {
	types := []health.ProbeType{
		health.ProbeTypeHTTP,
		health.ProbeTypeTCP,
		health.ProbeTypeExec,
	}
	seen := make(map[health.ProbeType]bool)
	for _, pt := range types {
		if pt == "" {
			t.Error("ProbeType constant must not be empty")
		}
		if seen[pt] {
			t.Errorf("duplicate ProbeType: %q", pt)
		}
		seen[pt] = true
	}
}

func TestNewHTTPHealthCheckNotNil(t *testing.T) {
	expected := health.HTTPExpected{
		StatusCodes:     []int{200, 204},
		MaxResponseTime: 5 * time.Second,
	}
	check := health.NewHTTPHealthCheck("http-check", expected)
	if check == nil {
		t.Fatal("NewHTTPHealthCheck should return non-nil")
	}
}

func TestHTTPHealthCheckName(t *testing.T) {
	check := health.NewHTTPHealthCheck("my-check", health.HTTPExpected{})
	if check.Name() != "my-check" {
		t.Errorf("expected Name() = %q, got %q", "my-check", check.Name())
	}
}

func TestHTTPHealthCheckEnabled(t *testing.T) {
	check := health.NewHTTPHealthCheck("check", health.HTTPExpected{})
	if !check.Enabled() {
		t.Error("expected new HTTPHealthCheck to be enabled")
	}
}

func TestNewTCPHealthCheckNotNil(t *testing.T) {
	check := health.NewTCPHealthCheck("tcp-check")
	if check == nil {
		t.Fatal("NewTCPHealthCheck should return non-nil")
	}
	if check.Name() != "tcp-check" {
		t.Errorf("expected Name() = %q, got %q", "tcp-check", check.Name())
	}
	if !check.Enabled() {
		t.Error("expected new TCPHealthCheck to be enabled")
	}
}

func TestNewMemoryHealthCheckNotNil(t *testing.T) {
	check := health.NewMemoryHealthCheck("mem-check", 1024*1024*1024)
	if check == nil {
		t.Fatal("NewMemoryHealthCheck should return non-nil")
	}
	if check.Name() != "mem-check" {
		t.Errorf("expected Name() = %q, got %q", "mem-check", check.Name())
	}
	if !check.Enabled() {
		t.Error("expected new MemoryHealthCheck to be enabled")
	}
}

func TestNewDiskHealthCheckNotNil(t *testing.T) {
	check := health.NewDiskHealthCheck("disk-check", "/tmp", 1024*1024*1024)
	if check == nil {
		t.Fatal("NewDiskHealthCheck should return non-nil")
	}
	if check.Name() != "disk-check" {
		t.Errorf("expected Name() = %q, got %q", "disk-check", check.Name())
	}
	if !check.Enabled() {
		t.Error("expected new DiskHealthCheck to be enabled")
	}
}

func TestNewProcessHealthCheckNotNil(t *testing.T) {
	check := health.NewProcessHealthCheck("proc-check")
	if check == nil {
		t.Fatal("NewProcessHealthCheck should return non-nil")
	}
	if check.Name() != "proc-check" {
		t.Errorf("expected Name() = %q, got %q", "proc-check", check.Name())
	}
	if !check.Enabled() {
		t.Error("expected new ProcessHealthCheck to be enabled")
	}
}

func TestNewReadinessProbe(t *testing.T) {
	cfg := health.ProbeConfig{
		Type:    health.ProbeTypeHTTP,
		Path:    "/ready",
		Port:    8080,
		Timeout: 5 * time.Second,
	}
	probe := health.NewReadinessProbe("readiness", cfg)
	if probe == nil {
		t.Fatal("NewReadinessProbe should return non-nil")
	}
	if probe.Name() != "readiness" {
		t.Errorf("expected Name() = %q, got %q", "readiness", probe.Name())
	}
}

func TestNewLivenessProbe(t *testing.T) {
	cfg := health.ProbeConfig{
		Type:    health.ProbeTypeTCP,
		Port:    8080,
		Timeout: 3 * time.Second,
	}
	probe := health.NewLivenessProbe("liveness", cfg)
	if probe == nil {
		t.Fatal("NewLivenessProbe should return non-nil")
	}
	if probe.Name() != "liveness" {
		t.Errorf("expected Name() = %q, got %q", "liveness", probe.Name())
	}
}

func TestCheckResultFields(t *testing.T) {
	result := &health.CheckResult{
		Status:   health.StatusHealthy,
		Message:  "all good",
		Metadata: map[string]interface{}{"key": "value"},
	}
	if result.Status != health.StatusHealthy {
		t.Errorf("unexpected Status: %q", result.Status)
	}
	if result.Message != "all good" {
		t.Errorf("unexpected Message: %q", result.Message)
	}
}
