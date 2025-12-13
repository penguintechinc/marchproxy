package acceleration

import (
	"marchproxy-l3l4/internal/acceleration/afxdp"

	"github.com/sirupsen/logrus"
)

// AFXDPHandler wraps the AF_XDP handler
type AFXDPHandler struct {
	handler *afxdp.Handler
}

// NewAFXDPHandler creates a new AF_XDP handler
func NewAFXDPHandler(device string, queueCount int, logger *logrus.Logger) (*AFXDPHandler, error) {
	handler, err := afxdp.NewHandler(device, queueCount, logger)
	if err != nil {
		return nil, err
	}

	return &AFXDPHandler{
		handler: handler,
	}, nil
}

// Start starts the AF_XDP handler
func (a *AFXDPHandler) Start() error {
	return a.handler.Start()
}

// Stop stops the AF_XDP handler
func (a *AFXDPHandler) Stop() {
	a.handler.Stop()
}

// GetStats returns AF_XDP statistics
func (a *AFXDPHandler) GetStats() map[string]interface{} {
	return a.handler.GetStats()
}
