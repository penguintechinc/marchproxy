package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marchproxy-dblb/internal/config"
	"marchproxy-dblb/internal/grpc"
	"marchproxy-dblb/internal/handlers"
	"marchproxy-dblb/internal/pool"
	"marchproxy-dblb/internal/security"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	version   = "1.0.0"
	buildTime = "development"
	gitCommit = "unknown"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	var configPath string

	rootCmd := &cobra.Command{
		Use:   "proxy-dblb",
		Short: "MarchProxy Database Load Balancer",
		Long: `MarchProxy DBLB - Database Load Balancer with:
- Multi-protocol database support (MySQL, PostgreSQL, MongoDB, Redis, MSSQL)
- Connection pooling and rate limiting
- SQL injection detection
- Per-route configuration
- gRPC-based module communication`,
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitCommit),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDBLB(configPath, logger)
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")

	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to start DBLB")
	}
}

func runDBLB(configPath string, logger *logrus.Logger) error {
	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
		"commit":     gitCommit,
	}).Info("Starting MarchProxy Database Load Balancer")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize security checker
	securityChecker := security.NewChecker(logger)
	logger.Info("Security checker initialized")

	// Initialize connection pool
	connectionPool := pool.NewPool(cfg.MaxConnectionsPerRoute, logger)
	logger.Info("Connection pool initialized")

	// Initialize database handlers
	handlerManager := handlers.NewManager(connectionPool, securityChecker, cfg, logger)

	// Register database protocol handlers
	if err := handlerManager.RegisterHandler("mysql", 3306); err != nil {
		logger.WithError(err).Warn("Failed to register MySQL handler")
	}

	if err := handlerManager.RegisterHandler("postgresql", 5432); err != nil {
		logger.WithError(err).Warn("Failed to register PostgreSQL handler")
	}

	if err := handlerManager.RegisterHandler("mongodb", 27017); err != nil {
		logger.WithError(err).Warn("Failed to register MongoDB handler")
	}

	if err := handlerManager.RegisterHandler("redis", 6379); err != nil {
		logger.WithError(err).Warn("Failed to register Redis handler")
	}

	if err := handlerManager.RegisterHandler("mssql", 1433); err != nil {
		logger.WithError(err).Warn("Failed to register MSSQL handler")
	}

	logger.Info("Database handlers registered")

	// Start all handlers
	if err := handlerManager.StartAll(ctx); err != nil {
		return fmt.Errorf("failed to start handlers: %w", err)
	}

	// Initialize gRPC server with ModuleService
	moduleService := grpc.NewModuleService(handlerManager, logger)
	grpcServer := grpc.NewServer(cfg.GRPCAddr, cfg.GRPCPort, moduleService, logger)

	// Start gRPC server in goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			logger.WithError(err).Error("gRPC server error")
		}
	}()

	logger.WithFields(logrus.Fields{
		"address": cfg.GRPCAddr,
		"port":    cfg.GRPCPort,
	}).Info("gRPC ModuleService server started")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start metrics/health server
	metricsMux := http.NewServeMux()

	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsMux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		stats := handlerManager.GetStats()
		poolStats := connectionPool.GetStats()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"version":"%s","handlers":%v,"pool":%v}`, version, stats, poolStats)
	})

	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: metricsMux,
	}

	go func() {
		logger.WithField("addr", cfg.MetricsAddr).Info("Starting metrics/health server")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Metrics server error")
		}
	}()

	logger.Info("MarchProxy DBLB started successfully")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Metrics server shutdown error")
	}

	if grpcServer != nil {
		if err := grpcServer.Stop(); err != nil {
			logger.WithError(err).Error("gRPC server shutdown error")
		}
	}

	if err := handlerManager.StopAll(); err != nil {
		logger.WithError(err).Error("Handlers shutdown error")
	}

	if err := connectionPool.Close(); err != nil {
		logger.WithError(err).Error("Connection pool shutdown error")
	}

	logger.Info("Shutdown complete")
	return nil
}
