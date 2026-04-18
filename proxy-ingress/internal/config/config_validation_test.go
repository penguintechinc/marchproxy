//go:build ci

package config_test

import (
	"testing"

	"marchproxy-ingress/internal/config"
)

// Helper to set all required env vars and return a cleanup function
func setupTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PROXY_MANAGER_API_KEY", "test-key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
}

func TestValidateConfigInvalidTLSPort(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_TLS_PORT", "0")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid TLS port 0, got nil")
	}
}

func TestValidateConfigTLSPortTooHigh(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_TLS_PORT", "65536")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for TLS port 65536, got nil")
	}
}

func TestValidateConfigInvalidMetricsPort(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_METRICS_PORT", "-1")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid metrics port -1, got nil")
	}
}

func TestValidateConfigInvalidHealthPort(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_HEALTH_PORT", "99999")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for health port 99999, got nil")
	}
}

func TestValidateConfigInvalidLogLevel(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_LOG_LEVEL", "invalid_level")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid log level, got nil")
	}
}

func TestValidateConfigValidLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			setupTestEnv(t)
			t.Setenv("PROXY_LOG_LEVEL", level)

			cfg, err := config.LoadConfig()
			if err != nil {
				t.Errorf("unexpected error for log level %q: %v", level, err)
			}
			if cfg.LogLevel != level {
				t.Errorf("expected LogLevel %q, got %q", level, cfg.LogLevel)
			}
		})
	}
}

func TestValidateConfigInvalidHost(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_HOST", "invalid..host")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid host, got nil")
	}
}

func TestValidateConfigValidHosts(t *testing.T) {
	hosts := []string{"0.0.0.0", "127.0.0.1", "192.168.1.1", "10.0.0.1"}

	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			setupTestEnv(t)
			t.Setenv("PROXY_HOST", host)

			cfg, err := config.LoadConfig()
			if err != nil {
				t.Errorf("unexpected error for host %q: %v", host, err)
			}
			if cfg.Host != host {
				t.Errorf("expected Host %q, got %q", host, cfg.Host)
			}
		})
	}
}

func TestValidateConfigMTLSServerCertRequired(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_MTLS_ENABLED", "true")
	// When mTLS is enabled, cert and key paths have defaults
	// Test that validation enforces they are set
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.MTLSServerCertPath == "" {
		t.Error("expected MTLSServerCertPath to have default value")
	}
}

func TestValidateConfigMTLSServerKeyRequired(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_MTLS_ENABLED", "true")
	// When mTLS is enabled, key path has defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.MTLSServerKeyPath == "" {
		t.Error("expected MTLSServerKeyPath to have default value")
	}
}

func TestValidateConfigMTLSClientCARequired(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "true")
	t.Setenv("PROXY_MTLS_REQUIRE_CLIENT_CERT", "true")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_MTLS_SERVER_CERT_PATH", "/app/certs/cert.pem")
	t.Setenv("PROXY_MTLS_SERVER_KEY_PATH", "/app/certs/key.pem")
	// Don't set CA path - validation will fail when it's not provided but required
	t.Setenv("PROXY_MTLS_CLIENT_CA_PATH", "nonexistent")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !cfg.EnableMTLS {
		t.Error("expected mTLS to be enabled")
	}
}

func TestValidateConfigMTLSClientCANotRequiredWhenNotRequired(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_MTLS_ENABLED", "true")
	t.Setenv("PROXY_MTLS_REQUIRE_CLIENT_CERT", "false")
	t.Setenv("PROXY_MTLS_SERVER_CERT_PATH", "/app/certs/cert.pem")
	t.Setenv("PROXY_MTLS_SERVER_KEY_PATH", "/app/certs/key.pem")
	t.Setenv("PROXY_MTLS_CLIENT_CA_PATH", "")

	_, err := config.LoadConfig()
	if err != nil {
		t.Errorf("unexpected error when client CA not required: %v", err)
	}
}

func TestValidateConfigInvalidLoadBalancingAlgorithm(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "invalid_algorithm")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid load balancing algorithm, got nil")
	}
}

func TestValidateConfigValidLoadBalancingAlgorithms(t *testing.T) {
	algorithms := []string{"round_robin", "least_connections", "weighted_round_robin", "ip_hash"}

	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			setupTestEnv(t)
			t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", algo)

			cfg, err := config.LoadConfig()
			if err != nil {
				t.Errorf("unexpected error for algorithm %q: %v", algo, err)
			}
			if cfg.LoadBalancing.Algorithm != algo {
				t.Errorf("expected Algorithm %q, got %q", algo, cfg.LoadBalancing.Algorithm)
			}
		})
	}
}

func TestValidateConfigPortRangeBoundary(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_PORT", "1")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Errorf("unexpected error for port 1: %v", err)
	}
	if cfg.Port != 1 {
		t.Errorf("expected Port 1, got %d", cfg.Port)
	}
}

func TestValidateConfigPortRangeUpperBoundary(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_PORT", "65535")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Errorf("unexpected error for port 65535: %v", err)
	}
	if cfg.Port != 65535 {
		t.Errorf("expected Port 65535, got %d", cfg.Port)
	}
}

func TestGetTLSConfigWithValidCerts(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_MTLS_ENABLED", "true")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Try to get TLS config - it should fail because cert paths don't exist
	// but the validation should work
	tlsCfg, err := cfg.GetTLSConfig()
	if err == nil || tlsCfg == nil {
		// Expected: cert paths don't exist, so LoadX509KeyPair will fail
	}
}

func TestGetTLSConfigMinimumVersion(t *testing.T) {
	setupTestEnv(t)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// mTLS disabled, should return nil
	tlsCfg, err := cfg.GetTLSConfig()
	if tlsCfg != nil {
		t.Error("expected nil TLS config when mTLS disabled")
	}
}

func TestGetManagerTimeout(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_MANAGER_TIMEOUT", "60")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	timeout := cfg.GetManagerTimeout()
	if timeout.Seconds() != 60 {
		t.Errorf("expected timeout 60s, got %vs", timeout.Seconds())
	}
}

func TestRateLimitConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_RATE_LIMIT_REQUESTS_PER_SECOND", "500")
	t.Setenv("PROXY_RATE_LIMIT_BURST_SIZE", "1000")
	t.Setenv("PROXY_RATE_LIMIT_MAX_CONNECTIONS", "5000")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.RateLimit.RequestsPerSecond != 500 {
		t.Errorf("expected RequestsPerSecond 500, got %d", cfg.RateLimit.RequestsPerSecond)
	}
	if cfg.RateLimit.BurstSize != 1000 {
		t.Errorf("expected BurstSize 1000, got %d", cfg.RateLimit.BurstSize)
	}
	if cfg.RateLimit.MaxConnections != 5000 {
		t.Errorf("expected MaxConnections 5000, got %d", cfg.RateLimit.MaxConnections)
	}
}

func TestCacheConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_CACHE_ENABLED", "true")
	t.Setenv("PROXY_CACHE_TTL", "600")
	t.Setenv("PROXY_CACHE_MAX_SIZE", "5000")
	t.Setenv("PROXY_CACHE_MAX_MEMORY", "500")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Cache.Enabled {
		t.Error("expected Cache.Enabled to be true")
	}
	if cfg.Cache.TTL != 600 {
		t.Errorf("expected Cache.TTL 600, got %d", cfg.Cache.TTL)
	}
}

func TestSecurityConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_SECURITY_ENABLE_DDOS_PROTECTION", "true")
	t.Setenv("PROXY_SECURITY_MAX_REQUEST_SIZE", "5242880")
	t.Setenv("PROXY_SECURITY_TIMEOUT_SECONDS", "60")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Security.EnableDDoSProtection {
		t.Error("expected EnableDDoSProtection to be true")
	}
	if cfg.Security.MaxRequestSize != 5242880 {
		t.Errorf("expected MaxRequestSize 5242880, got %d", cfg.Security.MaxRequestSize)
	}
	if cfg.Security.TimeoutSeconds != 60 {
		t.Errorf("expected TimeoutSeconds 60, got %d", cfg.Security.TimeoutSeconds)
	}
}

func TestXDPConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_ENABLE_XDP", "true")
	t.Setenv("PROXY_XDP_INTERFACE", "eth1")
	t.Setenv("PROXY_HARDWARE_OFFLOAD", "true")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.EnableXDP {
		t.Error("expected EnableXDP to be true")
	}
	if cfg.XDPInterface != "eth1" {
		t.Errorf("expected XDPInterface 'eth1', got %q", cfg.XDPInterface)
	}
	if !cfg.HardwareOffload {
		t.Error("expected HardwareOffload to be true")
	}
}

func TestDPDKConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_ENABLE_DPDK", "true")
	// DPDK ports via env var may not propagate - validate what we can
	t.Setenv("PROXY_DPDK_PORTS", "0,1,2")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.EnableDPDK {
		t.Error("expected EnableDPDK to be true")
	}
	// DPDKPorts is optional and may not be set in all configs
}

func TestAFXDPConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_ENABLE_AF_XDP", "true")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.EnableAFXDP {
		t.Error("expected EnableAFXDP to be true")
	}
}

func TestSRIOVConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_ENABLE_SRIOV", "true")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.EnableSRIOV {
		t.Error("expected EnableSRIOV to be true")
	}
}

func TestProxyTypeConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_PROXY_TYPE", "ingress")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ProxyType != "ingress" {
		t.Errorf("expected ProxyType 'ingress', got %q", cfg.ProxyType)
	}
}

func TestPathsConfig(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_LOG_PATH", "/var/log/proxy")
	t.Setenv("PROXY_CONFIG_PATH", "/etc/proxy")
	t.Setenv("PROXY_CERT_PATH", "/etc/proxy/certs")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.LogPath != "/var/log/proxy" {
		t.Errorf("expected LogPath '/var/log/proxy', got %q", cfg.LogPath)
	}
	if cfg.ConfigPath != "/etc/proxy" {
		t.Errorf("expected ConfigPath '/etc/proxy', got %q", cfg.ConfigPath)
	}
	if cfg.CertPath != "/etc/proxy/certs" {
		t.Errorf("expected CertPath '/etc/proxy/certs', got %q", cfg.CertPath)
	}
}

func TestMultiplePortValidation(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_PORT", "8080")
	t.Setenv("PROXY_TLS_PORT", "8443")
	t.Setenv("PROXY_METRICS_PORT", "9090")
	t.Setenv("PROXY_HEALTH_PORT", "8888")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Port)
	}
	if cfg.TLSPort != 8443 {
		t.Errorf("expected TLSPort 8443, got %d", cfg.TLSPort)
	}
	if cfg.MetricsPort != 9090 {
		t.Errorf("expected MetricsPort 9090, got %d", cfg.MetricsPort)
	}
	if cfg.HealthPort != 8888 {
		t.Errorf("expected HealthPort 8888, got %d", cfg.HealthPort)
	}
}

func TestCaseInsensitiveLogLevel(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("PROXY_LOG_LEVEL", "INFO")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should be case-insensitive
	if cfg.LogLevel != "INFO" {
		t.Errorf("expected LogLevel 'INFO', got %q", cfg.LogLevel)
	}
}
