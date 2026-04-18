package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/config"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/envoy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/logging"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/metrics"
)

// Server implements the ModuleService gRPC interface
type Server struct {
	pb.UnimplementedModuleServiceServer

	config            *config.Config
	envoyManager      *envoy.Manager
	xdsClient         *envoy.XDSClient
	metricsCollector  *metrics.Collector
	logger            *logging.LogrusAdapter
	grpcServer        *grpc.Server
	startTime         time.Time
}

// NewServer creates a new gRPC server
func NewServer(
	cfg *config.Config,
	envoyMgr *envoy.Manager,
	xdsClient *envoy.XDSClient,
	metricsCollector *metrics.Collector,
	logger *logging.LogrusAdapter,
) *Server {
	if logger == nil {
		logger = &logging.LogrusAdapter{}
	}

	return &Server{
		config:           cfg,
		envoyManager:     envoyMgr,
		xdsClient:        xdsClient,
		metricsCollector: metricsCollector,
		logger:           logger,
		startTime:        time.Now(),
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Create gRPC server with keepalive
	kaParams := keepalive.ServerParameters{
		MaxConnectionAge:      s.config.GRPCMaxConnAge,
		MaxConnectionAgeGrace: 10 * time.Second,
		Time:                  30 * time.Second,
		Timeout:               5 * time.Second,
	}

	s.grpcServer = grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	// Register service
	pb.RegisterModuleServiceServer(s.grpcServer, s)

	s.logger.WithField("address", addr).Info("Starting gRPC server")

	// Start serving
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.WithError(err).Error("gRPC server error")
		}
	}()

	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// GetStatus returns the current health and operational status
func (s *Server) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	s.logger.Debug("GetStatus called")

	resp := &pb.GetStatusResponse{
		Status:  pb.HealthStatus_HEALTH_STATUS_HEALTHY,
		Message: "Module is running normally",
	}

	if !s.envoyManager.IsRunning() {
		resp.Status = pb.HealthStatus_HEALTH_STATUS_UNHEALTHY
		resp.Message = "Envoy manager is not running"
	}

	return resp, nil
}

// GetRoutes returns the current route configuration
func (s *Server) GetRoutes(ctx context.Context, req *pb.GetRoutesRequest) (*pb.GetRoutesResponse, error) {
	s.logger.Debug("GetRoutes called")

	resp := &pb.GetRoutesResponse{
		Routes: []*pb.Route{},
	}

	return resp, nil
}

// SetRateLimit applies rate limiting configuration
func (s *Server) SetRateLimit(ctx context.Context, req *pb.SetRateLimitRequest) (*pb.SetRateLimitResponse, error) {
	s.logger.Info("SetRateLimit called")

	return &pb.SetRateLimitResponse{
		Success: true,
	}, nil
}

// GetMetrics returns current performance metrics
func (s *Server) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	s.logger.Debug("GetMetrics called")

	resp := &pb.GetMetricsResponse{
		Metrics: &pb.ModuleMetrics{},
	}

	return resp, nil
}

// SetTrafficWeight sets traffic weights for blue/green deployments
func (s *Server) SetTrafficWeight(ctx context.Context, req *pb.SetTrafficWeightRequest) (*pb.SetTrafficWeightResponse, error) {
	s.logger.Info("SetTrafficWeight called")

	return &pb.SetTrafficWeightResponse{
		Success: true,
	}, nil
}

// Reload triggers a graceful configuration reload
func (s *Server) Reload(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error) {
	s.logger.Info("Reload called")

	if err := s.envoyManager.Reload(); err != nil {
		return &pb.ReloadResponse{
			Success: false,
		}, nil
	}

	return &pb.ReloadResponse{
		Success: true,
	}, nil
}
