package config_test

import (
	"os"
	"testing"
	"time"

	"marchproxy-dblb/internal/config"
)

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

	if cfg.GRPCAddr != "0.0.0.0" {
		t.Errorf("expected GRPCAddr '0.0.0.0', got %q", cfg.GRPCAddr)
	}
	if cfg.GRPCPort != 50052 {
		t.Errorf("expected GRPCPort 50052, got %d", cfg.GRPCPort)
	}
	if cfg.MetricsAddr != ":7002" {
		t.Errorf("expected MetricsAddr ':7002', got %q", cfg.MetricsAddr)
	}
	if cfg.ManagerURL != "http://api-server:8000" {
		t.Errorf("expected ManagerURL 'http://api-server:8000', got %q", cfg.ManagerURL)
	}
}

func TestLoadConfigConnectionPoolingDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.MaxConnectionsPerRoute != 100 {
		t.Errorf("expected MaxConnectionsPerRoute 100, got %d", cfg.MaxConnectionsPerRoute)
	}
	if cfg.ConnectionIdleTimeout != 5*time.Minute {
		t.Errorf("expected ConnectionIdleTimeout 5m, got %v", cfg.ConnectionIdleTimeout)
	}
	if cfg.ConnectionMaxLifetime != 30*time.Minute {
		t.Errorf("expected ConnectionMaxLifetime 30m, got %v", cfg.ConnectionMaxLifetime)
	}
}

func TestLoadConfigRateLimitingDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.EnableRateLimiting {
		t.Error("expected EnableRateLimiting to be true by default")
	}
	if cfg.DefaultConnectionRate != 100.0 {
		t.Errorf("expected DefaultConnectionRate 100.0, got %f", cfg.DefaultConnectionRate)
	}
	if cfg.DefaultQueryRate != 1000.0 {
		t.Errorf("expected DefaultQueryRate 1000.0, got %f", cfg.DefaultQueryRate)
	}
}

func TestLoadConfigSecurityDefaults(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.EnableSQLInjectionDetection {
		t.Error("expected EnableSQLInjectionDetection to be true by default")
	}
	if !cfg.BlockSuspiciousQueries {
		t.Error("expected BlockSuspiciousQueries to be true by default")
	}
}

func TestLoadConfigLicensingDefaults(t *testing.T) {
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

func TestLoadConfigWithoutAPIKey(t *testing.T) {
	os.Unsetenv("CLUSTER_API_KEY")

	// Load should succeed, ClusterAPIKey will just be empty
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.ClusterAPIKey != "" {
		t.Errorf("expected empty ClusterAPIKey, got %s", cfg.ClusterAPIKey)
	}
}

func TestConfigValidateInvalidGRPCPort(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                0,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
		EnableSQLInjectionDetection: true,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid GRPCPort 0")
	}
}

func TestConfigValidateInvalidConnectionPooling(t *testing.T) {
	tests := []struct {
		name       string
		maxConns   int
		idleTime   time.Duration
		maxLifeTime time.Duration
		shouldFail bool
	}{
		{"negative max connections", -1, 5*time.Minute, 30*time.Minute, true},
		{"zero max connections", 0, 5*time.Minute, 30*time.Minute, true},
		{"negative idle timeout", 100, -1*time.Second, 30*time.Minute, true},
		{"negative max lifetime", 100, 5*time.Minute, -1*time.Second, true},
		{"valid", 100, 5*time.Minute, 30*time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ManagerURL:              "http://api-server:8000",
				ClusterAPIKey:           "key",
				GRPCPort:                50052,
				MaxConnectionsPerRoute:  tt.maxConns,
				ConnectionIdleTimeout:   tt.idleTime,
				ConnectionMaxLifetime:   tt.maxLifeTime,
				EnableRateLimiting:      true,
				DefaultConnectionRate:   100.0,
				DefaultQueryRate:        1000.0,
			}

			err := cfg.Validate()
			if tt.shouldFail && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.shouldFail && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigValidateRateLimiting(t *testing.T) {
	tests := []struct {
		name           string
		connRate       float64
		queryRate      float64
		shouldFail     bool
	}{
		{"negative conn rate", -1.0, 1000.0, true},
		{"zero conn rate", 0.0, 1000.0, true},
		{"negative query rate", 100.0, -1.0, true},
		{"zero query rate", 100.0, 0.0, true},
		{"valid", 100.0, 1000.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ManagerURL:              "http://api-server:8000",
				ClusterAPIKey:           "key",
				GRPCPort:                50052,
				MaxConnectionsPerRoute:  100,
				ConnectionIdleTimeout:   5 * time.Minute,
				ConnectionMaxLifetime:   30 * time.Minute,
				EnableRateLimiting:      true,
				DefaultConnectionRate:   tt.connRate,
				DefaultQueryRate:        tt.queryRate,
			}

			err := cfg.Validate()
			if tt.shouldFail && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.shouldFail && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigValidateRateLimitingDisabled(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      false,
		DefaultConnectionRate:   0.0,
		DefaultQueryRate:        0.0,
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigIsEnterpriseFeatureEnabledDevMode(t *testing.T) {
	withAPIKey(t)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.IsEnterpriseFeatureEnabled("any-feature") {
		t.Error("expected enterprise feature enabled in dev mode")
	}
}

func TestConfigIsEnterpriseFeatureEnabledReleaseMode(t *testing.T) {
	cfg := &config.Config{
		ReleaseMode: true,
		LicenseKey:  "",
	}

	if cfg.IsEnterpriseFeatureEnabled("advanced-security") {
		t.Error("expected enterprise feature disabled in release mode without license")
	}

	cfg.LicenseKey = "valid-license"
	if !cfg.IsEnterpriseFeatureEnabled("advanced-security") {
		t.Error("expected enterprise feature enabled in release mode with license")
	}
}

func TestConfigValidateValidConfig(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:              "http://api-server:8000",
		ClusterAPIKey:           "key",
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigValidateMinimalConfig(t *testing.T) {
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestConfigValidateNegativeRates(t *testing.T) {
	tests := []struct {
		name           string
		connRate       float64
		queryRate      float64
		enableRateLimit bool
		shouldFail     bool
	}{
		{"valid rates", 100.0, 1000.0, true, false},
		{"negative conn rate", -1.0, 1000.0, true, true},
		{"negative query rate", 100.0, -1.0, true, true},
		{"zero conn rate", 0.0, 1000.0, true, true},
		{"zero query rate", 100.0, 0.0, true, true},
		{"rate limit disabled", -1.0, -1.0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				GRPCPort:                50052,
				MaxConnectionsPerRoute:  100,
				ConnectionIdleTimeout:   5 * time.Minute,
				ConnectionMaxLifetime:   30 * time.Minute,
				EnableRateLimiting:      tt.enableRateLimit,
				DefaultConnectionRate:   tt.connRate,
				DefaultQueryRate:        tt.queryRate,
			}

			err := cfg.Validate()
			if (err != nil) != tt.shouldFail {
				t.Errorf("Validate() failed=%v, want=%v (err: %v)", err != nil, tt.shouldFail, err)
			}
		})
	}
}

func TestRouteConfigValidateProtocol(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		shouldFail bool
	}{
		{"mysql protocol", "mysql", false},
		{"postgresql protocol", "postgresql", false},
		{"mongodb protocol", "mongodb", false},
		{"redis protocol", "redis", false},
		{"mssql protocol", "mssql", false},
		{"invalid protocol", "oracle", true},
		{"empty protocol", "", true},
		{"wrong case", "MySQL", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &config.RouteConfig{
				Name:        "test",
				Protocol:    tt.protocol,
				ListenPort:  3306,
				BackendHost: "localhost",
				BackendPort: 3306,
			}

			err := route.Validate()
			if (err != nil) != tt.shouldFail {
				t.Errorf("Validate() failed=%v, want=%v (err: %v)", err != nil, tt.shouldFail, err)
			}
		})
	}
}

func TestRouteConfigValidatePorts(t *testing.T) {
	tests := []struct {
		name       string
		listenPort int
		backendPort int
		shouldFail bool
	}{
		{"valid ports", 3306, 3306, false},
		{"zero listen port", 0, 3306, true},
		{"negative listen port", -1, 3306, true},
		{"listen port too high", 65536, 3306, true},
		{"zero backend port", 3306, 0, true},
		{"negative backend port", 3306, -1, true},
		{"backend port too high", 3306, 65536, true},
		{"max valid port", 65535, 65535, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &config.RouteConfig{
				Name:        "test",
				Protocol:    "mysql",
				ListenPort:  tt.listenPort,
				BackendHost: "localhost",
				BackendPort: tt.backendPort,
			}

			err := route.Validate()
			if (err != nil) != tt.shouldFail {
				t.Errorf("Validate() failed=%v, want=%v (err: %v)", err != nil, tt.shouldFail, err)
			}
		})
	}
}

func TestRouteConfigValidateMissingName(t *testing.T) {
	route := &config.RouteConfig{
		Name:        "",
		Protocol:    "mysql",
		ListenPort:  3306,
		BackendHost: "localhost",
		BackendPort: 3306,
	}

	if err := route.Validate(); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestRouteConfigValidateMissingBackendHost(t *testing.T) {
	route := &config.RouteConfig{
		Name:        "test",
		Protocol:    "mysql",
		ListenPort:  3306,
		BackendHost: "",
		BackendPort: 3306,
	}

	if err := route.Validate(); err == nil {
		t.Error("expected error for missing backend host")
	}
}

func TestRouteConfigDefaultValues(t *testing.T) {
	route := &config.RouteConfig{
		Name:        "test",
		Protocol:    "mysql",
		ListenPort:  3306,
		BackendHost: "localhost",
		BackendPort: 3306,
		MaxConnections: 0,
		ConnectionRate: 0,
		QueryRate:      0,
	}

	if err := route.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	// Check defaults were applied
	if route.MaxConnections != 100 {
		t.Errorf("MaxConnections default = %d, want 100", route.MaxConnections)
	}
	if route.ConnectionRate != 100.0 {
		t.Errorf("ConnectionRate default = %f, want 100.0", route.ConnectionRate)
	}
	if route.QueryRate != 1000.0 {
		t.Errorf("QueryRate default = %f, want 1000.0", route.QueryRate)
	}
}

func TestConfigIsEnterpriseFeatureEnabledDevelopment(t *testing.T) {
	cfg := &config.Config{
		ReleaseMode: false,
		LicenseKey:  "",
	}

	// In development mode, all features are enabled
	if !cfg.IsEnterpriseFeatureEnabled("sso") {
		t.Error("SSO should be enabled in development mode")
	}

	if !cfg.IsEnterpriseFeatureEnabled("waddleai") {
		t.Error("WaddleAI should be enabled in development mode")
	}

	if !cfg.IsEnterpriseFeatureEnabled("any_feature") {
		t.Error("Any feature should be enabled in development mode")
	}
}

func TestConfigIsEnterpriseFeatureEnabledProduction(t *testing.T) {
	tests := []struct {
		name       string
		licenseKey string
		want       bool
	}{
		{"with license key", "valid-key-123", true},
		{"without license key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ReleaseMode: true,
				LicenseKey:  tt.licenseKey,
			}

			if enabled := cfg.IsEnterpriseFeatureEnabled("sso"); enabled != tt.want {
				t.Errorf("IsEnterpriseFeatureEnabled(sso) = %v, want %v", enabled, tt.want)
			}
		})
	}
}

func TestConfigMultipleRoutesValidation(t *testing.T) {
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
		Routes: []config.RouteConfig{
			{
				Name:        "mysql",
				Protocol:    "mysql",
				ListenPort:  3306,
				BackendHost: "db1.local",
				BackendPort: 3306,
			},
			{
				Name:        "postgres",
				Protocol:    "postgresql",
				ListenPort:  5432,
				BackendHost: "db2.local",
				BackendPort: 5432,
			},
			{
				Name:        "redis",
				Protocol:    "redis",
				ListenPort:  6379,
				BackendHost: "cache.local",
				BackendPort: 6379,
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestConfigMultipleRoutesWithInvalidRoute(t *testing.T) {
	cfg := &config.Config{
		GRPCPort:                50052,
		MaxConnectionsPerRoute:  100,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableRateLimiting:      true,
		DefaultConnectionRate:   100.0,
		DefaultQueryRate:        1000.0,
		Routes: []config.RouteConfig{
			{
				Name:        "mysql",
				Protocol:    "mysql",
				ListenPort:  3306,
				BackendHost: "db1.local",
				BackendPort: 3306,
			},
			{
				Name:        "",  // Invalid: empty name
				Protocol:    "postgresql",
				ListenPort:  5432,
				BackendHost: "db2.local",
				BackendPort: 5432,
			},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid route")
	}
}

func TestConfigEnvironmentVariableOverride(t *testing.T) {
	t.Setenv("CLUSTER_API_KEY", "env-api-key-123")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.ClusterAPIKey != "env-api-key-123" {
		t.Errorf("ClusterAPIKey = %q, want env-api-key-123", cfg.ClusterAPIKey)
	}
}

func TestLoadConfigTracingDefaults(t *testing.T) {
	t.Setenv("CLUSTER_API_KEY", "test-key")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.EnableTracing {
		t.Error("expected EnableTracing false by default")
	}

	if cfg.TraceSampleRate != 0.1 {
		t.Errorf("expected TraceSampleRate 0.1, got %f", cfg.TraceSampleRate)
	}

	if cfg.MetricsNamespace != "marchproxy_dblb" {
		t.Errorf("expected MetricsNamespace 'marchproxy_dblb', got %q", cfg.MetricsNamespace)
	}
}

func TestRouteConfigFullConfiguration(t *testing.T) {
	route := &config.RouteConfig{
		Name:           "mysql-prod",
		Protocol:       "mysql",
		ListenPort:     3306,
		BackendHost:    "mysql.prod.local",
		BackendPort:    3306,
		MaxConnections: 200,
		ConnectionRate: 500.0,
		QueryRate:      5000.0,
		EnableAuth:     true,
		Username:       "proxy_user",
		Password:       "secret",
		EnableSSL:      true,
		HealthCheckSQL: "SELECT 1",
	}

	if err := route.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}

	// Verify all fields set correctly
	if route.Name != "mysql-prod" {
		t.Errorf("Name = %q, want mysql-prod", route.Name)
	}
	if route.MaxConnections != 200 {
		t.Errorf("MaxConnections = %d, want 200", route.MaxConnections)
	}
	if route.ConnectionRate != 500.0 {
		t.Errorf("ConnectionRate = %f, want 500.0", route.ConnectionRate)
	}
}
