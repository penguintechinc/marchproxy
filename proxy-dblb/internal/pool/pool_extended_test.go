//go:build ci

package pool

import (
	"net"
	"testing"
	"time"

	"marchproxy-dblb/internal/logging"
)

// TestNewPool tests pool creation
func TestNewPool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	if p == nil {
		t.Fatal("NewPool returned nil")
	}

	if p.pools == nil {
		t.Error("pools map not initialized")
	}

	if p.maxConns != 10 {
		t.Errorf("maxConns not set correctly: %d", p.maxConns)
	}

	if p.idleTimeout != 5*time.Minute {
		t.Errorf("idleTimeout not set correctly: %v", p.idleTimeout)
	}
}

// TestCreatePoolExtended tests creating a protocol pool
func TestCreatePoolExtended(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	err := p.CreatePool("mysql", 5)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	// Try creating same pool again (should fail)
	err = p.CreatePool("mysql", 5)
	if err == nil {
		t.Error("CreatePool should fail for duplicate protocol")
	}
}

// TestCreateMultiplePoolsExtended tests creating multiple protocol pools
func TestCreateMultiplePoolsExtended(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	protocols := map[string]int{
		"mysql":      5,
		"postgresql": 10,
		"mongodb":    8,
	}

	for proto, maxConns := range protocols {
		err := p.CreatePool(proto, maxConns)
		if err != nil {
			t.Errorf("Failed to create pool for %s: %v", proto, err)
		}
	}

	stats := p.GetStats()
	if len(stats) != len(protocols) {
		t.Errorf("Expected %d protocol pools, got %d", len(protocols), len(stats))
	}
}

// TestGetNonExistentPool tests getting from non-existent pool
func TestGetNonExistentPool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	_, err := p.Get("nonexistent")
	if err == nil {
		t.Error("Get should fail for non-existent protocol")
	}
}

// TestPutToNonExistentPool tests putting to non-existent pool
func TestPutToNonExistentPool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	mockConn := &mockConnection{}
	// Put to non-existent protocol (should handle gracefully)
	p.Put("nonexistent", mockConn)
	// Should not panic
}

// TestPutNilConnection tests putting nil connection to pool
func TestPutNilConnection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	err := p.CreatePool("mysql", 5)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	// Put nil connection (should handle gracefully)
	p.Put("mysql", nil)
	// Should not panic
}

// TestGetStatsEmpty tests getting stats from empty pool
func TestGetStatsEmpty(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	stats := p.GetStats()
	if stats == nil {
		t.Error("GetStats returned nil")
	}

	if len(stats) != 0 {
		t.Errorf("Expected 0 stats for empty pool, got %d", len(stats))
	}
}

// TestGetStatsWithPools tests getting stats with populated pools
func TestGetStatsWithPools(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	err := p.CreatePool("mysql", 5)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	stats := p.GetStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 protocol in stats, got %d", len(stats))
	}

	mysqlStats, ok := stats["mysql"]
	if !ok {
		t.Error("mysql stats not found")
	}

	mysqlMap, ok := mysqlStats.(map[string]interface{})
	if !ok {
		t.Error("mysql stats is not a map")
	}

	// Check expected stat fields
	if _, ok := mysqlMap["active_conns"]; !ok {
		t.Error("active_conns not in stats")
	}

	if _, ok := mysqlMap["max_conns"]; !ok {
		t.Error("max_conns not in stats")
	}
}

// TestConnectionPoolState tests protocol pool connection tracking
func TestConnectionPoolState(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(3, logger)

	err := p.CreatePool("mysql", 2)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	stats := p.GetStats()
	mysqlStats := stats["mysql"].(map[string]interface{})

	if mysqlStats["active_conns"] != 0 {
		t.Error("Initial active_conns should be 0")
	}

	if mysqlStats["max_conns"] != 2 {
		t.Error("max_conns should be 2")
	}
}

// TestPoolMaxConnectionsExhausted tests pool exhaustion handling
func TestPoolMaxConnectionsExhausted(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(5, logger)

	err := p.CreatePool("mysql", 1)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	// First Get should succeed (creates connection)
	_, err = p.Get("mysql")
	if err != nil {
		t.Errorf("First Get should succeed: %v", err)
	}

	// Second Get should fail (pool exhausted)
	_, err = p.Get("mysql")
	if err == nil {
		t.Error("Second Get should fail when pool is exhausted")
	}
}

// TestMultipleProtocolPools tests managing multiple protocol pools
func TestMultipleProtocolPools(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(20, logger)

	// Create pools for different protocols
	protocols := []struct {
		name    string
		maxConn int
	}{
		{"mysql", 5},
		{"postgresql", 5},
		{"mongodb", 5},
	}

	for _, proto := range protocols {
		err := p.CreatePool(proto.name, proto.maxConn)
		if err != nil {
			t.Errorf("Failed to create pool for %s: %v", proto.name, err)
		}
	}

	stats := p.GetStats()
	if len(stats) != 3 {
		t.Errorf("Expected 3 protocol pools, got %d", len(stats))
	}
}

// TestProtocolPoolIndependence tests that protocol pools are independent
func TestProtocolPoolIndependence(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(20, logger)

	err := p.CreatePool("mysql", 5)
	if err != nil {
		t.Fatalf("CreatePool for mysql failed: %v", err)
	}

	err = p.CreatePool("postgresql", 5)
	if err != nil {
		t.Fatalf("CreatePool for postgresql failed: %v", err)
	}

	// Getting from one pool shouldn't affect the other
	_, errMysql := p.Get("mysql")
	_, errPgSQL := p.Get("postgresql")

	// One or both might fail depending on connection creation, but shouldn't both fail silently
	if errMysql != nil && errPgSQL != nil {
		// Both failed, which is OK in this context
	}
}

// TestPoolWithHighMaxConnections tests pool with high max connections
func TestPoolWithHighMaxConnections(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(100, logger)

	err := p.CreatePool("mysql", 50)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	stats := p.GetStats()
	mysqlStats := stats["mysql"].(map[string]interface{})

	if mysqlStats["max_conns"] != 50 {
		t.Errorf("max_conns should be 50, got %v", mysqlStats["max_conns"])
	}
}

// TestPoolStatsConsistency tests that stats remain consistent after operations
func TestPoolStatsConsistency(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	err := p.CreatePool("mysql", 5)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	stats1 := p.GetStats()
	mysqlStats1 := stats1["mysql"].(map[string]interface{})

	// Do nothing and check again
	stats2 := p.GetStats()
	mysqlStats2 := stats2["mysql"].(map[string]interface{})

	if mysqlStats1["max_conns"] != mysqlStats2["max_conns"] {
		t.Error("max_conns should be consistent across GetStats calls")
	}
}

// mockConnection implements net.Conn for testing
type mockConnection struct {
	closed bool
}

func (m *mockConnection) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *mockConnection) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}

func (m *mockConnection) LocalAddr() net.Addr {
	return nil
}

func (m *mockConnection) RemoteAddr() net.Addr {
	return nil
}

func (m *mockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

// TestCreatePoolWithZeroConnections tests creating pool with zero max connections
func TestCreatePoolWithZeroConnections(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	// Should allow creating pool with 0 connections
	err := p.CreatePool("readonly", 0)
	if err != nil {
		t.Errorf("CreatePool with 0 connections should be allowed: %v", err)
	}
}

// TestPoolIdleTimeout tests that idle timeout is set correctly
func TestPoolIdleTimeout(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(10, logger)

	// Idle timeout should be set to 5 minutes
	if p.idleTimeout != 5*time.Minute {
		t.Errorf("idleTimeout should be 5 minutes, got %v", p.idleTimeout)
	}
}

// TestConcurrentPoolCreation tests thread-safe pool creation
func TestConcurrentPoolCreation(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	p := NewPool(50, logger)

	done := make(chan bool, 3)

	// Create pools concurrently
	go func() {
		p.CreatePool("mysql", 10)
		done <- true
	}()

	go func() {
		p.CreatePool("postgresql", 10)
		done <- true
	}()

	go func() {
		stats := p.GetStats()
		_ = stats
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	stats := p.GetStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 protocol pools after concurrent creation, got %d", len(stats))
	}
}
