package config

import (
	"os"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	if config == nil {
		t.Fatal("Expected config to be created, got nil")
	}

	// Test default values
	if config.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", config.LogLevel)
	}

	if config.ListenPort != 8080 {
		t.Errorf("Expected default listen port 8080, got %d", config.ListenPort)
	}

	if config.AdminPort != 8081 {
		t.Errorf("Expected default admin port 8081, got %d", config.AdminPort)
	}

	if config.EnableMetrics != true {
		t.Error("Expected metrics to be enabled by default")
	}
}

// TestConfigStructFields validates all configuration structure fields
func TestConfigStructFields(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "L7Config fields",
			test: func(t *testing.T) {
				cfg := &L7Config{
					Enabled:         true,
					EnvoyBinary:     "/usr/bin/envoy",
					EnvoyAdminPort:  9001,
					HTTPListenPort:  8080,
					HTTPSListenPort: 8443,
					HTTP3Enabled:    true,
				}
				if !cfg.Enabled || cfg.EnvoyAdminPort != 9001 {
					t.Error("L7Config field validation failed")
				}
			},
		},
		{
			name: "ThreatConfig fields",
			test: func(t *testing.T) {
				cfg := &ThreatConfig{
					Enabled:               true,
					IPBlockingEnabled:     true,
					IPCacheSize:           1000,
					DomainBlockingEnabled: true,
					URLMatchEngine:        "re2",
				}
				if !cfg.Enabled || cfg.IPCacheSize != 1000 {
					t.Error("ThreatConfig field validation failed")
				}
			},
		},
		{
			name: "TLSInterceptConfig fields",
			test: func(t *testing.T) {
				cfg := &TLSInterceptConfig{
					Enabled:       true,
					Mode:          "mitm",
					CACertPath:    "/etc/certs/ca.crt",
					CertCacheSize: 5000,
				}
				if !cfg.Enabled || cfg.CertCacheSize != 5000 {
					t.Error("TLSInterceptConfig field validation failed")
				}
			},
		},
		{
			name: "ExtAuthConfig fields",
			test: func(t *testing.T) {
				cfg := &ExtAuthConfig{
					Enabled: true,
					Port:    9000,
					Host:    "127.0.0.1",
				}
				if !cfg.Enabled || cfg.Port != 9000 {
					t.Error("ExtAuthConfig field validation failed")
				}
			},
		},
		{
			name: "AccessControlConfig fields",
			test: func(t *testing.T) {
				cfg := &AccessControlConfig{
					Enabled:            true,
					DefaultRequireAuth: true,
					DefaultAllow:       false,
				}
				if !cfg.Enabled || cfg.DefaultAllow {
					t.Error("AccessControlConfig field validation failed")
				}
			},
		},
		{
			name: "LeversConfig fields",
			test: func(t *testing.T) {
				cfg := &LeversConfig{
					Enabled:    true,
					ListenAddr: "127.0.0.1:9010",
				}
				if !cfg.Enabled {
					t.Error("LeversConfig field validation failed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}

// TestConfigManagerSettings validates manager-related config
func TestConfigManagerSettings(t *testing.T) {
	config := NewConfig()
	config.ManagerURL = "http://manager:9000"
	config.ClusterAPIKey = "test-api-key"

	if config.ManagerURL != "http://manager:9000" {
		t.Errorf("expected ManagerURL 'http://manager:9000', got %s", config.ManagerURL)
	}
	if config.ClusterAPIKey != "test-api-key" {
		t.Errorf("expected ClusterAPIKey 'test-api-key', got %s", config.ClusterAPIKey)
	}
}

// TestConfigProxySettings validates proxy-related config
func TestConfigProxySettings(t *testing.T) {
	config := NewConfig()
	config.ProxyName = "proxy-egress-1"
	config.Hostname = "proxy.example.com"
	config.ListenPort = 9000
	config.AdminPort = 9001

	if config.ProxyName != "proxy-egress-1" {
		t.Errorf("expected ProxyName 'proxy-egress-1', got %s", config.ProxyName)
	}
	if config.ListenPort != 9000 {
		t.Errorf("expected ListenPort 9000, got %d", config.ListenPort)
	}
}

// TestConfigPerformanceSettings validates performance-related config
func TestConfigPerformanceSettings(t *testing.T) {
	config := NewConfig()
	config.EnableEBPF = true
	config.EnableMetrics = true
	config.WorkerThreads = 8

	if !config.EnableEBPF {
		t.Error("expected EnableEBPF to be true")
	}
	if config.WorkerThreads != 8 {
		t.Errorf("expected WorkerThreads 8, got %d", config.WorkerThreads)
	}
}

// TestConfigNetworkAcceleration validates network acceleration options
func TestConfigNetworkAcceleration(t *testing.T) {
	config := NewConfig()
	config.EnableDPDK = true
	config.EnableXDP = true
	config.EnableAFXDP = false
	config.DPDKDevices = "0000:01:00.0,0000:01:00.1"

	if !config.EnableDPDK {
		t.Error("expected EnableDPDK to be true")
	}
	if !config.EnableXDP {
		t.Error("expected EnableXDP to be true")
	}
	if config.EnableAFXDP {
		t.Error("expected EnableAFXDP to be false")
	}
	if config.DPDKDevices != "0000:01:00.0,0000:01:00.1" {
		t.Errorf("expected DPDKDevices '0000:01:00.0,0000:01:00.1', got %s", config.DPDKDevices)
	}
}

// TestConfigTLSSettings validates TLS configuration
func TestConfigTLSSettings(t *testing.T) {
	config := NewConfig()
	config.TLSCertPath = "/etc/certs/server.crt"
	config.TLSKeyPath = "/etc/certs/server.key"

	if config.TLSCertPath != "/etc/certs/server.crt" {
		t.Errorf("expected TLSCertPath '/etc/certs/server.crt', got %s", config.TLSCertPath)
	}
}

// TestConfigMTLSSettings validates mutual TLS configuration
func TestConfigMTLSSettings(t *testing.T) {
	config := NewConfig()
	config.EnableMTLS = true
	config.MTLSServerCertPath = "/etc/certs/mtls-server.crt"
	config.MTLSServerKeyPath = "/etc/certs/mtls-server.key"
	config.MTLSClientCAPath = "/etc/certs/client-ca.crt"
	config.MTLSRequireClientCert = true
	config.MTLSVerifyClientCert = true

	if !config.EnableMTLS {
		t.Error("expected EnableMTLS to be true")
	}
	if !config.MTLSRequireClientCert {
		t.Error("expected MTLSRequireClientCert to be true")
	}
}

// TestConfigRateLimiting validates rate limit configuration
func TestConfigRateLimiting(t *testing.T) {
	config := NewConfig()
	config.RateLimitEnabled = true
	config.RateLimitRPS = 1000

	if !config.RateLimitEnabled {
		t.Error("expected RateLimitEnabled to be true")
	}
	if config.RateLimitRPS != 1000 {
		t.Errorf("expected RateLimitRPS 1000, got %d", config.RateLimitRPS)
	}
}

// TestConfigKillKrillIntegration validates KillKrill configuration
func TestConfigKillKrillIntegration(t *testing.T) {
	config := NewConfig()
	config.KillKrillEnabled = true
	config.KillKrillLogEndpoint = "http://killkrill:8000/logs"
	config.KillKrillAPIKey = "test-api-key"
	config.KillKrillBatchSize = 100
	config.KillKrillFlushInterval = 5

	if !config.KillKrillEnabled {
		t.Error("expected KillKrillEnabled to be true")
	}
	if config.KillKrillBatchSize != 100 {
		t.Errorf("expected KillKrillBatchSize 100, got %d", config.KillKrillBatchSize)
	}
}

// TestConfigTimeouts validates timeout configuration
func TestConfigTimeouts(t *testing.T) {
	config := NewConfig()
	config.ConfigUpdateInterval = 30
	config.HeartbeatInterval = 60
	config.ConnectionTimeout = 10

	if config.ConfigUpdateInterval != 30 {
		t.Errorf("expected ConfigUpdateInterval 30, got %d", config.ConfigUpdateInterval)
	}
	if config.HeartbeatInterval != 60 {
		t.Errorf("expected HeartbeatInterval 60, got %d", config.HeartbeatInterval)
	}
	if config.ConnectionTimeout != 10 {
		t.Errorf("expected ConnectionTimeout 10, got %d", config.ConnectionTimeout)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("MANAGER_URL", "http://test-manager:8000")
	os.Setenv("CLUSTER_API_KEY", "test-api-key")
	os.Setenv("PROXY_NAME", "test-proxy")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LISTEN_PORT", "9090")
	os.Setenv("ENABLE_EBPF", "true")
	defer func() {
		os.Unsetenv("MANAGER_URL")
		os.Unsetenv("CLUSTER_API_KEY")
		os.Unsetenv("PROXY_NAME")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LISTEN_PORT")
		os.Unsetenv("ENABLE_EBPF")
	}()

	config := NewConfig()
	err := config.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load from environment: %v", err)
	}

	if config.ManagerURL != "http://test-manager:8000" {
		t.Errorf("Expected ManagerURL 'http://test-manager:8000', got %s", config.ManagerURL)
	}

	if config.ClusterAPIKey != "test-api-key" {
		t.Errorf("Expected ClusterAPIKey 'test-api-key', got %s", config.ClusterAPIKey)
	}

	if config.ProxyName != "test-proxy" {
		t.Errorf("Expected ProxyName 'test-proxy', got %s", config.ProxyName)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got %s", config.LogLevel)
	}

	if config.ListenPort != 9090 {
		t.Errorf("Expected ListenPort 9090, got %d", config.ListenPort)
	}

	if !config.EnableEBPF {
		t.Error("Expected EnableEBPF to be true")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary config file
	configContent := `
manager_url: "http://file-manager:8000"
cluster_api_key: "file-api-key"
proxy_name: "file-proxy"
log_level: "warn"
listen_port: 7070
admin_port: 7071
enable_metrics: false
enable_ebpf: true
enable_xdp: true
worker_threads: 8
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tmpFile.Close()

	config := NewConfig()
	err = config.LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	if config.ManagerURL != "http://file-manager:8000" {
		t.Errorf("Expected ManagerURL 'http://file-manager:8000', got %s", config.ManagerURL)
	}

	if config.ClusterAPIKey != "file-api-key" {
		t.Errorf("Expected ClusterAPIKey 'file-api-key', got %s", config.ClusterAPIKey)
	}

	if config.LogLevel != "warn" {
		t.Errorf("Expected LogLevel 'warn', got %s", config.LogLevel)
	}

	if config.ListenPort != 7070 {
		t.Errorf("Expected ListenPort 7070, got %d", config.ListenPort)
	}

	if config.AdminPort != 7071 {
		t.Errorf("Expected AdminPort 7071, got %d", config.AdminPort)
	}

	if config.EnableMetrics {
		t.Error("Expected EnableMetrics to be false")
	}

	if !config.EnableEBPF {
		t.Error("Expected EnableEBPF to be true")
	}

	if !config.EnableXDP {
		t.Error("Expected EnableXDP to be true")
	}

	if config.WorkerThreads != 8 {
		t.Errorf("Expected WorkerThreads 8, got %d", config.WorkerThreads)
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "valid-api-key",
				ProxyName:     "test-proxy",
				ListenPort:    8080,
				AdminPort:     8081,
			},
			expectError: false,
		},
		{
			name: "Missing manager URL",
			config: &Config{
				ClusterAPIKey: "valid-api-key",
				ProxyName:     "test-proxy",
				ListenPort:    8080,
				AdminPort:     8081,
			},
			expectError: true,
			errorMsg:    "manager_url",
		},
		{
			name: "Missing cluster API key",
			config: &Config{
				ManagerURL: "http://manager:8000",
				ProxyName:  "test-proxy",
				ListenPort: 8080,
				AdminPort:  8081,
			},
			expectError: true,
			errorMsg:    "cluster_api_key",
		},
		{
			name: "Invalid port range",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "valid-api-key",
				ProxyName:     "test-proxy",
				ListenPort:    70000, // Invalid port
				AdminPort:     8081,
			},
			expectError: true,
			errorMsg:    "port",
		},
		{
			name: "Same ports",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "valid-api-key",
				ProxyName:     "test-proxy",
				ListenPort:    8080,
				AdminPort:     8080, // Same as listen port
			},
			expectError: true,
			errorMsg:    "same port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}

			if tc.expectError && err != nil && tc.errorMsg != "" {
				if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.errorMsg, err)
				}
			}
		})
	}
}

func TestGetHostname(t *testing.T) {
	config := NewConfig()

	// Test with empty hostname (should get system hostname)
	hostname := config.GetHostname()
	if hostname == "" {
		t.Error("Expected hostname to be non-empty")
	}

	// Test with custom hostname
	config.Hostname = "custom-hostname"
	hostname = config.GetHostname()
	if hostname != "custom-hostname" {
		t.Errorf("Expected hostname 'custom-hostname', got %s", hostname)
	}
}

func TestIsAccelerationEnabled(t *testing.T) {
	config := NewConfig()

	// Test with no acceleration enabled
	if config.IsAccelerationEnabled() {
		t.Error("Expected acceleration to be disabled by default")
	}

	// Test with eBPF enabled
	config.EnableEBPF = true
	if !config.IsAccelerationEnabled() {
		t.Error("Expected acceleration to be enabled with eBPF")
	}

	// Test with XDP enabled
	config = NewConfig()
	config.EnableXDP = true
	if !config.IsAccelerationEnabled() {
		t.Error("Expected acceleration to be enabled with XDP")
	}

	// Test with DPDK enabled
	config = NewConfig()
	config.EnableDPDK = true
	if !config.IsAccelerationEnabled() {
		t.Error("Expected acceleration to be enabled with DPDK")
	}
}

func TestGetListenAddress(t *testing.T) {
	config := NewConfig()
	config.ListenPort = 8080

	address := config.GetListenAddress()
	expected := ":8080"
	if address != expected {
		t.Errorf("Expected listen address '%s', got %s", expected, address)
	}
}

func TestGetAdminAddress(t *testing.T) {
	config := NewConfig()
	config.AdminPort = 8081

	address := config.GetAdminAddress()
	expected := ":8081"
	if address != expected {
		t.Errorf("Expected admin address '%s', got %s", expected, address)
	}
}

func TestGetWorkerThreads(t *testing.T) {
	config := NewConfig()

	// Test default (should be number of CPUs)
	threads := config.GetWorkerThreads()
	if threads <= 0 {
		t.Error("Expected positive number of worker threads")
	}

	// Test custom value
	config.WorkerThreads = 16
	threads = config.GetWorkerThreads()
	if threads != 16 {
		t.Errorf("Expected 16 worker threads, got %d", threads)
	}
}

func TestIsTLSEnabled(t *testing.T) {
	config := NewConfig()

	// Test with no TLS
	if config.IsTLSEnabled() {
		t.Error("Expected TLS to be disabled by default")
	}

	// Test with TLS cert path only
	config.TLSCertPath = "/path/to/cert.pem"
	if config.IsTLSEnabled() {
		t.Error("Expected TLS to be disabled with only cert path")
	}

	// Test with both cert and key paths
	config.TLSKeyPath = "/path/to/key.pem"
	if !config.IsTLSEnabled() {
		t.Error("Expected TLS to be enabled with both cert and key paths")
	}
}

func TestIsMTLSEnabled(t *testing.T) {
	config := NewConfig()

	// Test with mTLS disabled
	if config.IsMTLSEnabled() {
		t.Error("Expected mTLS to be disabled by default")
	}

	// Test with mTLS enabled but missing paths
	config.EnableMTLS = true
	if config.IsMTLSEnabled() {
		t.Error("Expected mTLS to be disabled with missing cert paths")
	}

	// Test with all mTLS paths
	config.MTLSServerCertPath = "/path/to/server.crt"
	config.MTLSServerKeyPath = "/path/to/server.key"
	config.MTLSClientCAPath = "/path/to/ca.crt"
	if !config.IsMTLSEnabled() {
		t.Error("Expected mTLS to be enabled with all paths configured")
	}
}

func TestEnvironmentVariableParsing(t *testing.T) {
	// Test boolean parsing
	os.Setenv("ENABLE_METRICS", "false")
	os.Setenv("ENABLE_EBPF", "true")
	defer func() {
		os.Unsetenv("ENABLE_METRICS")
		os.Unsetenv("ENABLE_EBPF")
	}()

	config := NewConfig()
	err := config.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load from environment: %v", err)
	}

	if config.EnableMetrics {
		t.Error("Expected EnableMetrics to be false")
	}

	if !config.EnableEBPF {
		t.Error("Expected EnableEBPF to be true")
	}
}

func TestConfigPriority(t *testing.T) {
	// Set environment variable
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")

	// Create config file with different value
	configContent := `log_level: "debug"`
	tmpFile, err := os.CreateTemp("", "config_priority_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tmpFile.Close()

	config := NewConfig()

	// Load from file first
	err = config.LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	// Load from environment (should override file)
	err = config.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load from environment: %v", err)
	}

	// Environment should take precedence
	if config.LogLevel != "error" {
		t.Errorf("Expected LogLevel 'error' (from env), got %s", config.LogLevel)
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := &Config{
		ManagerURL:    "http://manager:8000",
		ClusterAPIKey: "valid-api-key",
		ProxyName:     "test-proxy",
		ListenPort:    8080,
		AdminPort:     8081,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}