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

	"marchproxy-l3l4/internal/zerotrust"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	rootCmd := &cobra.Command{
		Use:   "proxy-l3l4",
		Short: "MarchProxy L3/L4 Proxy with Zero-Trust Security",
		Long: `MarchProxy L3/L4 Proxy
High-performance L3/L4 proxy with zero-trust security features including:
- OPA policy enforcement
- mTLS certificate verification
- Immutable audit logging
- Compliance reporting (SOC2, HIPAA, PCI-DSS)`,
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", version, buildTime, gitCommit),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(logger)
		},
	}

	// Configuration flags
	rootCmd.PersistentFlags().String("config", "", "config file (default: /etc/marchproxy/config.yaml)")
	rootCmd.PersistentFlags().String("manager-url", "http://api-server:8000", "Manager API URL")
	rootCmd.PersistentFlags().String("cluster-api-key", "", "Cluster API key for authentication")
	rootCmd.PersistentFlags().String("opa-url", "http://opa:8181", "OPA server URL")
	rootCmd.PersistentFlags().String("audit-log-path", "/var/log/marchproxy/audit/audit.log", "Audit log file path")
	rootCmd.PersistentFlags().String("cert-path", "/etc/marchproxy/certs/server.crt", "Server certificate path")
	rootCmd.PersistentFlags().String("key-path", "/etc/marchproxy/certs/server.key", "Server key path")
	rootCmd.PersistentFlags().Bool("enable-zero-trust", true, "Enable zero-trust security features")
	rootCmd.PersistentFlags().String("bind-addr", ":8081", "Proxy bind address")
	rootCmd.PersistentFlags().String("metrics-addr", ":8082", "Metrics/health check bind address")

	// Bind flags to viper
	viper.BindPFlags(rootCmd.PersistentFlags())
	viper.AutomaticEnv()

	// Execute
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to start proxy")
	}
}

func runProxy(logger *logrus.Logger) error {
	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
		"commit":     gitCommit,
	}).Info("Starting MarchProxy L3/L4 Proxy")

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize zero-trust components if enabled
	var policyEnforcer *zerotrust.PolicyEnforcer
	var auditLogger *zerotrust.AuditLogger
	var certRotator *zerotrust.CertRotator
	var rbacEvaluator *zerotrust.RBACEvaluator
	// var complianceReporter *zerotrust.ComplianceReporter // Available for future use

	if viper.GetBool("enable-zero-trust") {
		logger.Info("Initializing zero-trust security features")

		// Check license for Enterprise features
		// In production, validate with license server
		licenseValid := true // Placeholder

		// Initialize audit logger
		auditLogPath := viper.GetString("audit-log-path")
		auditLogDir := filepath.Dir(auditLogPath)
		if err := os.MkdirAll(auditLogDir, 0755); err != nil {
			return fmt.Errorf("failed to create audit log directory: %w", err)
		}

		var err error
		auditLogger, err = zerotrust.NewAuditLogger(auditLogPath, logger)
		if err != nil {
			return fmt.Errorf("failed to initialize audit logger: %w", err)
		}
		defer auditLogger.Close()

		logger.Info("Audit logger initialized")

		// Initialize OPA policy enforcer
		opaURL := viper.GetString("opa-url")
		policyEnforcer, err = zerotrust.NewPolicyEnforcer(opaURL, logger, auditLogger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize OPA policy enforcer, continuing without policy enforcement")
		} else {
			policyEnforcer.SetLicenseStatus(licenseValid)
			logger.Info("OPA policy enforcer initialized")

			// Load default policies
			if err := loadDefaultPolicies(policyEnforcer); err != nil {
				logger.WithError(err).Warn("Failed to load default policies")
			}
		}

		// Initialize RBAC evaluator
		if policyEnforcer != nil {
			rbacEvaluator = zerotrust.NewRBACEvaluator(policyEnforcer, logger)

			// Load default roles
			loadDefaultRoles(rbacEvaluator)

			logger.Info("RBAC evaluator initialized")
		}

		// Initialize certificate rotator
		certPath := viper.GetString("cert-path")
		keyPath := viper.GetString("key-path")

		if _, err := os.Stat(certPath); err == nil {
			certRotator, err = zerotrust.NewCertRotator(certPath, keyPath, logger)
			if err != nil {
				logger.WithError(err).Warn("Failed to initialize certificate rotator")
			} else {
				certRotator.Start()
				defer certRotator.Stop()
				logger.Info("Certificate rotator initialized")
			}
		}

		// Initialize compliance reporter (available for future use)
		if auditLogger != nil {
			_ = zerotrust.NewComplianceReporter(auditLogger, logger)
			logger.Info("Compliance reporter available")
		}
	}

	// Start metrics/health server
	metricsAddr := viper.GetString("metrics-addr")
	metricsMux := http.NewServeMux()

	// Health check endpoint
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	metricsMux.Handle("/metrics", promhttp.Handler())

	// Zero-trust status endpoint
	metricsMux.HandleFunc("/zerotrust/status", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"enabled":             policyEnforcer != nil && policyEnforcer.IsEnabled(),
			"opa_connected":       policyEnforcer != nil,
			"audit_chain_valid":   auditLogger != nil && !auditLogger.IsChainBroken(),
			"cert_rotation_active": certRotator != nil,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Write JSON response (simplified)
		fmt.Fprintf(w, `{"enabled":%v,"opa_connected":%v,"audit_chain_valid":%v,"cert_rotation_active":%v}`,
			status["enabled"], status["opa_connected"], status["audit_chain_valid"], status["cert_rotation_active"])
	})

	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: metricsMux,
	}

	go func() {
		logger.WithField("addr", metricsAddr).Info("Starting metrics/health server")
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
			Reason:    "Proxy started successfully",
			Metadata: map[string]interface{}{
				"version":    version,
				"build_time": buildTime,
			},
		}
		if err := auditLogger.LogEvent(auditEvent); err != nil {
			logger.WithError(err).Warn("Failed to log startup event")
		}
	}

	logger.Info("MarchProxy L3/L4 Proxy started successfully")

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
			Reason:    "Proxy shutting down",
		}
		auditLogger.LogEvent(auditEvent)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Metrics server shutdown error")
	}

	if policyEnforcer != nil {
		policyEnforcer.Close()
	}

	logger.Info("Shutdown complete")
	return nil
}

func loadDefaultPolicies(enforcer *zerotrust.PolicyEnforcer) error {
	// Load RBAC policy
	rbacPolicy := `
package marchproxy.rbac

import rego.v1

default allow := false

allow if {
    input.user != ""
    true
}
`
	if err := enforcer.LoadPolicy("rbac", rbacPolicy); err != nil {
		return fmt.Errorf("failed to load RBAC policy: %w", err)
	}

	return nil
}

func loadDefaultRoles(evaluator *zerotrust.RBACEvaluator) {
	// Define default roles
	roles := []*zerotrust.Role{
		{
			Name: "admin",
			Permissions: []string{
				"*",
			},
			Description: "Full system access",
		},
		{
			Name: "operator",
			Permissions: []string{
				"read:*",
				"write:services",
			},
			Description: "Operator with read access and service management",
		},
		{
			Name: "viewer",
			Permissions: []string{
				"read:*",
			},
			Description: "Read-only access",
		},
	}

	evaluator.LoadRoles(roles)
}
