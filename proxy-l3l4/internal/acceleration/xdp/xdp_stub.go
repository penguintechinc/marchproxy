//go:build noxdp
// +build noxdp

package xdp

import (
	"fmt"

	"marchproxy-l3l4/internal/logging"
)

// XDPProgram represents an XDP eBPF program (stub version for noxdp builds)
type XDPProgram struct {
	device string
	logger *logging.LogrusAdapter
}

// NewXDPProgram creates a new XDP program (stub - returns error in noxdp builds)
func NewXDPProgram(device string, logger *logging.LogrusAdapter) *XDPProgram {
	return &XDPProgram{
		device: device,
		logger: logger,
	}
}

// Load loads the XDP program (stub)
func (p *XDPProgram) Load(filter string) error {
	return fmt.Errorf("XDP support disabled in this build (use -tags xdp to enable)")
}

// Unload unloads the XDP program (stub)
func (p *XDPProgram) Unload() error {
	return nil
}

// Attach attaches the XDP program (stub)
func (p *XDPProgram) Attach() error {
	return fmt.Errorf("XDP not supported")
}

// Detach detaches the XDP program (stub)
func (p *XDPProgram) Detach() error {
	return fmt.Errorf("XDP not supported")
}

// Close closes the XDP program (stub)
func (p *XDPProgram) Close() error {
	return nil
}

// GetStats returns XDP statistics (stub)
func (p *XDPProgram) GetStats() map[string]interface{} {
	return map[string]interface{}{"error": "XDP not supported"}
}
