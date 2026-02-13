// +build !xdp

package xdp

import (
	"log"
	"time"
)

// XDPManager handles XDP acceleration (fallback implementation)
type XDPManager struct {
	config  *XDPConfig
	running bool
	enabled bool
}

// XDPConfig holds XDP configuration
type XDPConfig struct {
	InterfaceName string
	Mode          string
	ProgramPath   string
	MapPinPath    string
}

// XDPStats holds XDP statistics
type XDPStats struct {
	TotalPackets   uint64
	PassedPackets  uint64
	DroppedPackets uint64
	TotalBytes     uint64
	LastUpdate     time.Time
}

// ServiceRule represents a service filtering rule for XDP
type ServiceRule struct {
	ServiceID    uint32
	IPAddr       uint32
	Port         uint16
	Protocol     uint8
	Action       uint8 // 0=drop, 1=pass, 2=redirect
	RedirectIP   uint32
	RedirectPort uint16
	AuthRequired uint8
	Reserved     uint8
}

// NewXDPManager creates a new XDP manager (fallback)
func NewXDPManager(config *XDPConfig) (*XDPManager, error) {
	return &XDPManager{
		config: config,
	}, nil
}

// Initialize initializes XDP (fallback - no-op)
func (xm *XDPManager) Initialize() error {
	log.Printf("XDP: Using fallback implementation (XDP not available)")
	return nil
}

// Start starts XDP processing (fallback - no-op)
func (xm *XDPManager) Start() error {
	xm.running = true
	log.Printf("XDP: Fallback implementation started")
	return nil
}

// Stop stops XDP processing (fallback - no-op)
func (xm *XDPManager) Stop() error {
	xm.running = false
	log.Printf("XDP: Fallback implementation stopped")
	return nil
}

// IsRunning returns whether XDP is running
func (xm *XDPManager) IsRunning() bool {
	return xm.running
}

// IsEnabled returns whether XDP is enabled
func (xm *XDPManager) IsEnabled() bool {
	return xm.enabled && xm.running
}

// ClearServiceRules clears all service rules (fallback - no-op)
func (xm *XDPManager) ClearServiceRules() error {
	log.Printf("XDP: ClearServiceRules called on fallback implementation")
	return nil
}

// AddServiceRule adds a service rule (fallback - no-op)
func (xm *XDPManager) AddServiceRule(ruleID uint32, rule *ServiceRule) error {
	log.Printf("XDP: AddServiceRule called on fallback implementation (rule %d)", ruleID)
	return nil
}

// GetStats returns XDP statistics (fallback - empty stats)
func (xm *XDPManager) GetStats() *XDPStats {
	return &XDPStats{
		LastUpdate: time.Now(),
	}
}

// UpdateRedirectMap updates the redirect map (fallback - no-op)
func (xm *XDPManager) UpdateRedirectMap(redirectMap map[uint32]uint32) error {
	log.Printf("XDP: UpdateRedirectMap called on fallback implementation")
	return nil
}

// GetConfig returns XDP configuration
func (xm *XDPManager) GetConfig() *XDPConfig {
	return xm.config
}