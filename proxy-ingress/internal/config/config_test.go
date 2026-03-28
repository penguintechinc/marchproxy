package config_test

import (
	"os"
	"testing"

	"marchproxy-ingress/internal/config"
)

// setRequiredEnvVars sets the minimum env vars to pass validation.
func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("PROXY_MANAGER_API_KEY", "test-api-key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
}

func TestDefaultConfig(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.ProxyType != "ingress" {
		t.Errorf("expected ProxyType 'ingress', got %q", cfg.ProxyType)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host '0.0.0.0', got %q", cfg.Host)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info', got %q", cfg.LogLevel)
	}
}

func TestConfigPortDefaults(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		got      int
		expected int
	}{
		{"Port", cfg.Port, 80},
		{"TLSPort", cfg.TLSPort, 443},
		{"MetricsPort", cfg.MetricsPort, 8082},
		{"HealthPort", cfg.HealthPort, 8083},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.got)
			}
		})
	}
}

func TestConfigManagerStruct(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("MANAGER_URL", "http://manager:9000")
	t.Setenv("CLUSTER_ID", "cluster-abc")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Manager.URL != "http://manager:9000" {
		t.Errorf("expected Manager.URL 'http://manager:9000', got %q", cfg.Manager.URL)
	}
	if cfg.Manager.ClusterID != "cluster-abc" {
		t.Errorf("expected Manager.ClusterID 'cluster-abc', got %q", cfg.Manager.ClusterID)
	}
	if cfg.Manager.RetryCount != 3 {
		t.Errorf("expected Manager.RetryCount 3, got %d", cfg.Manager.RetryCount)
	}
	if cfg.Manager.Timeout != 30 {
		t.Errorf("expected Manager.Timeout 30, got %d", cfg.Manager.Timeout)
	}
}

func TestConfigManagerTimeout(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	d := cfg.GetManagerTimeout()
	if d.Seconds() != 30 {
		t.Errorf("expected GetManagerTimeout() 30s, got %v", d)
	}
}

func TestConfigAPIKeyRequired(t *testing.T) {
	// Ensure no API key is set.
	os.Unsetenv("PROXY_MANAGER_API_KEY")
	os.Unsetenv("CLUSTER_API_KEY")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error when cluster API key is missing, got nil")
	}
}

func TestConfigInvalidPort(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_MTLS_ENABLED", "false")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_PORT", "0")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid port 0, got nil")
	}
}

func TestConfigEBPFDefaults(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if !cfg.EnableEBPF {
		t.Error("expected EnableEBPF to be true by default")
	}
	if cfg.EnableXDP {
		t.Error("expected EnableXDP to be false by default")
	}
	if cfg.XDPInterface != "eth0" {
		t.Errorf("expected XDPInterface 'eth0', got %q", cfg.XDPInterface)
	}
}

func TestConfigGetTLSConfigDisabled(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	// mTLS disabled, GetTLSConfig should return nil, nil
	tlsCfg, err := cfg.GetTLSConfig()
	if err != nil {
		t.Errorf("expected no error when mTLS disabled, got %v", err)
	}
	if tlsCfg != nil {
		t.Error("expected nil TLS config when mTLS is disabled")
	}
}

func TestConfigMTLSValidationRequiresPaths(t *testing.T) {
	t.Setenv("PROXY_MANAGER_API_KEY", "key")
	t.Setenv("PROXY_LOAD_BALANCING_ALGORITHM", "round_robin")
	t.Setenv("PROXY_MTLS_ENABLED", "true")
	t.Setenv("PROXY_MTLS_SERVER_CERT_PATH", "")
	t.Setenv("MTLS_SERVER_CERT_PATH", "")
	t.Setenv("PROXY_MTLS_SERVER_KEY_PATH", "")
	t.Setenv("MTLS_SERVER_KEY_PATH", "")
	t.Setenv("PROXY_MTLS_CLIENT_CA_PATH", "")
	t.Setenv("MTLS_CLIENT_CA_PATH", "")

	_, err := config.LoadConfig()
	if err == nil {
		t.Error("expected error when mTLS enabled but cert paths are empty")
	}
}
