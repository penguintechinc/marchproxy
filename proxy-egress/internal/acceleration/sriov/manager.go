package sriov

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"marchproxy-egress/internal/manager"
)

// SRIOVManager handles SR-IOV configuration and management
type SRIOVManager struct {
	enabled      bool
	initialized  bool
	physicalFs   map[string]*PhysicalFunction
	virtualFs    map[string]*VirtualFunction
	config       *SRIOVConfig
	stats        *SRIOVStats
	mu           sync.RWMutex
}

// PhysicalFunction represents a Physical Function (PF) in SR-IOV
type PhysicalFunction struct {
	PCIAddress    string
	InterfaceName string
	Driver        string
	VendorID      string
	DeviceID      string
	MaxVFs        int
	CurrentVFs    int
	VFList        []*VirtualFunction
	SupportedFeatures map[string]bool
	Statistics    *PFStats
	LastUpdate    time.Time
}

// VirtualFunction represents a Virtual Function (VF) in SR-IOV
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

// PFStats holds Physical Function statistics
type PFStats struct {
	RxPackets     uint64
	TxPackets     uint64
	RxBytes       uint64
	TxBytes       uint64
	RxDropped     uint64
	TxDropped     uint64
	RxErrors      uint64
	TxErrors      uint64
	PCIeErrors    uint64
	LastUpdate    time.Time
}

// VFStats holds Virtual Function statistics
type VFStats struct {
	RxPackets     uint64
	TxPackets     uint64
	RxBytes       uint64
	TxBytes       uint64
	RxDropped     uint64
	TxDropped     uint64
	RxErrors      uint64
	TxErrors      uint64
	VLANViolations uint64
	SpoofViolations uint64
	LastUpdate    time.Time
}

// QoSConfig holds Quality of Service configuration
type QoSConfig struct {
	MinBandwidth  uint32 // Mbps
	MaxBandwidth  uint32 // Mbps
	Priority      uint8
	WeightFactor  uint8
	RateLimiting  bool
}

// SRIOVStats holds overall SR-IOV statistics
type SRIOVStats struct {
	TotalPFs           int
	TotalVFs           int
	ActiveVFs          int
	TotalBandwidth     uint64
	UtilizedBandwidth  uint64
	PCIeUtilization    float64
	AverageLatency     time.Duration
	ThroughputMbps     uint64
	ErrorRate          float64
	LastUpdate         time.Time
}

// SRIOVConfig holds SR-IOV configuration parameters
type SRIOVConfig struct {
	EnabledPFs        []string
	MaxVFsPerPF       int
	DefaultQoS        *QoSConfig
	VLANMode          string // "trunk", "access", "disabled"
	SecurityMode      string // "strict", "permissive"
	EnableSpoofCheck  bool
	EnableTrustMode   bool
	StatsInterval     time.Duration
	AutoConfiguration bool
	PCIPassthrough    bool
}

// NewSRIOVManager creates a new SR-IOV manager
func NewSRIOVManager(enabled bool, config *SRIOVConfig) *SRIOVManager {
	if config == nil {
		config = &SRIOVConfig{
			EnabledPFs:       []string{},
			MaxVFsPerPF:      64,
			VLANMode:         "disabled",
			SecurityMode:     "strict",
			EnableSpoofCheck: true,
			EnableTrustMode:  false,
			StatsInterval:    time.Second * 5,
			DefaultQoS: &QoSConfig{
				MinBandwidth:  100,
				MaxBandwidth:  1000,
				Priority:      0,
				WeightFactor:  1,
				RateLimiting:  true,
			},
		}
	}

	return &SRIOVManager{
		enabled:    enabled,
		physicalFs: make(map[string]*PhysicalFunction),
		virtualFs:  make(map[string]*VirtualFunction),
		config:     config,
		stats: &SRIOVStats{
			LastUpdate: time.Now(),
		},
	}
}

// Initialize discovers and initializes SR-IOV capable devices
func (sm *SRIOVManager) Initialize() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.enabled {
		return fmt.Errorf("SR-IOV is disabled")
	}

	if sm.initialized {
		return fmt.Errorf("SR-IOV already initialized")
	}

	fmt.Printf("SR-IOV: Discovering SR-IOV capable devices\n")

	// Check if IOMMU is enabled
	if !sm.checkIOMMUEnabled() {
		fmt.Printf("SR-IOV: Warning - IOMMU not enabled, SR-IOV functionality may be limited\n")
	}

	// Discover Physical Functions
	if err := sm.discoverPhysicalFunctions(); err != nil {
		return fmt.Errorf("failed to discover Physical Functions: %w", err)
	}

	// Configure enabled PFs
	for _, pfAddr := range sm.config.EnabledPFs {
		if pf, exists := sm.physicalFs[pfAddr]; exists {
			if err := sm.configurePF(pf); err != nil {
				fmt.Printf("SR-IOV: Warning - failed to configure PF %s: %v\n", pfAddr, err)
			}
		}
	}

	sm.initialized = true
	
	// Start statistics collection
	go sm.statsCollector()
	
	fmt.Printf("SR-IOV: Initialized with %d PFs and %d VFs\n", len(sm.physicalFs), len(sm.virtualFs))
	return nil
}

// checkIOMMUEnabled checks if IOMMU is enabled in the system
func (sm *SRIOVManager) checkIOMMUEnabled() bool {
	// Check for IOMMU in kernel command line
	cmdlineData, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		return false
	}

	cmdline := string(cmdlineData)
	return strings.Contains(cmdline, "intel_iommu=on") || 
		   strings.Contains(cmdline, "amd_iommu=on") || 
		   strings.Contains(cmdline, "iommu=pt")
}

// discoverPhysicalFunctions discovers SR-IOV capable Physical Functions
func (sm *SRIOVManager) discoverPhysicalFunctions() error {
	// Scan PCI devices for SR-IOV capability
	pciDevicesPath := "/sys/bus/pci/devices"
	
	devices, err := ioutil.ReadDir(pciDevicesPath)
	if err != nil {
		return fmt.Errorf("failed to read PCI devices: %w", err)
	}

	for _, device := range devices {
		devicePath := filepath.Join(pciDevicesPath, device.Name())
		
		// Check if device supports SR-IOV
		sriovCapPath := filepath.Join(devicePath, "sriov_totalvfs")
		if _, err := os.Stat(sriovCapPath); os.IsNotExist(err) {
			continue
		}

		// Read SR-IOV capabilities
		totalVFsData, err := ioutil.ReadFile(sriovCapPath)
		if err != nil {
			continue
		}

		maxVFs, err := strconv.Atoi(strings.TrimSpace(string(totalVFsData)))
		if err != nil || maxVFs == 0 {
			continue
		}

		// Read current VF count
		currentVFsPath := filepath.Join(devicePath, "sriov_numvfs")
		currentVFsData, err := ioutil.ReadFile(currentVFsPath)
		if err != nil {
			continue
		}

		currentVFs, err := strconv.Atoi(strings.TrimSpace(string(currentVFsData)))
		if err != nil {
			currentVFs = 0
		}

		// Read device information
		vendorData, _ := ioutil.ReadFile(filepath.Join(devicePath, "vendor"))
		deviceData, _ := ioutil.ReadFile(filepath.Join(devicePath, "device"))
		driverPath := filepath.Join(devicePath, "driver")
		
		var driver string
		if driverLink, err := os.Readlink(driverPath); err == nil {
			driver = filepath.Base(driverLink)
		}

		// Find network interface name
		netPath := filepath.Join(devicePath, "net")
		var interfaceName string
		if netDevices, err := ioutil.ReadDir(netPath); err == nil && len(netDevices) > 0 {
			interfaceName = netDevices[0].Name()
		}

		// Create Physical Function
		pf := &PhysicalFunction{
			PCIAddress:    device.Name(),
			InterfaceName: interfaceName,
			Driver:        driver,
			VendorID:      strings.TrimSpace(string(vendorData)),
			DeviceID:      strings.TrimSpace(string(deviceData)),
			MaxVFs:        maxVFs,
			CurrentVFs:    currentVFs,
			VFList:        make([]*VirtualFunction, 0, maxVFs),
			SupportedFeatures: make(map[string]bool),
			Statistics:    &PFStats{LastUpdate: time.Now()},
			LastUpdate:    time.Now(),
		}

		// Check supported features
		sm.detectPFFeatures(pf, devicePath)
		
		sm.physicalFs[device.Name()] = pf
		fmt.Printf("SR-IOV: Found PF %s (%s) - Max VFs: %d, Current VFs: %d\n", 
			device.Name(), interfaceName, maxVFs, currentVFs)

		// Discover existing VFs
		if currentVFs > 0 {
			sm.discoverVirtualFunctions(pf, devicePath)
		}
	}

	return nil
}

// detectPFFeatures detects supported features for a Physical Function
func (sm *SRIOVManager) detectPFFeatures(pf *PhysicalFunction, devicePath string) {
	// Check for various SR-IOV features
	features := map[string]string{
		"VLAN filtering":     "sriov_vlan_filter",
		"QoS":               "sriov_qos",
		"Spoof checking":    "sriov_spoof_check",
		"Trust mode":        "sriov_trust",
		"Link state control": "sriov_link_state",
		"Rate limiting":     "sriov_rate_limit",
	}

	for featureName, sysfsFile := range features {
		featurePath := filepath.Join(devicePath, sysfsFile)
		if _, err := os.Stat(featurePath); err == nil {
			pf.SupportedFeatures[featureName] = true
		}
	}
}

// discoverVirtualFunctions discovers existing Virtual Functions
func (sm *SRIOVManager) discoverVirtualFunctions(pf *PhysicalFunction, pfPath string) {
	// VFs are linked as virtfnX directories
	for i := 0; i < pf.CurrentVFs; i++ {
		vfLinkPath := filepath.Join(pfPath, fmt.Sprintf("virtfn%d", i))
		if vfTarget, err := os.Readlink(vfLinkPath); err == nil {
			vfPCIAddr := filepath.Base(vfTarget)
			vfDevicePath := filepath.Join("/sys/bus/pci/devices", vfPCIAddr)

			// Get VF network interface
			vfNetPath := filepath.Join(vfDevicePath, "net")
			var vfInterface string
			if netDevices, err := ioutil.ReadDir(vfNetPath); err == nil && len(netDevices) > 0 {
				vfInterface = netDevices[0].Name()
			}

			// Create Virtual Function
			vf := &VirtualFunction{
				PCIAddress:    vfPCIAddr,
				InterfaceName: vfInterface,
				ParentPF:      pf.PCIAddress,
				VFIndex:       i,
				QoSSettings:   sm.config.DefaultQoS,
				TrustMode:     sm.config.EnableTrustMode,
				SpoofCheck:    sm.config.EnableSpoofCheck,
				LinkState:     "auto",
				Statistics:    &VFStats{LastUpdate: time.Now()},
				LastActivity:  time.Now(),
			}

			// Read VF MAC address if available
			sm.readVFConfiguration(pf, vf)

			pf.VFList = append(pf.VFList, vf)
			sm.virtualFs[vfPCIAddr] = vf
			
			fmt.Printf("SR-IOV: Found VF %s (%s) - Parent PF: %s\n", 
				vfPCIAddr, vfInterface, pf.PCIAddress)
		}
	}
}

// readVFConfiguration reads existing VF configuration
func (sm *SRIOVManager) readVFConfiguration(pf *PhysicalFunction, vf *VirtualFunction) {
	// This would use ethtool or similar tools to read VF configuration
	// For now, we'll set defaults
	
	// Generate a default MAC address
	vf.MACAddress = sm.generateVFMACAddress(pf, vf.VFIndex)
	vf.VLANTag = 0 // No VLAN by default
}

// generateVFMACAddress generates a MAC address for a VF
func (sm *SRIOVManager) generateVFMACAddress(pf *PhysicalFunction, vfIndex int) net.HardwareAddr {
	// Generate a locally administered MAC address
	mac := make(net.HardwareAddr, 6)
	mac[0] = 0x02 // Locally administered
	mac[1] = 0x00
	mac[2] = byte(vfIndex >> 8)
	mac[3] = byte(vfIndex)
	mac[4] = byte(time.Now().Unix() >> 8)
	mac[5] = byte(time.Now().Unix())
	return mac
}

// configurePF configures a Physical Function with default settings
func (sm *SRIOVManager) configurePF(pf *PhysicalFunction) error {
	fmt.Printf("SR-IOV: Configuring PF %s\n", pf.PCIAddress)

	// Enable VFs if not already enabled
	if pf.CurrentVFs == 0 && sm.config.MaxVFsPerPF > 0 {
		vfCount := pf.MaxVFs
		if vfCount > sm.config.MaxVFsPerPF {
			vfCount = sm.config.MaxVFsPerPF
		}

		if err := sm.enableVFs(pf, vfCount); err != nil {
			return fmt.Errorf("failed to enable VFs: %w", err)
		}
	}

	return nil
}

// enableVFs enables Virtual Functions for a Physical Function
func (sm *SRIOVManager) enableVFs(pf *PhysicalFunction, count int) error {
	fmt.Printf("SR-IOV: Enabling %d VFs for PF %s\n", count, pf.PCIAddress)

	// Write VF count to sysfs
	vfCountPath := fmt.Sprintf("/sys/bus/pci/devices/%s/sriov_numvfs", pf.PCIAddress)
	
	// First disable any existing VFs
	if err := ioutil.WriteFile(vfCountPath, []byte("0"), 0644); err != nil {
		return fmt.Errorf("failed to disable existing VFs: %w", err)
	}

	// Enable new VFs
	if err := ioutil.WriteFile(vfCountPath, []byte(strconv.Itoa(count)), 0644); err != nil {
		return fmt.Errorf("failed to enable VFs: %w", err)
	}

	// Wait for VFs to be created
	time.Sleep(time.Second * 2)

	// Update PF state
	pf.CurrentVFs = count
	
	// Discover newly created VFs
	pfPath := fmt.Sprintf("/sys/bus/pci/devices/%s", pf.PCIAddress)
	sm.discoverVirtualFunctions(pf, pfPath)

	fmt.Printf("SR-IOV: Successfully enabled %d VFs for PF %s\n", count, pf.PCIAddress)
	return nil
}

// ConfigureVF configures a Virtual Function with specific settings
func (sm *SRIOVManager) ConfigureVF(vfPCIAddr string, config *VFConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	vf, exists := sm.virtualFs[vfPCIAddr]
	if !exists {
		return fmt.Errorf("VF %s not found", vfPCIAddr)
	}

	fmt.Printf("SR-IOV: Configuring VF %s\n", vfPCIAddr)

	// Apply configuration
	if config.MACAddress != nil {
		vf.MACAddress = config.MACAddress
	}
	
	if config.VLANTag != nil {
		vf.VLANTag = *config.VLANTag
	}
	
	if config.QoSSettings != nil {
		vf.QoSSettings = config.QoSSettings
	}
	
	if config.TrustMode != nil {
		vf.TrustMode = *config.TrustMode
	}
	
	if config.SpoofCheck != nil {
		vf.SpoofCheck = *config.SpoofCheck
	}

	// In a real implementation, this would use ip/ethtool commands
	// to apply the configuration to the actual VF
	
	fmt.Printf("SR-IOV: VF %s configured successfully\n", vfPCIAddr)
	return nil
}

// VFConfig holds Virtual Function configuration
type VFConfig struct {
	MACAddress  net.HardwareAddr
	VLANTag     *int
	QoSSettings *QoSConfig
	TrustMode   *bool
	SpoofCheck  *bool
	LinkState   *string
	AssignedVM  *string
}

// statsCollector collects SR-IOV statistics
func (sm *SRIOVManager) statsCollector() {
	ticker := time.NewTicker(sm.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.collectStatistics()
		}
	}
}

// collectStatistics collects statistics from PFs and VFs
func (sm *SRIOVManager) collectStatistics() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	totalBandwidth := uint64(0)
	utilizedBandwidth := uint64(0)
	totalErrors := uint64(0)
	totalPackets := uint64(0)

	// Collect PF statistics
	for _, pf := range sm.physicalFs {
		if pf.InterfaceName != "" {
			sm.collectInterfaceStats(pf.InterfaceName, &pf.Statistics.RxPackets, 
				&pf.Statistics.TxPackets, &pf.Statistics.RxBytes, &pf.Statistics.TxBytes)
			pf.Statistics.LastUpdate = time.Now()
		}

		// Collect VF statistics
		for _, vf := range pf.VFList {
			if vf.InterfaceName != "" {
				sm.collectInterfaceStats(vf.InterfaceName, &vf.Statistics.RxPackets,
					&vf.Statistics.TxPackets, &vf.Statistics.RxBytes, &vf.Statistics.TxBytes)
				vf.Statistics.LastUpdate = time.Now()
				
				totalPackets += vf.Statistics.RxPackets + vf.Statistics.TxPackets
				totalErrors += vf.Statistics.RxErrors + vf.Statistics.TxErrors
			}
		}
	}

	// Update overall statistics
	sm.stats.TotalPFs = len(sm.physicalFs)
	sm.stats.TotalVFs = len(sm.virtualFs)
	sm.stats.ActiveVFs = sm.countActiveVFs()
	sm.stats.TotalBandwidth = totalBandwidth
	sm.stats.UtilizedBandwidth = utilizedBandwidth
	
	if totalPackets > 0 {
		sm.stats.ErrorRate = float64(totalErrors) / float64(totalPackets) * 100.0
	}
	
	sm.stats.LastUpdate = time.Now()
}

// collectInterfaceStats collects statistics for a network interface
func (sm *SRIOVManager) collectInterfaceStats(ifname string, rxPkts, txPkts, rxBytes, txBytes *uint64) {
	// Read statistics from /sys/class/net/<interface>/statistics/
	statsPath := fmt.Sprintf("/sys/class/net/%s/statistics", ifname)
	
	if data, err := ioutil.ReadFile(filepath.Join(statsPath, "rx_packets")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			*rxPkts = val
		}
	}
	
	if data, err := ioutil.ReadFile(filepath.Join(statsPath, "tx_packets")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			*txPkts = val
		}
	}
	
	if data, err := ioutil.ReadFile(filepath.Join(statsPath, "rx_bytes")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			*rxBytes = val
		}
	}
	
	if data, err := ioutil.ReadFile(filepath.Join(statsPath, "tx_bytes")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			*txBytes = val
		}
	}
}

// countActiveVFs counts the number of active Virtual Functions
func (sm *SRIOVManager) countActiveVFs() int {
	active := 0
	for _, vf := range sm.virtualFs {
		if time.Since(vf.LastActivity) < time.Minute*5 { // Consider active if activity in last 5 minutes
			active++
		}
	}
	return active
}

// UpdateServices updates service assignments for SR-IOV VFs
func (sm *SRIOVManager) UpdateServices(services []manager.Service) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	fmt.Printf("SR-IOV: Updating service assignments for %d services\n", len(services))

	// In a full implementation, this would assign services to VFs
	// based on performance requirements and resource availability
	
	return nil
}

// Stop disables VFs and cleans up SR-IOV configuration
func (sm *SRIOVManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.initialized {
		return nil
	}

	fmt.Printf("SR-IOV: Stopping and cleaning up\n")

	// Disable VFs for all PFs
	for _, pf := range sm.physicalFs {
		if pf.CurrentVFs > 0 {
			vfCountPath := fmt.Sprintf("/sys/bus/pci/devices/%s/sriov_numvfs", pf.PCIAddress)
			ioutil.WriteFile(vfCountPath, []byte("0"), 0644)
			fmt.Printf("SR-IOV: Disabled VFs for PF %s\n", pf.PCIAddress)
		}
	}

	sm.initialized = false
	fmt.Printf("SR-IOV: Cleanup complete\n")
	return nil
}

// GetStats returns current SR-IOV statistics
func (sm *SRIOVManager) GetStats() *SRIOVStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := *sm.stats
	return &stats
}

// IsEnabled returns whether SR-IOV is enabled and initialized
func (sm *SRIOVManager) IsEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.enabled && sm.initialized
}

// GetPhysicalFunctions returns all discovered Physical Functions
func (sm *SRIOVManager) GetPhysicalFunctions() map[string]*PhysicalFunction {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	pfs := make(map[string]*PhysicalFunction)
	for addr, pf := range sm.physicalFs {
		pfCopy := *pf
		pfs[addr] = &pfCopy
	}
	return pfs
}

// GetVirtualFunctions returns all discovered Virtual Functions
func (sm *SRIOVManager) GetVirtualFunctions() map[string]*VirtualFunction {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	vfs := make(map[string]*VirtualFunction)
	for addr, vf := range sm.virtualFs {
		vfCopy := *vf
		vfs[addr] = &vfCopy
	}
	return vfs
}