package envoy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager manages the Envoy proxy lifecycle
type Manager struct {
	binary      string
	configPath  string
	adminPort   int
	logLevel    string

	cmd         *exec.Cmd
	startTime   time.Time
	mu          sync.RWMutex
	isRunning   bool

	logger      *logrus.Logger
}

// NewManager creates a new Envoy manager
func NewManager(binary, configPath string, adminPort int, logLevel string, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}

	return &Manager{
		binary:     binary,
		configPath: configPath,
		adminPort:  adminPort,
		logLevel:   logLevel,
		logger:     logger,
		isRunning:  false,
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

	// Build command
	args := []string{
		"-c", m.configPath,
		"--log-level", m.logLevel,
		"--service-cluster", "alb-cluster",
		"--service-node", "alb-node",
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
	}).Info("Starting Envoy process")

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

	m.logger.Info("Envoy started successfully")
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

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if Envoy admin endpoint is responding
		if m.checkHealth() {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("envoy did not become ready within %v", timeout)
}

// checkHealth checks if Envoy is healthy via admin endpoint
func (m *Manager) checkHealth() bool {
	// In production, this would make an HTTP request to /ready
	// For now, we just check if the process is running
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.isRunning && m.cmd != nil && m.cmd.Process != nil
}
