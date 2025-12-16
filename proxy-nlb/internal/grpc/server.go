package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// NLBService defines the gRPC service interface for NLB
type NLBService interface {
	// RegisterModule registers a new module with the NLB
	RegisterModule(ctx context.Context, req *RegisterModuleRequest) (*RegisterModuleResponse, error)

	// UnregisterModule removes a module from the NLB
	UnregisterModule(ctx context.Context, req *UnregisterModuleRequest) (*UnregisterModuleResponse, error)

	// UpdateHealth updates the health status of a module
	UpdateHealth(ctx context.Context, req *HealthUpdateRequest) (*HealthUpdateResponse, error)

	// GetStats returns NLB statistics
	GetStats(ctx context.Context, req *StatsRequest) (*StatsResponse, error)
}

// Request and response types
type RegisterModuleRequest struct {
	ModuleName string
	Protocol   string
	Address    string
	Port       int32
	Version    string
	MaxConns   int32
}

type RegisterModuleResponse struct {
	Success bool
	Message string
	ModuleID string
}

type UnregisterModuleRequest struct {
	ModuleName string
	Protocol   string
}

type UnregisterModuleResponse struct {
	Success bool
	Message string
}

type HealthUpdateRequest struct {
	ModuleName string
	Healthy    bool
	Timestamp  int64
}

type HealthUpdateResponse struct {
	Success bool
	Message string
}

type StatsRequest struct {
	IncludeModules bool
	IncludeMetrics bool
}

type StatsResponse struct {
	Timestamp      int64
	TotalModules   int32
	HealthyModules int32
	TotalConns     int32
	Stats          map[string]string
}

// Server implements the NLB gRPC server
type Server struct {
	address     string
	port        int
	grpcServer  *grpc.Server
	healthServer *health.Server
	service     NLBService
	logger      *logrus.Logger
	listener    net.Listener
	mu          sync.RWMutex
	running     bool
}

// NewServer creates a new NLB gRPC server
func NewServer(address string, port int, service NLBService, logger *logrus.Logger) *Server {
	return &Server{
		address: address,
		port:    port,
		service: service,
		logger:  logger,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	addr := fmt.Sprintf("%s:%d", s.address, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener

	// Configure keepalive parameters
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}

	kaEnforcementPolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaEnforcementPolicy),
		grpc.MaxRecvMsgSize(16 * 1024 * 1024), // 16MB
		grpc.MaxSendMsgSize(16 * 1024 * 1024), // 16MB
	}

	s.grpcServer = grpc.NewServer(opts...)

	// Register health check service
	s.healthServer = health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Set initial health status
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("nlb.NLBService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for debugging
	reflection.Register(s.grpcServer)

	s.running = true
	s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"address": addr,
	}).Info("NLB gRPC server starting")

	// Start serving (blocking)
	if err := s.grpcServer.Serve(listener); err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping NLB gRPC server")

	// Mark as not serving
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus("nlb.NLBService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	// Graceful stop with timeout
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-stopped:
		s.logger.Info("NLB gRPC server stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Graceful stop timeout, forcing stop")
		s.grpcServer.Stop()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.running = false
	return nil
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.port
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf("%s:%d", s.address, s.port)
}

// MockNLBService is a mock implementation for testing
type MockNLBService struct {
	modules map[string]bool
	mu      sync.RWMutex
	logger  *logrus.Logger
}

// NewMockNLBService creates a mock NLB service
func NewMockNLBService(logger *logrus.Logger) *MockNLBService {
	return &MockNLBService{
		modules: make(map[string]bool),
		logger:  logger,
	}
}

// RegisterModule implements module registration
func (m *MockNLBService) RegisterModule(ctx context.Context, req *RegisterModuleRequest) (*RegisterModuleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	moduleID := fmt.Sprintf("%s-%s", req.Protocol, req.ModuleName)
	m.modules[moduleID] = true

	m.logger.WithFields(logrus.Fields{
		"module":   req.ModuleName,
		"protocol": req.Protocol,
		"address":  req.Address,
		"port":     req.Port,
	}).Info("Module registered")

	return &RegisterModuleResponse{
		Success:  true,
		Message:  "Module registered successfully",
		ModuleID: moduleID,
	}, nil
}

// UnregisterModule implements module unregistration
func (m *MockNLBService) UnregisterModule(ctx context.Context, req *UnregisterModuleRequest) (*UnregisterModuleResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	moduleID := fmt.Sprintf("%s-%s", req.Protocol, req.ModuleName)
	delete(m.modules, moduleID)

	m.logger.WithFields(logrus.Fields{
		"module":   req.ModuleName,
		"protocol": req.Protocol,
	}).Info("Module unregistered")

	return &UnregisterModuleResponse{
		Success: true,
		Message: "Module unregistered successfully",
	}, nil
}

// UpdateHealth implements health update
func (m *MockNLBService) UpdateHealth(ctx context.Context, req *HealthUpdateRequest) (*HealthUpdateResponse, error) {
	m.logger.WithFields(logrus.Fields{
		"module":  req.ModuleName,
		"healthy": req.Healthy,
	}).Debug("Health updated")

	return &HealthUpdateResponse{
		Success: true,
		Message: "Health updated",
	}, nil
}

// GetStats implements stats retrieval
func (m *MockNLBService) GetStats(ctx context.Context, req *StatsRequest) (*StatsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &StatsResponse{
		Timestamp:      time.Now().Unix(),
		TotalModules:   int32(len(m.modules)),
		HealthyModules: int32(len(m.modules)), // Simplified
		TotalConns:     0,
		Stats:          make(map[string]string),
	}, nil
}
