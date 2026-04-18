//go:build ci

package grpc_test

import (
	"context"
	"testing"
	"time"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/config"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/envoy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/grpc"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/logging"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/metrics"
)

func newTestServer(t *testing.T) *grpc.Server {
	cfg := &config.Config{
		ModuleID:        "test-alb",
		GRPCPort:        50051,
		GRPCMaxConnAge:  30 * time.Minute,
		EnvoyBinary:     "/usr/bin/true",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "localhost:18000",
	}

	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	mgr := envoy.NewManager(cfg.EnvoyBinary, cfg.EnvoyConfigPath, 9901, "info", logger)
	xdsClient := envoy.NewXDSClient(cfg.XDSServerAddr, logger)
	metricsCollector := metrics.NewCollector("localhost:9901", logger)

	return grpc.NewServer(cfg, mgr, xdsClient, metricsCollector, logger)
}

func TestNewServer(t *testing.T) {
	srv := newTestServer(t)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestGetStatus_Healthy(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	resp, err := srv.GetStatus(ctx, &pb.GetStatusRequest{})
	if err != nil {
		t.Fatalf("GetStatus() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// When envoy is not running, status should be UNHEALTHY
	if resp.Status != pb.HealthStatus_HEALTH_STATUS_UNHEALTHY {
		t.Errorf("expected UNHEALTHY status when envoy not running, got %v", resp.Status)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestGetRoutes_Empty(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	resp, err := srv.GetRoutes(ctx, &pb.GetRoutesRequest{})
	if err != nil {
		t.Fatalf("GetRoutes() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.Routes == nil {
		t.Error("expected non-nil routes slice")
	}

	if len(resp.Routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(resp.Routes))
	}
}

func TestSetRateLimit(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	req := &pb.SetRateLimitRequest{
		InstanceId: "test-instance",
	}

	resp, err := srv.SetRateLimit(ctx, req)
	if err != nil {
		t.Fatalf("SetRateLimit() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestGetMetrics(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	resp, err := srv.GetMetrics(ctx, &pb.GetMetricsRequest{})
	if err != nil {
		t.Fatalf("GetMetrics() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.Metrics == nil {
		t.Error("expected non-nil metrics")
	}
}

func TestSetTrafficWeight(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	req := &pb.SetTrafficWeightRequest{
		InstanceId: "test-instance",
		Version:    "v1.0",
		Weight:     80,
	}

	resp, err := srv.SetTrafficWeight(ctx, req)
	if err != nil {
		t.Fatalf("SetTrafficWeight() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestReload_NotRunning(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	resp, err := srv.Reload(ctx, &pb.ReloadRequest{})
	if err != nil {
		t.Fatalf("Reload() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// When envoy is not running, reload should fail
	if resp.Success {
		t.Error("expected success=false when envoy not running")
	}
}

func TestGetStatusContextCancellation(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	resp, err := srv.GetStatus(ctx, &pb.GetStatusRequest{})
	// GetStatus should still work since it doesn't use context for blocking operations
	if err != nil {
		t.Fatalf("GetStatus() unexpected error with cancelled context: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response even with cancelled context")
	}
}

func TestMultipleConcurrentRequests(t *testing.T) {
	srv := newTestServer(t)

	done := make(chan error, 3)

	go func() {
		_, err := srv.GetStatus(context.Background(), &pb.GetStatusRequest{})
		done <- err
	}()

	go func() {
		_, err := srv.GetRoutes(context.Background(), &pb.GetRoutesRequest{})
		done <- err
	}()

	go func() {
		_, err := srv.GetMetrics(context.Background(), &pb.GetMetricsRequest{})
		done <- err
	}()

	for i := 0; i < 3; i++ {
		err := <-done
		if err != nil {
			t.Errorf("concurrent request %d failed: %v", i, err)
		}
	}
}
