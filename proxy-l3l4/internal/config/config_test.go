package config_test

import (
	"os"
	"testing"
	"time"

	"marchproxy-l3l4/internal/config"
)

// loadDefaults calls Load with no config file but a valid CLUSTER_API_KEY
// env var so Validate() passes, then returns the resulting Config.
func loadDefaults(t *testing.T) *config.Config {
	t.Helper()
	t.Setenv("CLUSTER_API_KEY", "test-key")
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	return cfg
}

func TestLoad_DefaultBindAddr(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.BindAddr != ":8081" {
		t.Errorf("BindAddr = %q, want %q", cfg.BindAddr, ":8081")
	}
}

func TestLoad_DefaultMetricsAddr(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.MetricsAddr != ":8082" {
		t.Errorf("MetricsAddr = %q, want %q", cfg.MetricsAddr, ":8082")
	}
}

func TestLoad_DefaultManagerURL(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.ManagerURL != "http://api-server:8000" {
		t.Errorf("ManagerURL = %q, want %q", cfg.ManagerURL, "http://api-server:8000")
	}
}

func TestLoad_DefaultQoSBandwidth(t *testing.T) {
	cfg := loadDefaults(t)
	want := int64(1_000_000_000) // 1 Gbps
	if cfg.DefaultBandwidth != want {
		t.Errorf("DefaultBandwidth = %d, want %d", cfg.DefaultBandwidth, want)
	}
}

func TestLoad_DefaultQoSBurstSize(t *testing.T) {
	cfg := loadDefaults(t)
	want := int64(100_000_000) // 100 MB
	if cfg.BurstSize != want {
		t.Errorf("BurstSize = %d, want %d", cfg.BurstSize, want)
	}
}

func TestLoad_DefaultQoSEnabled(t *testing.T) {
	cfg := loadDefaults(t)
	if !cfg.EnableQoS {
		t.Errorf("EnableQoS = false, want true")
	}
}

func TestLoad_DefaultPriorityQueueDepth(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.PriorityQueueDepth != 1000 {
		t.Errorf("PriorityQueueDepth = %d, want 1000", cfg.PriorityQueueDepth)
	}
}

func TestLoad_DefaultZeroTrustEnabled(t *testing.T) {
	cfg := loadDefaults(t)
	if !cfg.EnableZeroTrust {
		t.Errorf("EnableZeroTrust = false, want true")
	}
}

func TestLoad_DefaultOpaURL(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.OpaURL != "http://opa:8181" {
		t.Errorf("OpaURL = %q, want %q", cfg.OpaURL, "http://opa:8181")
	}
}

func TestLoad_DefaultNUMADisabled(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.EnableNUMA {
		t.Errorf("EnableNUMA = true, want false")
	}
}

func TestLoad_DefaultWorkerThreads(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.WorkerThreads != 0 {
		t.Errorf("WorkerThreads = %d, want 0 (auto-detect)", cfg.WorkerThreads)
	}
}

func TestLoad_DefaultRoutingAlgorithm(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.RoutingAlgorithm != "latency" {
		t.Errorf("RoutingAlgorithm = %q, want %q", cfg.RoutingAlgorithm, "latency")
	}
}

func TestLoad_DefaultHealthCheckInterval(t *testing.T) {
	cfg := loadDefaults(t)
	want := 30 * time.Second
	if cfg.HealthCheckInterval != want {
		t.Errorf("HealthCheckInterval = %v, want %v", cfg.HealthCheckInterval, want)
	}
}

func TestLoad_DefaultAccelerationMode(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.AccelerationMode != "standard" {
		t.Errorf("AccelerationMode = %q, want %q", cfg.AccelerationMode, "standard")
	}
}

func TestLoad_DefaultAccelerationDisabled(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.EnableAcceleration {
		t.Errorf("EnableAcceleration = true, want false")
	}
}

func TestLoad_DefaultLicenseServer(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.LicenseServer != "https://license.penguintech.io" {
		t.Errorf("LicenseServer = %q, want %q", cfg.LicenseServer, "https://license.penguintech.io")
	}
}

func TestLoad_DefaultReleaseModeDisabled(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.ReleaseMode {
		t.Errorf("ReleaseMode = true, want false")
	}
}

func TestLoad_DefaultMetricsNamespace(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.MetricsNamespace != "marchproxy" {
		t.Errorf("MetricsNamespace = %q, want %q", cfg.MetricsNamespace, "marchproxy")
	}
}

func TestLoad_DefaultTraceSampleRate(t *testing.T) {
	cfg := loadDefaults(t)
	if cfg.TraceSampleRate != 0.1 {
		t.Errorf("TraceSampleRate = %f, want 0.1", cfg.TraceSampleRate)
	}
}

// TestValidate_MissingManagerURL exercises the Validate error path.
func TestValidate_MissingManagerURL(t *testing.T) {
	cfg := &config.Config{
		// ManagerURL intentionally empty
		DefaultBandwidth: 1,
		BurstSize:        1,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for empty ManagerURL, got nil")
	}
}

// TestValidate_MissingClusterAPIKey exercises the API-key path via env.
func TestValidate_MissingClusterAPIKey(t *testing.T) {
	os.Unsetenv("CLUSTER_API_KEY")
	cfg := &config.Config{
		ManagerURL:       "http://api-server:8000",
		DefaultBandwidth: 1,
		BurstSize:        1,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error when ClusterAPIKey and env are both empty, got nil")
	}
}

func TestValidate_QoSZeroBandwidth(t *testing.T) {
	t.Setenv("CLUSTER_API_KEY", "test-key")
	cfg := &config.Config{
		ManagerURL:       "http://api-server:8000",
		ClusterAPIKey:    "test-key",
		EnableQoS:        true,
		DefaultBandwidth: 0, // invalid
		BurstSize:        100,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for DefaultBandwidth=0, got nil")
	}
}

func TestValidate_QoSZeroBurstSize(t *testing.T) {
	t.Setenv("CLUSTER_API_KEY", "test-key")
	cfg := &config.Config{
		ManagerURL:       "http://api-server:8000",
		ClusterAPIKey:    "test-key",
		EnableQoS:        true,
		DefaultBandwidth: 100,
		BurstSize:        0, // invalid
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for BurstSize=0, got nil")
	}
}

func TestValidate_InvalidRoutingAlgorithm(t *testing.T) {
	t.Setenv("CLUSTER_API_KEY", "test-key")
	cfg := &config.Config{
		ManagerURL:       "http://api-server:8000",
		ClusterAPIKey:    "test-key",
		EnableQoS:        true,
		DefaultBandwidth: 1,
		BurstSize:        1,
		EnableMultiCloud: true,
		RoutingAlgorithm: "bogus",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for invalid RoutingAlgorithm, got nil")
	}
}

func TestValidate_ValidRoutingAlgorithms(t *testing.T) {
	validAlgos := []string{"latency", "cost", "geo", "roundrobin", "leastconn"}
	for _, algo := range validAlgos {
		t.Run(algo, func(t *testing.T) {
			t.Setenv("CLUSTER_API_KEY", "test-key")
			cfg := &config.Config{
				ManagerURL:       "http://api-server:8000",
				ClusterAPIKey:    "test-key",
				EnableQoS:        true,
				DefaultBandwidth: 1,
				BurstSize:        1,
				EnableMultiCloud: true,
				RoutingAlgorithm: algo,
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() unexpected error for algorithm %q: %v", algo, err)
			}
		})
	}
}

func TestValidate_InvalidAccelerationMode(t *testing.T) {
	t.Setenv("CLUSTER_API_KEY", "test-key")
	cfg := &config.Config{
		ManagerURL:         "http://api-server:8000",
		ClusterAPIKey:      "test-key",
		EnableQoS:          true,
		DefaultBandwidth:   1,
		BurstSize:          1,
		EnableAcceleration: true,
		AccelerationMode:   "turbo", // invalid
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for invalid AccelerationMode, got nil")
	}
}

func TestValidate_ValidAccelerationModes(t *testing.T) {
	validModes := []string{"standard", "xdp", "afxdp", "dpdk"}
	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("CLUSTER_API_KEY", "test-key")
			cfg := &config.Config{
				ManagerURL:         "http://api-server:8000",
				ClusterAPIKey:      "test-key",
				EnableQoS:          true,
				DefaultBandwidth:   1,
				BurstSize:          1,
				EnableAcceleration: true,
				AccelerationMode:   mode,
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() unexpected error for mode %q: %v", mode, err)
			}
		})
	}
}

func TestIsEnterpriseFeatureEnabled_DevMode(t *testing.T) {
	cfg := &config.Config{ReleaseMode: false}
	if !cfg.IsEnterpriseFeatureEnabled("any-feature") {
		t.Error("IsEnterpriseFeatureEnabled() = false in dev mode, want true")
	}
}

func TestIsEnterpriseFeatureEnabled_ReleaseModeWithKey(t *testing.T) {
	cfg := &config.Config{ReleaseMode: true, LicenseKey: "some-key"}
	if !cfg.IsEnterpriseFeatureEnabled("some-feature") {
		t.Error("IsEnterpriseFeatureEnabled() = false with license key, want true")
	}
}

func TestIsEnterpriseFeatureEnabled_ReleaseModeWithoutKey(t *testing.T) {
	cfg := &config.Config{ReleaseMode: true, LicenseKey: ""}
	if cfg.IsEnterpriseFeatureEnabled("some-feature") {
		t.Error("IsEnterpriseFeatureEnabled() = true without license key, want false")
	}
}

func TestBackendConfig_Fields(t *testing.T) {
	bc := config.BackendConfig{
		Name:     "backend-1",
		URL:      "http://10.0.0.1:8080",
		Weight:   5,
		Priority: 1,
		Cloud:    "aws",
		Region:   "us-east-1",
		Cost:     0.05,
		Timeout:  10 * time.Second,
	}
	if bc.Name != "backend-1" {
		t.Errorf("Name = %q, want %q", bc.Name, "backend-1")
	}
	if bc.Weight != 5 {
		t.Errorf("Weight = %d, want 5", bc.Weight)
	}
	if bc.Cloud != "aws" {
		t.Errorf("Cloud = %q, want %q", bc.Cloud, "aws")
	}
	if bc.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", bc.Timeout)
	}
}
