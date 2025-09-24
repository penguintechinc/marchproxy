package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
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