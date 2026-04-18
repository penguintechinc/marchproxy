//go:build ci
// +build ci

package xdp

import (
	"fmt"
	"sync"

	"marchproxy-l3l4/internal/logging"
)

// XDPProgram represents an XDP eBPF program (CI stub implementation)
type XDPProgram struct {
	mu sync.RWMutex

	device  string
	logger  *logging.LogrusAdapter
	loaded  bool

	// Statistics
	packetsProcessed uint64
	packetsDropped   uint64
	bytesProcessed   uint64
}

// NewXDPProgram creates a new XDP program instance
func NewXDPProgram(device string, logger *logging.LogrusAdapter) *XDPProgram {
	return &XDPProgram{
		device: device,
		logger: logger,
	}
}

// Load loads and attaches the XDP program
func (xdp *XDPProgram) Load(programPath string) error {
	xdp.mu.Lock()
	defer xdp.mu.Unlock()

	if xdp.loaded {
		return fmt.Errorf("XDP program already loaded")
	}

	xdp.logger.WithField("device", xdp.device).Info("XDP program loaded (CI stub)")
	xdp.loaded = true

	return nil
}

// Unload unloads the XDP program
func (xdp *XDPProgram) Unload() error {
	xdp.mu.Lock()
	defer xdp.mu.Unlock()

	if !xdp.loaded {
		return nil
	}

	xdp.loaded = false
	xdp.logger.WithField("device", xdp.device).Info("XDP program unloaded (CI stub)")

	return nil
}

// IsLoaded returns whether the program is loaded
func (xdp *XDPProgram) IsLoaded() bool {
	xdp.mu.RLock()
	defer xdp.mu.RUnlock()
	return xdp.loaded
}

// GetStats returns XDP statistics
func (xdp *XDPProgram) GetStats() map[string]interface{} {
	xdp.mu.RLock()
	defer xdp.mu.RUnlock()

	return map[string]interface{}{
		"device":            xdp.device,
		"loaded":            xdp.loaded,
		"packets_processed": xdp.packetsProcessed,
		"packets_dropped":   xdp.packetsDropped,
		"bytes_processed":   xdp.bytesProcessed,
		"stub":              true,
		"ci":                true,
	}
}

// UpdateStats updates internal statistics
func (xdp *XDPProgram) UpdateStats() error {
	xdp.mu.Lock()
	defer xdp.mu.Unlock()

	if !xdp.loaded {
		return fmt.Errorf("XDP program not loaded")
	}

	return nil
}
