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

func setupTestManager(t *testing.T) *Manager {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := pool.NewPool(10, logger)
	checker := security.NewChecker(logger)
	cfg := &config.Config{
		MaxConnectionsPerRoute: 10,
	}

	return NewManager(p, checker, cfg, logger)
}

// TestNewManagerExtended tests handler manager creation
func TestNewManagerExtended(t *testing.T) {
	m := setupTestManager(t)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.handlers == nil {
		t.Error("handlers map not initialized")
	}

	if m.pool == nil {
		t.Error("pool not set")
	}

	if m.securityChecker == nil {
		t.Error("securityChecker not set")
	}
}

// TestRegisterHandlerComprehensive tests registering a handler
func TestRegisterHandlerComprehensive(t *testing.T) {
	m := setupTestManager(t)

	err := m.RegisterHandler("mysql_test", 3306)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	// Try to register same protocol again (should fail)
	err = m.RegisterHandler("mysql_test", 3306)
	if err == nil {
		t.Error("RegisterHandler should fail for duplicate protocol")
	}
}

// TestRegisterMultipleHandlersExtended tests registering multiple handlers
func TestRegisterMultipleHandlersExtended(t *testing.T) {
	m := setupTestManager(t)

	protocols := map[string]int{
		"mysql":      3306,
		"postgresql": 5432,
		"mongodb":    27017,
	}

	for proto, port := range protocols {
		err := m.RegisterHandler(proto, port)
		if err != nil {
			t.Errorf("Failed to register handler for %s: %v", proto, err)
		}
	}

	stats := m.GetStats()
	if len(stats) != len(protocols) {
		t.Errorf("Expected %d handlers, got %d", len(protocols), len(stats))
	}
}

// TestStartAllHandlersExtended tests starting all registered handlers
func TestStartAllHandlersExtended(t *testing.T) {
	m := setupTestManager(t)

	err := m.RegisterHandler("mysql", 3306)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	ctx := context.Background()
	err = m.StartAll(ctx)
	// May succeed or fail depending on port availability, but shouldn't panic
	_ = err
}

// TestStopAllHandlersExtended tests stopping all handlers
func TestStopAllHandlersExtended(t *testing.T) {
	m := setupTestManager(t)

	err := m.RegisterHandler("mysql", 3306)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	err = m.StopAll()
	// Should not panic, might succeed or fail
	_ = err
}

// TestGetStatsHandlersExtended tests getting handler statistics
func TestGetStatsHandlersExtended(t *testing.T) {
	m := setupTestManager(t)

	err := m.RegisterHandler("mysql", 3306)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	stats := m.GetStats()
	if stats == nil {
		t.Error("GetStats returned nil")
	}

	// Stats should have the registered handler
	if len(stats) == 0 {
		t.Error("GetStats should return stats for registered handlers")
	}
}

// TestGetStatsEmptyManagerExtended tests getting stats from empty manager
func TestGetStatsEmptyManagerExtended(t *testing.T) {
	m := setupTestManager(t)

	stats := m.GetStats()
	if stats == nil {
		t.Error("GetStats returned nil for empty manager")
	}

	if len(stats) != 0 {
		t.Errorf("Expected empty stats, got %d entries", len(stats))
	}
}

// TestConcurrentHandlerRegistration tests thread-safety of handler registration
func TestConcurrentHandlerRegistration(t *testing.T) {
	m := setupTestManager(t)

	done := make(chan bool, 3)

	// Register handlers concurrently
	go func() {
		m.RegisterHandler("mysql", 3306)
		done <- true
	}()

	go func() {
		m.RegisterHandler("postgresql", 5432)
		done <- true
	}()

	go func() {
		stats := m.GetStats()
		_ = stats
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	stats := m.GetStats()
	if stats == nil {
		t.Error("Concurrent operations caused nil stats")
	}
}

// TestRegisterHandlerLocking tests that handler registration is properly locked
func TestRegisterHandlerLocking(t *testing.T) {
	m := setupTestManager(t)

	// Multiple registrations should be safe
	for i := 0; i < 5; i++ {
		proto := "test" + string(rune(i))
		err := m.RegisterHandler(proto, 9000+i)
		if err != nil {
			t.Errorf("RegisterHandler iteration %d failed: %v", i, err)
		}
	}

	stats := m.GetStats()
	if len(stats) != 5 {
		t.Errorf("Expected 5 handlers, got %d", len(stats))
	}
}

// TestStartAllWithMultipleHandlers tests starting multiple handlers
func TestStartAllWithMultipleHandlers(t *testing.T) {
	m := setupTestManager(t)

	protocols := []struct {
		name string
		port int
	}{
		{"mysql", 3306},
		{"postgresql", 5432},
	}

	for _, p := range protocols {
		err := m.RegisterHandler(p.name, p.port)
		if err != nil {
			t.Fatalf("Failed to register handler for %s: %v", p.name, err)
		}
	}

	ctx := context.Background()
	err := m.StartAll(ctx)
	// May succeed or fail, but should not panic
	_ = err
}

// TestStopAllWithMultipleHandlers tests stopping multiple handlers
func TestStopAllWithMultipleHandlers(t *testing.T) {
	m := setupTestManager(t)

	protocols := []struct {
		name string
		port int
	}{
		{"mysql", 3306},
		{"postgresql", 5432},
	}

	for _, p := range protocols {
		err := m.RegisterHandler(p.name, p.port)
		if err != nil {
			t.Fatalf("Failed to register handler for %s: %v", p.name, err)
		}
	}

	err := m.StopAll()
	// May succeed or fail, but should not panic
	_ = err
}
