//go:build ci

package config_test

import (
	"os"
	"testing"
	"time"

	"marchproxy-nlb/internal/config"
)

// withAPIKey sets the required CLUSTER_API_KEY for the duration of the test.
func withAPIKey(t *testing.T) {
	t.Helper()
	t.Setenv("CLUSTER_API_KEY", "test-api-key")
}

func TestLoadConfigDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.BindAddr != ":8080" {
		t.Errorf("expected BindAddr ':8080', got %q", cfg.BindAddr)
	}
	if cfg.GRPCAddr != "0.0.0.0" {
		t.Errorf("expected GRPCAddr '0.0.0.0', got %q", cfg.GRPCAddr)
	}
	if cfg.GRPCPort != 50051 {
		t.Errorf("expected GRPCPort 50051, got %d", cfg.GRPCPort)
	}
	if cfg.MetricsAddr != ":8082" {
		t.Errorf("expected MetricsAddr ':8082', got %q", cfg.MetricsAddr)
	}
	if cfg.ManagerURL != "http://api-server:8000" {
		t.Errorf("expected ManagerURL 'http://api-server:8000', got %q", cfg.ManagerURL)
	}
}

func TestConfigRateLimitingDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.EnableRateLimiting {
		t.Error("expected EnableRateLimiting to be true by default")
	}
	if cfg.DefaultRateLimit != 10000.0 {
		t.Errorf("expected DefaultRateLimit 10000.0, got %f", cfg.DefaultRateLimit)
	}
	if cfg.DefaultBurstSize != 20000.0 {
		t.Errorf("expected DefaultBurstSize 20000.0, got %f", cfg.DefaultBurstSize)
	}
}

func TestConfigAutoscalingDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.EnableAutoscaling {
		t.Error("expected EnableAutoscaling to be true by default")
	}
	if cfg.AutoscaleInterval != 30*time.Second {
		t.Errorf("expected AutoscaleInterval 30s, got %v", cfg.AutoscaleInterval)
	}
	if cfg.ScaleUpCooldown != 3*time.Minute {
		t.Errorf("expected ScaleUpCooldown 3m, got %v", cfg.ScaleUpCooldown)
	}
	if cfg.ScaleDownCooldown != 5*time.Minute {
		t.Errorf("expected ScaleDownCooldown 5m, got %v", cfg.ScaleDownCooldown)
	}
}

func TestConfigBlueGreenDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.EnableBlueGreen {
		t.Error("expected EnableBlueGreen to be true by default")
	}
	if cfg.CanaryStepSize != 10 {
		t.Errorf("expected CanaryStepSize 10, got %d", cfg.CanaryStepSize)
	}
}

func TestConfigModuleDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.MaxModulesPerProtocol != 50 {
		t.Errorf("expected MaxModulesPerProtocol 50, got %d", cfg.MaxModulesPerProtocol)
	}
	if cfg.MaxConnectionsPerModule != 10000 {
		t.Errorf("expected MaxConnectionsPerModule 10000, got %d", cfg.MaxConnectionsPerModule)
	}
	if !cfg.EnableConnectionPooling {
		t.Error("expected EnableConnectionPooling to be true by default")
	}
}

func TestConfigLicensingDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.LicenseServer != "https://license.penguintech.io" {
		t.Errorf("expected LicenseServer 'https://license.penguintech.io', got %q", cfg.LicenseServer)
	}
	if cfg.ReleaseMode {
		t.Error("expected ReleaseMode false by default")
	}
}

func TestConfigValidateRequiresAPIKey(t *testing.T) {
	os.Unsetenv("CLUSTER_API_KEY")

	_, err := config.Load("")
	if err == nil {
		t.Error("expected error when CLUSTER_API_KEY is missing")
	}
}

func TestConfigValidateInvalidGRPCPort(t *testing.T) {
	withAPIKey(t)

	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                0,
		DefaultRateLimit:        10000,
		DefaultBurstSize:        20000,
		EnableRateLimiting:      true,
		EnableAutoscaling:       true,
		AutoscaleInterval:       30 * time.Second,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		EnableBlueGreen:         true,
		CanaryStepSize:          10,
		CanaryStepDuration:      2 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid GRPCPort 0, got nil")
	}
}

func TestConfigIsEnterpriseFeatureEnabledDevMode(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	// In dev mode (ReleaseMode = false), all features available.
	if !cfg.IsEnterpriseFeatureEnabled("any-feature") {
		t.Error("expected enterprise feature enabled in dev mode")
	}
}

func TestConfigIsEnterpriseFeatureEnabledReleaseMode(t *testing.T) {
	cfg := &config.Config{
		ReleaseMode: true,
		LicenseKey:  "",
	}

	if cfg.IsEnterpriseFeatureEnabled("sso") {
		t.Error("expected enterprise feature disabled in release mode without license key")
	}

	cfg.LicenseKey = "valid-license"
	if !cfg.IsEnterpriseFeatureEnabled("sso") {
		t.Error("expected enterprise feature enabled in release mode with license key")
	}
}

func TestConfigValidateInvalidRateLimit(t *testing.T) {
	withAPIKey(t)

	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                50051,
		EnableRateLimiting:      true,
		DefaultRateLimit:        -1,
		DefaultBurstSize:        20000,
		EnableAutoscaling:       true,
		AutoscaleInterval:       30 * time.Second,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		EnableBlueGreen:         true,
		CanaryStepSize:          10,
		CanaryStepDuration:      2 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid DefaultRateLimit")
	}
}

func TestConfigValidateInvalidBurstSize(t *testing.T) {
	withAPIKey(t)

	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                50051,
		EnableRateLimiting:      true,
		DefaultRateLimit:        10000,
		DefaultBurstSize:        -1,
		EnableAutoscaling:       true,
		AutoscaleInterval:       30 * time.Second,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		EnableBlueGreen:         true,
		CanaryStepSize:          10,
		CanaryStepDuration:      2 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid DefaultBurstSize")
	}
}

func TestConfigValidateInvalidAutoscaleInterval(t *testing.T) {
	withAPIKey(t)

	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                50051,
		EnableRateLimiting:      true,
		DefaultRateLimit:        10000,
		DefaultBurstSize:        20000,
		EnableAutoscaling:       true,
		AutoscaleInterval:       -1,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		EnableBlueGreen:         true,
		CanaryStepSize:          10,
		CanaryStepDuration:      2 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid AutoscaleInterval")
	}
}

func TestConfigValidateInvalidCanaryStepSize(t *testing.T) {
	withAPIKey(t)

	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                50051,
		EnableRateLimiting:      true,
		DefaultRateLimit:        10000,
		DefaultBurstSize:        20000,
		EnableAutoscaling:       true,
		AutoscaleInterval:       30 * time.Second,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		EnableBlueGreen:         true,
		CanaryStepSize:          101,
		CanaryStepDuration:      2 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for CanaryStepSize > 100")
	}
}

func TestConfigValidateNoManagerURL(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:              "",
		ClusterAPIKey:           "key",
		GRPCPort:                50051,
		DefaultRateLimit:        10000,
		DefaultBurstSize:        20000,
		EnableAutoscaling:       true,
		AutoscaleInterval:       30 * time.Second,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error when ManagerURL is empty")
	}
}

func TestConfigValidateNoAPIKey(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "",
		GRPCPort:                50051,
		DefaultRateLimit:        10000,
		DefaultBurstSize:        20000,
		EnableAutoscaling:       true,
		AutoscaleInterval:       30 * time.Second,
		ScaleUpCooldown:         3 * time.Minute,
		ScaleDownCooldown:       5 * time.Minute,
		MaxModulesPerProtocol:   50,
		MaxConnectionsPerModule: 10000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error when ClusterAPIKey is empty")
	}
}
