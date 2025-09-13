// +build !sriov

package sriov

import (
	"fmt"
	"net"
	"time"

	"github.com/penguintech/marchproxy/internal/manager"
)

// SRIOVManager handles SR-IOV configuration and management (fallback implementation)
type SRIOVManager struct {
	enabled bool
	stats   *SRIOVStats
	config  *SRIOVConfig
}

// PhysicalFunction represents a Physical Function (fallback)
type PhysicalFunction struct {
	PCIAddress        string
	InterfaceName     string
	Driver            string
	VendorID          string
	DeviceID          string
	MaxVFs            int
	CurrentVFs        int
	VFList            []*VirtualFunction
	SupportedFeatures map[string]bool
	Statistics        *PFStats
	LastUpdate        time.Time
}

// VirtualFunction represents a Virtual Function (fallback)
type VirtualFunction struct {
	PCIAddress    string
	InterfaceName string
	ParentPF      string
	VFIndex       int
	MACAddress    net.HardwareAddr
	VLANTag       int
	QoSSettings   *QoSConfig
	TrustMode     bool
	SpoofCheck    bool
	LinkState     string
	Statistics    *VFStats
	AssignedVM    string
	LastActivity  time.Time
}

// PFStats holds Physical Function statistics (fallback)
type PFStats struct {
	RxPackets  uint64
	TxPackets  uint64
	RxBytes    uint64
	TxBytes    uint64
	RxDropped  uint64
	TxDropped  uint64
	RxErrors   uint64
	TxErrors   uint64
	PCIeErrors uint64
	LastUpdate time.Time
}

// VFStats holds Virtual Function statistics (fallback)
type VFStats struct {
	RxPackets       uint64
	TxPackets       uint64
	RxBytes         uint64
	TxBytes         uint64
	RxDropped       uint64
	TxDropped       uint64
	RxErrors        uint64
	TxErrors        uint64
	VLANViolations  uint64
	SpoofViolations uint64
	LastUpdate      time.Time
}

// QoSConfig holds Quality of Service configuration (fallback)
type QoSConfig struct {
	MinBandwidth uint32
	MaxBandwidth uint32
	Priority     uint8
	WeightFactor uint8
	RateLimiting bool
}

// SRIOVStats holds overall SR-IOV statistics (fallback)
type SRIOVStats struct {
	TotalPFs          int
	TotalVFs          int
	ActiveVFs         int
	TotalBandwidth    uint64
	UtilizedBandwidth uint64
	PCIeUtilization   float64
	AverageLatency    time.Duration
	ThroughputMbps    uint64
	ErrorRate         float64
	LastUpdate        time.Time
}

// SRIOVConfig holds SR-IOV configuration parameters (fallback)
type SRIOVConfig struct {
	EnabledPFs        []string
	MaxVFsPerPF       int
	DefaultQoS        *QoSConfig
	VLANMode          string
	SecurityMode      string
	EnableSpoofCheck  bool
	EnableTrustMode   bool
	StatsInterval     time.Duration
	AutoConfiguration bool
	PCIPassthrough    bool
}

// VFConfig holds Virtual Function configuration (fallback)
type VFConfig struct {
	MACAddress  net.HardwareAddr
	VLANTag     *int
	QoSSettings *QoSConfig
	TrustMode   *bool
	SpoofCheck  *bool
	LinkState   *string
	AssignedVM  *string
}

// NewSRIOVManager creates a new SR-IOV manager (fallback)
func NewSRIOVManager(enabled bool, config *SRIOVConfig) *SRIOVManager {
	return &SRIOVManager{
		enabled: false, // Always disabled in fallback mode
		stats: &SRIOVStats{
			LastUpdate: time.Now(),
		},
		config: config,
	}
}

// Initialize discovers and initializes SR-IOV capable devices (fallback)
func (sm *SRIOVManager) Initialize() error {
	fmt.Printf("SR-IOV: Fallback mode - SR-IOV support not compiled in\n")
	return nil
}

// ConfigureVF configures a Virtual Function (fallback)
func (sm *SRIOVManager) ConfigureVF(vfPCIAddr string, config *VFConfig) error {
	fmt.Printf("SR-IOV: Fallback mode - cannot configure VF %s\n", vfPCIAddr)
	return nil
}

// UpdateServices updates service assignments (fallback)
func (sm *SRIOVManager) UpdateServices(services []manager.Service) error {
	fmt.Printf("SR-IOV: Fallback mode - services update ignored\n")
	return nil
}

// Stop disables VFs and cleans up SR-IOV configuration (fallback)
func (sm *SRIOVManager) Stop() error {
	fmt.Printf("SR-IOV: Fallback mode - cleanup complete\n")
	return nil
}

// GetStats returns current SR-IOV statistics (fallback)
func (sm *SRIOVManager) GetStats() *SRIOVStats {
	return &SRIOVStats{
		LastUpdate: time.Now(),
	}
}

// IsEnabled returns whether SR-IOV is enabled (fallback)
func (sm *SRIOVManager) IsEnabled() bool {
	return false
}

// GetPhysicalFunctions returns all discovered Physical Functions (fallback)
func (sm *SRIOVManager) GetPhysicalFunctions() map[string]*PhysicalFunction {
	return make(map[string]*PhysicalFunction)
}

// GetVirtualFunctions returns all discovered Virtual Functions (fallback)
func (sm *SRIOVManager) GetVirtualFunctions() map[string]*VirtualFunction {
	return make(map[string]*VirtualFunction)
}