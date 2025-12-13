package acceleration

import (
	"marchproxy-l3l4/internal/acceleration/xdp"

	"github.com/sirupsen/logrus"
)

// XDPHandler wraps the XDP handler
type XDPHandler struct {
	handler *xdp.Handler
}

// NewXDPHandler creates a new XDP handler
func NewXDPHandler(device string, logger *logrus.Logger) (*XDPHandler, error) {
	handler, err := xdp.NewHandler(device, logger)
	if err != nil {
		return nil, err
	}

	return &XDPHandler{
		handler: handler,
	}, nil
}

// Start starts the XDP handler
func (x *XDPHandler) Start() error {
	return x.handler.Start()
}

// Stop stops the XDP handler
func (x *XDPHandler) Stop() {
	x.handler.Stop()
}

// GetStats returns XDP statistics
func (x *XDPHandler) GetStats() map[string]interface{} {
	return x.handler.GetStats()
}
