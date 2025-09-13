// +build !afxdp

package afxdp

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/penguintech/marchproxy/internal/manager"
)

// AFXDPManager handles AF_XDP socket management (fallback implementation)
type AFXDPManager struct {
	enabled bool
	stats   *AFXDPStats
	config  *AFXDPConfig
}

// AFXDPSocket represents an AF_XDP socket (fallback)
type AFXDPSocket struct {
	InterfaceName string
	QueueID       int
	SocketInfo    unsafe.Pointer
	UmemInfo      unsafe.Pointer
	Buffer        unsafe.Pointer
	BufferSize    uint64
	RxPackets     uint64
	TxPackets     uint64
	DroppedPkts   uint64
	InvalidPkts   uint64
	LastActivity  time.Time
}

// AFXDPWorker handles packet processing (fallback)
type AFXDPWorker struct {
	ID            int
	Socket        *AFXDPSocket
	ProcessingFn  func([]byte, *AFXDPSocket) bool
	BatchSize     int
	PollTimeout   time.Duration
	PacketsRx     uint64
	PacketsTx     uint64
	ProcessedPkts uint64
	ErrorCount    uint64
}

// AFXDPStats holds AF_XDP performance statistics (fallback)
type AFXDPStats struct {
	TotalRxPackets    uint64
	TotalTxPackets    uint64
	TotalDropped      uint64
	TotalInvalid      uint64
	ZeroCopyFrames    uint64
	UmemFillRing      uint64
	UmemCompRing      uint64
	SocketUtilization []float64
	FramesPerSecond   uint64
	BytesPerSecond    uint64
	LastUpdate        time.Time
}

// AFXDPConfig holds AF_XDP configuration parameters (fallback)
type AFXDPConfig struct {
	Interfaces     []string
	QueuesPerIf    int
	UmemFrameSize  uint32
	UmemFrameCount uint32
	BatchSize      int
	PollTimeout    time.Duration
	WorkerThreads  int
	ZeroCopyMode   bool
	BusyPolling    bool
	TxBudget       int
}

// PacketProcessor defines the interface for packet processing functions (fallback)
type PacketProcessor interface {
	ProcessPacket(data []byte, socket *AFXDPSocket) bool
}

// NewAFXDPManager creates a new AF_XDP manager (fallback)
func NewAFXDPManager(enabled bool, config *AFXDPConfig) *AFXDPManager {
	return &AFXDPManager{
		enabled: false, // Always disabled in fallback mode
		stats: &AFXDPStats{
			LastUpdate: time.Now(),
		},
		config: config,
	}
}

// Initialize sets up AF_XDP sockets and memory regions (fallback)
func (am *AFXDPManager) Initialize() error {
	fmt.Printf("AF_XDP: Fallback mode - AF_XDP support not compiled in\n")
	return nil
}

// StartWorkers starts worker threads for packet processing (fallback)
func (am *AFXDPManager) StartWorkers(processor PacketProcessor) error {
	fmt.Printf("AF_XDP: Fallback mode - no workers to start\n")
	return nil
}

// UpdateServices updates service rules for AF_XDP processing (fallback)
func (am *AFXDPManager) UpdateServices(services []manager.Service) error {
	fmt.Printf("AF_XDP: Fallback mode - services update ignored\n")
	return nil
}

// Stop stops all workers and cleans up resources (fallback)
func (am *AFXDPManager) Stop() error {
	fmt.Printf("AF_XDP: Fallback mode - cleanup complete\n")
	return nil
}

// GetStats returns current AF_XDP statistics (fallback)
func (am *AFXDPManager) GetStats() *AFXDPStats {
	return &AFXDPStats{
		LastUpdate: time.Now(),
	}
}

// IsEnabled returns whether AF_XDP is enabled (fallback)
func (am *AFXDPManager) IsEnabled() bool {
	return false
}

// GetSocketStats returns statistics for a specific socket (fallback)
func (am *AFXDPManager) GetSocketStats(interfaceName string, queueID int) (*AFXDPSocket, error) {
	return nil, fmt.Errorf("AF_XDP not available in fallback mode")
}

// GetWorkerStats returns statistics for all workers (fallback)
func (am *AFXDPManager) GetWorkerStats() []*AFXDPWorker {
	return []*AFXDPWorker{}
}

// SetZeroCopyMode enables or disables zero-copy mode (fallback)
func (am *AFXDPManager) SetZeroCopyMode(enabled bool) {
	fmt.Printf("AF_XDP: Fallback mode - zero-copy mode setting ignored\n")
}

// GetActiveInterfaces returns list of interfaces with active AF_XDP sockets (fallback)
func (am *AFXDPManager) GetActiveInterfaces() []string {
	return []string{}
}