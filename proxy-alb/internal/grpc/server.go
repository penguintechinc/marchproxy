package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/config"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/envoy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/metrics"
)

// Server implements the ModuleService gRPC interface
type Server struct {
	pb.UnimplementedModuleServiceServer

	config          *config.Config
	envoyManager    *envoy.Manager
	xdsClient       *envoy.XDSClient
	metricsCollector *metrics.Collector
	logger          *logrus.Logger

	grpcServer      *grpc.Server
	startTime       time.Time
}

// NewServer creates a new gRPC server
func NewServer(
	cfg *config.Config,
	envoyMgr *envoy.Manager,
	xdsClient *envoy.XDSClient,
	metricsCollector *metrics.Collector,
	logger *logrus.Logger,
) *Server {
	if logger == nil {
		logger = logrus.New()
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
func (s *Server) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	s.logger.Debug("GetStatus called")

	health := pb.HealthStatus_HEALTHY
	if !s.envoyManager.IsRunning() {
		health = pb.HealthStatus_UNHEALTHY
	}

	resp := &pb.StatusResponse{
		ModuleId:      s.config.ModuleID,
		ModuleType:    s.config.ModuleType,
		Version:       s.config.Version,
		Health:        health,
		UptimeSeconds: int64(s.envoyManager.Uptime().Seconds()),
		EnvoyVersion:  "v1.28.0", // Would be queried from Envoy admin API
		Metadata: map[string]string{
			"xds_server":   s.config.XDSServerAddr,
			"admin_port":   fmt.Sprintf("%d", s.config.EnvoyAdminPort),
			"listen_port":  fmt.Sprintf("%d", s.config.EnvoyListenPort),
		},
	}

	return resp, nil
}

// GetRoutes returns the current route configuration
func (s *Server) GetRoutes(ctx context.Context, req *pb.RoutesRequest) (*pb.RoutesResponse, error) {
	s.logger.WithField("cluster_id", req.ClusterId).Debug("GetRoutes called")

	// Fetch routes from xDS
	routes, err := s.xdsClient.GetRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch routes: %w", err)
	}

	// Convert to protobuf format
	pbRoutes := make([]*pb.RouteConfig, 0, len(routes))
	for _, route := range routes {
		pbRoute := &pb.RouteConfig{
			Name:        route.Name,
			Prefix:      route.Prefix,
			ClusterName: route.ClusterName,
			Hosts:       route.Hosts,
			TimeoutSeconds: int32(route.Timeout),
			Headers:     route.Headers,
			Enabled:     route.Enabled,
		}

		if route.RateLimit != nil {
			pbRoute.RateLimit = &pb.RateLimitConfig{
				RequestsPerSecond: int32(route.RateLimit.RequestsPerSecond),
				BurstSize:         int32(route.RateLimit.BurstSize),
				Enabled:           route.RateLimit.Enabled,
			}
		}

		pbRoutes = append(pbRoutes, pbRoute)
	}

	resp := &pb.RoutesResponse{
		Routes:  pbRoutes,
		Version: time.Now().Unix(),
	}

	return resp, nil
}

// ApplyRateLimit applies rate limiting configuration
func (s *Server) ApplyRateLimit(ctx context.Context, req *pb.RateLimitRequest) (*pb.RateLimitResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"route": req.RouteName,
		"rps":   req.Config.RequestsPerSecond,
	}).Info("ApplyRateLimit called")

	// Convert to internal format
	rateLimit := &envoy.RateLimitConfig{
		RequestsPerSecond: int(req.Config.RequestsPerSecond),
		BurstSize:         int(req.Config.BurstSize),
		Enabled:           req.Config.Enabled,
	}

	// Update via xDS
	if err := s.xdsClient.UpdateRouteRateLimit(req.RouteName, rateLimit); err != nil {
		return &pb.RateLimitResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to apply rate limit: %v", err),
		}, nil
	}

	return &pb.RateLimitResponse{
		Success: true,
		Message: "Rate limit applied successfully",
	}, nil
}

// GetMetrics returns current performance metrics
func (s *Server) GetMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	s.logger.Debug("GetMetrics called")

	// Collect metrics from Envoy
	m, err := s.metricsCollector.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Convert to protobuf format
	resp := &pb.MetricsResponse{
		Timestamp:         m.Timestamp,
		TotalConnections:  m.TotalConnections,
		ActiveConnections: m.ActiveConnections,
		TotalRequests:     m.TotalRequests,
		RequestsPerSecond: m.RequestsPerSecond,
		Latency: &pb.LatencyMetrics{
			P50Ms:  m.Latency.P50Ms,
			P90Ms:  m.Latency.P90Ms,
			P95Ms:  m.Latency.P95Ms,
			P99Ms:  m.Latency.P99Ms,
			AvgMs:  m.Latency.AvgMs,
		},
		StatusCodes: m.StatusCodes,
		Routes:      make(map[string]*pb.RouteMetrics),
	}

	// Convert route metrics
	for name, rm := range m.Routes {
		resp.Routes[name] = &pb.RouteMetrics{
			Requests:      rm.Requests,
			Errors:        rm.Errors,
			AvgLatencyMs:  rm.AvgLatencyMs,
		}
	}

	return resp, nil
}

// SetTrafficWeight sets traffic weights for blue/green deployments
func (s *Server) SetTrafficWeight(ctx context.Context, req *pb.TrafficWeightRequest) (*pb.TrafficWeightResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"route":   req.RouteName,
		"weights": len(req.Weights),
	}).Info("SetTrafficWeight called")

	// Convert weights to map
	weights := make(map[string]int)
	for _, w := range req.Weights {
		weights[w.BackendName] = int(w.Weight)
	}

	// Update via xDS
	if err := s.xdsClient.UpdateTrafficWeights(req.RouteName, weights); err != nil {
		return &pb.TrafficWeightResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to set traffic weights: %v", err),
		}, nil
	}

	return &pb.TrafficWeightResponse{
		Success:        true,
		Message:        "Traffic weights updated successfully",
		AppliedWeights: req.Weights,
	}, nil
}

// Reload triggers a graceful configuration reload
func (s *Server) Reload(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error) {
	s.logger.WithField("force", req.Force).Info("Reload called")

	// Trigger Envoy reload
	if err := s.envoyManager.Reload(); err != nil {
		return &pb.ReloadResponse{
			Success:         false,
			Message:         fmt.Sprintf("Reload failed: %v", err),
			ReloadTimestamp: time.Now().Unix(),
		}, nil
	}

	return &pb.ReloadResponse{
		Success:         true,
		Message:         "Configuration reloaded successfully",
		ReloadTimestamp: time.Now().Unix(),
	}, nil
}
