package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the ALB configuration
type Config struct {
	// Module identification
	ModuleID   string
	ModuleType string
	Version    string

	// Envoy configuration
	EnvoyBinary      string
	EnvoyConfigPath  string
	EnvoyAdminPort   int
	EnvoyListenPort  int
	EnvoyLogLevel    string

	// xDS configuration
	XDSServerAddr    string
	XDSNodeID        string
	XDSCluster       string
	XDSConnectTimeout time.Duration

	// gRPC server configuration
	GRPCPort         int
	GRPCMaxConnAge   time.Duration

	// Monitoring
	MetricsPort      int
	HealthCheckPort  int

	// Lifecycle
	ShutdownTimeout  time.Duration
	ReloadGracePeriod time.Duration

	// License
	LicenseKey       string
	ClusterAPIKey    string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ModuleID:         getEnv("MODULE_ID", "alb-1"),
		ModuleType:       "ALB",
		Version:          getEnv("VERSION", "v1.0.0"),

		EnvoyBinary:      getEnv("ENVOY_BINARY", "/usr/local/bin/envoy"),
		EnvoyConfigPath:  getEnv("ENVOY_CONFIG_PATH", "/etc/envoy/envoy.yaml"),
		EnvoyAdminPort:   getEnvAsInt("ENVOY_ADMIN_PORT", 9901),
		EnvoyListenPort:  getEnvAsInt("ENVOY_LISTEN_PORT", 10000),
		EnvoyLogLevel:    getEnv("ENVOY_LOG_LEVEL", "info"),

		XDSServerAddr:    getEnv("XDS_SERVER", "api-server:18000"),
		XDSNodeID:        getEnv("XDS_NODE_ID", "alb-node"),
		XDSCluster:       getEnv("XDS_CLUSTER", "marchproxy-cluster"),
		XDSConnectTimeout: getEnvAsDuration("XDS_CONNECT_TIMEOUT", 5*time.Second),

		GRPCPort:         getEnvAsInt("GRPC_PORT", 50051),
		GRPCMaxConnAge:   getEnvAsDuration("GRPC_MAX_CONN_AGE", 30*time.Minute),

		MetricsPort:      getEnvAsInt("METRICS_PORT", 9090),
		HealthCheckPort:  getEnvAsInt("HEALTH_PORT", 8080),

		ShutdownTimeout:  getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		ReloadGracePeriod: getEnvAsDuration("RELOAD_GRACE_PERIOD", 5*time.Second),

		LicenseKey:       getEnv("LICENSE_KEY", ""),
		ClusterAPIKey:    getEnv("CLUSTER_API_KEY", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ModuleID == "" {
		return fmt.Errorf("MODULE_ID cannot be empty")
	}

	if c.EnvoyBinary == "" {
		return fmt.Errorf("ENVOY_BINARY cannot be empty")
	}

	if c.EnvoyConfigPath == "" {
		return fmt.Errorf("ENVOY_CONFIG_PATH cannot be empty")
	}

	if c.XDSServerAddr == "" {
		return fmt.Errorf("XDS_SERVER cannot be empty")
	}

	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("GRPC_PORT must be between 1 and 65535")
	}

	if c.EnvoyAdminPort < 1 || c.EnvoyAdminPort > 65535 {
		return fmt.Errorf("ENVOY_ADMIN_PORT must be between 1 and 65535")
	}

	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvAsDuration gets an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
