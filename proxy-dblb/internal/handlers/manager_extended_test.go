//go:build ci

package handlers

import (
	"context"
	"testing"

	"marchproxy-dblb/internal/config"
	"marchproxy-dblb/internal/logging"
	"marchproxy-dblb/internal/pool"
	"marchproxy-dblb/internal/security"
)

func TestManagerGetHandler(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:               50052,
		MaxConnectionsPerRoute: 100,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	_ = manager.RegisterHandler("postgresql", 5432)

	handler, ok := manager.GetHandler("postgresql")
	if !ok {
		t.Error("GetHandler should return true for registered handler")
	}
	if handler == nil {
		t.Error("Handler is nil")
	}

	_, ok = manager.GetHandler("nonexistent")
	if ok {
		t.Error("GetHandler should return false for unregistered handler")
	}
}

func TestManagerStartAllError(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:               50052,
		MaxConnectionsPerRoute: 100,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	// Try to start with no handlers (should succeed)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := manager.StartAll(ctx)
	if err != nil {
		t.Errorf("StartAll with no handlers should not error: %v", err)
	}
}

func TestManagerDuplicateRegistration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:               50052,
		MaxConnectionsPerRoute: 100,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	err1 := manager.RegisterHandler("postgresql", 5432)
	if err1 != nil {
		t.Fatalf("First registration failed: %v", err1)
	}

	err2 := manager.RegisterHandler("postgresql", 5433)
	if err2 == nil {
		t.Error("Expected error for duplicate protocol registration")
	}
}

func TestManagerStopAllEmpty(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:               50052,
		MaxConnectionsPerRoute: 100,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	err := manager.StopAll()
	if err != nil {
		t.Errorf("StopAll with no handlers should not error: %v", err)
	}
}

func TestManagerGetStatsMultipleHandlers(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:               50052,
		MaxConnectionsPerRoute: 100,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	_ = manager.RegisterHandler("postgresql", 5432)
	_ = manager.RegisterHandler("mysql", 3306)
	_ = manager.RegisterHandler("mongodb", 27017)

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 handler stats, got %d", len(stats))
	}

	if _, ok := stats["postgresql"]; !ok {
		t.Error("postgresql stats not found")
	}
	if _, ok := stats["mysql"]; !ok {
		t.Error("mysql stats not found")
	}
	if _, ok := stats["mongodb"]; !ok {
		t.Error("mongodb stats not found")
	}
}

func TestManagerConcurrentRegistration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:               50052,
		MaxConnectionsPerRoute: 100,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	done := make(chan bool, 3)

	protocols := []string{"postgresql", "mysql", "mongodb"}
	ports := []int{5432, 3306, 27017}

	for i, proto := range protocols {
		go func(protocol string, port int) {
			err := manager.RegisterHandler(protocol, port)
			if err != nil {
				t.Logf("Registration failed: %v", err)
				done <- false
				return
			}
			done <- true
		}(proto, ports[i])
	}

	for i := 0; i < 3; i++ {
		success := <-done
		if !success {
			t.Error("Concurrent registration failed")
		}
	}

	if len(manager.handlers) != 3 {
		t.Errorf("Expected 3 handlers after concurrent registration, got %d", len(manager.handlers))
	}
}
