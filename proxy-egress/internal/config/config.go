// Package config handles configuration management for MarchProxy
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// L7Config holds configuration for L7 (HTTP) proxy via Envoy
type L7Config struct {
	Enabled         bool   `mapstructure:"enabled"`
	EnvoyBinary     string `mapstructure:"envoy_binary"`
	EnvoyConfigPath string `mapstructure:"envoy_config_path"`
	EnvoyAdminPort  int    `mapstructure:"envoy_admin_port"`
	HTTPListenPort  int    `mapstructure:"http_listen_port"`
	HTTPSListenPort int    `mapstructure:"https_listen_port"`
	HTTP3Enabled    bool   `mapstructure:"http3_enabled"` // EXPERIMENTAL
	LogLevel        string `mapstructure:"envoy_log_level"`
}

// ThreatConfig holds configuration for threat intelligence
type ThreatConfig struct {
	Enabled bool `mapstructure:"enabled"`

	// IP blocking
	IPBlockingEnabled bool `mapstructure:"ip_blocking_enabled"`
	IPCacheSize       int  `mapstructure:"ip_cache_size"`

	// Domain blocking
	DomainBlockingEnabled bool `mapstructure:"domain_blocking_enabled"`
	WildcardSupport       bool `mapstructure:"wildcard_support"`

	// URL matching
	URLMatchingEnabled bool   `mapstructure:"url_matching_enabled"`
	URLMatchEngine     string `mapstructure:"url_match_engine"` // "re2" or "boost"

	// DNS cache for resolved domain blocking
	DNSCacheEnabled bool          `mapstructure:"dns_cache_enabled"`
	DNSPositiveTTL  time.Duration `mapstructure:"dns_positive_ttl"`
	DNSNegativeTTL  time.Duration `mapstructure:"dns_negative_ttl"`
	DNSCacheSize    int           `mapstructure:"dns_cache_size"`
	DNSUpstream     []string      `mapstructure:"dns_upstream"`

	// Feed synchronization
	SyncMode         string        `mapstructure:"sync_mode"` // "grpc", "poll", or "both"
	SyncPollInterval time.Duration `mapstructure:"sync_poll_interval"`
	SyncGRPCEndpoint string        `mapstructure:"sync_grpc_endpoint"`
}

// TLSInterceptConfig holds configuration for TLS interception
type TLSInterceptConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Mode          string `mapstructure:"mode"` // "mitm" or "preconfigured"
	CACertPath    string `mapstructure:"ca_cert_path"`
	CAKeyPath     string `mapstructure:"ca_key_path"`
	CertCacheSize int    `mapstructure:"cert_cache_size"`

	// Per-domain and per-IP configuration (loaded from Manager API)
	// These are not directly set from config files but from the threat feed
}

// ExtAuthConfig holds configuration for the external authorization server
type ExtAuthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
}

// AccessControlConfig holds configuration for authentication-based access control
type AccessControlConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	DefaultRequireAuth bool `mapstructure:"default_require_auth"`
	DefaultAllow       bool `mapstructure:"default_allow"`
}

// Config holds all configuration for the proxy server
type Config struct {
	// Manager connection
	ManagerURL    string `mapstructure:"manager_url"`
	ClusterAPIKey string `mapstructure:"cluster_api_key"`

	// Proxy server settings
	ProxyName  string `mapstructure:"proxy_name"`
	Hostname   string `mapstructure:"hostname"`
	ListenPort int    `mapstructure:"listen_port"`
	AdminPort  int    `mapstructure:"admin_port"`

	// Logging configuration
	LogLevel       string `mapstructure:"log_level"`
	SyslogEndpoint string `mapstructure:"syslog_endpoint"`

	// Performance settings
	EnableEBPF    bool `mapstructure:"enable_ebpf"`
	EnableMetrics bool `mapstructure:"enable_metrics"`
	WorkerThreads int  `mapstructure:"worker_threads"`

	// Network acceleration (optional)
	EnableDPDK  bool   `mapstructure:"enable_dpdk"`
	EnableXDP   bool   `mapstructure:"enable_xdp"`
	EnableAFXDP bool   `mapstructure:"enable_af_xdp"`
	EnableSRIOV bool   `mapstructure:"enable_sriov"`
	DPDKDevices string `mapstructure:"dpdk_devices"`

	// TLS settings
	TLSCertPath string `mapstructure:"tls_cert_path"`
	TLSKeyPath  string `mapstructure:"tls_key_path"`

	// mTLS settings
	EnableMTLS            bool   `mapstructure:"enable_mtls"`
	MTLSServerCertPath    string `mapstructure:"mtls_server_cert_path"`
	MTLSServerKeyPath     string `mapstructure:"mtls_server_key_path"`
	MTLSClientCAPath      string `mapstructure:"mtls_client_ca_path"`
	MTLSClientCertPath    string `mapstructure:"mtls_client_cert_path"`
	MTLSClientKeyPath     string `mapstructure:"mtls_client_key_path"`
	MTLSRequireClientCert bool   `mapstructure:"mtls_require_client_cert"`
	MTLSVerifyClientCert  bool   `mapstructure:"mtls_verify_client_cert"`

	// License configuration
	LicenseKey string `mapstructure:"license_key"`

	// Timeouts and intervals
	ConfigUpdateInterval int `mapstructure:"config_update_interval"` // seconds
	HeartbeatInterval    int `mapstructure:"heartbeat_interval"`     // seconds
	ConnectionTimeout    int `mapstructure:"connection_timeout"`     // seconds

	// Rate limiting
	RateLimitEnabled bool `mapstructure:"rate_limit_enabled"`
	RateLimitRPS     int  `mapstructure:"rate_limit_rps"`

	// KillKrill integration
	KillKrillEnabled         bool   `mapstructure:"killkrill_enabled"`
	KillKrillLogEndpoint     string `mapstructure:"killkrill_log_endpoint"`
	KillKrillMetricsEndpoint string `mapstructure:"killkrill_metrics_endpoint"`
	KillKrillAPIKey          string `mapstructure:"killkrill_api_key"`
	KillKrillSourceName      string `mapstructure:"killkrill_source_name"`
	KillKrillApplication     string `mapstructure:"killkrill_application"`
	KillKrillBatchSize       int    `mapstructure:"killkrill_batch_size"`
	KillKrillFlushInterval   int    `mapstructure:"killkrill_flush_interval"`
	KillKrillTimeout         int    `mapstructure:"killkrill_timeout"`
	KillKrillUseHTTP3        bool   `mapstructure:"killkrill_use_http3"`
	KillKrillTLSInsecure     bool   `mapstructure:"killkrill_tls_insecure"`

	// L7 Configuration (Envoy-based HTTP proxy)
	L7 L7Config `mapstructure:"l7"`

	// Threat Intelligence Configuration
	Threat ThreatConfig `mapstructure:"threat"`

	// TLS Interception Configuration
	TLSIntercept TLSInterceptConfig `mapstructure:"tls_intercept"`

	// External Authorization Server Configuration
	ExtAuth ExtAuthConfig `mapstructure:"extauth"`

	// Access Control Configuration (authentication-based restrictions)
	AccessControl AccessControlConfig `mapstructure:"access_control"`
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	config := &Config{
		// Defaults
		LogLevel:       "info",
		ListenPort:     8080,
		AdminPort:      8081,
		EnableMetrics:  true,
		EnableEBPF:     false,
		Hostname:       getHostname(),

		// Timeout defaults
		ConnectionTimeout:    30,
		HeartbeatInterval:    60,
		ConfigUpdateInterval: 300,

		// L7 defaults
		L7: L7Config{
			Enabled:         false,
			EnvoyBinary:     "/usr/local/bin/envoy",
			EnvoyConfigPath: "/etc/envoy/envoy.yaml",
			EnvoyAdminPort:  9901,
			HTTPListenPort:  10000,
			HTTPSListenPort: 10443,
			HTTP3Enabled:    false,
			LogLevel:        "info",
		},

		// Threat defaults
		Threat: ThreatConfig{
			Enabled:               false,
			IPBlockingEnabled:     true,
			IPCacheSize:           100000,
			DomainBlockingEnabled: true,
			WildcardSupport:       true,
			URLMatchingEnabled:    true,
			URLMatchEngine:        "re2",
			DNSCacheEnabled:       true,
			DNSPositiveTTL:        5 * time.Minute,
			DNSNegativeTTL:        1 * time.Minute,
			DNSCacheSize:          10000,
			SyncMode:              "both",
			SyncPollInterval:      60 * time.Second,
		},

		// TLS Intercept defaults
		TLSIntercept: TLSInterceptConfig{
			Enabled:       false,
			Mode:          "mitm",
			CertCacheSize: 1000,
		},

		// ExtAuth defaults
		ExtAuth: ExtAuthConfig{
			Enabled: false,
			Port:    50051,
			Host:    "127.0.0.1",
		},

		// Access Control defaults
		AccessControl: AccessControlConfig{
			Enabled:            false,
			DefaultRequireAuth: false,
			DefaultAllow:       true,
		},
	}
	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check required fields
	if c.ManagerURL == "" {
		return fmt.Errorf("manager_url is required")
	}
	if c.ClusterAPIKey == "" {
		return fmt.Errorf("cluster_api_key is required")
	}

	// Check port range
	if c.ListenPort < 1 || c.ListenPort > 65535 {
		return fmt.Errorf("invalid listen port: %d (must be 1-65535)", c.ListenPort)
	}
	if c.AdminPort < 1 || c.AdminPort > 65535 {
		return fmt.Errorf("invalid admin port: %d (must be 1-65535)", c.AdminPort)
	}

	// Check port conflict
	if c.ListenPort == c.AdminPort {
		return fmt.Errorf("listen port and admin port cannot be the same port: %d", c.ListenPort)
	}

	return nil
}

// GetHostname returns the configured hostname or the system hostname
func (c *Config) GetHostname() string {
	if c.Hostname != "" {
		return c.Hostname
	}
	return getHostname()
}

// Load creates a new configuration from command line flags, environment variables, and config file
func Load(cmd *cobra.Command) (*Config, error) {
	v := viper.New()
	
	// Set defaults
	setDefaults(v)
	
	// Bind command line flags
	if err := bindFlags(v, cmd); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}
	
	// Set up environment variable handling
	v.SetEnvPrefix("MARCHPROXY")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	
	// Load config file if specified
	configFile, _ := cmd.Flags().GetString("config")
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}
	
	// Unmarshal into config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate and set derived values
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return &config, nil
}

func setDefaults(v *viper.Viper) {
	// Manager connection
	v.SetDefault("manager_url", os.Getenv("MANAGER_URL"))
	v.SetDefault("cluster_api_key", os.Getenv("CLUSTER_API_KEY"))

	// Proxy settings
	v.SetDefault("proxy_name", getHostname())
	v.SetDefault("hostname", getHostname())
	v.SetDefault("listen_port", 8080)
	v.SetDefault("admin_port", 8081)

	// Logging
	v.SetDefault("log_level", "INFO")
	v.SetDefault("syslog_endpoint", os.Getenv("SYSLOG_ENDPOINT"))

	// Performance
	v.SetDefault("enable_ebpf", true)
	v.SetDefault("enable_metrics", true)
	v.SetDefault("worker_threads", 0) // 0 = auto-detect based on CPU cores

	// Network acceleration (disabled by default)
	v.SetDefault("enable_dpdk", false)
	v.SetDefault("enable_xdp", false)
	v.SetDefault("enable_af_xdp", false)
	v.SetDefault("enable_sriov", false)
	v.SetDefault("dpdk_devices", "")

	// TLS
	v.SetDefault("tls_cert_path", "/app/certs/cert.pem")
	v.SetDefault("tls_key_path", "/app/certs/key.pem")

	// mTLS
	v.SetDefault("enable_mtls", getBoolEnv("MTLS_ENABLED", false))
	v.SetDefault("mtls_server_cert_path", os.Getenv("MTLS_SERVER_CERT_PATH"))
	v.SetDefault("mtls_server_key_path", os.Getenv("MTLS_SERVER_KEY_PATH"))
	v.SetDefault("mtls_client_ca_path", os.Getenv("MTLS_CLIENT_CA_PATH"))
	v.SetDefault("mtls_client_cert_path", os.Getenv("MTLS_CLIENT_CERT_PATH"))
	v.SetDefault("mtls_client_key_path", os.Getenv("MTLS_CLIENT_KEY_PATH"))
	v.SetDefault("mtls_require_client_cert", getBoolEnv("MTLS_REQUIRE_CLIENT_CERT", true))
	v.SetDefault("mtls_verify_client_cert", getBoolEnv("MTLS_VERIFY_CLIENT_CERT", true))

	// License
	v.SetDefault("license_key", os.Getenv("LICENSE_KEY"))

	// Intervals and timeouts
	v.SetDefault("config_update_interval", 60) // 60 seconds
	v.SetDefault("heartbeat_interval", 30)     // 30 seconds
	v.SetDefault("connection_timeout", 30)     // 30 seconds

	// Rate limiting
	v.SetDefault("rate_limit_enabled", false)
	v.SetDefault("rate_limit_rps", 1000)

	// KillKrill integration
	v.SetDefault("killkrill_enabled", getBoolEnv("KILLKRILL_ENABLED", false))
	v.SetDefault("killkrill_log_endpoint", os.Getenv("KILLKRILL_LOG_ENDPOINT"))
	v.SetDefault("killkrill_metrics_endpoint", os.Getenv("KILLKRILL_METRICS_ENDPOINT"))
	v.SetDefault("killkrill_api_key", os.Getenv("KILLKRILL_API_KEY"))
	v.SetDefault("killkrill_source_name", getEnvOrDefault("KILLKRILL_SOURCE_NAME", "marchproxy-"+getHostname()))
	v.SetDefault("killkrill_application", "proxy")
	v.SetDefault("killkrill_batch_size", getIntEnv("KILLKRILL_BATCH_SIZE", 100))
	v.SetDefault("killkrill_flush_interval", getIntEnv("KILLKRILL_FLUSH_INTERVAL", 10))
	v.SetDefault("killkrill_timeout", getIntEnv("KILLKRILL_TIMEOUT", 30))
	v.SetDefault("killkrill_use_http3", getBoolEnv("KILLKRILL_USE_HTTP3", true))
	v.SetDefault("killkrill_tls_insecure", getBoolEnv("KILLKRILL_TLS_INSECURE", false))

	// L7 Configuration (Envoy-based HTTP proxy)
	v.SetDefault("l7.enabled", getBoolEnv("ENVOY_ENABLED", false))
	v.SetDefault("l7.envoy_binary", getEnvOrDefault("ENVOY_BINARY", "/usr/local/bin/envoy"))
	v.SetDefault("l7.envoy_config_path", getEnvOrDefault("ENVOY_CONFIG_PATH", "/app/envoy/bootstrap.yaml"))
	v.SetDefault("l7.envoy_admin_port", getIntEnv("ENVOY_ADMIN_PORT", 9901))
	v.SetDefault("l7.http_listen_port", getIntEnv("ENVOY_HTTP_PORT", 10000))
	v.SetDefault("l7.https_listen_port", getIntEnv("ENVOY_HTTPS_PORT", 10443))
	v.SetDefault("l7.http3_enabled", getBoolEnv("ENVOY_HTTP3_ENABLED", false)) // EXPERIMENTAL - disabled by default
	v.SetDefault("l7.envoy_log_level", getEnvOrDefault("ENVOY_LOG_LEVEL", "info"))

	// Threat Intelligence Configuration
	v.SetDefault("threat.enabled", getBoolEnv("THREAT_ENABLED", true))
	v.SetDefault("threat.ip_blocking_enabled", getBoolEnv("THREAT_IP_BLOCKING_ENABLED", true))
	v.SetDefault("threat.ip_cache_size", getIntEnv("THREAT_IP_CACHE_SIZE", 100000))
	v.SetDefault("threat.domain_blocking_enabled", getBoolEnv("THREAT_DOMAIN_BLOCKING_ENABLED", true))
	v.SetDefault("threat.wildcard_support", getBoolEnv("THREAT_WILDCARD_SUPPORT", true))
	v.SetDefault("threat.url_matching_enabled", getBoolEnv("THREAT_URL_MATCHING_ENABLED", true))
	v.SetDefault("threat.url_match_engine", getEnvOrDefault("THREAT_URL_MATCH_ENGINE", "re2"))
	v.SetDefault("threat.dns_cache_enabled", getBoolEnv("THREAT_DNS_CACHE_ENABLED", true))
	v.SetDefault("threat.dns_positive_ttl", getDurationEnv("THREAT_DNS_POSITIVE_TTL", 5*time.Minute))
	v.SetDefault("threat.dns_negative_ttl", getDurationEnv("THREAT_DNS_NEGATIVE_TTL", 1*time.Minute))
	v.SetDefault("threat.dns_cache_size", getIntEnv("THREAT_DNS_CACHE_SIZE", 50000))
	v.SetDefault("threat.dns_upstream", getStringSliceEnv("THREAT_DNS_UPSTREAM", []string{"8.8.8.8:53", "1.1.1.1:53"}))
	v.SetDefault("threat.sync_mode", getEnvOrDefault("THREAT_SYNC_MODE", "both"))
	v.SetDefault("threat.sync_poll_interval", getDurationEnv("THREAT_SYNC_POLL_INTERVAL", 60*time.Second))
	v.SetDefault("threat.sync_grpc_endpoint", os.Getenv("THREAT_SYNC_GRPC_ENDPOINT"))

	// TLS Interception Configuration
	v.SetDefault("tls_intercept.enabled", getBoolEnv("TLS_INTERCEPT_ENABLED", false))
	v.SetDefault("tls_intercept.mode", getEnvOrDefault("TLS_INTERCEPT_MODE", "mitm"))
	v.SetDefault("tls_intercept.ca_cert_path", getEnvOrDefault("TLS_INTERCEPT_CA_CERT", "/app/certs/ca.crt"))
	v.SetDefault("tls_intercept.ca_key_path", getEnvOrDefault("TLS_INTERCEPT_CA_KEY", "/app/certs/ca.key"))
	v.SetDefault("tls_intercept.cert_cache_size", getIntEnv("TLS_INTERCEPT_CACHE_SIZE", 10000))

	// External Authorization Server Configuration
	v.SetDefault("extauth.enabled", getBoolEnv("EXTAUTH_ENABLED", true))
	v.SetDefault("extauth.port", getIntEnv("EXTAUTH_PORT", 9002))
	v.SetDefault("extauth.host", getEnvOrDefault("EXTAUTH_HOST", "127.0.0.1"))

	// Access Control Configuration
	v.SetDefault("access_control.enabled", getBoolEnv("ACCESS_CONTROL_ENABLED", false))
	v.SetDefault("access_control.default_require_auth", getBoolEnv("ACCESS_CONTROL_DEFAULT_REQUIRE_AUTH", false))
	v.SetDefault("access_control.default_allow", getBoolEnv("ACCESS_CONTROL_DEFAULT_ALLOW", true))
}

func bindFlags(v *viper.Viper, cmd *cobra.Command) error {
	// Bind specific flags that override config file and env vars
	flagBindings := map[string]string{
		"manager-url":      "manager_url",
		"cluster-api-key":  "cluster_api_key",
		"listen-port":      "listen_port",
		"admin-port":       "admin_port",
		"log-level":        "log_level",
		"enable-ebpf":      "enable_ebpf",
		"enable-metrics":   "enable_metrics",
	}
	
	for flag, configKey := range flagBindings {
		if err := v.BindPFlag(configKey, cmd.Flags().Lookup(flag)); err != nil {
			return err
		}
	}
	
	return nil
}

func validateConfig(config *Config) error {
	// Required settings
	if config.ManagerURL == "" {
		return fmt.Errorf("manager_url is required")
	}
	
	if config.ClusterAPIKey == "" {
		return fmt.Errorf("cluster_api_key is required")
	}
	
	if config.ProxyName == "" {
		return fmt.Errorf("proxy_name is required")
	}
	
	if config.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	
	// Port validation
	if config.ListenPort <= 0 || config.ListenPort > 65535 {
		return fmt.Errorf("invalid listen_port: %d", config.ListenPort)
	}
	
	if config.AdminPort <= 0 || config.AdminPort > 65535 {
		return fmt.Errorf("invalid admin_port: %d", config.AdminPort)
	}
	
	if config.ListenPort == config.AdminPort {
		return fmt.Errorf("listen_port and admin_port cannot be the same")
	}
	
	// Log level validation
	validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if config.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		return fmt.Errorf("invalid log_level: %s (must be one of: %v)", config.LogLevel, validLogLevels)
	}
	
	// Worker threads validation
	if config.WorkerThreads < 0 {
		return fmt.Errorf("worker_threads cannot be negative")
	}
	
	// Auto-detect worker threads if set to 0
	if config.WorkerThreads == 0 {
		// Use GOMAXPROCS or detect CPU cores
		if gomaxprocs := os.Getenv("GOMAXPROCS"); gomaxprocs != "" {
			if threads, err := strconv.Atoi(gomaxprocs); err == nil && threads > 0 {
				config.WorkerThreads = threads
			}
		}
		// If still 0, will be set based on runtime.NumCPU() in the proxy server
	}
	
	// Interval validation
	if config.ConfigUpdateInterval < 10 {
		return fmt.Errorf("config_update_interval must be at least 10 seconds")
	}
	
	if config.HeartbeatInterval < 5 {
		return fmt.Errorf("heartbeat_interval must be at least 5 seconds")
	}
	
	if config.ConnectionTimeout < 1 {
		return fmt.Errorf("connection_timeout must be at least 1 second")
	}
	
	// Rate limiting validation
	if config.RateLimitEnabled && config.RateLimitRPS <= 0 {
		return fmt.Errorf("rate_limit_rps must be positive when rate limiting is enabled")
	}

	// mTLS validation
	if config.EnableMTLS {
		if config.MTLSServerCertPath == "" {
			return fmt.Errorf("mtls_server_cert_path is required when mTLS is enabled")
		}
		if config.MTLSServerKeyPath == "" {
			return fmt.Errorf("mtls_server_key_path is required when mTLS is enabled")
		}
		if config.MTLSRequireClientCert && config.MTLSClientCAPath == "" {
			return fmt.Errorf("mtls_client_ca_path is required when client certificate validation is enabled")
		}
	}

	// L7 (Envoy) validation
	if config.L7.Enabled {
		if config.L7.EnvoyBinary == "" {
			return fmt.Errorf("l7.envoy_binary is required when L7 proxy is enabled")
		}
		if config.L7.EnvoyConfigPath == "" {
			return fmt.Errorf("l7.envoy_config_path is required when L7 proxy is enabled")
		}
		if config.L7.EnvoyAdminPort <= 0 || config.L7.EnvoyAdminPort > 65535 {
			return fmt.Errorf("invalid l7.envoy_admin_port: %d", config.L7.EnvoyAdminPort)
		}
		if config.L7.HTTPListenPort <= 0 || config.L7.HTTPListenPort > 65535 {
			return fmt.Errorf("invalid l7.http_listen_port: %d", config.L7.HTTPListenPort)
		}
		if config.L7.HTTPSListenPort <= 0 || config.L7.HTTPSListenPort > 65535 {
			return fmt.Errorf("invalid l7.https_listen_port: %d", config.L7.HTTPSListenPort)
		}
		// Validate log level for Envoy
		validEnvoyLogLevels := []string{"trace", "debug", "info", "warning", "error", "critical", "off"}
		isValidEnvoyLogLevel := false
		for _, level := range validEnvoyLogLevels {
			if config.L7.LogLevel == level {
				isValidEnvoyLogLevel = true
				break
			}
		}
		if !isValidEnvoyLogLevel {
			return fmt.Errorf("invalid l7.envoy_log_level: %s (must be one of: %v)", config.L7.LogLevel, validEnvoyLogLevels)
		}
	}

	// Threat intelligence validation
	if config.Threat.Enabled {
		if config.Threat.IPCacheSize < 0 {
			return fmt.Errorf("threat.ip_cache_size cannot be negative")
		}
		if config.Threat.DNSCacheSize < 0 {
			return fmt.Errorf("threat.dns_cache_size cannot be negative")
		}
		if config.Threat.SyncMode != "" && config.Threat.SyncMode != "grpc" && config.Threat.SyncMode != "poll" && config.Threat.SyncMode != "both" {
			return fmt.Errorf("invalid threat.sync_mode: %s (must be 'grpc', 'poll', or 'both')", config.Threat.SyncMode)
		}
		if config.Threat.URLMatchEngine != "" && config.Threat.URLMatchEngine != "re2" && config.Threat.URLMatchEngine != "boost" {
			return fmt.Errorf("invalid threat.url_match_engine: %s (must be 're2' or 'boost')", config.Threat.URLMatchEngine)
		}
	}

	// TLS interception validation
	if config.TLSIntercept.Enabled {
		if config.TLSIntercept.Mode != "mitm" && config.TLSIntercept.Mode != "preconfigured" {
			return fmt.Errorf("invalid tls_intercept.mode: %s (must be 'mitm' or 'preconfigured')", config.TLSIntercept.Mode)
		}
		if config.TLSIntercept.Mode == "mitm" {
			if config.TLSIntercept.CACertPath == "" {
				return fmt.Errorf("tls_intercept.ca_cert_path is required when TLS interception is in MITM mode")
			}
			if config.TLSIntercept.CAKeyPath == "" {
				return fmt.Errorf("tls_intercept.ca_key_path is required when TLS interception is in MITM mode")
			}
		}
		if config.TLSIntercept.CertCacheSize < 0 {
			return fmt.Errorf("tls_intercept.cert_cache_size cannot be negative")
		}
	}

	// External authorization validation
	if config.ExtAuth.Enabled {
		if config.ExtAuth.Port <= 0 || config.ExtAuth.Port > 65535 {
			return fmt.Errorf("invalid extauth.port: %d", config.ExtAuth.Port)
		}
	}

	return nil
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Split by comma and trim spaces
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

// GetListenAddress returns the full listen address for the proxy
func (c *Config) GetListenAddress() string {
	return fmt.Sprintf(":%d", c.ListenPort)
}

// GetAdminAddress returns the full admin/metrics address
func (c *Config) GetAdminAddress() string {
	return fmt.Sprintf(":%d", c.AdminPort)
}

// IsNetworkAccelerationEnabled returns true if any network acceleration is enabled
func (c *Config) IsNetworkAccelerationEnabled() bool {
	return c.EnableDPDK || c.EnableXDP || c.EnableAFXDP || c.EnableSRIOV
}

// GetCapabilities returns a list of enabled capabilities
func (c *Config) GetCapabilities() []string {
	capabilities := []string{}

	if c.EnableEBPF {
		capabilities = append(capabilities, "ebpf")
	}

	if c.EnableDPDK {
		capabilities = append(capabilities, "dpdk")
	}

	if c.EnableXDP {
		capabilities = append(capabilities, "xdp")
	}

	if c.EnableAFXDP {
		capabilities = append(capabilities, "af_xdp")
	}

	if c.EnableSRIOV {
		capabilities = append(capabilities, "sr_iov")
	}

	if c.EnableMTLS {
		capabilities = append(capabilities, "mtls")
	}

	// L7 capabilities
	if c.L7.Enabled {
		capabilities = append(capabilities, "l7_proxy")
		capabilities = append(capabilities, "http1")
		capabilities = append(capabilities, "http2")
		if c.L7.HTTP3Enabled {
			capabilities = append(capabilities, "http3_experimental")
		}
	}

	// Threat intelligence capabilities
	if c.Threat.Enabled {
		capabilities = append(capabilities, "threat_intel")
		if c.Threat.IPBlockingEnabled {
			capabilities = append(capabilities, "ip_blocking")
		}
		if c.Threat.DomainBlockingEnabled {
			capabilities = append(capabilities, "domain_blocking")
		}
		if c.Threat.URLMatchingEnabled {
			capabilities = append(capabilities, "url_matching")
		}
	}

	// TLS interception
	if c.TLSIntercept.Enabled {
		capabilities = append(capabilities, "tls_intercept")
	}

	// Access control
	if c.AccessControl.Enabled {
		capabilities = append(capabilities, "access_control")
	}

	return capabilities
}

// IsMTLSEnabled returns true if mTLS is enabled and properly configured
func (c *Config) IsMTLSEnabled() bool {
	if !c.EnableMTLS {
		return false
	}
	// mTLS requires server cert, server key, and client CA path
	return c.MTLSServerCertPath != "" && c.MTLSServerKeyPath != "" && c.MTLSClientCAPath != ""
}

// GetMTLSConfig returns the mTLS configuration paths
func (c *Config) GetMTLSConfig() (serverCert, serverKey, clientCA string) {
	return c.MTLSServerCertPath, c.MTLSServerKeyPath, c.MTLSClientCAPath
}

// RequiresClientCert returns true if client certificates are required
func (c *Config) RequiresClientCert() bool {
	return c.EnableMTLS && c.MTLSRequireClientCert
}

// ShouldVerifyClientCert returns true if client certificates should be verified
func (c *Config) ShouldVerifyClientCert() bool {
	return c.EnableMTLS && c.MTLSVerifyClientCert
}

// GetKillKrillConfig returns a KillKrill client configuration based on the proxy config
func (c *Config) GetKillKrillConfig() *map[string]interface{} {
	if !c.KillKrillEnabled {
		return nil
	}

	return &map[string]interface{}{
		"enabled":          c.KillKrillEnabled,
		"log_endpoint":     c.KillKrillLogEndpoint,
		"metrics_endpoint": c.KillKrillMetricsEndpoint,
		"api_key":          c.KillKrillAPIKey,
		"source_name":      c.KillKrillSourceName,
		"application":      c.KillKrillApplication,
		"batch_size":       c.KillKrillBatchSize,
		"flush_interval":   c.KillKrillFlushInterval,
		"timeout":          c.KillKrillTimeout,
		"use_http3":        c.KillKrillUseHTTP3,
		"tls_insecure":     c.KillKrillTLSInsecure,
	}
}

// IsL7Enabled returns true if L7 (HTTP) proxy is enabled
func (c *Config) IsL7Enabled() bool {
	return c.L7.Enabled
}

// IsHTTP3Enabled returns true if HTTP/3 (QUIC) is enabled (EXPERIMENTAL)
func (c *Config) IsHTTP3Enabled() bool {
	return c.L7.Enabled && c.L7.HTTP3Enabled
}

// GetEnvoyConfig returns the Envoy configuration
func (c *Config) GetEnvoyConfig() (binary, configPath string, adminPort int) {
	return c.L7.EnvoyBinary, c.L7.EnvoyConfigPath, c.L7.EnvoyAdminPort
}

// IsThreatEnabled returns true if threat intelligence is enabled
func (c *Config) IsThreatEnabled() bool {
	return c.Threat.Enabled
}

// IsTLSInterceptEnabled returns true if TLS interception is enabled
func (c *Config) IsTLSInterceptEnabled() bool {
	return c.TLSIntercept.Enabled
}

// GetTLSInterceptCAConfig returns the CA certificate paths for TLS interception
func (c *Config) GetTLSInterceptCAConfig() (certPath, keyPath string) {
	return c.TLSIntercept.CACertPath, c.TLSIntercept.CAKeyPath
}

// GetTLSInterceptMode returns the TLS interception mode ("mitm" or "preconfigured")
func (c *Config) GetTLSInterceptMode() string {
	return c.TLSIntercept.Mode
}

// IsAccessControlEnabled returns true if authentication-based access control is enabled
func (c *Config) IsAccessControlEnabled() bool {
	return c.AccessControl.Enabled
}

// GetExtAuthAddress returns the external authorization server address
func (c *Config) GetExtAuthAddress() string {
	return fmt.Sprintf("%s:%d", c.ExtAuth.Host, c.ExtAuth.Port)
}

// IsExtAuthEnabled returns true if external authorization is enabled
func (c *Config) IsExtAuthEnabled() bool {
	return c.ExtAuth.Enabled
}

// LoadFromEnvironment loads configuration from environment variables
func (c *Config) LoadFromEnvironment() error {
	// Load from environment variables with MARCHPROXY_ prefix
	if url := os.Getenv("MANAGER_URL"); url != "" {
		c.ManagerURL = url
	}
	if key := os.Getenv("CLUSTER_API_KEY"); key != "" {
		c.ClusterAPIKey = key
	}
	if name := os.Getenv("PROXY_NAME"); name != "" {
		c.ProxyName = name
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.LogLevel = level
	}
	if port := os.Getenv("LISTEN_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.ListenPort = p
		}
	}
	if ebpf := os.Getenv("ENABLE_EBPF"); ebpf != "" {
		c.EnableEBPF = ebpf == "true" || ebpf == "1"
	}
	if metrics := os.Getenv("ENABLE_METRICS"); metrics != "" {
		c.EnableMetrics = metrics == "true" || metrics == "1"
	}
	return nil
}

// LoadFromFile loads configuration from a YAML file
func (c *Config) LoadFromFile(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	if err := v.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// IsAccelerationEnabled returns true if any hardware acceleration is enabled
func (c *Config) IsAccelerationEnabled() bool {
	return c.EnableEBPF || c.EnableDPDK || c.EnableXDP || c.EnableAFXDP || c.EnableSRIOV
}

// GetWorkerThreads returns the number of worker threads
func (c *Config) GetWorkerThreads() int {
	if c.WorkerThreads > 0 {
		return c.WorkerThreads
	}
	return 4 // default
}

// IsTLSEnabled returns true if TLS is enabled
func (c *Config) IsTLSEnabled() bool {
	return c.TLSCertPath != "" && c.TLSKeyPath != ""
}