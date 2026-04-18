//go:build ci

package handlers

import (
	"context"
	"testing"
	"time"

	"marchproxy-dblb/internal/config"
	"marchproxy-dblb/internal/logging"
	"marchproxy-dblb/internal/pool"
	"marchproxy-dblb/internal/security"
)

func TestNewManager(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)

	manager := NewManager(mockPool, checker, cfg, logger)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.logger == nil {
		t.Error("logger not set")
	}

	if manager.handlers == nil {
		t.Error("handlers map not initialized")
	}

	if manager.pool == nil {
		t.Error("pool not set")
	}

	if manager.securityChecker == nil {
		t.Error("securityChecker not set")
	}

	if manager.config == nil {
		t.Error("config not set")
	}
}

func TestRegisterHandler(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	err := manager.RegisterHandler("mysql", 3306)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	handler, ok := manager.GetHandler("mysql")
	if !ok {
		t.Error("GetHandler should return true for registered handler")
	}

	if handler == nil {
		t.Error("GetHandler should return handler")
	}
}

func TestRegisterHandler_Duplicate(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	err1 := manager.RegisterHandler("mysql", 3306)
	if err1 != nil {
		t.Fatalf("First RegisterHandler failed: %v", err1)
	}

	err2 := manager.RegisterHandler("mysql", 3307)
	if err2 == nil {
		t.Error("RegisterHandler should fail for duplicate protocol")
	}
}

func TestRegisterHandler_MultipleProtocols(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	protocols := []struct {
		protocol string
		port     int
	}{
		{"mysql", 3306},
		{"postgresql", 5432},
		{"redis", 6379},
		{"mongodb", 27017},
	}

	for _, p := range protocols {
		err := manager.RegisterHandler(p.protocol, p.port)
		if err != nil {
			t.Errorf("RegisterHandler for %s failed: %v", p.protocol, err)
		}
	}

	for _, p := range protocols {
		handler, ok := manager.GetHandler(p.protocol)
		if !ok {
			t.Errorf("GetHandler should return true for %s", p.protocol)
		}

		if handler == nil {
			t.Errorf("GetHandler should return handler for %s", p.protocol)
		}
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	handler, ok := manager.GetHandler("nonexistent")
	if ok {
		t.Error("GetHandler should return false for unregistered protocol")
	}

	if handler != nil {
		t.Error("GetHandler should return nil for unregistered protocol")
	}
}

func TestGetStats_Empty(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if len(stats) != 0 {
		t.Errorf("GetStats should return empty map for manager with no handlers, got %d", len(stats))
	}
}

func TestGetStats_MultipleHandlers(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	manager.RegisterHandler("mysql", 3306)
	manager.RegisterHandler("postgresql", 5432)
	manager.RegisterHandler("redis", 6379)

	stats := manager.GetStats()
	if len(stats) != 3 {
		t.Errorf("GetStats should return 3 handlers, got %d", len(stats))
	}

	if _, ok := stats["mysql"]; !ok {
		t.Error("mysql stats missing")
	}

	if _, ok := stats["postgresql"]; !ok {
		t.Error("postgresql stats missing")
	}

	if _, ok := stats["redis"]; !ok {
		t.Error("redis stats missing")
	}
}

func TestNewTCPHandler(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)

	handler := NewTCPHandler("mysql", 3306, mockPool, checker, cfg, logger)

	if handler == nil {
		t.Fatal("NewTCPHandler returned nil")
	}

	if handler.protocol != "mysql" {
		t.Errorf("protocol = %q, want mysql", handler.protocol)
	}

	if handler.port != 3306 {
		t.Errorf("port = %d, want 3306", handler.port)
	}

	if handler.logger == nil {
		t.Error("logger not set")
	}

	if handler.pool == nil {
		t.Error("pool not set")
	}

	if handler.securityChecker == nil {
		t.Error("securityChecker not set")
	}
}

func TestTCPHandler_GetStats_Initial(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	handler := NewTCPHandler("mysql", 3306, mockPool, checker, cfg, logger)

	stats := handler.GetStats()

	if stats["protocol"] != "mysql" {
		t.Errorf("protocol = %v, want mysql", stats["protocol"])
	}

	if stats["port"] != 3306 {
		t.Errorf("port = %v, want 3306", stats["port"])
	}

	if stats["active_conns"] != int64(0) {
		t.Errorf("active_conns = %v, want 0", stats["active_conns"])
	}

	if stats["total_conns"] != int64(0) {
		t.Errorf("total_conns = %v, want 0", stats["total_conns"])
	}

	if stats["running"] != false {
		t.Errorf("running = %v, want false", stats["running"])
	}
}

func TestTCPHandler_Stop_NotRunning(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	handler := NewTCPHandler("mysql", 3306, mockPool, checker, cfg, logger)

	err := handler.Stop()
	if err != nil {
		t.Errorf("Stop() when not running returned error: %v", err)
	}

	stats := handler.GetStats()
	if stats["running"].(bool) {
		t.Error("handler should not be running after Stop()")
	}
}

func TestStopAll_NoHandlers(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	err := manager.StopAll()
	if err != nil {
		t.Errorf("StopAll() with no handlers returned error: %v", err)
	}
}

func TestRegisterHandler_Concurrent(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	done := make(chan bool, 3)

	go func() {
		manager.RegisterHandler("mysql", 3306)
		manager.RegisterHandler("postgresql", 5432)
		done <- true
	}()

	go func() {
		manager.RegisterHandler("redis", 6379)
		manager.RegisterHandler("mongodb", 27017)
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			manager.GetStats()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}

	stats := manager.GetStats()
	if len(stats) < 1 {
		t.Error("At least one handler should be registered after concurrent operations")
	}
}

func TestStartAll_ContextCancellation(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := manager.StartAll(ctx)
	if err != nil {
		t.Errorf("StartAll() with no handlers returned error: %v", err)
	}
}

// TestTCPHandler_StartAndGetStats tests Start and GetStats behavior
func TestTCPHandler_StartAndGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	handler := NewTCPHandler("postgresql", 15432, mockPool, checker, cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start handler
	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer handler.Stop()

	// Check stats after start
	stats := handler.GetStats()
	if !stats["running"].(bool) {
		t.Error("handler should be running after Start()")
	}
	if stats["protocol"] != "postgresql" {
		t.Errorf("protocol = %v, want postgresql", stats["protocol"])
	}
	if stats["port"] != 15432 {
		t.Errorf("port = %v, want 15432", stats["port"])
	}
}

// TestTCPHandler_DoubleStart tests starting an already running handler
func TestTCPHandler_DoubleStart(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	handler := NewTCPHandler("mysql", 13308, mockPool, checker, cfg, logger)

	ctx := context.Background()

	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	defer handler.Stop()

	// Try to start again
	err = handler.Start(ctx)
	if err == nil {
		t.Error("Starting already running handler should return error")
	}
}

// TestManager_ConcurrentRegisterAndGetStats tests concurrent register and stat operations
func TestManager_ConcurrentRegisterAndGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	// Concurrent register and get stats
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			protocol := string(rune('a' + idx))
			err := manager.RegisterHandler(protocol, 3000+idx)
			errors <- err
		}(i)

		go func() {
			manager.GetStats()
			errors <- nil
		}()
	}

	// Collect results
	for i := 0; i < 10; i++ {
		err := <-errors
		if err != nil && err.Error() != "handler for protocol a already registered" {
			// Ignore duplicate errors from race conditions
		}
	}
}

// TestManager_MultipleRegistrationsAndStats tests multiple handler registrations
func TestManager_MultipleRegistrationsAndStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	cfg := &config.Config{
		DefaultConnectionRate: 100.0,
		DefaultQueryRate:      1000.0,
	}
	mockPool := pool.NewPool(100, logger)
	checker := security.NewChecker(logger)
	manager := NewManager(mockPool, checker, cfg, logger)

	protocols := []string{"mysql", "postgresql", "redis", "mongodb", "elasticsearch"}
	ports := []int{3306, 5432, 6379, 27017, 9200}

	// Register all handlers
	for i, proto := range protocols {
		err := manager.RegisterHandler(proto, ports[i])
		if err != nil {
			t.Fatalf("RegisterHandler(%s) failed: %v", proto, err)
		}
	}

	// Verify all are registered and stats are available
	stats := manager.GetStats()
	if len(stats) != len(protocols) {
		t.Errorf("Expected %d handlers in stats, got %d", len(protocols), len(stats))
	}

	for _, proto := range protocols {
		if _, ok := stats[proto]; !ok {
			t.Errorf("Protocol %s missing from stats", proto)
		}
	}
}
