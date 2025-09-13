// +build !dpdk

package dpdk

import (
	"fmt"
	"time"
)

// DPDKManager handles DPDK initialization and packet processing (fallback implementation)
type DPDKManager struct {
	enabled bool
	stats   *DPDKStats
	config  *DPDKConfig
}

// DPDKPort represents a DPDK-enabled network port (fallback)
type DPDKPort struct {
	ID            uint16
	NbRxQueues    uint16
	NbTxQueues    uint16
	LinkStatus    bool
	RxPackets     uint64
	TxPackets     uint64
	RxBytes       uint64
	TxBytes       uint64
	RxDropped     uint64
	TxDropped     uint64
}

// DPDKPacket represents a packet in DPDK format (fallback)
type DPDKPacket struct {
	Data      []byte
	Length    uint16
	PortID    uint16
	QueueID   uint16
	Timestamp time.Time
}

// DPDKStats holds DPDK performance statistics (fallback)
type DPDKStats struct {
	TotalRxPackets    uint64
	TotalTxPackets    uint64
	TotalRxBytes      uint64
	TotalTxBytes      uint64
	TotalRxDropped    uint64
	TotalTxDropped    uint64
	WorkerUtilization []float64
	PacketsPerSecond  uint64
	BytesPerSecond    uint64
	LastUpdate        time.Time
}

// DPDKConfig holds DPDK configuration parameters (fallback)
type DPDKConfig struct {
	EALArgs           []string
	NbMbufs           uint32
	MempoolCacheSize  uint32
	DataRoomSize      uint16
	RxDescriptors     uint16
	TxDescriptors     uint16
	RxQueuesPerPort   uint16
	TxQueuesPerPort   uint16
	WorkerCores       []int
	BurstSize         uint16
	PrefetchOffset    uint8
}

// NewDPDKManager creates a new DPDK manager (fallback)
func NewDPDKManager(enabled bool, config *DPDKConfig) *DPDKManager {
	return &DPDKManager{
		enabled: false, // Always disabled in fallback mode
		stats: &DPDKStats{
			LastUpdate: time.Now(),
		},
		config: config,
	}
}

// Initialize initializes DPDK EAL and sets up memory pools (fallback)
func (dm *DPDKManager) Initialize() error {
	fmt.Printf("DPDK: Fallback mode - DPDK support not compiled in\n")
	return nil
}

// AddPort configures and starts a DPDK port (fallback)
func (dm *DPDKManager) AddPort(portID uint16) error {
	fmt.Printf("DPDK: Fallback mode - cannot add port %d\n", portID)
	return nil
}

// StartWorkers starts DPDK worker threads for packet processing (fallback)
func (dm *DPDKManager) StartWorkers() error {
	fmt.Printf("DPDK: Fallback mode - no workers to start\n")
	return nil
}

// Stop stops all DPDK workers and cleans up resources (fallback)
func (dm *DPDKManager) Stop() error {
	fmt.Printf("DPDK: Fallback mode - cleanup complete\n")
	return nil
}

// GetStats returns current DPDK statistics (fallback)
func (dm *DPDKManager) GetStats() *DPDKStats {
	return &DPDKStats{
		LastUpdate: time.Now(),
	}
}

// IsEnabled returns whether DPDK is enabled (fallback)
func (dm *DPDKManager) IsEnabled() bool {
	return false
}

// GetPortStats returns statistics for a specific port (fallback)
func (dm *DPDKManager) GetPortStats(portID uint16) (*DPDKPort, error) {
	return nil, fmt.Errorf("DPDK not available in fallback mode")
}