package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the NLB configuration
type Config struct {
	// Server settings
	BindAddr        string `mapstructure:"bind_addr"`
	GRPCAddr        string `mapstructure:"grpc_addr"`
	GRPCPort        int    `mapstructure:"grpc_port"`
	MetricsAddr     string `mapstructure:"metrics_addr"`

	// Manager connection
	ManagerURL      string `mapstructure:"manager_url"`
	ClusterAPIKey   string `mapstructure:"cluster_api_key"`
	RegistrationURL string `mapstructure:"registration_url"`

	// Traffic management
	EnableRateLimiting bool              `mapstructure:"enable_rate_limiting"`
	DefaultRateLimit   float64           `mapstructure:"default_rate_limit"`
	DefaultBurstSize   float64           `mapstructure:"default_burst_size"`
	RateLimitBuckets   []RateLimitConfig `mapstructure:"rate_limit_buckets"`

	// Autoscaling
	EnableAutoscaling      bool          `mapstructure:"enable_autoscaling"`
	AutoscaleInterval      time.Duration `mapstructure:"autoscale_interval"`
	ScaleUpCooldown        time.Duration `mapstructure:"scale_up_cooldown"`
	ScaleDownCooldown      time.Duration `mapstructure:"scale_down_cooldown"`

	// Blue/Green deployments
	EnableBlueGreen        bool          `mapstructure:"enable_bluegreen"`
	CanaryStepSize         int           `mapstructure:"canary_step_size"`
	CanaryStepDuration     time.Duration `mapstructure:"canary_step_duration"`

	// Module management
	MaxModulesPerProtocol  int           `mapstructure:"max_modules_per_protocol"`
	ModuleHealthCheckInterval time.Duration `mapstructure:"module_health_check_interval"`

	// Observability
	EnableTracing       bool   `mapstructure:"enable_tracing"`
	JaegerEndpoint      string `mapstructure:"jaeger_endpoint"`
	TraceSampleRate     float64 `mapstructure:"trace_sample_rate"`
	MetricsNamespace    string `mapstructure:"metrics_namespace"`

	// Licensing
	LicenseKey      string `mapstructure:"license_key"`
	LicenseServer   string `mapstructure:"license_server"`
	ReleaseMode     bool   `mapstructure:"release_mode"`

	// Advanced features
	EnableConnectionPooling bool `mapstructure:"enable_connection_pooling"`
	MaxConnectionsPerModule int  `mapstructure:"max_connections_per_module"`
}

// RateLimitConfig defines rate limiting for a specific bucket
type RateLimitConfig struct {
	Name       string  `mapstructure:"name"`
	Protocol   string  `mapstructure:"protocol"`
	Capacity   float64 `mapstructure:"capacity"`
	RefillRate float64 `mapstructure:"refill_rate"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Set defaults
	viper.SetDefault("bind_addr", ":8080")
	viper.SetDefault("grpc_addr", "0.0.0.0")
	viper.SetDefault("grpc_port", 50051)
	viper.SetDefault("metrics_addr", ":8082")
	viper.SetDefault("manager_url", "http://api-server:8000")

	// Rate limiting defaults
	viper.SetDefault("enable_rate_limiting", true)
	viper.SetDefault("default_rate_limit", 10000.0) // 10k requests per second
	viper.SetDefault("default_burst_size", 20000.0)

	// Autoscaling defaults
	viper.SetDefault("enable_autoscaling", true)
	viper.SetDefault("autoscale_interval", 30*time.Second)
	viper.SetDefault("scale_up_cooldown", 3*time.Minute)
	viper.SetDefault("scale_down_cooldown", 5*time.Minute)

	// Blue/Green defaults
	viper.SetDefault("enable_bluegreen", true)
	viper.SetDefault("canary_step_size", 10) // 10% increments
	viper.SetDefault("canary_step_duration", 2*time.Minute)

	// Module management defaults
	viper.SetDefault("max_modules_per_protocol", 50)
	viper.SetDefault("module_health_check_interval", 10*time.Second)

	// Observability defaults
	viper.SetDefault("enable_tracing", false)
	viper.SetDefault("trace_sample_rate", 0.1)
	viper.SetDefault("metrics_namespace", "marchproxy_nlb")

	// Licensing defaults
	viper.SetDefault("license_server", "https://license.penguintech.io")
	viper.SetDefault("release_mode", false)

	// Advanced features defaults
	viper.SetDefault("enable_connection_pooling", true)
	viper.SetDefault("max_connections_per_module", 10000)

	// Load config file if provided
	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables take precedence
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MARCHPROXY_NLB")

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override from environment if set
	if apiKey := os.Getenv("CLUSTER_API_KEY"); apiKey != "" {
		cfg.ClusterAPIKey = apiKey
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ManagerURL == "" {
		return fmt.Errorf("manager_url is required")
	}

	if c.ClusterAPIKey == "" {
		return fmt.Errorf("cluster_api_key is required")
	}

	if c.GRPCPort <= 0 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid grpc_port: must be 1-65535")
	}

	if c.EnableRateLimiting {
		if c.DefaultRateLimit <= 0 {
			return fmt.Errorf("default_rate_limit must be > 0")
		}
		if c.DefaultBurstSize <= 0 {
			return fmt.Errorf("default_burst_size must be > 0")
		}
	}

	if c.EnableAutoscaling {
		if c.AutoscaleInterval <= 0 {
			return fmt.Errorf("autoscale_interval must be > 0")
		}
		if c.ScaleUpCooldown <= 0 || c.ScaleDownCooldown <= 0 {
			return fmt.Errorf("scale cooldown periods must be > 0")
		}
	}

	if c.EnableBlueGreen {
		if c.CanaryStepSize <= 0 || c.CanaryStepSize > 100 {
			return fmt.Errorf("canary_step_size must be 1-100")
		}
		if c.CanaryStepDuration <= 0 {
			return fmt.Errorf("canary_step_duration must be > 0")
		}
	}

	if c.MaxModulesPerProtocol <= 0 {
		return fmt.Errorf("max_modules_per_protocol must be > 0")
	}

	if c.MaxConnectionsPerModule <= 0 {
		return fmt.Errorf("max_connections_per_module must be > 0")
	}

	return nil
}

// IsEnterpriseFeatureEnabled checks if an enterprise feature is enabled
func (c *Config) IsEnterpriseFeatureEnabled(feature string) bool {
	// In release mode, check license
	if c.ReleaseMode {
		// TODO: Validate with license server
		return c.LicenseKey != ""
	}
	// In development mode, all features available
	return true
}
