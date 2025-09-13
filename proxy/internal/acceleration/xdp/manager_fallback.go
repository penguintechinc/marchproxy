// +build !xdp

package xdp

import (
	"fmt"
	"time"

	"github.com/penguintech/marchproxy/internal/manager"
)

// XDPManager handles XDP program lifecycle and management (fallback implementation)
type XDPManager struct {
	enabled bool
	stats   *XDPStats
	config  *XDPConfig
}

// XDPInterface represents an interface with XDP attached (fallback)
type XDPInterface struct {
	Name          string
	Index         int
	ProgramFD     int
	AttachFlags   uint32
	RxPackets     uint64
	TxPackets     uint64
	DroppedPkts   uint64
	RedirectPkts  uint64
	LastUpdate    time.Time
}

// XDPStats holds XDP performance statistics (fallback)
type XDPStats struct {
	TotalPackets      uint64
	PassedPackets     uint64
	DroppedPackets    uint64
	RedirectedPackets uint64
	TCPPackets        uint64
	UDPPackets        uint64
	OtherPackets      uint64
	MalformedPackets  uint64
	PacketsPerSecond  uint64
	LastUpdate        time.Time
}

// XDPConfig holds XDP configuration parameters (fallback)
type XDPConfig struct {
	ProgramPath     string
	Interfaces      []string
	AttachMode      string
	ForceReplace    bool
	EnableStats     bool
	StatsInterval   time.Duration
	BurstSize       uint32
	BatchTimeout    time.Duration
}

// ServiceRule represents a service filtering rule for XDP (fallback)
type ServiceRule struct {
	ServiceID    uint32
	IPAddr       uint32
	Port         uint16
	Protocol     uint8
	Action       uint8
	RedirectIP   uint32
	RedirectPort uint16
	AuthRequired uint8
	Reserved     uint8
}

// NewXDPManager creates a new XDP manager (fallback)
func NewXDPManager(enabled bool, config *XDPConfig) *XDPManager {
	return &XDPManager{
		enabled: false, // Always disabled in fallback mode
		stats: &XDPStats{
			LastUpdate: time.Now(),
		},
		config: config,
	}
}

// Initialize loads the XDP program (fallback)
func (xm *XDPManager) Initialize() error {
	fmt.Printf("XDP: Fallback mode - XDP support not compiled in\n")
	return nil
}

// AttachToInterface attaches XDP program to a network interface (fallback)
func (xm *XDPManager) AttachToInterface(interfaceName string) error {
	fmt.Printf("XDP: Fallback mode - cannot attach to interface %s\n", interfaceName)
	return nil
}

// DetachFromInterface detaches XDP program from a network interface (fallback)
func (xm *XDPManager) DetachFromInterface(interfaceName string) error {
	fmt.Printf("XDP: Fallback mode - cannot detach from interface %s\n", interfaceName)
	return nil
}

// UpdateServices synchronizes services with XDP maps (fallback)
func (xm *XDPManager) UpdateServices(services []manager.Service) error {
	fmt.Printf("XDP: Fallback mode - services update ignored\n")
	return nil
}

// Stop detaches from all interfaces and unloads the program (fallback)
func (xm *XDPManager) Stop() error {
	fmt.Printf("XDP: Fallback mode - cleanup complete\n")
	return nil
}

// GetStats returns current XDP statistics (fallback)
func (xm *XDPManager) GetStats() *XDPStats {
	return &XDPStats{
		LastUpdate: time.Now(),
	}
}

// IsEnabled returns whether XDP is enabled (fallback)
func (xm *XDPManager) IsEnabled() bool {
	return false
}

// GetAttachedInterfaces returns list of interfaces with XDP attached (fallback)
func (xm *XDPManager) GetAttachedInterfaces() []string {
	return []string{}
}

// GetInterfaceStats returns statistics for a specific interface (fallback)
func (xm *XDPManager) GetInterfaceStats(interfaceName string) (*XDPInterface, error) {
	return nil, fmt.Errorf("XDP not available in fallback mode")
}