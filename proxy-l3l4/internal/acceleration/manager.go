package acceleration

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// AccelerationMode defines the acceleration tier
type AccelerationMode int

const (
	ModeStandard AccelerationMode = iota
	ModeXDP
	ModeAFXDP
	ModeDPDK
)

// Manager manages hardware acceleration features
type Manager struct {
	mode   AccelerationMode
	logger *logrus.Logger

	// Acceleration components
	xdpHandler   *XDPHandler
	afxdpHandler *AFXDPHandler
}

// NewManager creates a new acceleration manager
func NewManager(mode string, logger *logrus.Logger) (*Manager, error) {
	var accelMode AccelerationMode

	switch mode {
	case "standard":
		accelMode = ModeStandard
	case "xdp":
		accelMode = ModeXDP
	case "afxdp":
		accelMode = ModeAFXDP
	case "dpdk":
		accelMode = ModeDPDK
	default:
		return nil, fmt.Errorf("unknown acceleration mode: %s", mode)
	}

	manager := &Manager{
		mode:   accelMode,
		logger: logger,
	}

	logger.WithField("mode", mode).Info("Acceleration manager initialized")
	return manager, nil
}

// Initialize initializes the acceleration subsystem
func (m *Manager) Initialize(device string, queueCount int) error {
	switch m.mode {
	case ModeStandard:
		m.logger.Info("Using standard networking (no hardware acceleration)")
		return nil

	case ModeXDP:
		m.logger.WithField("device", device).Info("Initializing XDP acceleration")
		handler, err := NewXDPHandler(device, m.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize XDP: %w", err)
		}
		m.xdpHandler = handler
		return nil

	case ModeAFXDP:
		m.logger.WithFields(logrus.Fields{
			"device": device,
			"queues": queueCount,
		}).Info("Initializing AF_XDP acceleration")
		handler, err := NewAFXDPHandler(device, queueCount, m.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize AF_XDP: %w", err)
		}
		m.afxdpHandler = handler
		return nil

	case ModeDPDK:
		m.logger.Warn("DPDK mode not yet implemented, falling back to standard")
		m.mode = ModeStandard
		return nil

	default:
		return fmt.Errorf("unsupported acceleration mode: %d", m.mode)
	}
}

// Start starts the acceleration subsystem
func (m *Manager) Start() error {
	switch m.mode {
	case ModeXDP:
		if m.xdpHandler != nil {
			return m.xdpHandler.Start()
		}
	case ModeAFXDP:
		if m.afxdpHandler != nil {
			return m.afxdpHandler.Start()
		}
	}
	return nil
}

// Stop stops the acceleration subsystem
func (m *Manager) Stop() {
	switch m.mode {
	case ModeXDP:
		if m.xdpHandler != nil {
			m.xdpHandler.Stop()
		}
	case ModeAFXDP:
		if m.afxdpHandler != nil {
			m.afxdpHandler.Stop()
		}
	}
	m.logger.Info("Acceleration subsystem stopped")
}

// GetMode returns the current acceleration mode
func (m *Manager) GetMode() string {
	switch m.mode {
	case ModeStandard:
		return "standard"
	case ModeXDP:
		return "xdp"
	case ModeAFXDP:
		return "afxdp"
	case ModeDPDK:
		return "dpdk"
	default:
		return "unknown"
	}
}

// IsAccelerated returns true if hardware acceleration is enabled
func (m *Manager) IsAccelerated() bool {
	return m.mode != ModeStandard
}

// GetStats returns acceleration statistics
func (m *Manager) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"mode":        m.GetMode(),
		"accelerated": m.IsAccelerated(),
	}

	switch m.mode {
	case ModeXDP:
		if m.xdpHandler != nil {
			stats["xdp"] = m.xdpHandler.GetStats()
		}
	case ModeAFXDP:
		if m.afxdpHandler != nil {
			stats["afxdp"] = m.afxdpHandler.GetStats()
		}
	}

	return stats
}
