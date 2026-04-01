// Package main is the entry point for the AILB (AI Load Balancer) service.
//
// AILB routes AI/LLM requests across multiple providers (OpenAI, Anthropic,
// Gemini, Ollama, llama.cpp, Mistral, WaddleAI) with intelligent routing,
// rate limiting, usage billing, and security scanning.
//
// It exposes:
//   - HTTP server (default :8080) with OpenAI-compatible and Ollama-compatible endpoints
//   - gRPC server (default :50051) implementing the MarchProxy ModuleService interface
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/auth"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/billing"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/config"
	grpcserver "github.com/PenguinTech/MarchProxy/proxy-ailb/internal/grpc"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/handler"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/health"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/metrics"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration from environment.
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Configure structured logging.
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.SlogLevel(),
	})
	slog.SetDefault(slog.New(logHandler))

	slog.Info("starting AILB",
		"version", version,
		"http_port", cfg.HTTPPort,
		"grpc_port", cfg.GRPCPort,
		"providers", len(cfg.Providers),
		"routing_ai", cfg.RoutingAI,
	)

	// Create provider registry and register configured providers.
	registry := providers.NewRegistry()
	registerProviders(registry, cfg)

	if len(registry.Names()) == 0 {
		slog.Warn("no providers configured - AILB will return errors for all requests")
	}

	// Create router.
	strategy := router.ParseStrategy(cfg.DefaultStrategy)
	rtr := router.New(registry, strategy)

	// Create billing reporter (fire-and-forget to WaddleAI).
	reporter := billing.NewReporter(cfg.WaddleAIAddress)
	if err := reporter.Connect(); err != nil {
		slog.Warn("failed to connect usage reporter, billing disabled", "error", err)
	}
	defer reporter.Close()

	// Create metrics.
	m := metrics.New()

	// Create JWT validator.
	validator := auth.NewValidator(cfg.JWTSecret)

	// Create WaddleAI gRPC client for routing/security/memory.
	waddleAIClient := providers.NewWaddleAIGRPCClient(cfg.WaddleAIAddress, cfg.RoutingAI)
	defer waddleAIClient.Close()

	// Build HTTP mux.
	mux := buildMux(registry, rtr, reporter, m, validator, waddleAIClient, cfg.RoutingAI)

	// Context with cancellation for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture shutdown signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start gRPC server in background.
	grpcErrCh := make(chan error, 1)
	go func() {
		grpcErrCh <- grpcserver.Start(ctx, cfg.GRPCPort, registry, rtr, version)
	}()

	// Start HTTP server.
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	httpErrCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server starting", "addr", httpServer.Addr)
		httpErrCh <- httpServer.ListenAndServe()
	}()

	// Wait for shutdown signal or server error.
	select {
	case sig := <-sigCh:
		slog.Info("received shutdown signal", "signal", sig)
	case err := <-httpErrCh:
		if err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	case err := <-grpcErrCh:
		if err != nil {
			return fmt.Errorf("gRPC server error: %w", err)
		}
	}

	// Graceful shutdown.
	slog.Info("shutting down", "timeout", cfg.ShutdownTimeout)
	cancel() // Stop gRPC server.

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP server shutdown: %w", err)
	}

	slog.Info("shutdown complete")
	return nil
}

// buildMux constructs the HTTP request multiplexer with all routes and middleware.
func buildMux(
	registry *providers.Registry,
	rtr *router.Router,
	reporter *billing.Reporter,
	m *metrics.Metrics,
	validator *auth.Validator,
	waddleAIClient *providers.WaddleAIGRPCClient,
	routingAI bool,
) http.Handler {
	mux := http.NewServeMux()

	// Health and metrics (no auth required).
	mux.HandleFunc("GET /healthz", health.Handler())
	mux.Handle("GET /metrics", metrics.Handler())

	// OpenAI-compatible endpoints.
	chatHandler := handler.NewChatHandler(registry, rtr, reporter, m)
	chatHandler.SetWaddleAI(waddleAIClient, routingAI)
	modelsHandler := handler.NewModelsHandler(registry)
	mux.Handle("POST /v1/chat/completions", chatHandler)
	mux.Handle("GET /v1/models", modelsHandler)

	// Ollama-compatible endpoints.
	ollamaHandler := handler.NewOllamaHandler(registry, rtr, m)
	mux.HandleFunc("POST /api/chat", ollamaHandler.ChatHandler())
	mux.HandleFunc("POST /api/generate", ollamaHandler.GenerateHandler())
	mux.HandleFunc("GET /api/tags", ollamaHandler.TagsHandler())

	// Apply middleware stack (outermost first).
	var h http.Handler = mux
	h = handler.AuthMiddleware(validator)(h)
	h = handler.MetricsMiddleware(m)(h)
	h = handler.LoggingMiddleware(h)
	h = handler.RecoveryMiddleware(h)

	return h
}

// registerProviders creates and registers providers based on configuration.
func registerProviders(reg *providers.Registry, cfg *config.Config) {
	for name, pcfg := range cfg.Providers {
		if !pcfg.Enabled {
			continue
		}

		switch name {
		case "openai":
			reg.Register(providers.NewOpenAIProvider("openai", pcfg.APIKey, pcfg.BaseURL, pcfg.Models))
		case "anthropic":
			reg.Register(providers.NewAnthropicProvider(pcfg.APIKey, pcfg.Models))
		case "gemini":
			reg.Register(providers.NewGeminiProvider(pcfg.APIKey, pcfg.BaseURL, pcfg.Models))
		case "ollama":
			reg.Register(providers.NewOllamaProvider(pcfg.BaseURL, pcfg.Models))
		case "llamacpp":
			reg.Register(providers.NewLlamaCppProvider(pcfg.BaseURL, pcfg.Models))
		case "mistral":
			reg.Register(providers.NewMistralProvider(pcfg.APIKey, pcfg.BaseURL, pcfg.Models))
		case "waddleai":
			reg.Register(providers.NewWaddleAIProvider(pcfg.APIKey, pcfg.BaseURL, pcfg.Models))
		default:
			// Unknown providers with a base URL are treated as OpenAI-compatible.
			if pcfg.BaseURL != "" {
				reg.Register(providers.NewOpenAIProvider(name, pcfg.APIKey, pcfg.BaseURL, pcfg.Models))
			} else {
				slog.Warn("skipping unknown provider with no base URL", "provider", name)
			}
		}

		slog.Info("registered provider", "name", name)
	}
}
