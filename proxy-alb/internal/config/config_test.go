//go:build ci

package config_test

import (
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	// No special env vars – all defaults should apply.
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.ModuleID != "alb-1" {
		t.Errorf("expected ModuleID 'alb-1', got %q", cfg.ModuleID)
	}
	if cfg.ModuleType != "ALB" {
		t.Errorf("expected ModuleType 'ALB', got %q", cfg.ModuleType)
	}
	if cfg.EnvoyBinary != "/usr/local/bin/envoy" {
		t.Errorf("expected EnvoyBinary '/usr/local/bin/envoy', got %q", cfg.EnvoyBinary)
	}
	if cfg.EnvoyAdminPort != 9901 {
		t.Errorf("expected EnvoyAdminPort 9901, got %d", cfg.EnvoyAdminPort)
	}
	if cfg.EnvoyListenPort != 10000 {
		t.Errorf("expected EnvoyListenPort 10000, got %d", cfg.EnvoyListenPort)
	}
}

func TestLoadConfigXDSDefaults(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.XDSServerAddr != "api-server:18000" {
		t.Errorf("expected XDSServerAddr 'api-server:18000', got %q", cfg.XDSServerAddr)
	}
	if cfg.XDSNodeID != "alb-node" {
		t.Errorf("expected XDSNodeID 'alb-node', got %q", cfg.XDSNodeID)
	}
	if cfg.XDSCluster != "marchproxy-cluster" {
		t.Errorf("expected XDSCluster 'marchproxy-cluster', got %q", cfg.XDSCluster)
	}
	if cfg.XDSConnectTimeout != 5*time.Second {
		t.Errorf("expected XDSConnectTimeout 5s, got %v", cfg.XDSConnectTimeout)
	}
}

func TestLoadConfigPortDefaults(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.GRPCPort != 50051 {
		t.Errorf("expected GRPCPort 50051, got %d", cfg.GRPCPort)
	}
	if cfg.MetricsPort != 9090 {
		t.Errorf("expected MetricsPort 9090, got %d", cfg.MetricsPort)
	}
	if cfg.HealthCheckPort != 8080 {
		t.Errorf("expected HealthCheckPort 8080, got %d", cfg.HealthCheckPort)
	}
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	t.Setenv("MODULE_ID", "alb-test-42")
	t.Setenv("GRPC_PORT", "50099")
	t.Setenv("ENVOY_LOG_LEVEL", "debug")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.ModuleID != "alb-test-42" {
		t.Errorf("expected ModuleID 'alb-test-42', got %q", cfg.ModuleID)
	}
	if cfg.GRPCPort != 50099 {
		t.Errorf("expected GRPCPort 50099, got %d", cfg.GRPCPort)
	}
	if cfg.EnvoyLogLevel != "debug" {
		t.Errorf("expected EnvoyLogLevel 'debug', got %q", cfg.EnvoyLogLevel)
	}
}

func TestValidateConfigValid(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        50051,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidateConfigMissingModuleID(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        50051,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty ModuleID, got nil")
	}
}

func TestValidateConfigMissingEnvoyBinary(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        50051,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty EnvoyBinary, got nil")
	}
}

func TestValidateConfigMissingXDSServer(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "",
		GRPCPort:        50051,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty XDSServerAddr, got nil")
	}
}

func TestValidateConfigInvalidGRPCPort(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        0,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for GRPCPort 0, got nil")
	}
}

func TestLoadConfigShutdownTimeout(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("expected ShutdownTimeout 30s, got %v", cfg.ShutdownTimeout)
	}
	if cfg.ReloadGracePeriod != 5*time.Second {
		t.Errorf("expected ReloadGracePeriod 5s, got %v", cfg.ReloadGracePeriod)
	}
}

func TestValidateConfigInvalidAdminPort(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        50051,
		EnvoyAdminPort:  65536,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid EnvoyAdminPort")
	}
}

func TestValidateConfigInvalidGRPCPortHigh(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        65536,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for GRPCPort above 65535")
	}
}

func TestValidateConfigMissingEnvoyConfig(t *testing.T) {
	cfg := &config.Config{
		ModuleID:        "alb-1",
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "",
		XDSServerAddr:   "api-server:18000",
		GRPCPort:        50051,
		EnvoyAdminPort:  9901,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty EnvoyConfigPath")
	}
}

func TestLoadConfigWithBadPortEnvVar(t *testing.T) {
	t.Setenv("GRPC_PORT", "not-a-number")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	// Should default to 50051 when env var is invalid
	if cfg.GRPCPort != 50051 {
		t.Errorf("expected GRPCPort to default to 50051, got %d", cfg.GRPCPort)
	}
}

func TestLoadConfigWithBadDurationEnvVar(t *testing.T) {
	t.Setenv("XDS_CONNECT_TIMEOUT", "invalid-duration")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	// Should default to 5s when env var is invalid
	if cfg.XDSConnectTimeout != 5*time.Second {
		t.Errorf("expected XDSConnectTimeout to default to 5s, got %v", cfg.XDSConnectTimeout)
	}
}

func TestLoadConfigWithValidDuration(t *testing.T) {
	t.Setenv("XDS_CONNECT_TIMEOUT", "15s")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.XDSConnectTimeout != 15*time.Second {
		t.Errorf("expected XDSConnectTimeout 15s, got %v", cfg.XDSConnectTimeout)
	}
}

func TestLoadConfigLicenseAndAPIKey(t *testing.T) {
	t.Setenv("LICENSE_KEY", "test-license-key-123")
	t.Setenv("CLUSTER_API_KEY", "test-api-key-456")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.LicenseKey != "test-license-key-123" {
		t.Errorf("expected LicenseKey 'test-license-key-123', got %q", cfg.LicenseKey)
	}

	if cfg.ClusterAPIKey != "test-api-key-456" {
		t.Errorf("expected ClusterAPIKey 'test-api-key-456', got %q", cfg.ClusterAPIKey)
	}
}

func TestLoadConfigMultipleEnvVars(t *testing.T) {
	envVars := map[string]string{
		"MODULE_ID":            "alb-custom",
		"MODULE_TYPE":          "ALB_CUSTOM",
		"VERSION":              "v2.0.0",
		"ENVOY_BINARY":         "/opt/envoy",
		"ENVOY_CONFIG_PATH":    "/opt/config.yaml",
		"ENVOY_ADMIN_PORT":     "9902",
		"ENVOY_LOG_LEVEL":      "trace",
		"XDS_SERVER":           "xds.example.com:18000",
		"XDS_NODE_ID":          "custom-node",
		"XDS_CLUSTER":          "custom-cluster",
	}

	for key, value := range envVars {
		t.Setenv(key, value)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.ModuleID != "alb-custom" {
		t.Errorf("expected ModuleID 'alb-custom', got %q", cfg.ModuleID)
	}

	if cfg.Version != "v2.0.0" {
		t.Errorf("expected Version 'v2.0.0', got %q", cfg.Version)
	}

	if cfg.EnvoyBinary != "/opt/envoy" {
		t.Errorf("expected EnvoyBinary '/opt/envoy', got %q", cfg.EnvoyBinary)
	}

	if cfg.EnvoyAdminPort != 9902 {
		t.Errorf("expected EnvoyAdminPort 9902, got %d", cfg.EnvoyAdminPort)
	}

	if cfg.EnvoyLogLevel != "trace" {
		t.Errorf("expected EnvoyLogLevel 'trace', got %q", cfg.EnvoyLogLevel)
	}

	if cfg.XDSServerAddr != "xds.example.com:18000" {
		t.Errorf("expected XDSServerAddr 'xds.example.com:18000', got %q", cfg.XDSServerAddr)
	}

	if cfg.XDSNodeID != "custom-node" {
		t.Errorf("expected XDSNodeID 'custom-node', got %q", cfg.XDSNodeID)
	}
}
