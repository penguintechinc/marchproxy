package grpc_test

import (
	"context"
	"testing"
	"time"

	"marchproxy-nlb/internal/grpc"
	"marchproxy-nlb/internal/logging"
)

// MockNLBService is a test implementation of NLBService
type MockNLBService struct {
	registerCalled    bool
	unregisterCalled  bool
	updateHealthCalled bool
	getStatsCalled    bool
}

func (m *MockNLBService) RegisterModule(ctx context.Context, req *grpc.RegisterModuleRequest) (*grpc.RegisterModuleResponse, error) {
	m.registerCalled = true
	return &grpc.RegisterModuleResponse{
		Success:  true,
		Message:  "registered",
		ModuleID: "module-1",
	}, nil
}

func (m *MockNLBService) UnregisterModule(ctx context.Context, req *grpc.UnregisterModuleRequest) (*grpc.UnregisterModuleResponse, error) {
	m.unregisterCalled = true
	return &grpc.UnregisterModuleResponse{
		Success: true,
		Message: "unregistered",
	}, nil
}

func (m *MockNLBService) UpdateHealth(ctx context.Context, req *grpc.HealthUpdateRequest) (*grpc.HealthUpdateResponse, error) {
	m.updateHealthCalled = true
	return &grpc.HealthUpdateResponse{
		Success: true,
		Message: "health updated",
	}, nil
}

func (m *MockNLBService) GetStats(ctx context.Context, req *grpc.StatsRequest) (*grpc.StatsResponse, error) {
	m.getStatsCalled = true
	return &grpc.StatsResponse{
		Timestamp:      time.Now().Unix(),
		TotalModules:   1,
		HealthyModules: 1,
		TotalConns:     100,
		Stats:          map[string]string{"key": "value"},
	}, nil
}

func TestNewServer(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestServerStart(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)

	err := srv.Start()
	if err != nil {
		t.Fatalf("unexpected error starting server: %v", err)
	}
	defer srv.Stop()

	if !srv.IsRunning() {
		t.Error("expected server to be running")
	}
}

func TestServerGetPort(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}
	port := 9999

	srv := grpc.NewServer("127.0.0.1", port, svc, logger)
	if srv.GetPort() != port {
		t.Errorf("expected port %d, got %d", port, srv.GetPort())
	}
}

func TestServerGetAddress(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}
	addr := "127.0.0.1"

	srv := grpc.NewServer(addr, 9999, svc, logger)
	if srv.GetAddress() != addr {
		t.Errorf("expected address %s, got %s", addr, srv.GetAddress())
	}
}

func TestServerIsRunningBeforeStart(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 9999, svc, logger)
	if srv.IsRunning() {
		t.Error("expected server not to be running before start")
	}
}

func TestServerStop(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	srv.Stop()

	time.Sleep(100 * time.Millisecond)
	if srv.IsRunning() {
		t.Error("expected server not to be running after stop")
	}
}

func TestServerRegisterModule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	resp, err := svc.RegisterModule(context.Background(), &grpc.RegisterModuleRequest{
		ModuleName: "test-module",
		Protocol:   "http",
		Address:    "127.0.0.1",
		Port:       8000,
		Version:    "1.0.0",
		MaxConns:   1000,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected successful registration")
	}
	if svc.registerCalled != true {
		t.Error("expected RegisterModule to be called")
	}
}

func TestServerUnregisterModule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	resp, err := svc.UnregisterModule(context.Background(), &grpc.UnregisterModuleRequest{
		ModuleName: "test-module",
		Protocol:   "http",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected successful unregistration")
	}
	if svc.unregisterCalled != true {
		t.Error("expected UnregisterModule to be called")
	}
}

func TestServerUpdateHealth(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	resp, err := svc.UpdateHealth(context.Background(), &grpc.HealthUpdateRequest{
		ModuleName: "test-module",
		Healthy:    true,
		Timestamp:  time.Now().Unix(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected successful health update")
	}
	if svc.updateHealthCalled != true {
		t.Error("expected UpdateHealth to be called")
	}
}

func TestServerGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	svc := &MockNLBService{}

	srv := grpc.NewServer("127.0.0.1", 0, svc, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	resp, err := svc.GetStats(context.Background(), &grpc.StatsRequest{
		IncludeModules: true,
		IncludeMetrics: true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalModules != 1 {
		t.Errorf("expected TotalModules 1, got %d", resp.TotalModules)
	}
	if svc.getStatsCalled != true {
		t.Error("expected GetStats to be called")
	}
}
