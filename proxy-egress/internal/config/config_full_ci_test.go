//go:build ci
// +build ci

package config

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// NewConfig Tests
// ============================================================================

func TestNewConfig_DefaultValues(t *testing.T) {
	config := NewConfig()

	tests := []struct {
		name  string
		got   interface{}
		want  interface{}
	}{
		{"LogLevel", config.LogLevel, "info"},
		{"ListenPort", config.ListenPort, 8080},
		{"AdminPort", config.AdminPort, 8081},
		{"EnableMetrics", config.EnableMetrics, true},
		{"EnableEBPF", config.EnableEBPF, false},
		{"ConnectionTimeout", config.ConnectionTimeout, 30},
		{"HeartbeatInterval", config.HeartbeatInterval, 60},
		{"ConfigUpdateInterval", config.ConfigUpdateInterval, 300},
		{"L7Enabled", config.L7.Enabled, false},
		{"L7EnvoyBinary", config.L7.EnvoyBinary, "/usr/local/bin/envoy"},
		{"L7EnvoyAdminPort", config.L7.EnvoyAdminPort, 9901},
		{"L7HTTPListenPort", config.L7.HTTPListenPort, 10000},
		{"L7HTTPSListenPort", config.L7.HTTPSListenPort, 10443},
		{"L7LogLevel", config.L7.LogLevel, "info"},
		{"ThreatEnabled", config.Threat.Enabled, false},
		{"ThreatIPBlockingEnabled", config.Threat.IPBlockingEnabled, true},
		{"ThreatIPCacheSize", config.Threat.IPCacheSize, 100000},
		{"ThreatDomainBlockingEnabled", config.Threat.DomainBlockingEnabled, true},
		{"ThreatWildcardSupport", config.Threat.WildcardSupport, true},
		{"ThreatURLMatchingEnabled", config.Threat.URLMatchingEnabled, true},
		{"ThreatURLMatchEngine", config.Threat.URLMatchEngine, "re2"},
		{"ThreatDNSCacheEnabled", config.Threat.DNSCacheEnabled, true},
		{"ThreatDNSPositiveTTL", config.Threat.DNSPositiveTTL, 5 * time.Minute},
		{"ThreatDNSNegativeTTL", config.Threat.DNSNegativeTTL, 1 * time.Minute},
		{"ThreatDNSCacheSize", config.Threat.DNSCacheSize, 10000},
		{"ThreatSyncMode", config.Threat.SyncMode, "both"},
		{"ThreatSyncPollInterval", config.Threat.SyncPollInterval, 60 * time.Second},
		{"TLSInterceptEnabled", config.TLSIntercept.Enabled, false},
		{"TLSInterceptMode", config.TLSIntercept.Mode, "mitm"},
		{"TLSInterceptCertCacheSize", config.TLSIntercept.CertCacheSize, 1000},
		{"ExtAuthEnabled", config.ExtAuth.Enabled, false},
		{"ExtAuthPort", config.ExtAuth.Port, 50051},
		{"ExtAuthHost", config.ExtAuth.Host, "127.0.0.1"},
		{"AccessControlEnabled", config.AccessControl.Enabled, false},
		{"AccessControlDefaultAllow", config.AccessControl.DefaultAllow, true},
		{"LeversEnabled", config.Levers.Enabled, false},
		{"LeversListenAddr", config.Levers.ListenAddr, ":9003"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestNewConfig_NestedStructs(t *testing.T) {
	config := NewConfig()

	if config.L7.EnvoyBinary == "" {
		t.Error("L7Config should have default EnvoyBinary")
	}
	if config.Threat.IPCacheSize == 0 {
		t.Error("ThreatConfig should have non-zero IPCacheSize")
	}
	if config.TLSIntercept.Mode == "" {
		t.Error("TLSInterceptConfig should have default Mode")
	}
	if config.ExtAuth.Port == 0 {
		t.Error("ExtAuthConfig should have default Port")
	}
	if config.AccessControl.DefaultAllow == false {
		t.Error("AccessControlConfig should have DefaultAllow set")
	}
	if config.Levers.ListenAddr == "" {
		t.Error("LeversConfig should have default ListenAddr")
	}
}

// ============================================================================
// Config.Validate Tests
// ============================================================================

func TestConfig_Validate_Success(t *testing.T) {
	config := &Config{
		ManagerURL:    "http://manager:8000",
		ClusterAPIKey: "test-key-123",
		ListenPort:    8080,
		AdminPort:     8081,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestConfig_Validate_MissingManagerURL(t *testing.T) {
	config := &Config{
		ClusterAPIKey: "test-key",
		ListenPort:    8080,
		AdminPort:     8081,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should fail when ManagerURL is empty")
	}
	if !strings.Contains(err.Error(), "manager_url") {
		t.Errorf("error should mention manager_url, got: %v", err)
	}
}

func TestConfig_Validate_MissingClusterAPIKey(t *testing.T) {
	config := &Config{
		ManagerURL: "http://manager:8000",
		ListenPort: 8080,
		AdminPort:  8081,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should fail when ClusterAPIKey is empty")
	}
	if !strings.Contains(err.Error(), "cluster_api_key") {
		t.Errorf("error should mention cluster_api_key, got: %v", err)
	}
}

func TestConfig_Validate_InvalidListenPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		hasError bool
	}{
		{"ValidPort", 8080, false},
		{"MinPort", 1, false},
		{"MaxPort", 65535, false},
		{"ZeroPort", 0, true},
		{"NegativePort", -1, true},
		{"TooHighPort", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "key",
				ListenPort:    tt.port,
				AdminPort:     8081,
			}
			err := config.Validate()
			if tt.hasError && err == nil {
				t.Errorf("Validate() should fail for port %d", tt.port)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Validate() should succeed for port %d, got: %v", tt.port, err)
			}
		})
	}
}

func TestConfig_Validate_InvalidAdminPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		hasError bool
	}{
		{"ValidPort", 8081, false},
		{"MinPort", 1, false},
		{"MaxPort", 65535, false},
		{"ZeroPort", 0, true},
		{"TooHighPort", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "key",
				ListenPort:    8080,
				AdminPort:     tt.port,
			}
			err := config.Validate()
			if tt.hasError && err == nil {
				t.Errorf("Validate() should fail for admin port %d", tt.port)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Validate() should succeed for admin port %d, got: %v", tt.port, err)
			}
		})
	}
}

func TestConfig_Validate_PortConflict(t *testing.T) {
	config := &Config{
		ManagerURL:    "http://manager:8000",
		ClusterAPIKey: "key",
		ListenPort:    8080,
		AdminPort:     8080,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should fail when ListenPort equals AdminPort")
	}
	if !strings.Contains(err.Error(), "cannot be the same") {
		t.Errorf("error should mention port conflict, got: %v", err)
	}
}

// ============================================================================
// GetHostname Tests
// ============================================================================

func TestConfig_GetHostname_WithConfigured(t *testing.T) {
	config := &Config{
		Hostname: "my-custom-host",
	}

	hostname := config.GetHostname()
	if hostname != "my-custom-host" {
		t.Errorf("GetHostname() should return configured hostname, got: %v", hostname)
	}
}

func TestConfig_GetHostname_WithDefault(t *testing.T) {
	config := &Config{
		Hostname: "",
	}

	hostname := config.GetHostname()
	if hostname == "" {
		t.Error("GetHostname() should return system hostname when not configured")
	}
}

// ============================================================================
// Helper Functions Tests
// ============================================================================

func TestGetBoolEnv_TrueValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"true", "true"},
		{"TRUE", "TRUE"},
		{"True", "True"},
		{"1", "1"},
		{"yes", "yes"},
		{"YES", "YES"},
		{"on", "on"},
		{"ON", "ON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL", tt.value)
			result := getBoolEnv("TEST_BOOL", false)
			if !result {
				t.Errorf("getBoolEnv(%q) should return true", tt.value)
			}
		})
	}
}

func TestGetBoolEnv_FalseValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"false", "false"},
		{"FALSE", "FALSE"},
		{"False", "False"},
		{"0", "0"},
		{"no", "no"},
		{"NO", "NO"},
		{"off", "off"},
		{"OFF", "OFF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL", tt.value)
			result := getBoolEnv("TEST_BOOL", true)
			if result {
				t.Errorf("getBoolEnv(%q) should return false", tt.value)
			}
		})
	}
}

func TestGetBoolEnv_InvalidValue(t *testing.T) {
	t.Setenv("TEST_BOOL", "invalid")
	result := getBoolEnv("TEST_BOOL", true)
	if !result {
		t.Error("getBoolEnv() should return default when value is invalid")
	}
}

func TestGetBoolEnv_EmptyValue(t *testing.T) {
	t.Setenv("TEST_BOOL", "")
	result := getBoolEnv("TEST_BOOL", true)
	if !result {
		t.Error("getBoolEnv() should return default when value is empty")
	}
}

func TestGetBoolEnv_NotSet(t *testing.T) {
	os.Unsetenv("TEST_BOOL_NOT_SET")
	result := getBoolEnv("TEST_BOOL_NOT_SET", true)
	if !result {
		t.Error("getBoolEnv() should return default when variable is not set")
	}
}

func TestGetIntEnv_ValidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"positive", "42", 42},
		{"zero", "0", 0},
		{"negative", "-10", -10},
		{"large", "999999", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT", tt.value)
			result := getIntEnv("TEST_INT", 0)
			if result != tt.want {
				t.Errorf("getIntEnv(%q) got %d, want %d", tt.value, result, tt.want)
			}
		})
	}
}

func TestGetIntEnv_InvalidValue(t *testing.T) {
	t.Setenv("TEST_INT", "not-a-number")
	result := getIntEnv("TEST_INT", 99)
	if result != 99 {
		t.Error("getIntEnv() should return default for invalid value")
	}
}

func TestGetIntEnv_EmptyValue(t *testing.T) {
	t.Setenv("TEST_INT", "")
	result := getIntEnv("TEST_INT", 99)
	if result != 99 {
		t.Error("getIntEnv() should return default for empty value")
	}
}

func TestGetIntEnv_NotSet(t *testing.T) {
	os.Unsetenv("TEST_INT_NOT_SET")
	result := getIntEnv("TEST_INT_NOT_SET", 99)
	if result != 99 {
		t.Error("getIntEnv() should return default when not set")
	}
}

func TestGetEnvOrDefault_WithValue(t *testing.T) {
	t.Setenv("TEST_STR", "custom-value")
	result := getEnvOrDefault("TEST_STR", "default")
	if result != "custom-value" {
		t.Errorf("getEnvOrDefault() got %q, want %q", result, "custom-value")
	}
}

func TestGetEnvOrDefault_EmptyValue(t *testing.T) {
	t.Setenv("TEST_STR", "")
	result := getEnvOrDefault("TEST_STR", "default")
	if result != "default" {
		t.Errorf("getEnvOrDefault() got %q, want %q", result, "default")
	}
}

func TestGetEnvOrDefault_NotSet(t *testing.T) {
	os.Unsetenv("TEST_STR_NOT_SET")
	result := getEnvOrDefault("TEST_STR_NOT_SET", "default")
	if result != "default" {
		t.Errorf("getEnvOrDefault() got %q, want %q", result, "default")
	}
}

func TestGetDurationEnv_ValidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"seconds", "5s", 5 * time.Second},
		{"minutes", "2m", 2 * time.Minute},
		{"hours", "1h", 1 * time.Hour},
		{"zero", "0s", 0},
		{"complex", "1h30m45s", 1*time.Hour + 30*time.Minute + 45*time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_DUR", tt.value)
			result := getDurationEnv("TEST_DUR", 0)
			if result != tt.want {
				t.Errorf("getDurationEnv(%q) got %v, want %v", tt.value, result, tt.want)
			}
		})
	}
}

func TestGetDurationEnv_InvalidValue(t *testing.T) {
	t.Setenv("TEST_DUR", "invalid-duration")
	result := getDurationEnv("TEST_DUR", 10*time.Second)
	if result != 10*time.Second {
		t.Error("getDurationEnv() should return default for invalid value")
	}
}

func TestGetDurationEnv_EmptyValue(t *testing.T) {
	t.Setenv("TEST_DUR", "")
	result := getDurationEnv("TEST_DUR", 20*time.Second)
	if result != 20*time.Second {
		t.Error("getDurationEnv() should return default for empty value")
	}
}

func TestGetDurationEnv_NotSet(t *testing.T) {
	os.Unsetenv("TEST_DUR_NOT_SET")
	result := getDurationEnv("TEST_DUR_NOT_SET", 30*time.Second)
	if result != 30*time.Second {
		t.Error("getDurationEnv() should return default when not set")
	}
}

func TestGetStringSliceEnv_ValidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"single", "item1", []string{"item1"}},
		{"multiple", "item1,item2,item3", []string{"item1", "item2", "item3"}},
		{"withspaces", "item1 , item2 , item3", []string{"item1", "item2", "item3"}},
		{"trailingcomma", "item1,item2,", []string{"item1", "item2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_SLICE", tt.value)
			result := getStringSliceEnv("TEST_SLICE", nil)
			if len(result) != len(tt.want) {
				t.Errorf("getStringSliceEnv(%q) got %d items, want %d", tt.value, len(result), len(tt.want))
				return
			}
			for i, v := range result {
				if v != tt.want[i] {
					t.Errorf("getStringSliceEnv(%q)[%d] got %q, want %q", tt.value, i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetStringSliceEnv_EmptyValue(t *testing.T) {
	t.Setenv("TEST_SLICE", "")
	defaultVal := []string{"default1", "default2"}
	result := getStringSliceEnv("TEST_SLICE", defaultVal)
	if len(result) != 2 || result[0] != "default1" {
		t.Error("getStringSliceEnv() should return default for empty value")
	}
}

func TestGetStringSliceEnv_NotSet(t *testing.T) {
	os.Unsetenv("TEST_SLICE_NOT_SET")
	defaultVal := []string{"default1", "default2"}
	result := getStringSliceEnv("TEST_SLICE_NOT_SET", defaultVal)
	if len(result) != 2 || result[0] != "default1" {
		t.Error("getStringSliceEnv() should return default when not set")
	}
}

func TestGetStringSliceEnv_OnlyWhitespace(t *testing.T) {
	t.Setenv("TEST_SLICE", "  ,  , ,,  ")
	defaultVal := []string{"default"}
	result := getStringSliceEnv("TEST_SLICE", defaultVal)
	if len(result) != 1 || result[0] != "default" {
		t.Error("getStringSliceEnv() should return default when all items are empty")
	}
}

// ============================================================================
// Nested Config Struct Tests
// ============================================================================

func TestL7Config_ZeroValue(t *testing.T) {
	cfg := L7Config{}
	if cfg.Enabled {
		t.Error("L7Config.Enabled should be false by default")
	}
	if cfg.EnvoyBinary != "" {
		t.Error("L7Config.EnvoyBinary should be empty by default")
	}
	if cfg.HTTPListenPort != 0 {
		t.Error("L7Config.HTTPListenPort should be 0 by default")
	}
}

func TestThreatConfig_ZeroValue(t *testing.T) {
	cfg := ThreatConfig{}
	if cfg.Enabled {
		t.Error("ThreatConfig.Enabled should be false by default")
	}
	if cfg.IPCacheSize != 0 {
		t.Error("ThreatConfig.IPCacheSize should be 0 by default")
	}
}

func TestTLSInterceptConfig_ZeroValue(t *testing.T) {
	cfg := TLSInterceptConfig{}
	if cfg.Enabled {
		t.Error("TLSInterceptConfig.Enabled should be false by default")
	}
	if cfg.Mode != "" {
		t.Error("TLSInterceptConfig.Mode should be empty by default")
	}
}

func TestExtAuthConfig_ZeroValue(t *testing.T) {
	cfg := ExtAuthConfig{}
	if cfg.Enabled {
		t.Error("ExtAuthConfig.Enabled should be false by default")
	}
	if cfg.Port != 0 {
		t.Error("ExtAuthConfig.Port should be 0 by default")
	}
}

func TestAccessControlConfig_ZeroValue(t *testing.T) {
	cfg := AccessControlConfig{}
	if cfg.Enabled {
		t.Error("AccessControlConfig.Enabled should be false by default")
	}
}

func TestLeversConfig_ZeroValue(t *testing.T) {
	cfg := LeversConfig{}
	if cfg.Enabled {
		t.Error("LeversConfig.Enabled should be false by default")
	}
	if cfg.ListenAddr != "" {
		t.Error("LeversConfig.ListenAddr should be empty by default")
	}
}

// ============================================================================
// Validation Specific Tests
// ============================================================================

func TestValidateConfig_WorkerThreadsNegative(t *testing.T) {
	config := &Config{
		ManagerURL:            "http://manager:8000",
		ClusterAPIKey:         "key",
		ProxyName:             "proxy",
		Hostname:              "host",
		ListenPort:            8080,
		AdminPort:             8081,
		LogLevel:              "INFO",
		ConfigUpdateInterval:  300,
		HeartbeatInterval:     30,
		ConnectionTimeout:     30,
		WorkerThreads:         -5,
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("validateConfig() should fail for negative WorkerThreads")
	}
	if !strings.Contains(err.Error(), "worker_threads") {
		t.Errorf("error should mention worker_threads, got: %v", err)
	}
}

func TestValidateConfig_WorkerThreadsZero(t *testing.T) {
	// WorkerThreads == 0 is valid (auto-detect)
	config := &Config{
		ManagerURL:            "http://manager:8000",
		ClusterAPIKey:         "key",
		ProxyName:             "proxy",
		Hostname:              "host",
		ListenPort:            8080,
		AdminPort:             8081,
		LogLevel:              "INFO",
		ConfigUpdateInterval:  300,
		HeartbeatInterval:     30,
		ConnectionTimeout:     30,
		WorkerThreads:         0,
	}

	err := validateConfig(config)
	if err != nil {
		t.Errorf("validateConfig() should succeed for WorkerThreads=0, got: %v", err)
	}
}

func TestValidateConfig_InvalidLogLevel(t *testing.T) {
	config := &Config{
		ManagerURL:            "http://manager:8000",
		ClusterAPIKey:         "key",
		ProxyName:             "proxy",
		Hostname:              "host",
		ListenPort:            8080,
		AdminPort:             8081,
		LogLevel:              "INVALID",
		ConfigUpdateInterval:  300,
		HeartbeatInterval:     30,
		ConnectionTimeout:     30,
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("validateConfig() should fail for invalid log level")
	}
	if !strings.Contains(err.Error(), "log_level") {
		t.Errorf("error should mention log_level, got: %v", err)
	}
}

func TestValidateConfig_ValidLogLevels(t *testing.T) {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			config := &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "key",
				ProxyName:             "proxy",
				Hostname:              "host",
				ListenPort:            8080,
				AdminPort:             8081,
				LogLevel:              level,
				ConfigUpdateInterval:  300,
				HeartbeatInterval:     30,
				ConnectionTimeout:     30,
			}

			err := validateConfig(config)
			if err != nil {
				t.Errorf("validateConfig() should succeed for log level %s, got: %v", level, err)
			}
		})
	}
}

func TestValidateConfig_IntervalValidation(t *testing.T) {
	tests := []struct {
		name                  string
		configUpdateInterval  int
		heartbeatInterval     int
		connectionTimeout     int
		expectError           bool
		errorPattern          string
	}{
		{
			"ValidIntervals",
			300, 30, 30,
			false, "",
		},
		{
			"ConfigUpdateIntervalTooLow",
			5, 30, 30,
			true, "config_update_interval",
		},
		{
			"HeartbeatIntervalTooLow",
			300, 3, 30,
			true, "heartbeat_interval",
		},
		{
			"ConnectionTimeoutTooLow",
			300, 30, 0,
			true, "connection_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:           "http://manager:8000",
				ClusterAPIKey:        "key",
				ProxyName:            "proxy",
				Hostname:             "host",
				ListenPort:           8080,
				AdminPort:            8081,
				LogLevel:             "INFO",
				ConfigUpdateInterval: tt.configUpdateInterval,
				HeartbeatInterval:    tt.heartbeatInterval,
				ConnectionTimeout:    tt.connectionTimeout,
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
			if tt.expectError && err != nil && !strings.Contains(err.Error(), tt.errorPattern) {
				t.Errorf("error should mention %q, got: %v", tt.errorPattern, err)
			}
		})
	}
}

func TestValidateConfig_RateLimitingValidation(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		rps          int
		expectError  bool
	}{
		{"NotEnabled", false, 0, false},
		{"EnabledWithValid", true, 100, false},
		{"EnabledWithZero", true, 0, true},
		{"EnabledWithNegative", true, -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:           "http://manager:8000",
				ClusterAPIKey:        "key",
				ProxyName:            "proxy",
				Hostname:             "host",
				ListenPort:           8080,
				AdminPort:            8081,
				LogLevel:             "INFO",
				ConfigUpdateInterval: 300,
				HeartbeatInterval:    30,
				ConnectionTimeout:    30,
				RateLimitEnabled:     tt.enabled,
				RateLimitRPS:         tt.rps,
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_MTLSValidation(t *testing.T) {
	tests := []struct {
		name             string
		enabled          bool
		serverCertPath   string
		serverKeyPath    string
		requireClientCert bool
		clientCAPath     string
		expectError      bool
		errorPattern     string
	}{
		{
			"NotEnabled",
			false, "", "", false, "",
			false, "",
		},
		{
			"EnabledWithAllPaths",
			true, "/cert", "/key", true, "/ca",
			false, "",
		},
		{
			"MissingServerCert",
			true, "", "/key", false, "",
			true, "mtls_server_cert_path",
		},
		{
			"MissingServerKey",
			true, "/cert", "", false, "",
			true, "mtls_server_key_path",
		},
		{
			"RequireClientCertButNoCA",
			true, "/cert", "/key", true, "",
			true, "mtls_client_ca_path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "key",
				ProxyName:             "proxy",
				Hostname:              "host",
				ListenPort:            8080,
				AdminPort:             8081,
				LogLevel:              "INFO",
				ConfigUpdateInterval:  300,
				HeartbeatInterval:     30,
				ConnectionTimeout:     30,
				EnableMTLS:            tt.enabled,
				MTLSServerCertPath:    tt.serverCertPath,
				MTLSServerKeyPath:     tt.serverKeyPath,
				MTLSRequireClientCert: tt.requireClientCert,
				MTLSClientCAPath:      tt.clientCAPath,
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_L7Validation(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		binary       string
		configPath   string
		adminPort    int
		httpPort     int
		httpsPort    int
		logLevel     string
		expectError  bool
		errorPattern string
	}{
		{
			"NotEnabled",
			false, "", "", 0, 0, 0, "",
			false, "",
		},
		{
			"EnabledWithValidConfig",
			true, "/usr/bin/envoy", "/etc/envoy.yaml", 9901, 10000, 10443, "info",
			false, "",
		},
		{
			"MissingBinary",
			true, "", "/etc/envoy.yaml", 9901, 10000, 10443, "info",
			true, "envoy_binary",
		},
		{
			"MissingConfigPath",
			true, "/usr/bin/envoy", "", 9901, 10000, 10443, "info",
			true, "envoy_config_path",
		},
		{
			"InvalidAdminPort",
			true, "/usr/bin/envoy", "/etc/envoy.yaml", 70000, 10000, 10443, "info",
			true, "envoy_admin_port",
		},
		{
			"InvalidHTTPPort",
			true, "/usr/bin/envoy", "/etc/envoy.yaml", 9901, 0, 10443, "info",
			true, "http_listen_port",
		},
		{
			"InvalidHTTPSPort",
			true, "/usr/bin/envoy", "/etc/envoy.yaml", 9901, 10000, -1, "info",
			true, "https_listen_port",
		},
		{
			"InvalidLogLevel",
			true, "/usr/bin/envoy", "/etc/envoy.yaml", 9901, 10000, 10443, "invalid",
			true, "envoy_log_level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "key",
				ProxyName:             "proxy",
				Hostname:              "host",
				ListenPort:            8080,
				AdminPort:             8081,
				LogLevel:              "INFO",
				ConfigUpdateInterval:  300,
				HeartbeatInterval:     30,
				ConnectionTimeout:     30,
				L7: L7Config{
					Enabled:         tt.enabled,
					EnvoyBinary:     tt.binary,
					EnvoyConfigPath: tt.configPath,
					EnvoyAdminPort:  tt.adminPort,
					HTTPListenPort:  tt.httpPort,
					HTTPSListenPort: tt.httpsPort,
					LogLevel:        tt.logLevel,
				},
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_ThreatValidation(t *testing.T) {
	tests := []struct {
		name               string
		enabled            bool
		ipCacheSize        int
		dnsCacheSize       int
		syncMode           string
		urlMatchEngine     string
		expectError        bool
		errorPattern       string
	}{
		{
			"NotEnabled",
			false, -1, -1, "invalid", "invalid",
			false, "",
		},
		{
			"EnabledWithValidConfig",
			true, 100000, 50000, "both", "re2",
			false, "",
		},
		{
			"NegativeIPCacheSize",
			true, -1, 50000, "both", "re2",
			true, "ip_cache_size",
		},
		{
			"NegativeDNSCacheSize",
			true, 100000, -1, "both", "re2",
			true, "dns_cache_size",
		},
		{
			"InvalidSyncMode",
			true, 100000, 50000, "invalid", "re2",
			true, "sync_mode",
		},
		{
			"InvalidURLMatchEngine",
			true, 100000, 50000, "both", "invalid",
			true, "url_match_engine",
		},
		{
			"ValidSyncModeGRPC",
			true, 100000, 50000, "grpc", "re2",
			false, "",
		},
		{
			"ValidSyncModePoll",
			true, 100000, 50000, "poll", "re2",
			false, "",
		},
		{
			"ValidURLMatchEngineBoost",
			true, 100000, 50000, "both", "boost",
			false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "key",
				ProxyName:             "proxy",
				Hostname:              "host",
				ListenPort:            8080,
				AdminPort:             8081,
				LogLevel:              "INFO",
				ConfigUpdateInterval:  300,
				HeartbeatInterval:     30,
				ConnectionTimeout:     30,
				Threat: ThreatConfig{
					Enabled:        tt.enabled,
					IPCacheSize:    tt.ipCacheSize,
					DNSCacheSize:   tt.dnsCacheSize,
					SyncMode:       tt.syncMode,
					URLMatchEngine: tt.urlMatchEngine,
				},
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_TLSInterceptValidation(t *testing.T) {
	tests := []struct {
		name               string
		enabled            bool
		mode               string
		caCertPath         string
		caKeyPath          string
		certCacheSize      int
		expectError        bool
		errorPattern       string
	}{
		{
			"NotEnabled",
			false, "", "", "", 0,
			false, "",
		},
		{
			"EnabledWithPreconfigured",
			true, "preconfigured", "", "", 1000,
			false, "",
		},
		{
			"EnabledWithMITMAndPaths",
			true, "mitm", "/cert", "/key", 1000,
			false, "",
		},
		{
			"InvalidMode",
			true, "invalid", "/cert", "/key", 1000,
			true, "mode",
		},
		{
			"MITMMissingCert",
			true, "mitm", "", "/key", 1000,
			true, "ca_cert_path",
		},
		{
			"MITMMissingKey",
			true, "mitm", "/cert", "", 1000,
			true, "ca_key_path",
		},
		{
			"NegativeCertCacheSize",
			true, "mitm", "/cert", "/key", -1,
			true, "cert_cache_size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "key",
				ProxyName:             "proxy",
				Hostname:              "host",
				ListenPort:            8080,
				AdminPort:             8081,
				LogLevel:              "INFO",
				ConfigUpdateInterval:  300,
				HeartbeatInterval:     30,
				ConnectionTimeout:     30,
				TLSIntercept: TLSInterceptConfig{
					Enabled:       tt.enabled,
					Mode:          tt.mode,
					CACertPath:    tt.caCertPath,
					CAKeyPath:     tt.caKeyPath,
					CertCacheSize: tt.certCacheSize,
				},
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
		})
	}
}

func TestValidateConfig_ExtAuthValidation(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		port        int
		expectError bool
		errorPattern string
	}{
		{"NotEnabled", false, 0, false, ""},
		{"EnabledWithValidPort", true, 9002, false, ""},
		{"EnabledWithInvalidPort", true, 70000, true, "port"},
		{"EnabledWithZeroPort", true, 0, true, "port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "key",
				ProxyName:             "proxy",
				Hostname:              "host",
				ListenPort:            8080,
				AdminPort:             8081,
				LogLevel:              "INFO",
				ConfigUpdateInterval:  300,
				HeartbeatInterval:     30,
				ConnectionTimeout:     30,
				ExtAuth: ExtAuthConfig{
					Enabled: tt.enabled,
					Port:    tt.port,
				},
			}

			err := validateConfig(config)
			if tt.expectError && err == nil {
				t.Error("validateConfig() should fail")
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateConfig() should succeed, got: %v", err)
			}
		})
	}
}

// ============================================================================
// Helper Function - getHostname Tests
// ============================================================================

func TestGetHostname_Success(t *testing.T) {
	hostname := getHostname()
	if hostname == "" {
		t.Error("getHostname() should not return empty string")
	}
	if hostname == "unknown" {
		// This is the fallback, which is OK but not ideal
		t.Logf("getHostname() returned fallback value 'unknown'")
	}
}

// ============================================================================
// Getter/Query Methods Tests
// ============================================================================

func TestConfig_GetListenAddress(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{"StandardPort", 8080, ":8080"},
		{"HighPort", 65535, ":65535"},
		{"LowPort", 1, ":1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{ListenPort: tt.port}
			result := config.GetListenAddress()
			if result != tt.want {
				t.Errorf("GetListenAddress() got %q, want %q", result, tt.want)
			}
		})
	}
}

func TestConfig_GetAdminAddress(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{"StandardPort", 8081, ":8081"},
		{"HighPort", 65535, ":65535"},
		{"LowPort", 1, ":1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{AdminPort: tt.port}
			result := config.GetAdminAddress()
			if result != tt.want {
				t.Errorf("GetAdminAddress() got %q, want %q", result, tt.want)
			}
		})
	}
}

func TestConfig_IsNetworkAccelerationEnabled(t *testing.T) {
	tests := []struct {
		name     string
		dpdk     bool
		xdp      bool
		afxdp    bool
		sriov    bool
		expected bool
	}{
		{"AllDisabled", false, false, false, false, false},
		{"DPDKEnabled", true, false, false, false, true},
		{"XDPEnabled", false, true, false, false, true},
		{"AFXDPEnabled", false, false, true, false, true},
		{"SRIOVEnabled", false, false, false, true, true},
		{"MultipleEnabled", true, true, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableDPDK:  tt.dpdk,
				EnableXDP:   tt.xdp,
				EnableAFXDP: tt.afxdp,
				EnableSRIOV: tt.sriov,
			}
			result := config.IsNetworkAccelerationEnabled()
			if result != tt.expected {
				t.Errorf("IsNetworkAccelerationEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetCapabilities(t *testing.T) {
	config := &Config{
		EnableEBPF:  true,
		EnableDPDK:  true,
		EnableXDP:   false,
		EnableAFXDP: false,
		EnableSRIOV: false,
		EnableMTLS:  true,
		L7: L7Config{
			Enabled:       true,
			HTTP3Enabled:  true,
		},
		Threat: ThreatConfig{
			Enabled:               true,
			IPBlockingEnabled:     true,
			DomainBlockingEnabled: false,
			URLMatchingEnabled:    true,
		},
		TLSIntercept: TLSInterceptConfig{
			Enabled: true,
		},
		AccessControl: AccessControlConfig{
			Enabled: false,
		},
	}

	caps := config.GetCapabilities()

	expectedCaps := map[string]bool{
		"ebpf": true, "dpdk": true, "mtls": true, "l7_proxy": true,
		"http1": true, "http2": true, "http3_experimental": true,
		"threat_intel": true, "ip_blocking": true, "url_matching": true,
		"tls_intercept": true,
	}

	notExpected := map[string]bool{
		"xdp": true, "af_xdp": true, "sr_iov": true,
		"domain_blocking": true, "access_control": true,
	}

	for _, cap := range caps {
		if notExpected[cap] {
			t.Errorf("capability %q should not be present", cap)
		}
		if expectedCaps[cap] {
			expectedCaps[cap] = false
		}
	}

	for cap, found := range expectedCaps {
		if found {
			t.Errorf("capability %q not found in capabilities", cap)
		}
	}
}

func TestConfig_IsMTLSEnabled(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		certPath  string
		keyPath   string
		caPath    string
		expected  bool
	}{
		{"Disabled", false, "/cert", "/key", "/ca", false},
		{"EnabledAllPaths", true, "/cert", "/key", "/ca", true},
		{"EnabledMissingCert", true, "", "/key", "/ca", false},
		{"EnabledMissingKey", true, "/cert", "", "/ca", false},
		{"EnabledMissingCA", true, "/cert", "/key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableMTLS:       tt.enabled,
				MTLSServerCertPath: tt.certPath,
				MTLSServerKeyPath:  tt.keyPath,
				MTLSClientCAPath:   tt.caPath,
			}
			result := config.IsMTLSEnabled()
			if result != tt.expected {
				t.Errorf("IsMTLSEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetMTLSConfig(t *testing.T) {
	config := &Config{
		MTLSServerCertPath: "/cert.pem",
		MTLSServerKeyPath:  "/key.pem",
		MTLSClientCAPath:   "/ca.pem",
	}

	cert, key, ca := config.GetMTLSConfig()
	if cert != "/cert.pem" {
		t.Errorf("cert got %q, want /cert.pem", cert)
	}
	if key != "/key.pem" {
		t.Errorf("key got %q, want /key.pem", key)
	}
	if ca != "/ca.pem" {
		t.Errorf("ca got %q, want /ca.pem", ca)
	}
}

func TestConfig_RequiresClientCert(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		require    bool
		expected   bool
	}{
		{"DisabledAndRequired", false, true, false},
		{"EnabledAndRequired", true, true, true},
		{"EnabledAndNotRequired", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableMTLS:            tt.enabled,
				MTLSRequireClientCert: tt.require,
			}
			result := config.RequiresClientCert()
			if result != tt.expected {
				t.Errorf("RequiresClientCert() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_ShouldVerifyClientCert(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		verify   bool
		expected bool
	}{
		{"DisabledAndVerify", false, true, false},
		{"EnabledAndVerify", true, true, true},
		{"EnabledAndNoVerify", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableMTLS:           tt.enabled,
				MTLSVerifyClientCert: tt.verify,
			}
			result := config.ShouldVerifyClientCert()
			if result != tt.expected {
				t.Errorf("ShouldVerifyClientCert() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetKillKrillConfig(t *testing.T) {
	config := &Config{
		KillKrillEnabled:         true,
		KillKrillLogEndpoint:     "http://killkrill:8000",
		KillKrillMetricsEndpoint: "http://killkrill:9000",
		KillKrillAPIKey:          "api-key-123",
		KillKrillSourceName:      "proxy-1",
		KillKrillApplication:     "proxy",
		KillKrillBatchSize:       100,
		KillKrillFlushInterval:   10,
		KillKrillTimeout:         30,
		KillKrillUseHTTP3:        true,
		KillKrillTLSInsecure:     false,
	}

	result := config.GetKillKrillConfig()
	if result == nil {
		t.Error("GetKillKrillConfig() should not return nil when enabled")
	}
	if (*result)["enabled"] != true {
		t.Error("GetKillKrillConfig() should have enabled=true")
	}
}

func TestConfig_GetKillKrillConfig_Disabled(t *testing.T) {
	config := &Config{
		KillKrillEnabled: false,
	}

	result := config.GetKillKrillConfig()
	if result != nil {
		t.Error("GetKillKrillConfig() should return nil when disabled")
	}
}

func TestConfig_IsL7Enabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"Disabled", false, false},
		{"Enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				L7: L7Config{Enabled: tt.enabled},
			}
			result := config.IsL7Enabled()
			if result != tt.expected {
				t.Errorf("IsL7Enabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_IsHTTP3Enabled(t *testing.T) {
	tests := []struct {
		name            string
		l7Enabled       bool
		http3Enabled    bool
		expected        bool
	}{
		{"L7DisabledHTTP3Disabled", false, false, false},
		{"L7EnabledHTTP3Disabled", true, false, false},
		{"L7DisabledHTTP3Enabled", false, true, false},
		{"L7EnabledHTTP3Enabled", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				L7: L7Config{
					Enabled:      tt.l7Enabled,
					HTTP3Enabled: tt.http3Enabled,
				},
			}
			result := config.IsHTTP3Enabled()
			if result != tt.expected {
				t.Errorf("IsHTTP3Enabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetEnvoyConfig(t *testing.T) {
	config := &Config{
		L7: L7Config{
			EnvoyBinary:    "/usr/bin/envoy",
			EnvoyConfigPath: "/etc/envoy.yaml",
			EnvoyAdminPort: 9901,
		},
	}

	binary, configPath, adminPort := config.GetEnvoyConfig()
	if binary != "/usr/bin/envoy" {
		t.Errorf("binary got %q, want /usr/bin/envoy", binary)
	}
	if configPath != "/etc/envoy.yaml" {
		t.Errorf("configPath got %q, want /etc/envoy.yaml", configPath)
	}
	if adminPort != 9901 {
		t.Errorf("adminPort got %d, want 9901", adminPort)
	}
}

func TestConfig_IsThreatEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"Disabled", false, false},
		{"Enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Threat: ThreatConfig{Enabled: tt.enabled},
			}
			result := config.IsThreatEnabled()
			if result != tt.expected {
				t.Errorf("IsThreatEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_IsTLSInterceptEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"Disabled", false, false},
		{"Enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				TLSIntercept: TLSInterceptConfig{Enabled: tt.enabled},
			}
			result := config.IsTLSInterceptEnabled()
			if result != tt.expected {
				t.Errorf("IsTLSInterceptEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetTLSInterceptCAConfig(t *testing.T) {
	config := &Config{
		TLSIntercept: TLSInterceptConfig{
			CACertPath: "/cert.pem",
			CAKeyPath:  "/key.pem",
		},
	}

	cert, key := config.GetTLSInterceptCAConfig()
	if cert != "/cert.pem" {
		t.Errorf("cert got %q, want /cert.pem", cert)
	}
	if key != "/key.pem" {
		t.Errorf("key got %q, want /key.pem", key)
	}
}

func TestConfig_GetTLSInterceptMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{"MITM", "mitm", "mitm"},
		{"Preconfigured", "preconfigured", "preconfigured"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				TLSIntercept: TLSInterceptConfig{Mode: tt.mode},
			}
			result := config.GetTLSInterceptMode()
			if result != tt.expected {
				t.Errorf("GetTLSInterceptMode() got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConfig_IsAccessControlEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"Disabled", false, false},
		{"Enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				AccessControl: AccessControlConfig{Enabled: tt.enabled},
			}
			result := config.IsAccessControlEnabled()
			if result != tt.expected {
				t.Errorf("IsAccessControlEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetExtAuthAddress(t *testing.T) {
	config := &Config{
		ExtAuth: ExtAuthConfig{
			Host: "127.0.0.1",
			Port: 9002,
		},
	}

	result := config.GetExtAuthAddress()
	expected := "127.0.0.1:9002"
	if result != expected {
		t.Errorf("GetExtAuthAddress() got %q, want %q", result, expected)
	}
}

func TestConfig_IsExtAuthEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"Disabled", false, false},
		{"Enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ExtAuth: ExtAuthConfig{Enabled: tt.enabled},
			}
			result := config.IsExtAuthEnabled()
			if result != tt.expected {
				t.Errorf("IsExtAuthEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_IsAccelerationEnabled(t *testing.T) {
	tests := []struct {
		name     string
		ebpf     bool
		dpdk     bool
		xdp      bool
		afxdp    bool
		sriov    bool
		expected bool
	}{
		{"AllDisabled", false, false, false, false, false, false},
		{"EBPFEnabled", true, false, false, false, false, true},
		{"DPDKEnabled", false, true, false, false, false, true},
		{"XDPEnabled", false, false, true, false, false, true},
		{"MultipleEnabled", true, true, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableEBPF:  tt.ebpf,
				EnableDPDK:  tt.dpdk,
				EnableXDP:   tt.xdp,
				EnableAFXDP: tt.afxdp,
				EnableSRIOV: tt.sriov,
			}
			result := config.IsAccelerationEnabled()
			if result != tt.expected {
				t.Errorf("IsAccelerationEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_GetWorkerThreads(t *testing.T) {
	tests := []struct {
		name     string
		threads  int
		expected int
	}{
		{"Zero", 0, 4},
		{"Positive", 8, 8},
		{"High", 64, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				WorkerThreads: tt.threads,
			}
			result := config.GetWorkerThreads()
			if result != tt.expected {
				t.Errorf("GetWorkerThreads() got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestConfig_IsTLSEnabled(t *testing.T) {
	tests := []struct {
		name      string
		certPath  string
		keyPath   string
		expected  bool
	}{
		{"BothEmpty", "", "", false},
		{"OnlyCert", "/cert.pem", "", false},
		{"OnlyKey", "", "/key.pem", false},
		{"BothPresent", "/cert.pem", "/key.pem", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				TLSCertPath: tt.certPath,
				TLSKeyPath:  tt.keyPath,
			}
			result := config.IsTLSEnabled()
			if result != tt.expected {
				t.Errorf("IsTLSEnabled() got %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Edge Cases and Integration Tests
// ============================================================================

func TestConfig_MultipleErrorConditions(t *testing.T) {
	// Test a config with multiple issues to ensure we catch at least one
	config := &Config{
		ManagerURL:    "",
		ClusterAPIKey: "",
		ListenPort:    70000,
		AdminPort:     70000,
		LogLevel:      "INVALID",
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() should fail with multiple issues")
	}
}

func TestConfig_FieldTypes(t *testing.T) {
	config := NewConfig()

	// Verify field types are correct
	_ = config.ListenPort + 1            // int
	_ = config.LogLevel + ""              // string
	_ = config.EnableMetrics && true     // bool
	_ = config.Threat.DNSPositiveTTL / 2 // time.Duration
	_ = config.L7.HTTPListenPort + 1      // int
}

func TestGetIntEnv_Boundaries(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"MaxInt", strconv.Itoa(1<<31 - 1), 1<<31 - 1},
		{"MinInt", "-" + strconv.Itoa(1<<31 - 1), -(1<<31 - 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_BOUNDARY", tt.value)
			result := getIntEnv("TEST_INT_BOUNDARY", 0)
			if result != tt.want {
				t.Errorf("getIntEnv(%q) got %d, want %d", tt.value, result, tt.want)
			}
		})
	}
}

// ============================================================================
// GetCapabilities Edge Cases
// ============================================================================

func TestConfig_GetCapabilities_Empty(t *testing.T) {
	config := &Config{}
	caps := config.GetCapabilities()
	if len(caps) != 0 {
		t.Errorf("GetCapabilities() should return empty slice for empty config, got %d items", len(caps))
	}
}

func TestConfig_GetCapabilities_OnlyEBPF(t *testing.T) {
	config := &Config{
		EnableEBPF: true,
	}
	caps := config.GetCapabilities()
	if len(caps) != 1 || caps[0] != "ebpf" {
		t.Errorf("GetCapabilities() got %v, want [ebpf]", caps)
	}
}

// ============================================================================
// LoadFromFile Edge Cases
// ============================================================================

func TestConfig_LoadFromFile_InvalidPath(t *testing.T) {
	config := NewConfig()
	err := config.LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadFromFile() should fail for non-existent file")
	}
}

// ============================================================================
// Validation Coverage - Additional Cases
// ============================================================================

func TestValidateConfig_ProxyNameRequired(t *testing.T) {
	config := &Config{
		ManagerURL:            "http://manager:8000",
		ClusterAPIKey:         "key",
		ProxyName:             "",
		Hostname:              "host",
		ListenPort:            8080,
		AdminPort:             8081,
		LogLevel:              "INFO",
		ConfigUpdateInterval:  300,
		HeartbeatInterval:     30,
		ConnectionTimeout:     30,
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("validateConfig() should fail when ProxyName is empty")
	}
	if !strings.Contains(err.Error(), "proxy_name") {
		t.Errorf("error should mention proxy_name, got: %v", err)
	}
}

func TestValidateConfig_HostnameRequired(t *testing.T) {
	config := &Config{
		ManagerURL:            "http://manager:8000",
		ClusterAPIKey:         "key",
		ProxyName:             "proxy",
		Hostname:              "",
		ListenPort:            8080,
		AdminPort:             8081,
		LogLevel:              "INFO",
		ConfigUpdateInterval:  300,
		HeartbeatInterval:     30,
		ConnectionTimeout:     30,
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("validateConfig() should fail when Hostname is empty")
	}
	if !strings.Contains(err.Error(), "hostname") {
		t.Errorf("error should mention hostname, got: %v", err)
	}
}
