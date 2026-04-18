package pool_test

import (
	"net"
	"testing"

	"marchproxy-dblb/internal/logging"
	"marchproxy-dblb/internal/pool"
)

func TestNewPool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)
	if p == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestPoolCreateProtocol(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	err := p.CreatePool("mysql", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPoolCreateProtocolSmallMax(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	// Small max connections is allowed
	err := p.CreatePool("mysql", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPoolCreateProtocolDuplicate(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	p.CreatePool("mysql", 50)
	err := p.CreatePool("mysql", 60)
	if err == nil {
		t.Fatal("expected error when adding duplicate protocol")
	}
}

func TestPoolGetNonexistentProtocol(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	_, err := p.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent protocol")
	}
}

func TestPoolStatsInitial(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	stats := p.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}

func TestPoolClose(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	p.CreatePool("mysql", 50)
	p.Close()

	// After closing, getting connections should fail
	_, err := p.Get("mysql")
	if err == nil {
		t.Error("expected error after pool close")
	}
}

func TestPoolMultipleProtocols(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := pool.NewPool(100, logger)

	protocols := []string{"mysql", "postgresql", "mongodb"}
	for _, proto := range protocols {
		err := p.CreatePool(proto, 50)
		if err != nil {
			t.Fatalf("unexpected error adding protocol %s: %v", proto, err)
		}
	}

	stats := p.GetStats()
	if len(stats) != len(protocols) {
		t.Errorf("expected %d protocol stats, got %d", len(protocols), len(stats))
	}
}

// Mock connection for testing
type MockConn struct {
	closed bool
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *MockConn) Close() error {
	m.closed = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return nil
}

func (m *MockConn) RemoteAddr() net.Addr {
	return nil
}

func (m *MockConn) SetDeadline(t interface{}) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t interface{}) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t interface{}) error {
	return nil
}
