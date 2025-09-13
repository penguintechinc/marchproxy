// +build !numa

package numa

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/penguintech/marchproxy/internal/manager"
)

// NUMAManager handles NUMA-aware memory management (fallback implementation)
type NUMAManager struct {
	enabled bool
	stats   *NUMAStats
	config  *NUMAConfig
}

// NUMATopology represents the NUMA topology (fallback)
type NUMATopology struct {
	NumNodes    int
	NumCPUs     int
	NodesOnline []int
	CPUsOnline  []int
	NodeCPUMap  map[int][]int
	CPUNodeMap  map[int]int
	Distances   map[int]map[int]int
	MemoryInfo  map[int]*NodeMemoryInfo
	LastScan    time.Time
}

// NUMANode represents a NUMA node (fallback)
type NUMANode struct {
	ID             int
	CPUs           []int
	MemoryTotal    uint64
	MemoryFree     uint64
	MemoryUsed     uint64
	HugePagesTotal int
	HugePagesFree  int
	HugePagesSize  int
	Allocations    []*MemoryAllocation
	LoadAverage    float64
	Utilization    float64
	LastUpdate     time.Time
}

// NodeMemoryInfo holds detailed memory information (fallback)
type NodeMemoryInfo struct {
	MemTotal     uint64
	MemFree      uint64
	MemAvailable uint64
	Buffers      uint64
	Cached       uint64
	Active       uint64
	Inactive     uint64
	Dirty        uint64
	WriteBack    uint64
	Slab         uint64
	PageTables   uint64
	LastUpdate   time.Time
}

// MemoryPolicy defines memory allocation policies (fallback)
type MemoryPolicy struct {
	Name        string
	Policy      int
	NodeMask    uint64
	Flags       uint32
	Description string
}

// MemoryAllocation tracks memory allocations (fallback)
type MemoryAllocation struct {
	Address   uintptr
	Size      uint64
	Node      int
	Policy    string
	Timestamp time.Time
	ProcessID int
	ThreadID  int
	Purpose   string
}

// NUMAStats holds NUMA performance statistics (fallback)
type NUMAStats struct {
	TotalAllocations   uint64
	TotalDeallocations uint64
	CrossNodeAccesses  uint64
	LocalNodeAccesses  uint64
	NumaHitRatio       float64
	MemoryMigrations   uint64
	PageFaults         uint64
	RemotePageFaults   uint64
	InterleaveHit      uint64
	InterleaveMiss     uint64
	PreferredNodeHit   uint64
	PreferredNodeMiss  uint64
	LastUpdate         time.Time
}

// NUMAConfig holds NUMA configuration parameters (fallback)
type NUMAConfig struct {
	EnableAutoBinding   bool
	DefaultPolicy       string
	HugePagesEnabled    bool
	HugePagesSize       int
	PreferredNodes      []int
	ExcludedNodes       []int
	MemoryMigration     bool
	AutoBalance         bool
	StatsInterval       time.Duration
	AllocationThreshold uint64
	LocalityThreshold   float64
}

// NewNUMAManager creates a new NUMA manager (fallback)
func NewNUMAManager(enabled bool, config *NUMAConfig) *NUMAManager {
	return &NUMAManager{
		enabled: false, // Always disabled in fallback mode
		stats: &NUMAStats{
			LastUpdate: time.Now(),
		},
		config: config,
	}
}

// Initialize discovers NUMA topology (fallback)
func (nm *NUMAManager) Initialize() error {
	fmt.Printf("NUMA: Fallback mode - NUMA support not compiled in\n")
	return nil
}

// AllocateMemory allocates memory (fallback - uses regular malloc)
func (nm *NUMAManager) AllocateMemory(size uint64, policy string, nodeHint int) (unsafe.Pointer, error) {
	// Use regular memory allocation
	ptr := unsafe.Pointer(make([]byte, size))
	if ptr == nil {
		return nil, fmt.Errorf("failed to allocate memory")
	}
	return ptr, nil
}

// FreeMemory frees memory (fallback)
func (nm *NUMAManager) FreeMemory(ptr unsafe.Pointer) error {
	// Memory will be freed by garbage collector
	fmt.Printf("NUMA: Fallback mode - memory will be freed by GC\n")
	return nil
}

// SetThreadAffinity sets thread affinity (fallback)
func (nm *NUMAManager) SetThreadAffinity(nodeID int) error {
	fmt.Printf("NUMA: Fallback mode - thread affinity not supported\n")
	return nil
}

// UpdateServices updates service placement (fallback)
func (nm *NUMAManager) UpdateServices(services []manager.Service) error {
	fmt.Printf("NUMA: Fallback mode - service optimization ignored\n")
	return nil
}

// GetOptimalNode returns optimal node (fallback)
func (nm *NUMAManager) GetOptimalNode(workloadType string, requiredMemory uint64) (int, error) {
	return 0, nil // Always return node 0
}

// Stop cleans up resources (fallback)
func (nm *NUMAManager) Stop() error {
	fmt.Printf("NUMA: Fallback mode - cleanup complete\n")
	return nil
}

// GetStats returns NUMA statistics (fallback)
func (nm *NUMAManager) GetStats() *NUMAStats {
	return &NUMAStats{
		LastUpdate: time.Now(),
	}
}

// IsEnabled returns whether NUMA is enabled (fallback)
func (nm *NUMAManager) IsEnabled() bool {
	return false
}

// GetTopology returns NUMA topology (fallback)
func (nm *NUMAManager) GetTopology() *NUMATopology {
	return nil
}

// GetNodes returns NUMA nodes (fallback)
func (nm *NUMAManager) GetNodes() map[int]*NUMANode {
	return make(map[int]*NUMANode)
}

// GetNodeDistance returns distance between nodes (fallback)
func (nm *NUMAManager) GetNodeDistance(node1, node2 int) (int, error) {
	return -1, fmt.Errorf("NUMA not available in fallback mode")
}