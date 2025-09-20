// +build xdp

package ratelimit

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
	"unsafe"

	"github.com/MarchProxy/proxy/internal/logging"
	"github.com/MarchProxy/proxy/internal/metrics"
)

// #cgo CFLAGS: -I/usr/include/bpf -I../../../ebpf/include
// #cgo LDFLAGS: -lbpf -lelf -lz
// #include <stdio.h>
// #include <stdlib.h>
// #include <bpf/bpf.h>
// #include <bpf/libbpf.h>
//
// struct bpf_object* load_bpf_program(const char *filename);
// int attach_xdp_to_interface(const char *ifname, int prog_fd, unsigned int flags);
// int detach_xdp_from_interface(const char *ifname);
// int get_map_fd_by_name(struct bpf_object *obj, const char *name);
// void close_bpf_object(struct bpf_object *obj);
import "C"

// RateLimitConfig represents the XDP rate limiting configuration
type RateLimitConfig struct {
	Enabled          bool   `json:"enabled"`
	GlobalPPSLimit   uint32 `json:"global_pps_limit"`
	PerIPPPSLimit    uint32 `json:"per_ip_pps_limit"`
	WindowSizeNS     uint32 `json:"window_size_ns"`
	BurstAllowance   uint32 `json:"burst_allowance"`
	Action           uint32 `json:"action"` // 0=PASS, 1=DROP, 2=RATE_LIMIT
}

// IPRateState represents per-IP rate limiting state
type IPRateState struct {
	LastUpdateNS   uint64 `json:"last_update_ns"`
	PacketCount    uint32 `json:"packet_count"`
	TotalPackets   uint32 `json:"total_packets"`
	DroppedPackets uint32 `json:"dropped_packets"`
	BurstTokens    uint32 `json:"burst_tokens"`
}

// GlobalRateState represents global rate limiting state
type GlobalRateState struct {
	LastUpdateNS   uint64 `json:"last_update_ns"`
	PacketCount    uint32 `json:"packet_count"`
	TotalPackets   uint32 `json:"total_packets"`
	DroppedPackets uint32 `json:"dropped_packets"`
}

// RateLimitStats represents rate limiting statistics
type RateLimitStats struct {
	TotalPackets     uint64 `json:"total_packets"`
	PassedPackets    uint64 `json:"passed_packets"`
	DroppedPackets   uint64 `json:"dropped_packets"`
	RateLimitedIPs   uint64 `json:"rate_limited_ips"`
	GlobalDrops      uint64 `json:"global_drops"`
	PerIPDrops       uint64 `json:"per_ip_drops"`
}

// XDPRateLimiter manages XDP-based rate limiting
type XDPRateLimiter struct {
	enabled           bool
	config            *RateLimitConfig
	program           *C.struct_bpf_object
	attachedInterfaces map[string]bool
	configMapFD       int
	stateMapFD        int
	globalStateMapFD  int
	statsMapFD        int
	licenseMapFD      int

	logger   logging.Logger
	metrics  metrics.Collector
	mu       sync.RWMutex
}

// NewXDPRateLimiter creates a new XDP-based rate limiter
func NewXDPRateLimiter(logger logging.Logger, metricsCollector metrics.Collector) *XDPRateLimiter {
	return &XDPRateLimiter{
		enabled:            false,
		attachedInterfaces: make(map[string]bool),
		logger:             logger,
		metrics:            metricsCollector,
	}
}

// LoadProgram loads the XDP rate limiting program
func (rl *XDPRateLimiter) LoadProgram(programPath string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.program != nil {
		return fmt.Errorf("XDP program already loaded")
	}

	cPath := C.CString(programPath)
	defer C.free(unsafe.Pointer(cPath))

	rl.program = C.load_bpf_program(cPath)
	if rl.program == nil {
		return fmt.Errorf("failed to load XDP rate limiting program from %s", programPath)
	}

	// Get map file descriptors
	var err error
	if rl.configMapFD, err = rl.getMapFD("rate_limit_config_map"); err != nil {
		return fmt.Errorf("failed to get config map FD: %v", err)
	}

	if rl.stateMapFD, err = rl.getMapFD("ip_rate_state_map"); err != nil {
		return fmt.Errorf("failed to get state map FD: %v", err)
	}

	if rl.globalStateMapFD, err = rl.getMapFD("global_rate_state_map"); err != nil {
		return fmt.Errorf("failed to get global state map FD: %v", err)
	}

	if rl.statsMapFD, err = rl.getMapFD("rate_limit_stats_map"); err != nil {
		return fmt.Errorf("failed to get stats map FD: %v", err)
	}

	if rl.licenseMapFD, err = rl.getMapFD("enterprise_license_map"); err != nil {
		return fmt.Errorf("failed to get license map FD: %v", err)
	}

	rl.logger.Info("XDP rate limiting program loaded successfully")
	return nil
}

// AttachToInterface attaches the XDP program to a network interface
func (rl *XDPRateLimiter) AttachToInterface(interfaceName string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.program == nil {
		return fmt.Errorf("XDP program not loaded")
	}

	if rl.attachedInterfaces[interfaceName] {
		return fmt.Errorf("already attached to interface %s", interfaceName)
	}

	// Get program FD
	progFD := rl.getProgramFD("xdp_rate_limiter")
	if progFD < 0 {
		return fmt.Errorf("failed to get program FD")
	}

	cIfName := C.CString(interfaceName)
	defer C.free(unsafe.Pointer(cIfName))

	// Attach with XDP_FLAGS_SKB_MODE for compatibility
	ret := C.attach_xdp_to_interface(cIfName, C.int(progFD), 2) // XDP_FLAGS_SKB_MODE
	if ret != 0 {
		// Try with hardware offload if available
		ret = C.attach_xdp_to_interface(cIfName, C.int(progFD), 8) // XDP_FLAGS_HW_MODE
		if ret != 0 {
			// Fall back to driver mode
			ret = C.attach_xdp_to_interface(cIfName, C.int(progFD), 4) // XDP_FLAGS_DRV_MODE
			if ret != 0 {
				return fmt.Errorf("failed to attach XDP program to interface %s: %d", interfaceName, ret)
			}
		}
	}

	rl.attachedInterfaces[interfaceName] = true
	rl.logger.Info("XDP rate limiter attached to interface", "interface", interfaceName)

	// Update metrics
	rl.metrics.GaugeSet("xdp_attached_interfaces", float64(len(rl.attachedInterfaces)))

	return nil
}

// DetachFromInterface detaches the XDP program from a network interface
func (rl *XDPRateLimiter) DetachFromInterface(interfaceName string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.attachedInterfaces[interfaceName] {
		return fmt.Errorf("not attached to interface %s", interfaceName)
	}

	cIfName := C.CString(interfaceName)
	defer C.free(unsafe.Pointer(cIfName))

	ret := C.detach_xdp_from_interface(cIfName)
	if ret != 0 {
		return fmt.Errorf("failed to detach XDP program from interface %s: %d", interfaceName, ret)
	}

	delete(rl.attachedInterfaces, interfaceName)
	rl.logger.Info("XDP rate limiter detached from interface", "interface", interfaceName)

	// Update metrics
	rl.metrics.GaugeSet("xdp_attached_interfaces", float64(len(rl.attachedInterfaces)))

	return nil
}

// UpdateConfig updates the rate limiting configuration
func (rl *XDPRateLimiter) UpdateConfig(config *RateLimitConfig) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.configMapFD <= 0 {
		return fmt.Errorf("config map not available")
	}

	rl.config = config

	// Convert to C structure
	cConfig := struct {
		Enabled          uint32
		GlobalPPSLimit   uint32
		PerIPPPSLimit    uint32
		WindowSizeNS     uint32
		BurstAllowance   uint32
		Action           uint32
	}{
		Enabled:          boolToUint32(config.Enabled),
		GlobalPPSLimit:   config.GlobalPPSLimit,
		PerIPPPSLimit:    config.PerIPPPSLimit,
		WindowSizeNS:     config.WindowSizeNS,
		BurstAllowance:   config.BurstAllowance,
		Action:           config.Action,
	}

	// Update the map
	key := uint32(0)
	ret := C.bpf_map_update_elem(C.int(rl.configMapFD), unsafe.Pointer(&key), unsafe.Pointer(&cConfig), 0)
	if ret != 0 {
		return fmt.Errorf("failed to update rate limit config: %d", ret)
	}

	rl.enabled = config.Enabled
	rl.logger.Info("Rate limiting configuration updated", "config", config)

	// Update metrics
	rl.metrics.GaugeSet("rate_limit_enabled", boolToFloat64(config.Enabled))
	rl.metrics.GaugeSet("rate_limit_global_pps", float64(config.GlobalPPSLimit))
	rl.metrics.GaugeSet("rate_limit_per_ip_pps", float64(config.PerIPPPSLimit))

	return nil
}

// SetEnterpriseLicense updates the Enterprise license status in XDP
func (rl *XDPRateLimiter) SetEnterpriseLicense(enabled bool) error {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rl.licenseMapFD <= 0 {
		return fmt.Errorf("license map not available")
	}

	key := uint32(0)
	value := boolToUint32(enabled)

	ret := C.bpf_map_update_elem(C.int(rl.licenseMapFD), unsafe.Pointer(&key), unsafe.Pointer(&value), 0)
	if ret != 0 {
		return fmt.Errorf("failed to update enterprise license status: %d", ret)
	}

	rl.logger.Info("Enterprise license status updated in XDP", "enabled", enabled)
	rl.metrics.GaugeSet("enterprise_license_active", boolToFloat64(enabled))

	return nil
}

// GetStats retrieves current rate limiting statistics
func (rl *XDPRateLimiter) GetStats() (*RateLimitStats, error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rl.statsMapFD <= 0 {
		return nil, fmt.Errorf("stats map not available")
	}

	key := uint32(0)
	var stats RateLimitStats

	ret := C.bpf_map_lookup_elem(C.int(rl.statsMapFD), unsafe.Pointer(&key), unsafe.Pointer(&stats))
	if ret != 0 {
		return nil, fmt.Errorf("failed to get rate limiting stats: %d", ret)
	}

	// Update Prometheus metrics
	rl.metrics.CounterSet("rate_limit_total_packets", float64(stats.TotalPackets))
	rl.metrics.CounterSet("rate_limit_passed_packets", float64(stats.PassedPackets))
	rl.metrics.CounterSet("rate_limit_dropped_packets", float64(stats.DroppedPackets))
	rl.metrics.GaugeSet("rate_limit_active_ips", float64(stats.RateLimitedIPs))

	return &stats, nil
}

// GetIPState retrieves rate limiting state for a specific IP
func (rl *XDPRateLimiter) GetIPState(ip net.IP) (*IPRateState, error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rl.stateMapFD <= 0 {
		return nil, fmt.Errorf("state map not available")
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return nil, fmt.Errorf("only IPv4 addresses supported")
	}

	// Convert IP to uint32 (network byte order)
	key := uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])

	var state IPRateState
	ret := C.bpf_map_lookup_elem(C.int(rl.stateMapFD), unsafe.Pointer(&key), unsafe.Pointer(&state))
	if ret != 0 {
		return nil, fmt.Errorf("IP state not found or map lookup failed: %d", ret)
	}

	return &state, nil
}

// ClearIPState removes rate limiting state for a specific IP
func (rl *XDPRateLimiter) ClearIPState(ip net.IP) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.stateMapFD <= 0 {
		return fmt.Errorf("state map not available")
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return fmt.Errorf("only IPv4 addresses supported")
	}

	// Convert IP to uint32 (network byte order)
	key := uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])

	ret := C.bpf_map_delete_elem(C.int(rl.stateMapFD), unsafe.Pointer(&key))
	if ret != 0 {
		return fmt.Errorf("failed to clear IP state: %d", ret)
	}

	rl.logger.Debug("Cleared rate limiting state", "ip", ip.String())
	return nil
}

// Close cleans up the XDP rate limiter
func (rl *XDPRateLimiter) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Detach from all interfaces
	for ifName := range rl.attachedInterfaces {
		cIfName := C.CString(ifName)
		C.detach_xdp_from_interface(cIfName)
		C.free(unsafe.Pointer(cIfName))
	}

	// Close BPF object
	if rl.program != nil {
		C.close_bpf_object(rl.program)
		rl.program = nil
	}

	rl.attachedInterfaces = make(map[string]bool)
	rl.enabled = false

	rl.logger.Info("XDP rate limiter closed")
	return nil
}

// Helper functions

func (rl *XDPRateLimiter) getMapFD(mapName string) (int, error) {
	if rl.program == nil {
		return -1, fmt.Errorf("BPF program not loaded")
	}

	cMapName := C.CString(mapName)
	defer C.free(unsafe.Pointer(cMapName))

	fd := C.get_map_fd_by_name(rl.program, cMapName)
	if fd < 0 {
		return -1, fmt.Errorf("failed to get map FD for %s", mapName)
	}

	return int(fd), nil
}

func (rl *XDPRateLimiter) getProgramFD(progName string) int {
	// This would be implemented using libbpf to get program FD
	// For now, return a placeholder
	return -1
}

func boolToUint32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// IsEnabled returns whether rate limiting is currently enabled
func (rl *XDPRateLimiter) IsEnabled() bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.enabled
}

// GetAttachedInterfaces returns a list of interfaces with XDP attached
func (rl *XDPRateLimiter) GetAttachedInterfaces() []string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	interfaces := make([]string, 0, len(rl.attachedInterfaces))
	for ifName := range rl.attachedInterfaces {
		interfaces = append(interfaces, ifName)
	}
	return interfaces
}

// GetConfig returns the current rate limiting configuration
func (rl *XDPRateLimiter) GetConfig() *RateLimitConfig {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rl.config == nil {
		return nil
	}

	// Return a copy to prevent external modification
	configCopy := *rl.config
	return &configCopy
}