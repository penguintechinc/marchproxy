package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the complete proxy configuration
type Config struct {
	// Manager connection
	ManagerURL      string `mapstructure:"manager_url"`
	ClusterAPIKey   string `mapstructure:"cluster_api_key"`
	RegistrationURL string `mapstructure:"registration_url"`

	// Server settings
	BindAddr    string `mapstructure:"bind_addr"`
	MetricsAddr string `mapstructure:"metrics_addr"`

	// Zero-Trust settings
	EnableZeroTrust bool   `mapstructure:"enable_zero_trust"`
	OpaURL          string `mapstructure:"opa_url"`
	AuditLogPath    string `mapstructure:"audit_log_path"`
	CertPath        string `mapstructure:"cert_path"`
	KeyPath         string `mapstructure:"key_path"`

	// NUMA settings
	EnableNUMA         bool  `mapstructure:"enable_numa"`
	NumaNode           int   `mapstructure:"numa_node"`
	CPUAffinityEnabled bool  `mapstructure:"cpu_affinity_enabled"`
	WorkerThreads      int   `mapstructure:"worker_threads"`

	// QoS settings
	EnableQoS          bool              `mapstructure:"enable_qos"`
	DefaultBandwidth   int64             `mapstructure:"default_bandwidth"`
	BurstSize          int64             `mapstructure:"burst_size"`
	PriorityQueueDepth int               `mapstructure:"priority_queue_depth"`
	DSCPMarking        map[string]uint8  `mapstructure:"dscp_marking"`

	// Multi-Cloud routing
	EnableMultiCloud   bool              `mapstructure:"enable_multicloud"`
	RoutingAlgorithm   string            `mapstructure:"routing_algorithm"`
	HealthCheckEnabled bool              `mapstructure:"health_check_enabled"`
	HealthCheckInterval time.Duration    `mapstructure:"health_check_interval"`
	CostOptimization   bool              `mapstructure:"cost_optimization"`
	Backends           []BackendConfig   `mapstructure:"backends"`

	// Observability
	EnableTracing    bool   `mapstructure:"enable_tracing"`
	JaegerEndpoint   string `mapstructure:"jaeger_endpoint"`
	TraceSampleRate  float64 `mapstructure:"trace_sample_rate"`
	MetricsNamespace string `mapstructure:"metrics_namespace"`

	// Acceleration
	EnableAcceleration bool   `mapstructure:"enable_acceleration"`
	AccelerationMode   string `mapstructure:"acceleration_mode"`
	XDPDevice          string `mapstructure:"xdp_device"`
	AFXDPQueueCount    int    `mapstructure:"afxdp_queue_count"`
	DPDKEnabled        bool   `mapstructure:"dpdk_enabled"`

	// Licensing
	LicenseKey      string `mapstructure:"license_key"`
	LicenseServer   string `mapstructure:"license_server"`
	ReleaseMode     bool   `mapstructure:"release_mode"`
}

// BackendConfig represents a backend server configuration
type BackendConfig struct {
	Name     string        `mapstructure:"name"`
	URL      string        `mapstructure:"url"`
	Weight   int           `mapstructure:"weight"`
	Priority int           `mapstructure:"priority"`
	Cloud    string        `mapstructure:"cloud"`
	Region   string        `mapstructure:"region"`
	Cost     float64       `mapstructure:"cost"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Set defaults
	viper.SetDefault("manager_url", "http://api-server:8000")
	viper.SetDefault("bind_addr", ":8081")
	viper.SetDefault("metrics_addr", ":8082")
	viper.SetDefault("enable_zero_trust", true)
	viper.SetDefault("opa_url", "http://opa:8181")
	viper.SetDefault("audit_log_path", "/var/log/marchproxy/audit/audit.log")
	viper.SetDefault("enable_numa", false)
	viper.SetDefault("worker_threads", 0) // Auto-detect
	viper.SetDefault("enable_qos", true)
	viper.SetDefault("default_bandwidth", 1000000000) // 1 Gbps
	viper.SetDefault("burst_size", 100000000)         // 100 MB
	viper.SetDefault("priority_queue_depth", 1000)
	viper.SetDefault("enable_multicloud", false)
	viper.SetDefault("routing_algorithm", "latency")
	viper.SetDefault("health_check_enabled", true)
	viper.SetDefault("health_check_interval", 30*time.Second)
	viper.SetDefault("cost_optimization", false)
	viper.SetDefault("enable_tracing", false)
	viper.SetDefault("trace_sample_rate", 0.1)
	viper.SetDefault("metrics_namespace", "marchproxy")
	viper.SetDefault("enable_acceleration", false)
	viper.SetDefault("acceleration_mode", "standard")
	viper.SetDefault("afxdp_queue_count", 4)
	viper.SetDefault("dpdk_enabled", false)
	viper.SetDefault("license_server", "https://license.penguintech.io")
	viper.SetDefault("release_mode", false)

	// Default DSCP mappings
	viper.SetDefault("dscp_marking", map[string]uint8{
		"P0": 46, // EF (Expedited Forwarding)
		"P1": 34, // AF41
		"P2": 18, // AF21
		"P3": 0,  // BE (Best Effort)
	})

	// Load config file if provided
	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables take precedence
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MARCHPROXY")

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
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

	if c.ClusterAPIKey == "" && os.Getenv("CLUSTER_API_KEY") == "" {
		return fmt.Errorf("cluster_api_key is required")
	}

	if c.EnableNUMA && c.NumaNode < 0 {
		return fmt.Errorf("invalid numa_node: must be >= 0")
	}

	if c.EnableQoS {
		if c.DefaultBandwidth <= 0 {
			return fmt.Errorf("default_bandwidth must be > 0")
		}
		if c.BurstSize <= 0 {
			return fmt.Errorf("burst_size must be > 0")
		}
	}

	if c.EnableMultiCloud {
		if c.RoutingAlgorithm == "" {
			return fmt.Errorf("routing_algorithm is required when multicloud is enabled")
		}
		validAlgos := map[string]bool{
			"latency": true, "cost": true, "geo": true, "roundrobin": true, "leastconn": true,
		}
		if !validAlgos[c.RoutingAlgorithm] {
			return fmt.Errorf("invalid routing_algorithm: %s", c.RoutingAlgorithm)
		}
	}

	if c.EnableAcceleration {
		validModes := map[string]bool{
			"standard": true, "xdp": true, "afxdp": true, "dpdk": true,
		}
		if !validModes[c.AccelerationMode] {
			return fmt.Errorf("invalid acceleration_mode: %s", c.AccelerationMode)
		}
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
