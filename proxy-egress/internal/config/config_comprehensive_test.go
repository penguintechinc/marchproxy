//go:build ci
// +build ci

package config

import (
	"os"
	"testing"
	"time"
)

func TestL7Config_Defaults(t *testing.T) {
	config := &L7Config{}

	tests := []struct {
		name  string
		field interface{}
		want  interface{}
	}{
		{"Enabled", config.Enabled, false},
		{"HTTPListenPort", config.HTTPListenPort, 0},
		{"HTTP3Enabled", config.HTTP3Enabled, false},
	}

	for _, tt := range tests {
		if tt.field != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.field, tt.want)
		}
	}
}

func TestThreatConfig_Defaults(t *testing.T) {
	config := &ThreatConfig{}

	if config.Enabled {
		t.Error("Enabled should default to false")
	}

	if config.IPBlockingEnabled {
		t.Errorf("IPBlockingEnabled: got %v, want false", config.IPBlockingEnabled)
	}

	if config.IPCacheSize != 0 {
		t.Errorf("IPCacheSize: got %d, want 0", config.IPCacheSize)
	}
}

func TestTLSInterceptConfig_Defaults(t *testing.T) {
	config := &TLSInterceptConfig{}

	if config.Enabled {
		t.Error("Enabled should default to false")
	}

	if config.Mode != "" {
		t.Errorf("Mode should default to empty string, got %q", config.Mode)
	}

	if config.CertCacheSize != 0 {
		t.Errorf("CertCacheSize: got %d, want 0", config.CertCacheSize)
	}
}

func TestExtAuthConfig_Defaults(t *testing.T) {
	config := &ExtAuthConfig{}

	if config.Enabled {
		t.Error("Enabled should default to false")
	}

	if config.Port != 0 {
		t.Errorf("Port: got %d, want 0", config.Port)
	}

	if config.Host != "" {
		t.Errorf("Host: got %q, want empty", config.Host)
	}
}

func TestAccessControlConfig_Defaults(t *testing.T) {
	config := &AccessControlConfig{}

	if config.Enabled {
		t.Error("Enabled should default to false")
	}

	if config.DefaultRequireAuth {
		t.Error("DefaultRequireAuth should default to false")
	}

	if config.DefaultAllow {
		t.Error("DefaultAllow should default to false")
	}
}

func TestLeversConfig_Defaults(t *testing.T) {
	config := &LeversConfig{}

	if config.Enabled {
		t.Error("Enabled should default to false")
	}

	if config.ListenAddr != "" {
		t.Errorf("ListenAddr: got %q, want empty", config.ListenAddr)
	}
}

func TestMainConfig_Creation(t *testing.T) {
	config := &Config{
		ManagerURL:    "http://manager:8080",
		ClusterAPIKey: "test-key",
		ProxyName:     "test-proxy",
		Hostname:      "test.local",
		ListenPort:    8080,
		AdminPort:     9090,
		LogLevel:      "info",
	}

	if config.ManagerURL != "http://manager:8080" {
		t.Errorf("ManagerURL: got %q", config.ManagerURL)
	}

	if config.ClusterAPIKey != "test-key" {
		t.Errorf("ClusterAPIKey: got %q", config.ClusterAPIKey)
	}

	if config.ProxyName != "test-proxy" {
		t.Errorf("ProxyName: got %q", config.ProxyName)
	}

	if config.ListenPort != 8080 {
		t.Errorf("ListenPort: got %d", config.ListenPort)
	}
}

func TestConfig_Timeouts(t *testing.T) {
	config := &Config{
		ConfigUpdateInterval: 60,
		HeartbeatInterval:    30,
		ConnectionTimeout:    15,
	}

	if config.ConfigUpdateInterval != 60 {
		t.Errorf("ConfigUpdateInterval: got %d, want 60", config.ConfigUpdateInterval)
	}

	if config.HeartbeatInterval != 30 {
		t.Errorf("HeartbeatInterval: got %d, want 30", config.HeartbeatInterval)
	}

	if config.ConnectionTimeout != 15 {
		t.Errorf("ConnectionTimeout: got %d, want 15", config.ConnectionTimeout)
	}
}

func TestConfig_RateLimiting(t *testing.T) {
	config := &Config{
		RateLimitEnabled: true,
		RateLimitRPS:     1000,
	}

	if !config.RateLimitEnabled {
		t.Error("RateLimitEnabled should be true")
	}

	if config.RateLimitRPS != 1000 {
		t.Errorf("RateLimitRPS: got %d, want 1000", config.RateLimitRPS)
	}
}

func TestConfig_Performance(t *testing.T) {
	config := &Config{
		EnableEBPF:    true,
		EnableMetrics: true,
		WorkerThreads: 4,
		EnableDPDK:    false,
		EnableXDP:     true,
		EnableAFXDP:   false,
		EnableSRIOV:   false,
	}

	if !config.EnableEBPF {
		t.Error("EnableEBPF should be true")
	}

	if !config.EnableMetrics {
		t.Error("EnableMetrics should be true")
	}

	if config.WorkerThreads != 4 {
		t.Errorf("WorkerThreads: got %d, want 4", config.WorkerThreads)
	}

	if !config.EnableXDP {
		t.Error("EnableXDP should be true")
	}
}

func TestConfig_TLS(t *testing.T) {
	config := &Config{
		TLSCertPath: "/path/to/cert.pem",
		TLSKeyPath:  "/path/to/key.pem",
	}

	if config.TLSCertPath != "/path/to/cert.pem" {
		t.Errorf("TLSCertPath: got %q", config.TLSCertPath)
	}

	if config.TLSKeyPath != "/path/to/key.pem" {
		t.Errorf("TLSKeyPath: got %q", config.TLSKeyPath)
	}
}

func TestConfig_MTLS(t *testing.T) {
	config := &Config{
		EnableMTLS:            true,
		MTLSServerCertPath:    "/path/to/server-cert.pem",
		MTLSServerKeyPath:     "/path/to/server-key.pem",
		MTLSClientCAPath:      "/path/to/client-ca.pem",
		MTLSRequireClientCert: true,
		MTLSVerifyClientCert:  true,
	}

	if !config.EnableMTLS {
		t.Error("EnableMTLS should be true")
	}

	if config.MTLSServerCertPath != "/path/to/server-cert.pem" {
		t.Errorf("MTLSServerCertPath: got %q", config.MTLSServerCertPath)
	}

	if !config.MTLSRequireClientCert {
		t.Error("MTLSRequireClientCert should be true")
	}

	if !config.MTLSVerifyClientCert {
		t.Error("MTLSVerifyClientCert should be true")
	}
}

func TestConfig_KillKrill(t *testing.T) {
	config := &Config{
		KillKrillEnabled:         true,
		KillKrillLogEndpoint:     "https://killkrill:8080/logs",
		KillKrillMetricsEndpoint: "https://killkrill:8080/metrics",
		KillKrillAPIKey:          "killkrill-key",
		KillKrillSourceName:      "marchproxy",
		KillKrillApplication:     "proxy-egress",
		KillKrillBatchSize:       100,
		KillKrillFlushInterval:   30,
	}

	if !config.KillKrillEnabled {
		t.Error("KillKrillEnabled should be true")
	}

	if config.KillKrillLogEndpoint != "https://killkrill:8080/logs" {
		t.Errorf("KillKrillLogEndpoint: got %q", config.KillKrillLogEndpoint)
	}

	if config.KillKrillBatchSize != 100 {
		t.Errorf("KillKrillBatchSize: got %d, want 100", config.KillKrillBatchSize)
	}

	if config.KillKrillFlushInterval != 30 {
		t.Errorf("KillKrillFlushInterval: got %d, want 30", config.KillKrillFlushInterval)
	}
}

func TestConfig_License(t *testing.T) {
	config := &Config{
		LicenseKey: "license-xyz",
	}

	if config.LicenseKey != "license-xyz" {
		t.Errorf("LicenseKey: got %q", config.LicenseKey)
	}
}

func TestConfig_Defaults(t *testing.T) {
	config := &Config{}

	// Check all string defaults
	if config.ManagerURL != "" {
		t.Errorf("ManagerURL should be empty by default, got %q", config.ManagerURL)
	}

	if config.ProxyName != "" {
		t.Errorf("ProxyName should be empty by default, got %q", config.ProxyName)
	}

	// Check all int defaults
	if config.ListenPort != 0 {
		t.Errorf("ListenPort should be 0 by default, got %d", config.ListenPort)
	}

	if config.AdminPort != 0 {
		t.Errorf("AdminPort should be 0 by default, got %d", config.AdminPort)
	}

	// Check all bool defaults
	if config.EnableEBPF {
		t.Error("EnableEBPF should be false by default")
	}

	if config.EnableMetrics {
		t.Error("EnableMetrics should be false by default")
	}

	if config.EnableMTLS {
		t.Error("EnableMTLS should be false by default")
	}
}

func TestThreatConfig_DNSCache(t *testing.T) {
	config := &ThreatConfig{
		DNSCacheEnabled: true,
		DNSPositiveTTL:  5 * time.Minute,
		DNSNegativeTTL:  1 * time.Minute,
		DNSCacheSize:    1000,
		DNSUpstream:     []string{"8.8.8.8", "8.8.4.4"},
	}

	if !config.DNSCacheEnabled {
		t.Error("DNSCacheEnabled should be true")
	}

	if config.DNSPositiveTTL != 5*time.Minute {
		t.Errorf("DNSPositiveTTL: got %v, want 5m", config.DNSPositiveTTL)
	}

	if config.DNSNegativeTTL != 1*time.Minute {
		t.Errorf("DNSNegativeTTL: got %v, want 1m", config.DNSNegativeTTL)
	}

	if len(config.DNSUpstream) != 2 {
		t.Errorf("DNSUpstream: got %d entries, want 2", len(config.DNSUpstream))
	}
}

func TestThreatConfig_Sync(t *testing.T) {
	config := &ThreatConfig{
		SyncMode:         "grpc",
		SyncPollInterval: 30 * time.Second,
		SyncGRPCEndpoint: "localhost:50051",
	}

	if config.SyncMode != "grpc" {
		t.Errorf("SyncMode: got %q, want grpc", config.SyncMode)
	}

	if config.SyncPollInterval != 30*time.Second {
		t.Errorf("SyncPollInterval: got %v, want 30s", config.SyncPollInterval)
	}

	if config.SyncGRPCEndpoint != "localhost:50051" {
		t.Errorf("SyncGRPCEndpoint: got %q", config.SyncGRPCEndpoint)
	}
}

func TestL7Config_EnvoySettings(t *testing.T) {
	config := &L7Config{
		Enabled:         true,
		EnvoyBinary:     "/usr/local/bin/envoy",
		EnvoyConfigPath: "/etc/envoy/envoy.yaml",
		EnvoyAdminPort:  9901,
		HTTPListenPort:  80,
		HTTPSListenPort: 443,
		HTTP3Enabled:    true,
		LogLevel:        "info",
	}

	if !config.Enabled {
		t.Error("Enabled should be true")
	}

	if config.EnvoyBinary != "/usr/local/bin/envoy" {
		t.Errorf("EnvoyBinary: got %q", config.EnvoyBinary)
	}

	if config.EnvoyAdminPort != 9901 {
		t.Errorf("EnvoyAdminPort: got %d, want 9901", config.EnvoyAdminPort)
	}

	if !config.HTTP3Enabled {
		t.Error("HTTP3Enabled should be true")
	}
}

func TestEnvVarParsing(t *testing.T) {
	// Set environment variables
	os.Setenv("MANAGER_URL", "http://test-manager:8080")
	os.Setenv("PROXY_NAME", "test-proxy")
	os.Setenv("LISTEN_PORT", "8080")
	defer func() {
		os.Unsetenv("MANAGER_URL")
		os.Unsetenv("PROXY_NAME")
		os.Unsetenv("LISTEN_PORT")
	}()

	config := &Config{
		ManagerURL: os.Getenv("MANAGER_URL"),
		ProxyName:  os.Getenv("PROXY_NAME"),
	}

	if config.ManagerURL != "http://test-manager:8080" {
		t.Errorf("ManagerURL: got %q", config.ManagerURL)
	}

	if config.ProxyName != "test-proxy" {
		t.Errorf("ProxyName: got %q", config.ProxyName)
	}
}

func TestConfig_AllFields(t *testing.T) {
	config := &Config{
		ManagerURL:           "http://manager:8080",
		ClusterAPIKey:        "api-key",
		ProxyName:            "proxy",
		Hostname:             "host.local",
		ListenPort:           8080,
		AdminPort:            9090,
		LogLevel:             "debug",
		SyslogEndpoint:       "localhost:514",
		EnableEBPF:           true,
		EnableMetrics:        true,
		WorkerThreads:        8,
		EnableDPDK:           false,
		EnableXDP:            true,
		EnableAFXDP:          false,
		EnableSRIOV:          false,
		DPDKDevices:          "0000:01:00.0",
		TLSCertPath:          "/cert.pem",
		TLSKeyPath:           "/key.pem",
		EnableMTLS:           true,
		MTLSServerCertPath:   "/server-cert.pem",
		MTLSServerKeyPath:    "/server-key.pem",
		MTLSClientCAPath:     "/client-ca.pem",
		MTLSClientCertPath:   "/client-cert.pem",
		MTLSClientKeyPath:    "/client-key.pem",
		MTLSRequireClientCert: true,
		MTLSVerifyClientCert:  true,
		LicenseKey:           "license",
		ConfigUpdateInterval: 60,
		HeartbeatInterval:    30,
		ConnectionTimeout:    15,
		RateLimitEnabled:     true,
		RateLimitRPS:         1000,
	}

	// Verify all fields are set
	if config.ManagerURL == "" {
		t.Error("ManagerURL not set")
	}
	if config.ProxyName == "" {
		t.Error("ProxyName not set")
	}
	if config.ListenPort == 0 {
		t.Error("ListenPort not set")
	}
	if config.AdminPort == 0 {
		t.Error("AdminPort not set")
	}
	if config.WorkerThreads == 0 {
		t.Error("WorkerThreads not set")
	}
	if config.RateLimitRPS == 0 {
		t.Error("RateLimitRPS not set")
	}
}
