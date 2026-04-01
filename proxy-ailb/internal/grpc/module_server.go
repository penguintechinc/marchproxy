// Package grpcserver implements the ModuleService gRPC interface for AILB.
package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

// ModuleServer implements pb.ModuleServiceServer for the AILB module.
type ModuleServer struct {
	pb.UnimplementedModuleServiceServer
	registry  *providers.Registry
	router    *router.Router
	startTime time.Time
	moduleID  string
	version   string
}

// NewModuleServer creates a new ModuleServer.
func NewModuleServer(reg *providers.Registry, rtr *router.Router, version string) *ModuleServer {
	moduleID := os.Getenv("MODULE_ID")
	if moduleID == "" {
		moduleID = "ailb-1"
	}
	return &ModuleServer{
		registry:  reg,
		router:    rtr,
		startTime: time.Now(),
		moduleID:  moduleID,
		version:   version,
	}
}

// GetStatus returns module health and status information.
func (s *ModuleServer) GetStatus(_ context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	now := timestamppb.Now()
	return &pb.GetStatusResponse{
		Instance: &pb.ModuleInstance{
			InstanceId:   s.moduleID,
			ModuleType:   pb.ModuleType_MODULE_TYPE_AILB,
			HealthStatus: pb.HealthStatus_HEALTH_STATUS_HEALTHY,
			Version:      s.version,
			StartedAt:    timestamppb.New(s.startTime),
			Metadata: map[string]string{
				"num_providers": fmt.Sprintf("%d", len(s.registry.Names())),
			},
		},
		Status:      pb.HealthStatus_HEALTH_STATUS_HEALTHY,
		Message:     "AILB is healthy",
		LastUpdated: now,
		Details: map[string]string{
			"module_id":      s.moduleID,
			"module_type":    "AILB",
			"version":        s.version,
			"num_providers":  fmt.Sprintf("%d", len(s.registry.Names())),
			"uptime_seconds": fmt.Sprintf("%d", int(time.Since(s.startTime).Seconds())),
		},
	}, nil
}

// HealthCheck performs a health check on this module instance.
func (s *ModuleServer) HealthCheck(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	checks := map[string]string{
		"providers": fmt.Sprintf("%d registered", len(s.registry.Names())),
		"uptime":    fmt.Sprintf("%ds", int(time.Since(s.startTime).Seconds())),
	}

	status := pb.HealthStatus_HEALTH_STATUS_HEALTHY
	msg := "all checks passed"
	if len(s.registry.Names()) == 0 {
		status = pb.HealthStatus_HEALTH_STATUS_DEGRADED
		msg = "no providers registered"
	}

	return &pb.HealthCheckResponse{
		Status:    status,
		Message:   msg,
		Checks:    checks,
		CheckedAt: timestamppb.Now(),
	}, nil
}

// GetStats returns detailed statistics for this module instance.
func (s *ModuleServer) GetStats(_ context.Context, _ *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	routerStats := s.router.Stats()

	customStats := make(map[string]string)
	for provider, stats := range routerStats {
		for k, v := range stats {
			customStats[fmt.Sprintf("%s.%s", provider, k)] = fmt.Sprintf("%v", v)
		}
	}

	return &pb.GetStatsResponse{
		Stats: &pb.ModuleStats{
			InstanceId:  s.moduleID,
			CustomStats: customStats,
			Since:       timestamppb.New(s.startTime),
		},
	}, nil
}

// GetConfig returns the current configuration.
func (s *ModuleServer) GetConfig(_ context.Context, _ *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	providerNames := s.registry.Names()
	configData := fmt.Sprintf(`{"module_id":%q,"version":%q,"module_type":"AILB","providers":%d}`,
		s.moduleID, s.version, len(providerNames))

	configs := []*pb.ConfigUpdate{
		{
			ConfigId:   "ailb-main",
			ConfigType: "json",
			ConfigData: []byte(configData),
			Metadata: map[string]string{
				"module_id": s.moduleID,
				"version":   s.version,
			},
			UpdatedAt: timestamppb.New(s.startTime),
		},
	}

	return &pb.GetConfigResponse{
		Configs:     configs,
		LastUpdated: timestamppb.New(s.startTime),
	}, nil
}

// Start starts the gRPC server on the given port. It blocks until the
// context is cancelled.
func Start(ctx context.Context, port int, reg *providers.Registry, rtr *router.Router, version string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", port, err)
	}

	srv := grpc.NewServer()

	// Register ModuleService.
	moduleSrv := NewModuleServer(reg, rtr, version)
	pb.RegisterModuleServiceServer(srv, moduleSrv)

	// Register standard gRPC health service.
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	slog.Info("gRPC server starting", "port", port)

	// Shutdown on context cancellation.
	go func() {
		<-ctx.Done()
		slog.Info("gRPC server shutting down")
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}
