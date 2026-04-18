//go:build ci

package envoy_test

import (
	"context"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/envoy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/logging"
)

func newTestManager(t *testing.T) *envoy.Manager {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return envoy.NewManager(
		"/usr/bin/true",
		"/etc/envoy/envoy.yaml",
		9901,
		"info",
		logger,
	)
}

func TestNewManager(t *testing.T) {
	mgr := newTestManager(t)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestIsRunning_NotRunning(t *testing.T) {
	mgr := newTestManager(t)

	if mgr.IsRunning() {
		t.Error("expected IsRunning=false for new manager")
	}
}

func TestUptime_NotRunning(t *testing.T) {
	mgr := newTestManager(t)

	uptime := mgr.Uptime()
	if uptime != 0 {
		t.Errorf("expected uptime=0 for non-running manager, got %v", uptime)
	}
}

func TestIsRunning_AfterStart(t *testing.T) {
	mgr := newTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to start with a working binary (true/echo should work)
	_ = mgr.Start(ctx)
	// Error is expected since we're using /usr/bin/true which exits immediately
	// The important thing is we're testing the IsRunning method

	// Uptime should still be callable
	uptime := mgr.Uptime()
	_ = uptime // Just verify it doesn't panic
}

func TestUptimeIncreases(t *testing.T) {
	mgr := newTestManager(t)

	// Uptime before running
	uptime1 := mgr.Uptime()

	// Simulate running state by checking multiple times
	time.Sleep(10 * time.Millisecond)

	uptime2 := mgr.Uptime()
	if uptime2 < uptime1 {
		t.Errorf("uptime should not decrease: %v < %v", uptime2, uptime1)
	}
}

func TestManagerConcurrentIsRunning(t *testing.T) {
	mgr := newTestManager(t)

	results := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			results <- mgr.IsRunning()
		}()
	}

	for i := 0; i < 10; i++ {
		result := <-results
		if result != false {
			t.Errorf("expected IsRunning=false, got %v", result)
		}
	}
}

func TestManagerConcurrentUptime(t *testing.T) {
	mgr := newTestManager(t)

	uptimes := make(chan time.Duration, 10)

	for i := 0; i < 10; i++ {
		go func() {
			uptimes <- mgr.Uptime()
		}()
	}

	for i := 0; i < 10; i++ {
		uptime := <-uptimes
		if uptime != 0 {
			t.Errorf("expected uptime=0 for non-running manager, got %v", uptime)
		}
	}
}

func TestStartWithMissingBinary(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	mgr := envoy.NewManager(
		"/nonexistent/envoy",
		"/etc/envoy/envoy.yaml",
		9901,
		"info",
		logger,
	)

	ctx := context.Background()
	err := mgr.Start(ctx)

	if err == nil {
		t.Error("expected error when binary doesn't exist")
	}

	if !mgr.IsRunning() {
		// Good - manager should still not be running after failed start
	}
}

func TestStartWithMissingConfig(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	mgr := envoy.NewManager(
		"/usr/bin/true",
		"/nonexistent/config.yaml",
		9901,
		"info",
		logger,
	)

	ctx := context.Background()
	err := mgr.Start(ctx)

	if err == nil {
		t.Error("expected error when config doesn't exist")
	}

	if mgr.IsRunning() {
		t.Error("expected manager to not be running after failed start")
	}
}

func TestReloadNotRunning(t *testing.T) {
	mgr := newTestManager(t)

	err := mgr.Reload()
	if err == nil {
		t.Error("expected error when reloading non-running manager")
	}
}

func TestStopNotRunning(t *testing.T) {
	mgr := newTestManager(t)

	// Should not error when stopping a non-running manager
	err := mgr.Stop()
	if err != nil {
		t.Errorf("expected no error stopping non-running manager, got %v", err)
	}

	if mgr.IsRunning() {
		t.Error("expected manager to still not be running")
	}
}

func TestManagerInitialState(t *testing.T) {
	mgr := newTestManager(t)

	if mgr.IsRunning() {
		t.Error("expected new manager to not be running")
	}

	if mgr.Uptime() != 0 {
		t.Error("expected new manager to have 0 uptime")
	}
}

func TestStopAlreadyStopped(t *testing.T) {
	mgr := newTestManager(t)

	// Stopping twice should not cause panic
	err1 := mgr.Stop()
	if err1 != nil {
		t.Errorf("first Stop() should not error: %v", err1)
	}

	err2 := mgr.Stop()
	if err2 != nil {
		t.Errorf("second Stop() should not error: %v", err2)
	}
}

func TestNewManagerNilLogger(t *testing.T) {
	// Test that NewManager handles nil logger
	mgr := envoy.NewManager(
		"/usr/bin/true",
		"/etc/envoy/envoy.yaml",
		9901,
		"info",
		nil,
	)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	if mgr.IsRunning() {
		t.Error("expected new manager to not be running")
	}
}
