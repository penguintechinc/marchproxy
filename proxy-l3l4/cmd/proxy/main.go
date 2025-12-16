package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"marchproxy-l3l4/internal/acceleration"
	"marchproxy-l3l4/internal/config"
	"marchproxy-l3l4/internal/multicloud"
	"marchproxy-l3l4/internal/numa"
	"marchproxy-l3l4/internal/observability"
	"marchproxy-l3l4/internal/qos"
	"marchproxy-l3l4/internal/zerotrust"

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
		Use:   "proxy-l3l4",
		Short: "MarchProxy L3/L4 Enhanced Proxy",
		Long: `MarchProxy L3/L4 Proxy - Enterprise-grade network proxy with:
- NUMA-aware architecture
- QoS traffic shaping with priority queues
- Multi-cloud intelligent routing
- Hardware acceleration (XDP, AF_XDP)
- Distributed tracing and metrics
- Zero-trust security features`,
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitCommit),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(configPath, logger)
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")

	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to start proxy")
	}
}

func runProxy(configPath string, logger *logrus.Logger) error {
	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
		"commit":     gitCommit,
	}).Info("Starting MarchProxy L3/L4 Enhanced Proxy")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize observability
	var metrics *observability.Metrics
	var tracer *observability.Tracer

	metrics = observability.NewMetrics(cfg.MetricsNamespace)
	logger.Info("Metrics initialized")

	if cfg.EnableTracing {
		tracer, err = observability.NewTracer("marchproxy-l3l4", cfg.JaegerEndpoint, cfg.TraceSampleRate, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize tracing, continuing without it")
		} else {
			defer tracer.Shutdown(ctx)
			logger.Info("Distributed tracing initialized")
		}
	}

	// Initialize NUMA manager
	var numaManager *numa.Manager
	if cfg.EnableNUMA {
		numaManager = numa.NewManager(logger)
		if err := numaManager.Initialize(); err != nil {
			logger.WithError(err).Warn("NUMA initialization failed")
		} else if numaManager.IsEnabled() {
			// Allocate workers across NUMA nodes
			workerCount := cfg.WorkerThreads
			if workerCount == 0 {
				workerCount = numaManager.GetTopology().TotalCPUs
			}
			allocations, err := numaManager.AllocateWorkers(workerCount)
			if err != nil {
				logger.WithError(err).Warn("Failed to allocate NUMA workers")
			} else {
				logger.WithField("workers", len(allocations)).Info("NUMA workers allocated")
				metrics.NumaWorkers.Set(float64(len(allocations)))
				metrics.NumaNodesActive.Set(float64(numaManager.GetTopology().NodeCount))
			}
		}
	}

	// Initialize QoS traffic shaper
	var trafficShaper *qos.TrafficShaper
	if cfg.EnableQoS {
		trafficShaper = qos.NewTrafficShaper(
			cfg.DefaultBandwidth,
			cfg.BurstSize,
			cfg.PriorityQueueDepth,
			cfg.DSCPMarking,
			logger,
		)
		trafficShaper.Start()
		logger.Info("QoS traffic shaper started")
	}

	// Initialize multi-cloud router
	var mcRouter *multicloud.Router
	if cfg.EnableMultiCloud && len(cfg.Backends) > 0 {
		backends := make([]*multicloud.Backend, len(cfg.Backends))
		for i, b := range cfg.Backends {
			backends[i] = &multicloud.Backend{
				Name:     b.Name,
				URL:      b.URL,
				Weight:   b.Weight,
				Priority: b.Priority,
				Cloud:    b.Cloud,
				Region:   b.Region,
				Cost:     b.Cost,
				Healthy:  true,
			}
		}

		mcRouter, err = multicloud.NewRouter(cfg.RoutingAlgorithm, backends, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize multi-cloud router")
		} else {
			if err := mcRouter.Start(); err != nil {
				logger.WithError(err).Warn("Failed to start multi-cloud router")
			} else {
				logger.Info("Multi-cloud router started")
			}
		}
	}

	// Initialize hardware acceleration
	var accelManager *acceleration.Manager
	if cfg.EnableAcceleration {
		accelManager, err = acceleration.NewManager(cfg.AccelerationMode, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize acceleration")
		} else {
			if err := accelManager.Initialize(cfg.XDPDevice, cfg.AFXDPQueueCount); err != nil {
				logger.WithError(err).Warn("Failed to initialize acceleration hardware")
			} else if err := accelManager.Start(); err != nil {
				logger.WithError(err).Warn("Failed to start acceleration")
			} else {
				logger.WithField("mode", accelManager.GetMode()).Info("Hardware acceleration started")
			}
		}
	}

	// Initialize zero-trust components
	var policyEnforcer *zerotrust.PolicyEnforcer
	var auditLogger *zerotrust.AuditLogger
	var certRotator *zerotrust.CertRotator

	if cfg.EnableZeroTrust {
		logger.Info("Initializing zero-trust security features")

		auditLogDir := filepath.Dir(cfg.AuditLogPath)
		if err := os.MkdirAll(auditLogDir, 0755); err != nil {
			return fmt.Errorf("failed to create audit log directory: %w", err)
		}

		auditLogger, err = zerotrust.NewAuditLogger(cfg.AuditLogPath, logger)
		if err != nil {
			return fmt.Errorf("failed to initialize audit logger: %w", err)
		}
		defer auditLogger.Close()

		policyEnforcer, err = zerotrust.NewPolicyEnforcer(cfg.OpaURL, logger, auditLogger)
		if err != nil {
			logger.WithError(err).Warn("OPA policy enforcer unavailable")
		} else {
			policyEnforcer.SetLicenseStatus(cfg.IsEnterpriseFeatureEnabled("zero-trust"))
		}

		if _, err := os.Stat(cfg.CertPath); err == nil {
			certRotator, err = zerotrust.NewCertRotator(cfg.CertPath, cfg.KeyPath, logger)
			if err != nil {
				logger.WithError(err).Warn("Certificate rotator unavailable")
			} else {
				certRotator.Start()
				defer certRotator.Stop()
			}
		}
	}

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
			"version":     version,
			"uptime":      time.Since(time.Now()).Seconds(),
			"qos_enabled": cfg.EnableQoS,
			"numa_enabled": cfg.EnableNUMA,
			"multicloud_enabled": cfg.EnableMultiCloud,
			"acceleration_mode": "standard",
		}

		if accelManager != nil {
			status["acceleration_mode"] = accelManager.GetMode()
			status["acceleration_stats"] = accelManager.GetStats()
		}

		if numaManager != nil && numaManager.IsEnabled() {
			status["numa_stats"] = numaManager.Stats()
		}

		if trafficShaper != nil {
			status["qos_stats"] = trafficShaper.GetStats()
		}

		if mcRouter != nil {
			status["routing_stats"] = mcRouter.GetStats()
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

	// Log startup event
	if auditLogger != nil {
		auditEvent := &zerotrust.AuditEvent{
			Timestamp: time.Now(),
			EventType: "system_startup",
			Service:   "proxy-l3l4",
			Action:    "start",
			Resource:  "system",
			SourceIP:  "localhost",
			Allowed:   true,
			Reason:    "Enhanced proxy started",
			Metadata: map[string]interface{}{
				"version":            version,
				"qos_enabled":        cfg.EnableQoS,
				"numa_enabled":       cfg.EnableNUMA,
				"multicloud_enabled": cfg.EnableMultiCloud,
				"acceleration_mode":  "standard",
			},
		}
		if accelManager != nil {
			auditEvent.Metadata["acceleration_mode"] = accelManager.GetMode()
		}
		auditLogger.LogEvent(auditEvent)
	}

	logger.Info("MarchProxy L3/L4 Enhanced Proxy started successfully")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down...")

	// Log shutdown event
	if auditLogger != nil {
		auditEvent := &zerotrust.AuditEvent{
			Timestamp: time.Now(),
			EventType: "system_shutdown",
			Service:   "proxy-l3l4",
			Action:    "stop",
			Resource:  "system",
			SourceIP:  "localhost",
			Allowed:   true,
			Reason:    "Enhanced proxy shutting down",
		}
		auditLogger.LogEvent(auditEvent)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Metrics server shutdown error")
	}

	if mcRouter != nil {
		mcRouter.Stop()
	}

	if accelManager != nil {
		accelManager.Stop()
	}

	if policyEnforcer != nil {
		policyEnforcer.Close()
	}

	logger.Info("Shutdown complete")
	return nil
}
