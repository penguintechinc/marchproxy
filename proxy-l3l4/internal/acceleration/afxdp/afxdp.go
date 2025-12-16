package afxdp

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// Socket represents an AF_XDP socket (stub implementation)
type Socket struct {
	mu sync.RWMutex

	device     string
	queueID    int
	logger     *logrus.Logger
	configured bool

	// Statistics
	packetsRx uint64
	packetsTx uint64
	bytesRx   uint64
	bytesTx   uint64
}

// NewSocket creates a new AF_XDP socket
func NewSocket(device string, queueID int, logger *logrus.Logger) *Socket {
	return &Socket{
		device:  device,
		queueID: queueID,
		logger:  logger,
	}
}

// Configure configures the AF_XDP socket
func (s *Socket) Configure() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.configured {
		return fmt.Errorf("socket already configured")
	}

	s.configured = true
	s.logger.WithFields(logrus.Fields{
		"device":   s.device,
		"queue_id": s.queueID,
	}).Info("AF_XDP socket configured (stub)")

	return nil
}

// Close closes the AF_XDP socket
func (s *Socket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.configured {
		return nil
	}

	s.configured = false
	s.logger.WithField("queue_id", s.queueID).Info("AF_XDP socket closed (stub)")

	return nil
}

// IsConfigured returns whether the socket is configured
func (s *Socket) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configured
}

// GetStats returns socket statistics
func (s *Socket) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"device":     s.device,
		"queue_id":   s.queueID,
		"configured": s.configured,
		"packets_rx": s.packetsRx,
		"packets_tx": s.packetsTx,
		"bytes_rx":   s.bytesRx,
		"bytes_tx":   s.bytesTx,
		"stub":       true,
	}
}

// Poll polls the socket for packets
func (s *Socket) Poll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.configured {
		return fmt.Errorf("socket not configured")
	}

	return nil
}
