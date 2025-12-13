package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marchproxy-nlb/internal/config"
	"marchproxy-nlb/internal/grpc"
	"marchproxy-nlb/internal/nlb"

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
		Use:   "proxy-nlb",
		Short: "MarchProxy Network Load Balancer",
		Long: `MarchProxy NLB - Unified Network Load Balancer with:
- Protocol detection and intelligent routing
- Rate limiting with token bucket algorithm
- Autoscaling orchestration
- Blue/green deployments
- gRPC-based module communication`,
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitCommit),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNLB(configPath, logger)
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")

	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to start NLB")
	}
}

func runNLB(configPath string, logger *logrus.Logger) error {
	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
		"commit":     gitCommit,
	}).Info("Starting MarchProxy Network Load Balancer")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize router
	router := nlb.NewRouter(logger)
	logger.Info("Traffic router initialized")

	// Initialize rate limiter
	var rateLimiter *nlb.RateLimiter
	if cfg.EnableRateLimiting {
		rateLimiter = nlb.NewRateLimiter(logger)

		// Add configured buckets
		for _, bucket := range cfg.RateLimitBuckets {
			protocol := parseProtocol(bucket.Protocol)
			if err := rateLimiter.AddBucket(bucket.Name, protocol, bucket.Capacity, bucket.RefillRate); err != nil {
				logger.WithError(err).Warn("Failed to add rate limit bucket")
			}
		}

		logger.Info("Rate limiter initialized")
	}

	// Initialize autoscaler
	var autoscaler *nlb.Autoscaler
	if cfg.EnableAutoscaling {
		autoscaler = nlb.NewAutoscaler(router, logger)

		// Set default policies for common protocols
		protocols := []nlb.Protocol{
			nlb.ProtocolHTTP,
			nlb.ProtocolMySQL,
			nlb.ProtocolPostgreSQL,
			nlb.ProtocolMongoDB,
			nlb.ProtocolRedis,
			nlb.ProtocolRTMP,
		}

		for _, protocol := range protocols {
			policy := nlb.DefaultScalingPolicy(protocol)
			policy.ScaleUpCooldown = cfg.ScaleUpCooldown
			policy.ScaleDownCooldown = cfg.ScaleDownCooldown
			if err := autoscaler.SetPolicy(policy); err != nil {
				logger.WithError(err).Warn("Failed to set autoscaling policy")
			}
		}

		if err := autoscaler.Start(); err != nil {
			logger.WithError(err).Warn("Failed to start autoscaler")
		} else {
			logger.Info("Autoscaler started")
		}
	}

	// Initialize blue/green controller
	var blueGreenController *nlb.BlueGreenController
	if cfg.EnableBlueGreen {
		blueGreenController = nlb.NewBlueGreenController(router, logger)
		logger.Info("Blue/Green controller initialized")
	}

	// Initialize gRPC client pool
	var clientPool *grpc.ClientPool
	if cfg.EnableConnectionPooling {
		clientPool = grpc.NewClientPool(logger)
		logger.Info("gRPC client pool initialized")
	}

	// Initialize gRPC server
	mockService := grpc.NewMockNLBService(logger)
	grpcServer := grpc.NewServer(cfg.GRPCAddr, cfg.GRPCPort, mockService, logger)

	// Start gRPC server in goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			logger.WithError(err).Error("gRPC server error")
		}
	}()

	logger.WithFields(logrus.Fields{
		"address": cfg.GRPCAddr,
		"port":    cfg.GRPCPort,
	}).Info("gRPC server started")

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
		status := map[string]interface{}{
			"version":            version,
			"uptime":             time.Since(time.Now()).Seconds(),
			"rate_limiting":      cfg.EnableRateLimiting,
			"autoscaling":        cfg.EnableAutoscaling,
			"bluegreen":          cfg.EnableBlueGreen,
			"connection_pooling": cfg.EnableConnectionPooling,
		}

		if router != nil {
			status["router_stats"] = router.GetStats()
		}

		if rateLimiter != nil {
			status["ratelimit_stats"] = rateLimiter.GetAllStats()
		}

		if autoscaler != nil {
			status["autoscaler_stats"] = autoscaler.GetStats()
		}

		if blueGreenController != nil {
			status["bluegreen_stats"] = blueGreenController.GetStats()
		}

		if clientPool != nil {
			status["client_pool_stats"] = clientPool.GetStats()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":%v}`, status)
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

	logger.Info("MarchProxy NLB started successfully")

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

	if autoscaler != nil {
		autoscaler.Stop()
	}

	if blueGreenController != nil {
		blueGreenController.Stop()
	}

	if clientPool != nil {
		if err := clientPool.Close(); err != nil {
			logger.WithError(err).Error("Client pool shutdown error")
		}
	}

	logger.Info("Shutdown complete")
	return nil
}

// parseProtocol converts string to Protocol enum
func parseProtocol(protocolStr string) nlb.Protocol {
	switch protocolStr {
	case "http", "HTTP":
		return nlb.ProtocolHTTP
	case "mysql", "MySQL":
		return nlb.ProtocolMySQL
	case "postgresql", "PostgreSQL":
		return nlb.ProtocolPostgreSQL
	case "mongodb", "MongoDB":
		return nlb.ProtocolMongoDB
	case "redis", "Redis":
		return nlb.ProtocolRedis
	case "rtmp", "RTMP":
		return nlb.ProtocolRTMP
	default:
		return nlb.ProtocolUnknown
	}
}
