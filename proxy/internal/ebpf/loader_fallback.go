// +build !cgo

package ebpf

import (
	"fmt"
)

// BPFLoader manages eBPF program loading (fallback implementation without CGO)
type BPFLoader struct {
	programPath string
	loaded      bool
}

// NewBPFLoader creates a new eBPF program loader (fallback)
func NewBPFLoader(programPath string) *BPFLoader {
	return &BPFLoader{
		programPath: programPath,
		loaded:      false,
	}
}

// LoadProgram loads the eBPF program (fallback - mock implementation)
func (l *BPFLoader) LoadProgram() error {
	fmt.Printf("eBPF: Mock loading program from %s (CGO not available)\n", l.programPath)
	l.loaded = true
	return nil
}

// UnloadProgram unloads the eBPF program (fallback)
func (l *BPFLoader) UnloadProgram() error {
	fmt.Printf("eBPF: Mock unloading program (CGO not available)\n")
	l.loaded = false
	return nil
}

// UpdateServiceRule updates a service rule (fallback - no-op)
func (l *BPFLoader) UpdateServiceRule(ruleID uint32, rule *ServiceRule) error {
	if !l.loaded {
		return fmt.Errorf("eBPF program not loaded")
	}
	// Mock implementation - just log the operation
	fmt.Printf("eBPF: Mock updating rule %d (CGO not available)\n", ruleID)
	return nil
}

// DeleteServiceRule deletes a service rule (fallback - no-op)
func (l *BPFLoader) DeleteServiceRule(ruleID uint32) error {
	if !l.loaded {
		return fmt.Errorf("eBPF program not loaded")
	}
	fmt.Printf("eBPF: Mock deleting rule %d (CGO not available)\n", ruleID)
	return nil
}

// GetStatistics retrieves statistics (fallback - mock data)
func (l *BPFLoader) GetStatistics() (*EBPFStatistics, error) {
	if !l.loaded {
		return nil, fmt.Errorf("eBPF program not loaded")
	}

	// Return mock statistics
	return &EBPFStatistics{
		TotalPackets:     1000,
		TCPPackets:       800,
		UDPPackets:       150,
		DroppedPackets:   50,
		AllowedPackets:   700,
		UserspacePackets: 250,
	}, nil
}

// IsLoaded returns true if the eBPF program is loaded (fallback)
func (l *BPFLoader) IsLoaded() bool {
	return l.loaded
}

// GetProgramFD returns the program file descriptor (fallback - always -1)
func (l *BPFLoader) GetProgramFD() int {
	return -1
}

// ServiceRule represents a service rule for eBPF map (fallback)
type ServiceRule struct {
	ServiceID uint32
	IPAddr    uint32
	Port      uint16
	Protocol  uint8
	Action    uint8
}

// EBPFStatistics represents statistics from eBPF program (fallback)
type EBPFStatistics struct {
	TotalPackets     uint64
	TCPPackets       uint64
	UDPPackets       uint64
	DroppedPackets   uint64
	AllowedPackets   uint64
	UserspacePackets uint64
}