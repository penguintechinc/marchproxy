package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatal("default config should not be nil")
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("expected HTTPPort 8080, got %d", cfg.HTTPPort)
	}
	if cfg.GRPCPort != 50051 {
		t.Errorf("expected GRPCPort 50051, got %d", cfg.GRPCPort)
	}
	if cfg.RoutingAI != false {
		t.Errorf("expected RoutingAI false, got %v", cfg.RoutingAI)
	}
	if cfg.DefaultStrategy != "round_robin" {
		t.Errorf("expected DefaultStrategy round_robin, got %s", cfg.DefaultStrategy)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel info, got %s", cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("expected ShutdownTimeout 30s, got %v", cfg.ShutdownTimeout)
	}
}

func TestLoadFromEnvHTTPPort(t *testing.T) {
	t.Setenv("AILB_HTTP_PORT", "9000")
	defer os.Unsetenv("AILB_HTTP_PORT")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPPort != 9000 {
		t.Errorf("expected HTTPPort 9000, got %d", cfg.HTTPPort)
	}
}

func TestLoadFromEnvGRPCPort(t *testing.T) {
	t.Setenv("AILB_GRPC_PORT", "60000")
	defer os.Unsetenv("AILB_GRPC_PORT")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GRPCPort != 60000 {
		t.Errorf("expected GRPCPort 60000, got %d", cfg.GRPCPort)
	}
}

func TestLoadFromEnvInvalidHTTPPort(t *testing.T) {
	t.Setenv("AILB_HTTP_PORT", "invalid")
	defer os.Unsetenv("AILB_HTTP_PORT")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid HTTPPort, got nil")
	}
}

func TestLoadFromEnvRoutingAI(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"false", "false", false},
		{"0", "0", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AILB_ROUTING_AI", tt.value)
			cfg, err := config.LoadFromEnv()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.RoutingAI != tt.expected {
				t.Errorf("expected RoutingAI %v, got %v", tt.expected, cfg.RoutingAI)
			}
			os.Unsetenv("AILB_ROUTING_AI")
		})
	}
}

func TestLoadFromEnvEnableMemory(t *testing.T) {
	t.Setenv("AILB_ENABLE_MEMORY", "true")
	defer os.Unsetenv("AILB_ENABLE_MEMORY")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.EnableMemory {
		t.Errorf("expected EnableMemory true, got %v", cfg.EnableMemory)
	}
}

func TestLoadFromEnvWaddleAIAddress(t *testing.T) {
	t.Setenv("AILB_WADDLEAI_ADDRESS", "custom:50051")
	defer os.Unsetenv("AILB_WADDLEAI_ADDRESS")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WaddleAIAddress != "custom:50051" {
		t.Errorf("expected WaddleAIAddress custom:50051, got %s", cfg.WaddleAIAddress)
	}
}

func TestLoadFromEnvDefaultStrategy(t *testing.T) {
	t.Setenv("AILB_DEFAULT_STRATEGY", "cost_optimized")
	defer os.Unsetenv("AILB_DEFAULT_STRATEGY")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultStrategy != "cost_optimized" {
		t.Errorf("expected DefaultStrategy cost_optimized, got %s", cfg.DefaultStrategy)
	}
}

func TestLoadFromEnvJWTSecret(t *testing.T) {
	t.Setenv("AILB_JWT_SECRET", "test-secret-123")
	defer os.Unsetenv("AILB_JWT_SECRET")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.JWTSecret != "test-secret-123" {
		t.Errorf("expected JWTSecret test-secret-123, got %s", cfg.JWTSecret)
	}
}

func TestLoadFromEnvSecurityPolicy(t *testing.T) {
	t.Setenv("AILB_SECURITY_POLICY", "strict")
	defer os.Unsetenv("AILB_SECURITY_POLICY")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SecurityPolicy != "strict" {
		t.Errorf("expected SecurityPolicy strict, got %s", cfg.SecurityPolicy)
	}
}

func TestLoadFromEnvLogLevel(t *testing.T) {
	t.Setenv("AILB_LOG_LEVEL", "debug")
	defer os.Unsetenv("AILB_LOG_LEVEL")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug, got %s", cfg.LogLevel)
	}
}

func TestLoadFromEnvShutdownTimeout(t *testing.T) {
	t.Setenv("AILB_SHUTDOWN_TIMEOUT", "45s")
	defer os.Unsetenv("AILB_SHUTDOWN_TIMEOUT")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ShutdownTimeout != 45*time.Second {
		t.Errorf("expected ShutdownTimeout 45s, got %v", cfg.ShutdownTimeout)
	}
}

func TestLoadFromEnvInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("AILB_SHUTDOWN_TIMEOUT", "invalid")
	defer os.Unsetenv("AILB_SHUTDOWN_TIMEOUT")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid ShutdownTimeout, got nil")
	}
}

func TestLoadFromEnvProviders(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-123")
	t.Setenv("OPENAI_MODELS", "gpt-4,gpt-3.5-turbo")
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_MODELS")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	openaiProvider, exists := cfg.Providers["openai"]
	if !exists {
		t.Fatal("expected openai provider to be loaded")
	}
	if openaiProvider.APIKey != "sk-test-123" {
		t.Errorf("expected openai APIKey sk-test-123, got %s", openaiProvider.APIKey)
	}
	if len(openaiProvider.Models) != 2 {
		t.Errorf("expected 2 openai models, got %d", len(openaiProvider.Models))
	}
	if openaiProvider.Models[0] != "gpt-4" {
		t.Errorf("expected first model gpt-4, got %s", openaiProvider.Models[0])
	}

	anthropicProvider, exists := cfg.Providers["anthropic"]
	if !exists {
		t.Fatal("expected anthropic provider to be loaded")
	}
	if anthropicProvider.APIKey != "sk-ant-test" {
		t.Errorf("expected anthropic APIKey sk-ant-test, got %s", anthropicProvider.APIKey)
	}
}

func TestLoadFromEnvOllamaLocal(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")
	t.Setenv("OLLAMA_MODELS", "llama2")
	defer func() {
		os.Unsetenv("OLLAMA_BASE_URL")
		os.Unsetenv("OLLAMA_MODELS")
	}()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ollamaProvider, exists := cfg.Providers["ollama"]
	if !exists {
		t.Fatal("expected ollama provider to be loaded")
	}
	if ollamaProvider.BaseURL != "http://localhost:11434" {
		t.Errorf("expected ollama BaseURL http://localhost:11434, got %s", ollamaProvider.BaseURL)
	}
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		expected  string
	}{
		{"debug", "debug", "DEBUG"},
		{"info", "info", "INFO"},
		{"warn", "warn", "WARN"},
		{"warning", "warning", "WARN"},
		{"error", "error", "ERROR"},
		{"invalid", "invalid", "INFO"},
		{"empty", "", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{LogLevel: tt.logLevel}
			level := cfg.SlogLevel()
			if level.String() != tt.expected {
				t.Errorf("expected log level %s, got %s", tt.expected, level.String())
			}
		})
	}
}

func TestProviderConfigDefaults(t *testing.T) {
	cfg := config.DefaultConfig()
	if len(cfg.Providers) != 0 {
		t.Errorf("expected empty providers map, got %d providers", len(cfg.Providers))
	}
}
