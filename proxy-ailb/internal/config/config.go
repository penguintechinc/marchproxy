// Package config provides Viper-based configuration for the AILB service.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// ProviderConfig holds configuration for a single LLM provider.
type ProviderConfig struct {
	Name    string   `mapstructure:"name"`
	APIKey  string   `mapstructure:"api_key"`
	BaseURL string   `mapstructure:"base_url"`
	Models  []string `mapstructure:"models"`
	Enabled bool     `mapstructure:"enabled"`
}

// Config holds all AILB configuration.
type Config struct {
	HTTPPort        int                       `mapstructure:"http_port"`
	GRPCPort        int                       `mapstructure:"grpc_port"`
	RoutingAI       bool                      `mapstructure:"routing_ai"`
	WaddleAIAddress string                    `mapstructure:"waddleai_address"`
	DefaultStrategy string                    `mapstructure:"default_strategy"`
	EnableMemory    bool                      `mapstructure:"enable_memory"`
	JWTSecret       string                    `mapstructure:"jwt_secret"`
	SecurityPolicy  string                    `mapstructure:"security_policy"`
	LogLevel        string                    `mapstructure:"log_level"`
	ShutdownTimeout time.Duration             `mapstructure:"shutdown_timeout"`
	Providers       map[string]ProviderConfig `mapstructure:"providers"`
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		HTTPPort:        8080,
		GRPCPort:        50051,
		RoutingAI:       false,
		WaddleAIAddress: "waddleai:50051",
		DefaultStrategy: "round_robin",
		EnableMemory:    false,
		JWTSecret:       "",
		SecurityPolicy:  "balanced",
		LogLevel:        "info",
		ShutdownTimeout: 30 * time.Second,
		Providers:       make(map[string]ProviderConfig),
	}
}

// LoadFromEnv populates Config from environment variables.
// Environment variable mapping:
//
//	AILB_HTTP_PORT, AILB_GRPC_PORT, AILB_ROUTING_AI, AILB_WADDLEAI_ADDRESS,
//	AILB_DEFAULT_STRATEGY, AILB_ENABLE_MEMORY, AILB_JWT_SECRET,
//	AILB_SECURITY_POLICY, AILB_LOG_LEVEL, AILB_SHUTDOWN_TIMEOUT
//
// Provider-specific:
//
//	OPENAI_API_KEY, OPENAI_BASE_URL, OPENAI_MODELS (comma-separated)
//	ANTHROPIC_API_KEY, ANTHROPIC_MODELS
//	GEMINI_API_KEY, GEMINI_MODELS
//	OLLAMA_BASE_URL, OLLAMA_MODELS
//	LLAMACPP_BASE_URL, LLAMACPP_MODELS
//	MISTRAL_API_KEY, MISTRAL_BASE_URL, MISTRAL_MODELS
//	WADDLEAI_API_KEY, WADDLEAI_BASE_URL, WADDLEAI_MODELS
func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	if v := os.Getenv("AILB_HTTP_PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &cfg.HTTPPort); err != nil {
			return nil, fmt.Errorf("invalid AILB_HTTP_PORT %q: %w", v, err)
		}
	}
	if v := os.Getenv("AILB_GRPC_PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &cfg.GRPCPort); err != nil {
			return nil, fmt.Errorf("invalid AILB_GRPC_PORT %q: %w", v, err)
		}
	}
	if v := os.Getenv("AILB_ROUTING_AI"); v != "" {
		cfg.RoutingAI = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("AILB_WADDLEAI_ADDRESS"); v != "" {
		cfg.WaddleAIAddress = v
	}
	if v := os.Getenv("AILB_DEFAULT_STRATEGY"); v != "" {
		cfg.DefaultStrategy = v
	}
	if v := os.Getenv("AILB_ENABLE_MEMORY"); v != "" {
		cfg.EnableMemory = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("AILB_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("AILB_SECURITY_POLICY"); v != "" {
		cfg.SecurityPolicy = v
	}
	if v := os.Getenv("AILB_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("AILB_SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid AILB_SHUTDOWN_TIMEOUT %q: %w", v, err)
		}
		cfg.ShutdownTimeout = d
	}

	// Load provider configurations
	cfg.loadProviders()

	return cfg, nil
}

// loadProviders reads provider-specific environment variables.
func (c *Config) loadProviders() {
	type providerEnv struct {
		name      string
		keyEnv    string
		baseEnv   string
		modelsEnv string
		defBase   string
	}

	providers := []providerEnv{
		{"openai", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODELS", "https://api.openai.com/v1"},
		{"anthropic", "ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "ANTHROPIC_MODELS", "https://api.anthropic.com"},
		{"gemini", "GEMINI_API_KEY", "GEMINI_BASE_URL", "GEMINI_MODELS", "https://generativelanguage.googleapis.com"},
		{"ollama", "", "OLLAMA_BASE_URL", "OLLAMA_MODELS", "http://localhost:11434"},
		{"llamacpp", "", "LLAMACPP_BASE_URL", "LLAMACPP_MODELS", "http://localhost:8081"},
		{"mistral", "MISTRAL_API_KEY", "MISTRAL_BASE_URL", "MISTRAL_MODELS", "https://api.mistral.ai/v1"},
		{"waddleai", "WADDLEAI_API_KEY", "WADDLEAI_BASE_URL", "WADDLEAI_MODELS", ""},
	}

	for _, p := range providers {
		apiKey := os.Getenv(p.keyEnv)
		baseURL := getEnvOrDefault(p.baseEnv, p.defBase)
		modelsStr := os.Getenv(p.modelsEnv)

		// Skip providers with no API key and no base URL override (except local ones)
		hasKey := apiKey != ""
		hasBaseOverride := os.Getenv(p.baseEnv) != ""
		isLocal := p.name == "ollama" || p.name == "llamacpp"

		if !hasKey && !hasBaseOverride && !isLocal {
			continue
		}

		var models []string
		if modelsStr != "" {
			for _, m := range strings.Split(modelsStr, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					models = append(models, m)
				}
			}
		}

		c.Providers[p.name] = ProviderConfig{
			Name:    p.name,
			APIKey:  apiKey,
			BaseURL: baseURL,
			Models:  models,
			Enabled: true,
		}
	}
}

// LogLevel returns the slog.Level for the configured log level string.
func (c *Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
