package numa

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Topology represents the NUMA topology of the system
type Topology struct {
	NodeCount   int
	CPUsPerNode map[int][]int
	TotalCPUs   int
}

// DetectTopology detects the NUMA topology of the system
func DetectTopology() (*Topology, error) {
	switch runtime.GOOS {
	case "linux":
		return detectLinuxTopology()
	case "darwin":
		return detectDarwinTopology()
	case "windows":
		return detectWindowsTopology()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// detectLinuxTopology detects NUMA topology on Linux
func detectLinuxTopology() (*Topology, error) {
	// Check if NUMA is available
	if _, err := os.Stat("/sys/devices/system/node"); os.IsNotExist(err) {
		return nil, fmt.Errorf("NUMA not available on this system")
	}

	// Read NUMA nodes
	entries, err := os.ReadDir("/sys/devices/system/node")
	if err != nil {
		return nil, fmt.Errorf("failed to read NUMA nodes: %w", err)
	}

	topology := &Topology{
		CPUsPerNode: make(map[int][]int),
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "node") {
			continue
		}

		// Extract node ID
		nodeIDStr := strings.TrimPrefix(name, "node")
		nodeID, err := strconv.Atoi(nodeIDStr)
		if err != nil {
			continue
		}

		// Read CPUs for this node
		cpuListPath := fmt.Sprintf("/sys/devices/system/node/%s/cpulist", name)
		cpuListData, err := os.ReadFile(cpuListPath)
		if err != nil {
			continue
		}

		cpuList := strings.TrimSpace(string(cpuListData))
		cpus, err := parseCPUList(cpuList)
		if err != nil {
			continue
		}

		topology.CPUsPerNode[nodeID] = cpus
		topology.NodeCount++
		topology.TotalCPUs += len(cpus)
	}

	if topology.NodeCount == 0 {
		return nil, fmt.Errorf("no NUMA nodes found")
	}

	return topology, nil
}

// detectDarwinTopology detects NUMA topology on macOS
func detectDarwinTopology() (*Topology, error) {
	// macOS doesn't have traditional NUMA, but we can use CPU count
	cpuCount := runtime.NumCPU()

	topology := &Topology{
		NodeCount:   1,
		CPUsPerNode: make(map[int][]int),
		TotalCPUs:   cpuCount,
	}

	// All CPUs in single node
	cpus := make([]int, cpuCount)
	for i := 0; i < cpuCount; i++ {
		cpus[i] = i
	}
	topology.CPUsPerNode[0] = cpus

	return topology, nil
}

// detectWindowsTopology detects NUMA topology on Windows
func detectWindowsTopology() (*Topology, error) {
	// Simplified Windows support - treat as single node
	cpuCount := runtime.NumCPU()

	topology := &Topology{
		NodeCount:   1,
		CPUsPerNode: make(map[int][]int),
		TotalCPUs:   cpuCount,
	}

	// All CPUs in single node
	cpus := make([]int, cpuCount)
	for i := 0; i < cpuCount; i++ {
		cpus[i] = i
	}
	topology.CPUsPerNode[0] = cpus

	return topology, nil
}

// parseCPUList parses a Linux CPU list (e.g., "0-3,8-11")
func parseCPUList(cpuList string) ([]int, error) {
	var cpus []int

	parts := strings.Split(cpuList, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Range
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid CPU range: %s", part)
			}

			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid CPU range start: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid CPU range end: %s", rangeParts[1])
			}

			for i := start; i <= end; i++ {
				cpus = append(cpus, i)
			}
		} else {
			// Single CPU
			cpu, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid CPU ID: %s", part)
			}
			cpus = append(cpus, cpu)
		}
	}

	return cpus, nil
}

// GetNodeCPUs returns the CPU IDs for a given NUMA node
func (t *Topology) GetNodeCPUs(nodeID int) ([]int, error) {
	cpus, ok := t.CPUsPerNode[nodeID]
	if !ok {
		return nil, fmt.Errorf("NUMA node %d not found", nodeID)
	}
	return cpus, nil
}

// GetCPUNode returns the NUMA node ID for a given CPU
func (t *Topology) GetCPUNode(cpuID int) (int, error) {
	for nodeID, cpus := range t.CPUsPerNode {
		for _, cpu := range cpus {
			if cpu == cpuID {
				return nodeID, nil
			}
		}
	}
	return -1, fmt.Errorf("CPU %d not found in any NUMA node", cpuID)
}
