//go:build linux

package xdp

import (
	"fmt"
	"os"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// XDPProgram represents an XDP eBPF program
type XDPProgram struct {
	mu sync.RWMutex

	device  string
	logger  *logrus.Logger
	loaded  bool
	link    link.Link
	objects *xdpObjects

	// Statistics
	packetsProcessed uint64
	packetsDropped   uint64
	bytesProcessed   uint64
}

// xdpObjects represents the eBPF objects
type xdpObjects struct {
	Program *ebpf.Program `ebpf:"xdp_prog"`
	Stats   *ebpf.Map     `ebpf:"xdp_stats_map"`
}

// NewXDPProgram creates a new XDP program instance
func NewXDPProgram(device string, logger *logrus.Logger) *XDPProgram {
	return &XDPProgram{
		device: device,
		logger: logger,
	}
}

// Load loads and attaches the XDP program
func (xdp *XDPProgram) Load(programPath string) error {
	xdp.mu.Lock()
	defer xdp.mu.Unlock()

	if xdp.loaded {
		return fmt.Errorf("XDP program already loaded")
	}

	// Load the eBPF program
	spec, err := ebpf.LoadCollectionSpec(programPath)
	if err != nil {
		// If we can't load from file, try to use embedded or generate simple program
		xdp.logger.WithError(err).Warn("Failed to load XDP program from file, using default")
		return xdp.loadDefaultProgram()
	}

	// Load eBPF objects
	objs := &xdpObjects{}
	if err := spec.LoadAndAssign(objs, nil); err != nil {
		return fmt.Errorf("loading eBPF objects: %w", err)
	}

	// Get the network interface
	iface, err := netlink.LinkByName(xdp.device)
	if err != nil {
		objs.Program.Close()
		return fmt.Errorf("getting interface %s: %w", xdp.device, err)
	}

	// Attach XDP program to interface
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.Program,
		Interface: iface.Attrs().Index,
		Flags:     link.XDPGenericMode, // Use generic mode for compatibility
	})
	if err != nil {
		objs.Program.Close()
		return fmt.Errorf("attaching XDP program: %w", err)
	}

	xdp.link = l
	xdp.objects = objs
	xdp.loaded = true

	xdp.logger.WithFields(logrus.Fields{
		"device":  xdp.device,
		"program": programPath,
	}).Info("XDP program loaded and attached")

	return nil
}

// loadDefaultProgram loads a simple pass-through XDP program
func (xdp *XDPProgram) loadDefaultProgram() error {
	// Create a simple XDP program that passes all packets
	// This is a minimal implementation for when no custom program is provided
	spec := &ebpf.ProgramSpec{
		Type: ebpf.XDP,
		Instructions: []ebpf.Instruction{
			// XDP_PASS = 2
			ebpf.LoadImm(ebpf.R0, 2, ebpf.DWord),
			ebpf.Return(),
		},
		License: "GPL",
	}

	prog, err := ebpf.NewProgram(spec)
	if err != nil {
		return fmt.Errorf("creating default XDP program: %w", err)
	}

	// Get the network interface
	iface, err := netlink.LinkByName(xdp.device)
	if err != nil {
		prog.Close()
		return fmt.Errorf("getting interface %s: %w", xdp.device, err)
	}

	// Attach XDP program to interface
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: iface.Attrs().Index,
		Flags:     link.XDPGenericMode,
	})
	if err != nil {
		prog.Close()
		return fmt.Errorf("attaching default XDP program: %w", err)
	}

	xdp.link = l
	xdp.objects = &xdpObjects{
		Program: prog,
	}
	xdp.loaded = true

	xdp.logger.WithField("device", xdp.device).Info("Default XDP program loaded")
	return nil
}

// Unload unloads the XDP program
func (xdp *XDPProgram) Unload() error {
	xdp.mu.Lock()
	defer xdp.mu.Unlock()

	if !xdp.loaded {
		return nil
	}

	// Close the link (detaches the program)
	if xdp.link != nil {
		if err := xdp.link.Close(); err != nil {
			xdp.logger.WithError(err).Error("Error closing XDP link")
		}
		xdp.link = nil
	}

	// Close the program and maps
	if xdp.objects != nil {
		if xdp.objects.Program != nil {
			xdp.objects.Program.Close()
		}
		if xdp.objects.Stats != nil {
			xdp.objects.Stats.Close()
		}
		xdp.objects = nil
	}

	xdp.loaded = false
	xdp.logger.WithField("device", xdp.device).Info("XDP program unloaded")

	return nil
}

// IsLoaded returns whether the program is loaded
func (xdp *XDPProgram) IsLoaded() bool {
	xdp.mu.RLock()
	defer xdp.mu.RUnlock()
	return xdp.loaded
}

// GetStats returns XDP statistics
func (xdp *XDPProgram) GetStats() map[string]interface{} {
	xdp.mu.RLock()
	defer xdp.mu.RUnlock()

	return map[string]interface{}{
		"device":            xdp.device,
		"loaded":            xdp.loaded,
		"packets_processed": xdp.packetsProcessed,
		"packets_dropped":   xdp.packetsDropped,
		"bytes_processed":   xdp.bytesProcessed,
	}
}

// UpdateStats updates internal statistics from eBPF maps
func (xdp *XDPProgram) UpdateStats() error {
	xdp.mu.Lock()
	defer xdp.mu.Unlock()

	if !xdp.loaded {
		return fmt.Errorf("XDP program not loaded")
	}

	// If we have a stats map, read from it
	if xdp.objects != nil && xdp.objects.Stats != nil {
		var stats struct {
			PacketsProcessed uint64
			PacketsDropped   uint64
			BytesProcessed   uint64
		}

		key := uint32(0)
		if err := xdp.objects.Stats.Lookup(&key, &stats); err != nil {
			// Map might not exist or have data yet
			return nil
		}

		xdp.packetsProcessed = stats.PacketsProcessed
		xdp.packetsDropped = stats.PacketsDropped
		xdp.bytesProcessed = stats.BytesProcessed
	}

	return nil
}

// GetProgramFD returns the file descriptor of the loaded program
func (xdp *XDPProgram) GetProgramFD() (*os.File, error) {
	xdp.mu.RLock()
	defer xdp.mu.RUnlock()

	if !xdp.loaded || xdp.objects == nil || xdp.objects.Program == nil {
		return nil, fmt.Errorf("XDP program not loaded")
	}

	return xdp.objects.Program.FD().File("xdp_prog")
}
