package afxdp

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Handler manages multiple AF_XDP sockets
type Handler struct {
	device     string
	queueCount int
	sockets    []*Socket
	logger     *logrus.Logger
}

// NewHandler creates a new AF_XDP handler
func NewHandler(device string, queueCount int, logger *logrus.Logger) (*Handler, error) {
	handler := &Handler{
		device:     device,
		queueCount: queueCount,
		sockets:    make([]*Socket, 0, queueCount),
		logger:     logger,
	}

	// Create sockets for each queue
	for i := 0; i < queueCount; i++ {
		socket := NewSocket(device, i, logger)
		handler.sockets = append(handler.sockets, socket)
	}

	logger.WithFields(logrus.Fields{
		"device": device,
		"queues": queueCount,
	}).Info("AF_XDP handler created")

	return handler, nil
}

// Start starts the AF_XDP handler
func (h *Handler) Start() error {
	// Configure all sockets
	for _, socket := range h.sockets {
		if err := socket.Configure(); err != nil {
			return fmt.Errorf("failed to configure socket: %w", err)
		}
	}

	h.logger.Info("AF_XDP handler started")
	return nil
}

// Stop stops the AF_XDP handler
func (h *Handler) Stop() {
	for _, socket := range h.sockets {
		if err := socket.Close(); err != nil {
			h.logger.WithError(err).Error("Failed to close socket")
		}
	}
	h.logger.Info("AF_XDP handler stopped")
}

// GetStats returns AF_XDP statistics
func (h *Handler) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"device":      h.device,
		"queue_count": h.queueCount,
	}

	sockets := make([]map[string]interface{}, 0, len(h.sockets))
	for _, socket := range h.sockets {
		sockets = append(sockets, socket.GetStats())
	}
	stats["sockets"] = sockets

	return stats
}
