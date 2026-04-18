//go:build ci

package config

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errMsg    string
	}{
		{
			name: "Valid config",
			config: &Config{
				ManagerURL:             "http://manager:8000",
				ClusterAPIKey:          "test-key",
				GRPCPort:               50051,
				EnableRateLimiting:     true,
				DefaultRateLimit:       10000.0,
				DefaultBurstSize:       20000.0,
				EnableAutoscaling:      true,
				AutoscaleInterval:      30 * time.Second,
				ScaleUpCooldown:        3 * time.Minute,
				ScaleDownCooldown:      5 * time.Minute,
				EnableBlueGreen:        true,
				CanaryStepSize:         10,
				CanaryStepDuration:     2 * time.Minute,
				MaxModulesPerProtocol:  50,
				MaxConnectionsPerModule: 10000,
			},
			wantError: false,
		},
		{
			name: "Missing manager URL",
			config: &Config{
				ManagerURL:    "",
				ClusterAPIKey: "test-key",
				GRPCPort:      50051,
			},
			wantError: true,
			errMsg:    "manager_url is required",
		},
		{
			name: "Missing API key",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "",
				GRPCPort:      50051,
			},
			wantError: true,
			errMsg:    "cluster_api_key is required",
		},
		{
			name: "Invalid gRPC port - zero",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "test-key",
				GRPCPort:      0,
			},
			wantError: true,
			errMsg:    "invalid grpc_port",
		},
		{
			name: "Invalid gRPC port - too high",
			config: &Config{
				ManagerURL:    "http://manager:8000",
				ClusterAPIKey: "test-key",
				GRPCPort:      65536,
			},
			wantError: true,
			errMsg:    "invalid grpc_port",
		},
		{
			name: "Invalid rate limit - disabled config allowed",
			config: &Config{
				ManagerURL:              "http://manager:8000",
				ClusterAPIKey:           "test-key",
				GRPCPort:                50051,
				EnableRateLimiting:      false,
				DefaultRateLimit:        0,
				DefaultBurstSize:        0,
				MaxModulesPerProtocol:   50,
				MaxConnectionsPerModule: 10000,
			},
			wantError: false,
		},
		{
			name: "Invalid rate limit - zero rate",
			config: &Config{
				ManagerURL:         "http://manager:8000",
				ClusterAPIKey:      "test-key",
				GRPCPort:           50051,
				EnableRateLimiting: true,
				DefaultRateLimit:   0,
				DefaultBurstSize:   20000.0,
			},
			wantError: true,
			errMsg:    "default_rate_limit must be > 0",
		},
		{
			name: "Invalid rate limit - zero burst",
			config: &Config{
				ManagerURL:         "http://manager:8000",
				ClusterAPIKey:      "test-key",
				GRPCPort:           50051,
				EnableRateLimiting: true,
				DefaultRateLimit:   10000.0,
				DefaultBurstSize:   0,
			},
			wantError: true,
			errMsg:    "default_burst_size must be > 0",
		},
		{
			name: "Invalid autoscale interval",
			config: &Config{
				ManagerURL:        "http://manager:8000",
				ClusterAPIKey:     "test-key",
				GRPCPort:          50051,
				EnableAutoscaling: true,
				AutoscaleInterval: 0,
			},
			wantError: true,
			errMsg:    "autoscale_interval must be > 0",
		},
		{
			name: "Invalid scale up cooldown",
			config: &Config{
				ManagerURL:        "http://manager:8000",
				ClusterAPIKey:     "test-key",
				GRPCPort:          50051,
				EnableAutoscaling: true,
				AutoscaleInterval: 30 * time.Second,
				ScaleUpCooldown:   0,
				ScaleDownCooldown: 5 * time.Minute,
			},
			wantError: true,
			errMsg:    "scale cooldown",
		},
		{
			name: "Invalid canary step size - zero",
			config: &Config{
				ManagerURL:         "http://manager:8000",
				ClusterAPIKey:      "test-key",
				GRPCPort:           50051,
				EnableBlueGreen:    true,
				CanaryStepSize:     0,
				CanaryStepDuration: 2 * time.Minute,
			},
			wantError: true,
			errMsg:    "canary_step_size must be 1-100",
		},
		{
			name: "Invalid canary step size - too high",
			config: &Config{
				ManagerURL:         "http://manager:8000",
				ClusterAPIKey:      "test-key",
				GRPCPort:           50051,
				EnableBlueGreen:    true,
				CanaryStepSize:     101,
				CanaryStepDuration: 2 * time.Minute,
			},
			wantError: true,
			errMsg:    "canary_step_size must be 1-100",
		},
		{
			name: "Invalid canary step duration",
			config: &Config{
				ManagerURL:         "http://manager:8000",
				ClusterAPIKey:      "test-key",
				GRPCPort:           50051,
				EnableBlueGreen:    true,
				CanaryStepSize:     10,
				CanaryStepDuration: 0,
			},
			wantError: true,
			errMsg:    "canary_step_duration must be > 0",
		},
		{
			name: "Invalid max modules per protocol",
			config: &Config{
				ManagerURL:            "http://manager:8000",
				ClusterAPIKey:         "test-key",
				GRPCPort:              50051,
				MaxModulesPerProtocol: 0,
			},
			wantError: true,
			errMsg:    "max_modules_per_protocol must be > 0",
		},
		{
			name: "Invalid max connections per module",
			config: &Config{
				ManagerURL:             "http://manager:8000",
				ClusterAPIKey:          "test-key",
				GRPCPort:               50051,
				MaxModulesPerProtocol:  50,
				MaxConnectionsPerModule: 0,
			},
			wantError: true,
			errMsg:    "max_connections_per_module must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want message containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestIsEnterpriseFeatureEnabled(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		feature     string
		expected    bool
	}{
		{
			name: "Dev mode - all features enabled",
			config: &Config{
				ReleaseMode: false,
				LicenseKey:  "",
			},
			feature:  "sso",
			expected: true,
		},
		{
			name: "Release mode with license",
			config: &Config{
				ReleaseMode: true,
				LicenseKey:  "valid-key",
			},
			feature:  "sso",
			expected: true,
		},
		{
			name: "Release mode without license",
			config: &Config{
				ReleaseMode: true,
				LicenseKey:  "",
			},
			feature:  "sso",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnterpriseFeatureEnabled(tt.feature)
			if got != tt.expected {
				t.Errorf("IsEnterpriseFeatureEnabled(%s) = %v, want %v", tt.feature, got, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
