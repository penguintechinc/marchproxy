package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/config"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/envoy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/grpc"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/metrics"
)

var (
	version   = "v1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Setup logger
	logger := setupLogger()

	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
	}).Info("Starting MarchProxy ALB")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	logger.WithFields(logrus.Fields{
		"module_id":   cfg.ModuleID,
		"grpc_port":   cfg.GRPCPort,
		"xds_server":  cfg.XDSServerAddr,
	}).Info("Configuration loaded")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	envoyManager := envoy.NewManager(
		cfg.EnvoyBinary,
		cfg.EnvoyConfigPath,
		cfg.EnvoyAdminPort,
		cfg.EnvoyLogLevel,
		logger,
	)

	xdsClient := envoy.NewXDSClient(cfg.XDSServerAddr, logger)

	metricsCollector := metrics.NewCollector(
		fmt.Sprintf("localhost:%d", cfg.EnvoyAdminPort),
		logger,
	)

	grpcServer := grpc.NewServer(
		cfg,
		envoyManager,
		xdsClient,
		metricsCollector,
		logger,
	)

	// Start Envoy proxy
	logger.Info("Starting Envoy proxy")
	if err := envoyManager.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to start Envoy")
	}

	// Start gRPC server
	logger.Info("Starting gRPC server")
	if err := grpcServer.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start gRPC server")
	}

	// Start health check endpoint
	go startHealthCheckServer(cfg.HealthCheckPort, envoyManager, logger)

	// Start metrics endpoint
	go startMetricsServer(cfg.MetricsPort, metricsCollector, logger)

	logger.Info("ALB started successfully")

	// Wait for shutdown signal
	waitForShutdown(ctx, cancel, cfg, envoyManager, grpcServer, logger)
}

// setupLogger configures the logger
func setupLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger
}

// startHealthCheckServer starts HTTP health check endpoint
func startHealthCheckServer(port int, envoyMgr *envoy.Manager, logger *logrus.Logger) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if envoyMgr.IsRunning() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Envoy not running")
		}
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if envoyMgr.IsRunning() && envoyMgr.Uptime() > 5*time.Second {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Ready")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Not ready")
		}
	})

	addr := fmt.Sprintf(":%d", port)
	logger.WithField("address", addr).Info("Starting health check server")

	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.WithError(err).Error("Health check server error")
	}
}

// startMetricsServer starts Prometheus metrics endpoint
func startMetricsServer(port int, collector *metrics.Collector, logger *logrus.Logger) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		m, err := collector.GetMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write Prometheus format metrics
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP alb_total_connections Total number of connections\n")
		fmt.Fprintf(w, "# TYPE alb_total_connections counter\n")
		fmt.Fprintf(w, "alb_total_connections %d\n", m.TotalConnections)

		fmt.Fprintf(w, "# HELP alb_active_connections Active connections\n")
		fmt.Fprintf(w, "# TYPE alb_active_connections gauge\n")
		fmt.Fprintf(w, "alb_active_connections %d\n", m.ActiveConnections)

		fmt.Fprintf(w, "# HELP alb_total_requests Total number of requests\n")
		fmt.Fprintf(w, "# TYPE alb_total_requests counter\n")
		fmt.Fprintf(w, "alb_total_requests %d\n", m.TotalRequests)

		fmt.Fprintf(w, "# HELP alb_requests_per_second Requests per second\n")
		fmt.Fprintf(w, "# TYPE alb_requests_per_second gauge\n")
		fmt.Fprintf(w, "alb_requests_per_second %d\n", m.RequestsPerSecond)

		fmt.Fprintf(w, "# HELP alb_latency_ms Request latency in milliseconds\n")
		fmt.Fprintf(w, "# TYPE alb_latency_ms summary\n")
		fmt.Fprintf(w, "alb_latency_ms{quantile=\"0.5\"} %.2f\n", m.Latency.P50Ms)
		fmt.Fprintf(w, "alb_latency_ms{quantile=\"0.9\"} %.2f\n", m.Latency.P90Ms)
		fmt.Fprintf(w, "alb_latency_ms{quantile=\"0.95\"} %.2f\n", m.Latency.P95Ms)
		fmt.Fprintf(w, "alb_latency_ms{quantile=\"0.99\"} %.2f\n", m.Latency.P99Ms)

		// Status codes
		for code, count := range m.StatusCodes {
			fmt.Fprintf(w, "alb_responses_total{status=\"%s\"} %d\n", code, count)
		}
	})

	addr := fmt.Sprintf(":%d", port)
	logger.WithField("address", addr).Info("Starting metrics server")

	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.WithError(err).Error("Metrics server error")
	}
}

// waitForShutdown waits for termination signal and performs graceful shutdown
func waitForShutdown(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *config.Config,
	envoyMgr *envoy.Manager,
	grpcSrv *grpc.Server,
	logger *logrus.Logger,
) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.WithField("signal", sig).Info("Received shutdown signal")

	// Cancel context to stop any running operations
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	// Stop gRPC server
	logger.Info("Stopping gRPC server")
	grpcSrv.Stop()

	// Stop Envoy
	logger.Info("Stopping Envoy proxy")
	if err := envoyMgr.Stop(); err != nil {
		logger.WithError(err).Error("Error stopping Envoy")
	}

	// Wait for shutdown or timeout
	<-shutdownCtx.Done()

	if shutdownCtx.Err() == context.DeadlineExceeded {
		logger.Warn("Shutdown timeout exceeded, forcing exit")
	} else {
		logger.Info("Graceful shutdown completed")
	}
}
