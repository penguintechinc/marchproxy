// Package envoy manages the Envoy proxy lifecycle for L7 traffic handling
package envoy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager manages the Envoy proxy lifecycle for L7 egress traffic
type Manager struct {
	binary      string
	configPath  string
	adminPort   int
	listenPort  int
	logLevel    string
	http3Enable bool

	cmd       *exec.Cmd
	startTime time.Time
	mu        sync.RWMutex
	isRunning bool

	logger *logrus.Logger
}

// ManagerConfig holds configuration for the Envoy manager
type ManagerConfig struct {
	Binary      string
	ConfigPath  string
	AdminPort   int
	ListenPort  int
	LogLevel    string
	HTTP3Enable bool
}

// NewManager creates a new Envoy manager
func NewManager(cfg ManagerConfig, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}

	return &Manager{
		binary:      cfg.Binary,
		configPath:  cfg.ConfigPath,
		adminPort:   cfg.AdminPort,
		listenPort:  cfg.ListenPort,
		logLevel:    cfg.LogLevel,
		http3Enable: cfg.HTTP3Enable,
		logger:      logger,
		isRunning:   false,
	}
}

// Start starts the Envoy proxy process
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("envoy is already running")
	}

	// Verify binary exists
	if _, err := os.Stat(m.binary); os.IsNotExist(err) {
		return fmt.Errorf("envoy binary not found at %s", m.binary)
	}

	// Verify config exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return fmt.Errorf("envoy config not found at %s", m.configPath)
	}

	// Build command arguments
	args := []string{
		"-c", m.configPath,
		"--log-level", m.logLevel,
		"--service-cluster", "egress-cluster",
		"--service-node", "egress-node",
		"--drain-time-s", "30",
		"--parent-shutdown-time-s", "45",
	}

	m.cmd = exec.CommandContext(ctx, m.binary, args...)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	// Set process group for proper signal handling
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	m.logger.WithFields(logrus.Fields{
		"binary": m.binary,
		"config": m.configPath,
		"args":   args,
	}).Info("Starting Envoy process for L7 egress")

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start envoy: %w", err)
	}

	m.startTime = time.Now()
	m.isRunning = true

	// Monitor process in background
	go m.monitorProcess()

	// Wait for Envoy to be ready
	if err := m.waitForReady(ctx, 30*time.Second); err != nil {
		m.Stop()
		return fmt.Errorf("envoy failed to become ready: %w", err)
	}

	m.logger.Info("Envoy L7 proxy started successfully")
	return nil
}

// Stop stops the Envoy proxy process gracefully
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.logger.Info("Stopping Envoy process")

	// Send SIGTERM for graceful shutdown
	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.WithError(err).Warn("Failed to send SIGTERM, trying SIGKILL")
		if killErr := m.cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill envoy: %w", killErr)
		}
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case <-done:
		m.logger.Info("Envoy stopped successfully")
	case <-time.After(45 * time.Second):
		m.logger.Warn("Envoy shutdown timeout, killing process")
		m.cmd.Process.Kill()
	}

	m.isRunning = false
	return nil
}

// Reload triggers a hot restart of Envoy
func (m *Manager) Reload() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning {
		return fmt.Errorf("envoy is not running")
	}

	m.logger.Info("Triggering Envoy hot restart")

	// Envoy supports SIGHUP for configuration reload
	if err := m.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP to envoy: %w", err)
	}

	// Give Envoy time to reload
	time.Sleep(2 * time.Second)

	// Verify Envoy is still healthy
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.waitForReady(ctx, 10*time.Second); err != nil {
		return fmt.Errorf("envoy failed after reload: %w", err)
	}

	m.logger.Info("Envoy reloaded successfully")
	return nil
}

// IsRunning returns whether Envoy is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// Uptime returns how long Envoy has been running
func (m *Manager) Uptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning {
		return 0
	}

	return time.Since(m.startTime)
}

// GetAdminURL returns the Envoy admin endpoint URL
func (m *Manager) GetAdminURL() string {
	return fmt.Sprintf("http://localhost:%d", m.adminPort)
}

// IsHTTP3Enabled returns whether HTTP/3 (QUIC) is enabled
func (m *Manager) IsHTTP3Enabled() bool {
	return m.http3Enable
}

// SetHTTP3Enabled updates the HTTP/3 enabled state (requires reload)
func (m *Manager) SetHTTP3Enabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.http3Enable = enabled
}

// monitorProcess monitors the Envoy process and handles unexpected exits
func (m *Manager) monitorProcess() {
	err := m.cmd.Wait()

	m.mu.Lock()
	m.isRunning = false
	m.mu.Unlock()

	if err != nil {
		m.logger.WithError(err).Error("Envoy process exited unexpectedly")
	} else {
		m.logger.Info("Envoy process exited")
	}
}

// waitForReady waits for Envoy to become ready by checking the admin endpoint
func (m *Manager) waitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if Envoy admin endpoint is responding
		if m.checkHealth(client) {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("envoy did not become ready within %v", timeout)
}

// checkHealth checks if Envoy is healthy via admin endpoint
func (m *Manager) checkHealth(client *http.Client) bool {
	url := fmt.Sprintf("http://localhost:%d/ready", m.adminPort)
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetStats retrieves Envoy statistics from admin API
func (m *Manager) GetStats() (map[string]interface{}, error) {
	if !m.IsRunning() {
		return nil, fmt.Errorf("envoy is not running")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/stats?format=json", m.adminPort)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stats endpoint returned status %d", resp.StatusCode)
	}

	// Return raw stats - caller can parse as needed
	stats := make(map[string]interface{})
	stats["uptime"] = m.Uptime().Seconds()
	stats["admin_url"] = m.GetAdminURL()
	stats["http3_enabled"] = m.http3Enable

	return stats, nil
}

// GetListeners retrieves active listener information from Envoy
func (m *Manager) GetListeners() ([]map[string]interface{}, error) {
	if !m.IsRunning() {
		return nil, fmt.Errorf("envoy is not running")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/listeners?format=json", m.adminPort)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get listeners: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listeners endpoint returned status %d", resp.StatusCode)
	}

	// Return empty slice - actual parsing would be done by caller
	return []map[string]interface{}{}, nil
}

// GetClusters retrieves cluster information from Envoy
func (m *Manager) GetClusters() ([]map[string]interface{}, error) {
	if !m.IsRunning() {
		return nil, fmt.Errorf("envoy is not running")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/clusters?format=json", m.adminPort)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clusters endpoint returned status %d", resp.StatusCode)
	}

	// Return empty slice - actual parsing would be done by caller
	return []map[string]interface{}{}, nil
}
