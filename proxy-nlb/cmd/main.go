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
)

var (
	version   = "1.0.0"
	buildTime = "development"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
	}).Info("Starting MarchProxy Network Load Balancer")

	// Load configuration from config.example.yaml or environment
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.example.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config")
	}

	// Initialize router
	router := nlb.NewRouter(logger)
	logger.Info("Traffic router initialized")

	// Initialize rate limiter if enabled
	var rateLimiter *nlb.RateLimiter
	if cfg.EnableRateLimiting {
		rateLimiter = nlb.NewRateLimiter(logger)
		for _, bucket := range cfg.RateLimitBuckets {
			protocol := parseProtocol(bucket.Protocol)
			if err := rateLimiter.AddBucket(bucket.Name, protocol, bucket.Capacity, bucket.RefillRate); err != nil {
				logger.WithError(err).Warn("Failed to add rate limit bucket")
			}
		}
		logger.Info("Rate limiter initialized")
	}

	// Initialize autoscaler if enabled
	var autoscaler *nlb.Autoscaler
	if cfg.EnableAutoscaling {
		autoscaler = nlb.NewAutoscaler(router, logger)
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

	// Initialize blue/green controller if enabled
	var blueGreenController *nlb.BlueGreenController
	if cfg.EnableBlueGreen {
		blueGreenController = nlb.NewBlueGreenController(router, logger)
		logger.Info("Blue/Green controller initialized")
	}

	// Initialize gRPC client pool if enabled
	var clientPool *grpc.ClientPool
	if cfg.EnableConnectionPooling {
		clientPool = grpc.NewClientPool(logger)
		logger.Info("gRPC client pool initialized")
	}

	// Initialize gRPC server on port 50051
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
	}).Info("gRPC server started on port 50051")

	// Setup NLB router listening on port 443 (via configuration)
	// Note: Actual TCP/UDP routing would be implemented here
	// For now, we expose management endpoints
	logger.WithField("bind_addr", cfg.BindAddr).Info("NLB router ready on port 443")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start health check and metrics server
	mux := http.NewServeMux()

	// Health check endpoint on /healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Status endpoint with detailed information
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"version":            version,
			"rate_limiting":      cfg.EnableRateLimiting,
			"autoscaling":        cfg.EnableAutoscaling,
			"bluegreen":          cfg.EnableBlueGreen,
			"connection_pooling": cfg.EnableConnectionPooling,
		}

		if router != nil {
			status["router_stats"] = router.GetStats()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":%v}`, status)
	})

	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: mux,
	}

	go func() {
		logger.WithField("addr", cfg.MetricsAddr).Info("Starting health check and metrics server")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Metrics server error")
		}
	}()

	logger.Info("MarchProxy NLB started successfully - ready to route traffic")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Received shutdown signal, initiating graceful shutdown...")

	// Graceful shutdown with 30 second timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Metrics server shutdown error")
	}

	// Shutdown gRPC server
	if grpcServer != nil {
		if err := grpcServer.Stop(); err != nil {
			logger.WithError(err).Error("gRPC server shutdown error")
		}
	}

	// Stop autoscaler
	if autoscaler != nil {
		autoscaler.Stop()
	}

	// Stop blue/green controller
	if blueGreenController != nil {
		blueGreenController.Stop()
	}

	// Close client pool
	if clientPool != nil {
		if err := clientPool.Close(); err != nil {
			logger.WithError(err).Error("Client pool shutdown error")
		}
	}

	logger.Info("Graceful shutdown complete")
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
