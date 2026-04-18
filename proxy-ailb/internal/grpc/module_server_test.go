//go:build ci

package grpcserver_test

import (
	"context"
	"os"
	"testing"
	"time"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/grpc"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

// mockProvider implements providers.Provider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Chat(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return &providers.ChatResponse{
		Content:  "test response",
		Provider: m.name,
	}, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{
		{ID: m.name + "-model", Object: "model", Created: 1000, OwnedBy: m.name, Provider: m.name},
	}, nil
}

func (m *mockProvider) SupportsStreaming() bool {
	return false
}

func TestNewModuleServer(t *testing.T) {
	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)

	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	if server == nil {
		t.Error("expected module server to be created")
	}
}

func TestNewModuleServerWithModuleID(t *testing.T) {
	// Set MODULE_ID environment variable
	os.Setenv("MODULE_ID", "test-module-123")
	defer os.Unsetenv("MODULE_ID")

	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)

	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	if server == nil {
		t.Error("expected module server to be created")
	}
}

func TestNewModuleServerDefaultModuleID(t *testing.T) {
	// Ensure MODULE_ID is not set
	os.Unsetenv("MODULE_ID")

	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)

	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	if server == nil {
		t.Error("expected module server to be created with default MODULE_ID")
	}
}

func TestGetStatus(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "provider1"})
	registry.Register(&mockProvider{name: "provider2"})

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.GetStatus(context.Background(), &pb.GetStatusRequest{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response")
	}

	if resp.Instance == nil {
		t.Error("expected instance in response")
	}

	if resp.Instance.ModuleType != pb.ModuleType_MODULE_TYPE_AILB {
		t.Errorf("expected MODULE_TYPE_AILB, got %v", resp.Instance.ModuleType)
	}

	if resp.Status != pb.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Errorf("expected HEALTH_STATUS_HEALTHY, got %v", resp.Status)
	}

	if resp.Instance.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", resp.Instance.Version)
	}
}

func TestGetStatusMetadata(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "p1"})
	registry.Register(&mockProvider{name: "p2"})

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "2.1.3")

	resp, err := server.GetStatus(context.Background(), &pb.GetStatusRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Instance.Metadata == nil {
		t.Error("expected metadata in instance")
	}

	numProvidersStr, ok := resp.Instance.Metadata["num_providers"]
	if !ok {
		t.Error("expected num_providers in metadata")
	}

	if numProvidersStr != "2" {
		t.Errorf("expected 2 providers, got %s", numProvidersStr)
	}
}

func TestHealthCheck(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "test"})

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.HealthCheck(context.Background(), &pb.HealthCheckRequest{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response")
	}

	if resp.Status != pb.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Errorf("expected HEALTH_STATUS_HEALTHY, got %v", resp.Status)
	}

	if resp.Checks == nil {
		t.Error("expected checks in response")
	}

	if _, ok := resp.Checks["providers"]; !ok {
		t.Error("expected providers check")
	}
}

func TestHealthCheckNilCheck(t *testing.T) {
	registry := providers.NewRegistry()
	// No providers registered

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.HealthCheck(context.Background(), &pb.HealthCheckRequest{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp.Status != pb.HealthStatus_HEALTH_STATUS_DEGRADED {
		t.Errorf("expected HEALTH_STATUS_DEGRADED when no providers, got %v", resp.Status)
	}
}

func TestHealthCheckChecks(t *testing.T) {
	registry := providers.NewRegistry()
	for i := 0; i < 3; i++ {
		name := "provider" + string(rune('0'+i))
		registry.Register(&mockProvider{name: name})
	}

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.HealthCheck(context.Background(), &pb.HealthCheckRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkStr, ok := resp.Checks["providers"]
	if !ok {
		t.Fatal("expected providers check")
	}

	if checkStr != "3 registered" {
		t.Errorf("expected '3 registered', got %s", checkStr)
	}
}

func TestGetStats(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "test-provider"})

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.GetStats(context.Background(), &pb.GetStatsRequest{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response")
	}

	if resp.Stats == nil {
		t.Error("expected stats in response")
	}

	if resp.Stats.CustomStats == nil {
		t.Error("expected custom stats")
	}
}

func TestGetConfig(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "provider1"})

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.GetConfig(context.Background(), &pb.GetConfigRequest{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response")
	}

	if len(resp.Configs) == 0 {
		t.Error("expected configs in response")
	}

	config := resp.Configs[0]
	if config.ConfigId != "ailb-main" {
		t.Errorf("expected config_id 'ailb-main', got %s", config.ConfigId)
	}

	if config.ConfigType != "json" {
		t.Errorf("expected config_type 'json', got %s", config.ConfigType)
	}

	if len(config.ConfigData) == 0 {
		t.Error("expected config data")
	}

	if config.Metadata == nil {
		t.Error("expected metadata in config")
	}
}

func TestGetConfigMetadata(t *testing.T) {
	os.Setenv("MODULE_ID", "custom-ailb")
	defer os.Unsetenv("MODULE_ID")

	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "2.5.0")

	resp, err := server.GetConfig(context.Background(), &pb.GetConfigRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config := resp.Configs[0]
	if config.Metadata["version"] != "2.5.0" {
		t.Errorf("expected version 2.5.0 in metadata, got %s", config.Metadata["version"])
	}

	if config.Metadata["module_id"] != "custom-ailb" {
		t.Errorf("expected module_id custom-ailb in metadata, got %s", config.Metadata["module_id"])
	}
}

func TestGetStatusTimestamps(t *testing.T) {
	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.GetStatus(context.Background(), &pb.GetStatusRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.LastUpdated == nil {
		t.Error("expected LastUpdated timestamp")
	}

	if resp.Instance.StartedAt == nil {
		t.Error("expected StartedAt timestamp")
	}
}

func TestHealthCheckTimestamp(t *testing.T) {
	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.HealthCheck(context.Background(), &pb.HealthCheckRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.CheckedAt == nil {
		t.Error("expected CheckedAt timestamp")
	}
}

func TestMultipleProviders(t *testing.T) {
	registry := providers.NewRegistry()
	for i := 0; i < 5; i++ {
		name := "provider" + string(rune('0'+i))
		registry.Register(&mockProvider{name: name})
	}

	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	resp, err := server.GetStatus(context.Background(), &pb.GetStatusRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numProvidersStr := resp.Instance.Metadata["num_providers"]
	if numProvidersStr != "5" {
		t.Errorf("expected 5 providers, got %s", numProvidersStr)
	}
}

func TestGetStatusUptime(t *testing.T) {
	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	// Sleep a bit to ensure uptime is > 0
	time.Sleep(10 * time.Millisecond)

	resp, err := server.GetStatus(context.Background(), &pb.GetStatusRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uptimeStr := resp.Details["uptime_seconds"]
	if uptimeStr == "" {
		t.Error("expected uptime_seconds in details")
	}
}

func TestHealthCheckUptime(t *testing.T) {
	registry := providers.NewRegistry()
	rtr := router.New(registry, router.StrategyRoundRobin)
	server := grpcserver.NewModuleServer(registry, rtr, "1.0.0")

	// Sleep a bit to ensure uptime is > 0
	time.Sleep(10 * time.Millisecond)

	resp, err := server.HealthCheck(context.Background(), &pb.HealthCheckRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uptimeStr, ok := resp.Checks["uptime"]
	if !ok {
		t.Error("expected uptime check")
	}

	if uptimeStr == "" {
		t.Error("expected non-empty uptime value")
	}
}
