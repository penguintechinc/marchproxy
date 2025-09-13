// Package config handles configuration management for MarchProxy
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the proxy server
type Config struct {
	// Manager connection
	ManagerURL     string `mapstructure:"manager_url"`
	ClusterAPIKey  string `mapstructure:"cluster_api_key"`
	
	// Proxy server settings
	ProxyName      string `mapstructure:"proxy_name"`
	Hostname       string `mapstructure:"hostname"`
	ListenPort     int    `mapstructure:"listen_port"`
	AdminPort      int    `mapstructure:"admin_port"`
	
	// Logging configuration
	LogLevel       string `mapstructure:"log_level"`
	SyslogEndpoint string `mapstructure:"syslog_endpoint"`
	
	// Performance settings
	EnableEBPF     bool `mapstructure:"enable_ebpf"`
	EnableMetrics  bool `mapstructure:"enable_metrics"`
	WorkerThreads  int  `mapstructure:"worker_threads"`
	
	// Network acceleration (optional)
	EnableDPDK     bool   `mapstructure:"enable_dpdk"`
	EnableXDP      bool   `mapstructure:"enable_xdp"`
	EnableAFXDP    bool   `mapstructure:"enable_af_xdp"`
	EnableSRIOV    bool   `mapstructure:"enable_sriov"`
	DPDKDevices    string `mapstructure:"dpdk_devices"`
	
	// TLS settings
	TLSCertPath    string `mapstructure:"tls_cert_path"`
	TLSKeyPath     string `mapstructure:"tls_key_path"`
	
	// License configuration
	LicenseKey     string `mapstructure:"license_key"`
	
	// Timeouts and intervals
	ConfigUpdateInterval int `mapstructure:"config_update_interval"` // seconds
	HeartbeatInterval    int `mapstructure:"heartbeat_interval"`     // seconds
	ConnectionTimeout    int `mapstructure:"connection_timeout"`     // seconds
	
	// Rate limiting
	RateLimitEnabled bool `mapstructure:"rate_limit_enabled"`
	RateLimitRPS     int  `mapstructure:"rate_limit_rps"`
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
	
	// License
	v.SetDefault("license_key", os.Getenv("LICENSE_KEY"))
	
	// Intervals and timeouts
	v.SetDefault("config_update_interval", 60) // 60 seconds
	v.SetDefault("heartbeat_interval", 30)     // 30 seconds
	v.SetDefault("connection_timeout", 30)     // 30 seconds
	
	// Rate limiting
	v.SetDefault("rate_limit_enabled", false)
	v.SetDefault("rate_limit_rps", 1000)
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
	
	return nil
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
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
	
	return capabilities
}