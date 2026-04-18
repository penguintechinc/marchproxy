//go:build ci

package config

import (
	"os"
	"testing"
)

// TestLoadFromEnvironment_BasicVars tests loading basic environment variables
func TestLoadFromEnvironment_BasicVars(t *testing.T) {
	os.Setenv("MANAGER_URL", "http://manager:9000")
	os.Setenv("CLUSTER_API_KEY", "key123")
	os.Setenv("PROXY_NAME", "egress-proxy-1")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("MANAGER_URL")
		os.Unsetenv("CLUSTER_API_KEY")
		os.Unsetenv("PROXY_NAME")
		os.Unsetenv("LOG_LEVEL")
	}()

	config := NewConfig()
	err := config.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load from environment: %v", err)
	}

	if config.ManagerURL != "http://manager:9000" {
		t.Errorf("ManagerURL: got %q, want %q", config.ManagerURL, "http://manager:9000")
	}
	if config.ProxyName != "egress-proxy-1" {
		t.Errorf("ProxyName: got %q, want %q", config.ProxyName, "egress-proxy-1")
	}
}

// TestLoadFromFile_SimpleConfig tests loading a simple config file
func TestLoadFromFile_SimpleConfig(t *testing.T) {
	configContent := `
manager_url: "http://file-manager:8000"
cluster_api_key: "file-key"
proxy_name: "file-proxy"
log_level: "warn"
listen_port: 7070
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	config := NewConfig()
	err = config.LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	if config.ManagerURL != "http://file-manager:8000" {
		t.Errorf("ManagerURL: got %q, want %q", config.ManagerURL, "http://file-manager:8000")
	}
	if config.ListenPort != 7070 {
		t.Errorf("ListenPort: got %d, want 7070", config.ListenPort)
	}
}

// TestValidateConfig_InvalidCases tests validation failures
func TestValidateConfig_InvalidCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "ValidConfig",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "key",
				ProxyName:     "proxy",
				ListenPort:    8080,
				AdminPort:     8081,
			},
			expectError: false,
		},
		{
			name: "MissingManagerURL",
			config: &Config{
				ClusterAPIKey: "key",
				ProxyName:     "proxy",
				ListenPort:    8080,
				AdminPort:     8081,
			},
			expectError: true,
		},
		{
			name: "InvalidPort",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "key",
				ProxyName:     "proxy",
				ListenPort:    70000,
				AdminPort:     8081,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate: expectError=%v, got err=%v", tt.expectError, err)
			}
		})
	}
}

// TestGetHostname tests hostname retrieval
func TestGetHostname_Values(t *testing.T) {
	config := NewConfig()
	config.Hostname = "custom.example.com"
	
	hostname := config.GetHostname()
	if hostname != "custom.example.com" {
		t.Errorf("GetHostname: got %q, want %q", hostname, "custom.example.com")
	}
}

// TestGetListenAddress tests listen address formatting
func TestGetListenAddress_Ports(t *testing.T) {
	tests := []struct {
		port     int
		expected string
	}{
		{8080, ":8080"},
		{9000, ":9000"},
		{80, ":80"},
	}

	for _, tt := range tests {
		config := NewConfig()
		config.ListenPort = tt.port
		addr := config.GetListenAddress()
		if addr != tt.expected {
			t.Errorf("Port %d: got %q, want %q", tt.port, addr, tt.expected)
		}
	}
}

// TestGetAdminAddress tests admin address formatting
func TestGetAdminAddress_Ports(t *testing.T) {
	tests := []struct {
		port     int
		expected string
	}{
		{8081, ":8081"},
		{9001, ":9001"},
	}

	for _, tt := range tests {
		config := NewConfig()
		config.AdminPort = tt.port
		addr := config.GetAdminAddress()
		if addr != tt.expected {
			t.Errorf("Port %d: got %q, want %q", tt.port, addr, tt.expected)
		}
	}
}

// TestIsAccelerationEnabled tests acceleration flag checking
func TestIsAccelerationEnabled_Flags(t *testing.T) {
	tests := []struct {
		name    string
		ebpf    bool
		xdp     bool
		dpdk    bool
		enabled bool
	}{
		{"none", false, false, false, false},
		{"ebpf", true, false, false, true},
		{"xdp", false, true, false, true},
		{"dpdk", false, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig()
			config.EnableEBPF = tt.ebpf
			config.EnableXDP = tt.xdp
			config.EnableDPDK = tt.dpdk
			result := config.IsAccelerationEnabled()
			if result != tt.enabled {
				t.Errorf("IsAccelerationEnabled: got %v, want %v", result, tt.enabled)
			}
		})
	}
}

// TestIsTLSEnabled tests TLS detection
func TestIsTLSEnabled_Combinations(t *testing.T) {
	tests := []struct {
		name  string
		cert  string
		key   string
		valid bool
	}{
		{"none", "", "", false},
		{"certOnly", "/path/cert.pem", "", false},
		{"both", "/path/cert.pem", "/path/key.pem", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig()
			config.TLSCertPath = tt.cert
			config.TLSKeyPath = tt.key
			result := config.IsTLSEnabled()
			if result != tt.valid {
				t.Errorf("IsTLSEnabled: got %v, want %v", result, tt.valid)
			}
		})
	}
}

// TestIsMTLSEnabled tests mTLS detection
func TestIsMTLSEnabled_Combinations(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		certPath string
		keyPath  string
		caPath   string
		valid    bool
	}{
		{"disabled", false, "", "", "", false},
		{"enabled_nopath", true, "", "", "", false},
		{"enabled_fullpath", true, "/cert", "/key", "/ca", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig()
			config.EnableMTLS = tt.enabled
			config.MTLSServerCertPath = tt.certPath
			config.MTLSServerKeyPath = tt.keyPath
			config.MTLSClientCAPath = tt.caPath
			result := config.IsMTLSEnabled()
			if result != tt.valid {
				t.Errorf("IsMTLSEnabled: got %v, want %v", result, tt.valid)
			}
		})
	}
}

// TestGetMTLSConfig tests mTLS config retrieval
func TestGetMTLSConfig_Paths(t *testing.T) {
	config := NewConfig()
	config.MTLSServerCertPath = "/path/server.crt"
	config.MTLSServerKeyPath = "/path/server.key"
	config.MTLSClientCAPath = "/path/ca.crt"

	cert, key, ca := config.GetMTLSConfig()
	if cert != "/path/server.crt" {
		t.Errorf("ServerCert: got %q, want %q", cert, "/path/server.crt")
	}
	if key != "/path/server.key" {
		t.Errorf("ServerKey: got %q, want %q", key, "/path/server.key")
	}
	if ca != "/path/ca.crt" {
		t.Errorf("ClientCA: got %q, want %q", ca, "/path/ca.crt")
	}
}

// TestLoadFromFile_NonExistent tests error handling
func TestLoadFromFile_NonExistent(t *testing.T) {
	config := NewConfig()
	err := config.LoadFromFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("LoadFromFile: expected error for nonexistent file")
	}
}

// BenchmarkLoadFromFile benchmarks file loading
func BenchmarkLoadFromFile(b *testing.B) {
	configContent := `
manager_url: "http://manager:8000"
cluster_api_key: "test-key"
proxy_name: "test-proxy"
listen_port: 8080
`

	tmpFile, err := os.CreateTemp("", "bench_config_*.yaml")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		b.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := NewConfig()
		config.LoadFromFile(tmpFile.Name())
	}
}

// BenchmarkValidateConfig benchmarks validation
func BenchmarkValidateConfig(b *testing.B) {
	config := &Config{
		ManagerURL:    "http://manager:8000",
		ClusterAPIKey: "test-key",
		ProxyName:     "test-proxy",
		ListenPort:    8080,
		AdminPort:     8081,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}
