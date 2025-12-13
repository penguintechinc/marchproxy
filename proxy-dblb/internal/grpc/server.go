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

// ModuleService defines the ModuleService gRPC interface
// This is a simplified interface until we integrate the actual proto files
type ModuleService interface {
	GetStatus(ctx context.Context) (map[string]interface{}, error)
	Reload(ctx context.Context, graceful bool) error
	Shutdown(ctx context.Context, graceful bool) error
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
	HealthCheck(ctx context.Context) (string, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// Server implements the DBLB gRPC server
type Server struct {
	address      string
	port         int
	grpcServer   *grpc.Server
	healthServer *health.Server
	service      ModuleService
	logger       *logrus.Logger
	listener     net.Listener
	mu           sync.RWMutex
	running      bool
}

// NewServer creates a new DBLB gRPC server
func NewServer(address string, port int, service ModuleService, logger *logrus.Logger) *Server {
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
	s.healthServer.SetServingStatus("dblb.ModuleService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for debugging
	reflection.Register(s.grpcServer)

	s.running = true
	s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"address": addr,
	}).Info("DBLB gRPC server starting")

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

	s.logger.Info("Stopping DBLB gRPC server")

	// Mark as not serving
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus("dblb.ModuleService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
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
		s.logger.Info("DBLB gRPC server stopped gracefully")
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
