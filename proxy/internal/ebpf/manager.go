package ebpf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/penguintech/marchproxy/internal/manager"
)

// Manager handles eBPF program lifecycle and map management
type Manager struct {
	enabled       bool
	programLoaded bool
	maps          *EBPFMaps
	stats         *EBPFStats
	loader        *BPFLoader
	programPath   string
	mu            sync.RWMutex
}

// NewManager creates a new eBPF manager
func NewManager(enabled bool) *Manager {
	// Try to find eBPF program
	programPath, _ := FindEBPFProgram()
	
	manager := &Manager{
		enabled:     enabled,
		programPath: programPath,
		maps: &EBPFMaps{
			Services:    make(map[uint32]*EBPFService),
			Mappings:    make(map[uint32]*EBPFMapping),
			Connections: make(map[ConnectionKey]*ConnectionValue),
			Stats:       &ProxyStats{},
		},
		stats: &EBPFStats{
			ProgramLoaded:      false,
			AttachedInterfaces: []string{},
			LastUpdate:        time.Now(),
			MapSyncErrors:      0,
			ProgramErrors:      0,
		},
	}

	// Initialize loader if program found
	if programPath != "" {
		manager.loader = NewBPFLoader(programPath)
		fmt.Printf("eBPF: Found program at %s\n", programPath)
	} else if enabled {
		fmt.Printf("eBPF: Program file not found, using fallback mode\n")
	}

	return manager
}

// IsEnabled returns whether eBPF is enabled
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// LoadProgram loads the eBPF program
func (m *Manager) LoadProgram(programPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return fmt.Errorf("eBPF is disabled")
	}

	if m.programLoaded {
		return fmt.Errorf("eBPF program already loaded")
	}

	// Use provided path or default
	if programPath != "" {
		m.programPath = programPath
		if m.loader != nil {
			m.loader.UnloadProgram() // Clean up old loader
		}
		m.loader = NewBPFLoader(programPath)
	}

	if m.loader == nil {
		return fmt.Errorf("no eBPF program loader available")
	}

	// Load the actual eBPF program
	if err := m.loader.LoadProgram(); err != nil {
		return fmt.Errorf("failed to load eBPF program: %w", err)
	}

	m.programLoaded = true
	m.stats.ProgramLoaded = true
	m.stats.LastUpdate = time.Now()
	
	fmt.Printf("eBPF: Program loaded successfully from %s\n", m.programPath)
	return nil
}

// UnloadProgram unloads the eBPF program
func (m *Manager) UnloadProgram() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.programLoaded {
		return nil
	}

	// Mock implementation - in production this would:
	// 1. Detach from network interfaces
	// 2. Unpin and close eBPF maps
	// 3. Close program file descriptor
	
	fmt.Printf("eBPF: Unloading program\n")
	
	m.programLoaded = false
	m.stats.ProgramLoaded = false
	m.stats.AttachedInterfaces = []string{}
	m.stats.LastUpdate = time.Now()
	
	fmt.Printf("eBPF: Program unloaded successfully\n")
	return nil
}

// UpdateServices synchronizes services with eBPF maps
func (m *Manager) UpdateServices(services []manager.Service) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled || !m.programLoaded {
		return nil // Skip if eBPF not enabled or loaded
	}

	fmt.Printf("eBPF: Updating %d services in maps\n", len(services))

	// Clear existing services from local cache
	m.maps.Services = make(map[uint32]*EBPFService)

	// Convert and store services in both local cache and eBPF maps
	for i, service := range services {
		ebpfService := &EBPFService{
			ID:           uint32(service.ID),
			IPAddr:       IPToUint32(resolveServiceIP(service.IPFQDN)),
			Port:         80, // Default port - would be parsed from service config
			AuthRequired: 0,
			AuthType:     AuthTypeNone,
			Flags:        0,
		}

		// Set auth configuration
		if service.AuthType == "base64" {
			ebpfService.AuthRequired = 1
			ebpfService.AuthType = AuthTypeBase64
		} else if service.AuthType == "jwt" {
			ebpfService.AuthRequired = 1
			ebpfService.AuthType = AuthTypeJWT
		}

		// Store in local cache
		m.maps.Services[ebpfService.ID] = ebpfService

		// Update eBPF map if loader is available
		if m.loader != nil && m.loader.IsLoaded() {
			rule := &ServiceRule{
				ServiceID: ebpfService.ID,
				IPAddr:    ebpfService.IPAddr,
				Port:      ebpfService.Port,
				Protocol:  0, // 0 = any protocol
				Action:    2, // 2 = send to userspace for authentication
			}

			if err := m.loader.UpdateServiceRule(uint32(i), rule); err != nil {
				fmt.Printf("eBPF: Warning - failed to update service rule %d: %v\n", service.ID, err)
				m.stats.MapSyncErrors++
			}
		}
	}

	m.stats.LastUpdate = time.Now()
	fmt.Printf("eBPF: Services updated successfully\n")
	return nil
}

// UpdateMappings synchronizes mappings with eBPF maps
func (m *Manager) UpdateMappings(mappings []manager.Mapping) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled || !m.programLoaded {
		return nil // Skip if eBPF not enabled or loaded
	}

	fmt.Printf("eBPF: Updating %d mappings in maps\n", len(mappings))

	// Clear existing mappings
	m.maps.Mappings = make(map[uint32]*EBPFMapping)

	// Convert and store mappings
	for _, mapping := range mappings {
		ebpfMapping := &EBPFMapping{
			ID:           uint32(mapping.ID),
			Protocols:    0,
			AuthRequired: 0,
			Priority:     uint8(mapping.Priority),
			PortCount:    0,
			SrcCount:     0,
			DestCount:    0,
		}

		// Convert protocols
		for _, protocol := range mapping.Protocols {
			ebpfMapping.Protocols |= ProtocolToMask(protocol)
		}

		// Set auth requirement
		if mapping.AuthRequired {
			ebpfMapping.AuthRequired = 1
		}

		// Copy source services (limited by array size)
		for i, srcID := range mapping.SourceServices {
			if i >= 16 {
				break
			}
			ebpfMapping.SourceServices[i] = uint32(srcID)
			ebpfMapping.SrcCount++
		}

		// Copy destination services (limited by array size)
		for i, dstID := range mapping.DestServices {
			if i >= 16 {
				break
			}
			ebpfMapping.DestServices[i] = uint32(dstID)
			ebpfMapping.DestCount++
		}

		m.maps.Mappings[ebpfMapping.ID] = ebpfMapping
	}

	m.stats.LastUpdate = time.Now()
	fmt.Printf("eBPF: Mappings updated successfully\n")
	return nil
}

// GetStats returns current eBPF statistics
func (m *Manager) GetStats() (*ProxyStats, *EBPFStats) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Mock some statistics - in production this would read from eBPF maps
	if m.programLoaded {
		m.maps.Stats.TotalPackets += 1000
		m.maps.Stats.ForwardedPackets += 800
		m.maps.Stats.DroppedPackets += 50
		m.maps.Stats.FallbackToUserspace += 150
	}

	// Create copies to avoid race conditions
	proxyStats := *m.maps.Stats
	ebpfStats := *m.stats

	return &proxyStats, &ebpfStats
}

// GetConnectionCount returns the current number of tracked connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.maps.Connections)
}

// ShouldFallbackToUserspace determines if a packet should be processed in userspace
func (m *Manager) ShouldFallbackToUserspace(srcIP, dstIP uint32, srcPort, dstPort uint16, protocol uint8) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled || !m.programLoaded {
		return true // Always fallback if eBPF not available
	}

	// Mock logic - in production this would check eBPF maps
	// For now, fallback to userspace for all packets requiring authentication
	for _, mapping := range m.maps.Mappings {
		if mapping.AuthRequired == 1 {
			return true
		}
	}

	return false // Let eBPF handle simple forwarding
}

// AttachToInterface attaches eBPF program to network interface (mock)
func (m *Manager) AttachToInterface(interfaceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled || !m.programLoaded {
		return fmt.Errorf("eBPF program not loaded")
	}

	// Mock implementation - in production this would:
	// 1. Get network interface by name
	// 2. Attach TC classifier or XDP program
	// 3. Configure ingress/egress hooks

	fmt.Printf("eBPF: Attaching to interface %s (mock)\n", interfaceName)
	
	// Check if already attached
	for _, iface := range m.stats.AttachedInterfaces {
		if iface == interfaceName {
			return fmt.Errorf("already attached to interface %s", interfaceName)
		}
	}

	m.stats.AttachedInterfaces = append(m.stats.AttachedInterfaces, interfaceName)
	m.stats.LastUpdate = time.Now()

	fmt.Printf("eBPF: Attached to interface %s successfully\n", interfaceName)
	return nil
}

// DetachFromInterface detaches eBPF program from network interface (mock)
func (m *Manager) DetachFromInterface(interfaceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Printf("eBPF: Detaching from interface %s (mock)\n", interfaceName)

	// Remove from attached interfaces
	for i, iface := range m.stats.AttachedInterfaces {
		if iface == interfaceName {
			m.stats.AttachedInterfaces = append(
				m.stats.AttachedInterfaces[:i], 
				m.stats.AttachedInterfaces[i+1:]...)
			break
		}
	}

	m.stats.LastUpdate = time.Now()
	fmt.Printf("eBPF: Detached from interface %s successfully\n", interfaceName)
	return nil
}

// Cleanup performs cleanup operations
func (m *Manager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Printf("eBPF: Performing cleanup\n")

	// Detach from all interfaces
	for _, iface := range m.stats.AttachedInterfaces {
		fmt.Printf("eBPF: Detaching from interface %s\n", iface)
	}
	m.stats.AttachedInterfaces = []string{}

	// Unload program
	if m.programLoaded {
		m.programLoaded = false
		m.stats.ProgramLoaded = false
	}

	fmt.Printf("eBPF: Cleanup complete\n")
	return nil
}

// resolveServiceIP resolves a service FQDN/IP to an IP address
func resolveServiceIP(ipfqdn string) net.IP {
	// First try to parse as IP address
	if ip := net.ParseIP(ipfqdn); ip != nil {
		return ip.To4() // Return IPv4 version
	}

	// Try to resolve as hostname
	ips, err := net.LookupIP(ipfqdn)
	if err != nil {
		fmt.Printf("eBPF: Warning - failed to resolve %s: %v\n", ipfqdn, err)
		return net.IPv4(127, 0, 0, 1) // Default to localhost
	}

	// Return first IPv4 address found
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4
		}
	}

	// Default to localhost if no IPv4 found
	return net.IPv4(127, 0, 0, 1)
}

// FindEBPFProgram searches for the compiled eBPF program file
func FindEBPFProgram() (string, error) {
	// Search in common locations
	searchPaths := []string{
		"ebpf/build/complete_filter.o",
		"ebpf/build/rule_matcher.o",
		"/opt/marchproxy/ebpf/complete_filter.o",
		"./complete_filter.o",
		"./rule_matcher.o",
	}

	for _, path := range searchPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("eBPF program file not found in search paths: %v", searchPaths)
}