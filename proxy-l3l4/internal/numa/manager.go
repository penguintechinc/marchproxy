package numa

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
)

// Manager handles NUMA topology detection and CPU affinity
type Manager struct {
	topology *Topology
	logger   *logrus.Logger
	enabled  bool
}

// NewManager creates a new NUMA manager
func NewManager(logger *logrus.Logger) *Manager {
	return &Manager{
		logger:  logger,
		enabled: false,
	}
}

// Initialize initializes NUMA support
func (m *Manager) Initialize() error {
	m.logger.Info("Initializing NUMA support")

	// Detect NUMA topology
	topology, err := DetectTopology()
	if err != nil {
		m.logger.WithError(err).Warn("Failed to detect NUMA topology, running in non-NUMA mode")
		m.enabled = false
		return nil
	}

	m.topology = topology
	m.enabled = true

	m.logger.WithFields(logrus.Fields{
		"nodes":        topology.NodeCount,
		"cpus_per_node": topology.CPUsPerNode,
		"total_cpus":   topology.TotalCPUs,
		"platform":     runtime.GOOS,
	}).Info("NUMA topology detected")

	return nil
}

// IsEnabled returns whether NUMA is enabled and available
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// GetTopology returns the NUMA topology
func (m *Manager) GetTopology() *Topology {
	return m.topology
}

// BindToNode binds the current goroutine to a specific NUMA node
func (m *Manager) BindToNode(nodeID int) error {
	if !m.enabled {
		return fmt.Errorf("NUMA not enabled")
	}

	if nodeID < 0 || nodeID >= m.topology.NodeCount {
		return fmt.Errorf("invalid NUMA node ID: %d", nodeID)
	}

	// Get CPUs for this node
	cpus, err := m.topology.GetNodeCPUs(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get CPUs for node %d: %w", nodeID, err)
	}

	// Set CPU affinity
	if err := SetCPUAffinity(cpus); err != nil {
		return fmt.Errorf("failed to set CPU affinity: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"node": nodeID,
		"cpus": cpus,
	}).Debug("Bound to NUMA node")

	return nil
}

// BindCurrentThread binds the current OS thread to specific CPUs
func (m *Manager) BindCurrentThread(cpuIDs []int) error {
	if !m.enabled {
		return fmt.Errorf("NUMA not enabled")
	}

	// Lock OS thread to ensure affinity stays
	runtime.LockOSThread()

	if err := SetCPUAffinity(cpuIDs); err != nil {
		runtime.UnlockOSThread()
		return fmt.Errorf("failed to set CPU affinity: %w", err)
	}

	m.logger.WithField("cpus", cpuIDs).Debug("Bound thread to CPUs")
	return nil
}

// GetOptimalNode returns the optimal NUMA node for the current workload
func (m *Manager) GetOptimalNode() int {
	if !m.enabled || m.topology == nil {
		return 0
	}

	// Simple strategy: use node 0 for now
	// In production, this could be based on memory allocation, network cards, etc.
	return 0
}

// AllocateWorkers allocates worker threads across NUMA nodes
func (m *Manager) AllocateWorkers(workerCount int) ([]WorkerAllocation, error) {
	if !m.enabled || m.topology == nil {
		// No NUMA, allocate workers without affinity
		allocations := make([]WorkerAllocation, workerCount)
		for i := 0; i < workerCount; i++ {
			allocations[i] = WorkerAllocation{
				WorkerID: i,
				NodeID:   0,
				CPUIDs:   []int{},
			}
		}
		return allocations, nil
	}

	// Distribute workers evenly across NUMA nodes
	allocations := make([]WorkerAllocation, workerCount)
	nodeCount := m.topology.NodeCount
	workersPerNode := workerCount / nodeCount
	remainder := workerCount % nodeCount

	workerID := 0
	for nodeID := 0; nodeID < nodeCount; nodeID++ {
		workersForThisNode := workersPerNode
		if nodeID < remainder {
			workersForThisNode++
		}

		cpus, err := m.topology.GetNodeCPUs(nodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get CPUs for node %d: %w", nodeID, err)
		}

		cpusPerWorker := len(cpus) / workersForThisNode
		if cpusPerWorker == 0 {
			cpusPerWorker = 1
		}

		for w := 0; w < workersForThisNode; w++ {
			// Assign CPUs to this worker
			startCPU := w * cpusPerWorker
			endCPU := startCPU + cpusPerWorker
			if endCPU > len(cpus) {
				endCPU = len(cpus)
			}

			allocations[workerID] = WorkerAllocation{
				WorkerID: workerID,
				NodeID:   nodeID,
				CPUIDs:   cpus[startCPU:endCPU],
			}
			workerID++
		}
	}

	m.logger.WithFields(logrus.Fields{
		"total_workers": workerCount,
		"nodes":         nodeCount,
	}).Info("Allocated workers across NUMA nodes")

	return allocations, nil
}

// WorkerAllocation represents a worker thread's NUMA allocation
type WorkerAllocation struct {
	WorkerID int
	NodeID   int
	CPUIDs   []int
}

// Stats returns NUMA statistics
func (m *Manager) Stats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled": m.enabled,
	}

	if m.enabled && m.topology != nil {
		stats["nodes"] = m.topology.NodeCount
		stats["cpus_per_node"] = m.topology.CPUsPerNode
		stats["total_cpus"] = m.topology.TotalCPUs
	}

	return stats
}
