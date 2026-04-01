package xdp

import (
	"fmt"

	"marchproxy-l3l4/internal/logging"
)

// Handler wraps XDP functionality
type Handler struct {
	program *XDPProgram
	logger  *logging.LogrusAdapter
}

// NewHandler creates a new XDP handler
func NewHandler(device string, logger *logging.LogrusAdapter) (*Handler, error) {
	program := NewXDPProgram(device, logger)

	return &Handler{
		program: program,
		logger:  logger,
	}, nil
}

// Start starts the XDP handler
func (h *Handler) Start() error {
	// Load XDP program
	if err := h.program.Load(""); err != nil {
		return fmt.Errorf("failed to load XDP program: %w", err)
	}

	h.logger.Info("XDP handler started")
	return nil
}

// Stop stops the XDP handler
func (h *Handler) Stop() {
	if err := h.program.Unload(); err != nil {
		h.logger.WithError(err).Error("Failed to unload XDP program")
	}
	h.logger.Info("XDP handler stopped")
}

// GetStats returns XDP statistics
func (h *Handler) GetStats() map[string]interface{} {
	return h.program.GetStats()
}
