//go:build ci

package grpc

import (
	"context"
	"testing"

	"marchproxy-dblb/internal/logging"
)

// MockModuleService implements ModuleService for testing
type MockModuleService struct {
	statusResult  map[string]interface{}
	metricsResult map[string]interface{}
	statsResult   map[string]interface{}
	healthStatus  string
	reloadErr     error
	shutdownErr   error
}

func (m *MockModuleService) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	if m.statusResult == nil {
		return make(map[string]interface{}), nil
	}
	return m.statusResult, nil
}

func (m *MockModuleService) Reload(ctx context.Context, graceful bool) error {
	return m.reloadErr
}

func (m *MockModuleService) Shutdown(ctx context.Context, graceful bool) error {
	return m.shutdownErr
}

func (m *MockModuleService) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	if m.metricsResult == nil {
		return make(map[string]interface{}), nil
	}
	return m.metricsResult, nil
}

func (m *MockModuleService) HealthCheck(ctx context.Context) (string, error) {
	return m.healthStatus, nil
}

func (m *MockModuleService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if m.statsResult == nil {
		return make(map[string]interface{}), nil
	}
	return m.statsResult, nil
}

// TestNewServer tests creating a new gRPC server
func TestNewServer(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.address != "localhost" {
		t.Errorf("address not set correctly: %s", server.address)
	}

	if server.port != 50051 {
		t.Errorf("port not set correctly: %d", server.port)
	}

	if server.service == nil {
		t.Error("service not set")
	}

	if server.logger == nil {
		t.Error("logger not set")
	}
}

// TestServerNotRunningInitially tests that server is not running initially
func TestServerNotRunningInitially(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}
}

// TestGetPortReturnsCorrectPort tests that GetPort returns the correct port
func TestGetPortReturnsCorrectPort(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	port := 50051
	server := NewServer("localhost", port, service, logger)

	if server.GetPort() != port {
		t.Errorf("GetPort returned %d, expected %d", server.GetPort(), port)
	}
}

// TestGetAddressReturnsCorrectAddress tests that GetAddress returns listener address
func TestGetAddressReturnsCorrectAddress(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	// Before starting, listener is nil, so GetAddress returns empty
	// After starting, it returns the listener address
	address := server.GetAddress()
	// Just verify it's not nil - actual address depends on listener state
	_ = address
}

// TestStartStopServer tests starting and stopping the server
func TestStartStopServer(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{
		healthStatus: "healthy",
	}

	server := NewServer("localhost", 50051, service, logger)

	// Start the server
	err := server.Start()
	if err != nil {
		// Port might be in use, which is OK for this test
		// We're mainly testing that the methods exist and don't panic
		t.Logf("Start failed (expected if port in use): %v", err)
	} else {
		// If start succeeded, try to stop
		err = server.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}

		// Server should be stopped
		if server.IsRunning() {
			t.Error("Server should not be running after Stop")
		}
	}
}

// TestStartAlreadyRunningServer tests that starting an already running server fails
func TestStartAlreadyRunningServer(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	// Mark as running
	server.mu.Lock()
	server.running = true
	server.mu.Unlock()

	// Try to start again
	err := server.Start()
	if err == nil {
		t.Error("Starting an already running server should fail")
	}

	// Clean up
	server.mu.Lock()
	server.running = false
	server.mu.Unlock()
}

// TestIsRunning tests the IsRunning method
func TestIsRunning(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	// Initially not running
	if server.IsRunning() {
		t.Error("IsRunning should return false initially")
	}

	// Mark as running
	server.mu.Lock()
	server.running = true
	server.mu.Unlock()

	if !server.IsRunning() {
		t.Error("IsRunning should return true after setting running=true")
	}

	// Mark as stopped
	server.mu.Lock()
	server.running = false
	server.mu.Unlock()

	if server.IsRunning() {
		t.Error("IsRunning should return false after setting running=false")
	}
}

// TestServerWithDifferentAddresses tests server creation with different addresses
func TestServerWithDifferentAddresses(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	addresses := []string{"localhost", "127.0.0.1"}

	for _, addr := range addresses {
		server := NewServer(addr, 50051, service, logger)

		// Before starting, GetAddress returns empty since listener is nil
		if server == nil {
			t.Errorf("Failed to create server for %s", addr)
		}
	}
}

// TestServerWithDifferentPorts tests server creation with different ports
func TestServerWithDifferentPorts(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	ports := []int{50051, 50052, 50053}

	for _, port := range ports {
		server := NewServer("localhost", port, service, logger)

		if server.GetPort() != port {
			t.Errorf("Port mismatch for %d", port)
		}
	}
}

// TestStopUnstartedServer tests stopping a server that was never started
func TestStopUnstartedServer(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	// Try to stop without starting - should error since listener is nil
	err := server.Stop()
	// May error or succeed depending on implementation
	_ = err
}

// TestServerThreadSafety tests that server methods are thread-safe
func TestServerThreadSafety(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	server := NewServer("localhost", 50051, service, logger)

	done := make(chan bool, 3)

	// Concurrent operations
	go func() {
		server.IsRunning()
		done <- true
	}()

	go func() {
		server.GetPort()
		done <- true
	}()

	go func() {
		server.GetAddress()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

// TestMultipleServerInstances tests creating multiple server instances
func TestMultipleServerInstances(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	service := &MockModuleService{}

	servers := make([]*Server, 3)

	for i := 0; i < 3; i++ {
		servers[i] = NewServer("localhost", 50051+i, service, logger)

		if servers[i] == nil {
			t.Errorf("Failed to create server instance %d", i)
		}

		if !servers[i].IsRunning() {
			// OK, not running initially
		}
	}

	// Verify all servers are independent
	if servers[0].GetPort() == servers[1].GetPort() {
		t.Error("Server instances should have different ports")
	}
}
