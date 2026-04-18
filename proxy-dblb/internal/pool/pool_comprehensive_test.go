//go:build ci

package pool

import (
	"net"
	"testing"
	"time"

	"marchproxy-dblb/internal/logging"
)

type mockConn struct {
	closed bool
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestCreatePool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(10, logger)

	tests := []struct {
		name      string
		protocol  string
		maxConns  int
		expectErr bool
	}{
		{"postgresql", "postgresql", 5, false},
		{"mysql", "mysql", 10, false},
		{"duplicate", "postgresql", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "duplicate" {
				_ = p.CreatePool("postgresql", 5)
			}

			err := p.CreatePool(tt.protocol, tt.maxConns)
			if (err != nil) != tt.expectErr {
				t.Errorf("CreatePool error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestPoolGetNonexistent(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(10, logger)

	_, err := p.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent protocol")
	}
}

func TestPoolGetExhausted(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(1, logger)
	_ = p.CreatePool("test", 1)

	// Fill the pool with one connection
	conn, err := p.Get("test")
	if err != nil {
		t.Fatalf("First Get failed: %v", err)
	}

	// Try to get another when exhausted (not in pool)
	_, err = p.Get("test")
	if err == nil {
		t.Error("Expected error when pool exhausted")
	}

	p.Put("test", conn)
}

func TestPoolPutReturnsToPool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(5, logger)
	_ = p.CreatePool("test", 5)

	// Get a connection
	conn1, err := p.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Put it back
	p.Put("test", conn1)

	// Should be able to get it again
	conn2, err := p.Get("test")
	if err != nil {
		t.Errorf("Second Get failed: %v", err)
	}
	if conn2 == nil {
		t.Error("Retrieved connection is nil")
	}

	p.Put("test", conn2)
}

func TestPoolPutNonexistentProtocol(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(5, logger)

	conn := &mockConn{}
	// Should not panic or error
	p.Put("nonexistent", conn)
	if !conn.closed {
		t.Error("Connection not closed when put to nonexistent pool")
	}
}

func TestPoolPutNilConnection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(5, logger)
	_ = p.CreatePool("test", 5)

	// Should handle nil gracefully
	p.Put("test", nil)
}

func TestGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(10, logger)
	_ = p.CreatePool("postgresql", 5)

	stats := p.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	pgStats, ok := stats["postgresql"]
	if !ok {
		t.Fatal("postgresql stats not found")
	}

	statsMap, ok := pgStats.(map[string]interface{})
	if !ok {
		t.Fatal("Stats is not a map")
	}

	if _, ok := statsMap["active_conns"]; !ok {
		t.Error("active_conns stat missing")
	}
	if _, ok := statsMap["total_conns"]; !ok {
		t.Error("total_conns stat missing")
	}
}

func TestClosePool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(10, logger)
	_ = p.CreatePool("test", 5)

	conn := &mockConn{}
	p.Put("test", conn)

	err := p.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestPoolConcurrentAccess(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(10, logger)
	_ = p.CreatePool("test", 10)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			conn, err := p.Get("test")
			if err != nil {
				t.Logf("Worker %d Get failed: %v", id, err)
				done <- false
				return
			}
			p.Put("test", conn)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		success := <-done
		if !success {
			t.Error("Concurrent access failed")
		}
	}
}
