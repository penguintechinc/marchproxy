package qos

import (
	"fmt"
	"sync"
)

// DSCP values (RFC 2474)
const (
	DSCP_CS0  uint8 = 0  // Class Selector 0 (Best Effort)
	DSCP_CS1  uint8 = 8  // Class Selector 1
	DSCP_AF11 uint8 = 10 // Assured Forwarding 11
	DSCP_AF12 uint8 = 12 // Assured Forwarding 12
	DSCP_AF13 uint8 = 14 // Assured Forwarding 13
	DSCP_CS2  uint8 = 16 // Class Selector 2
	DSCP_AF21 uint8 = 18 // Assured Forwarding 21
	DSCP_AF22 uint8 = 20 // Assured Forwarding 22
	DSCP_AF23 uint8 = 22 // Assured Forwarding 23
	DSCP_CS3  uint8 = 24 // Class Selector 3
	DSCP_AF31 uint8 = 26 // Assured Forwarding 31
	DSCP_AF32 uint8 = 28 // Assured Forwarding 32
	DSCP_AF33 uint8 = 30 // Assured Forwarding 33
	DSCP_CS4  uint8 = 32 // Class Selector 4
	DSCP_AF41 uint8 = 34 // Assured Forwarding 41
	DSCP_AF42 uint8 = 36 // Assured Forwarding 42
	DSCP_AF43 uint8 = 38 // Assured Forwarding 43
	DSCP_CS5  uint8 = 40 // Class Selector 5
	DSCP_EF   uint8 = 46 // Expedited Forwarding
	DSCP_CS6  uint8 = 48 // Class Selector 6
	DSCP_CS7  uint8 = 56 // Class Selector 7
)

// DSCPMarker handles DSCP marking for packets
type DSCPMarker struct {
	mu sync.RWMutex

	// Priority to DSCP mapping
	mapping map[int]uint8
}

// NewDSCPMarker creates a new DSCP marker
func NewDSCPMarker(customMapping map[string]uint8) *DSCPMarker {
	marker := &DSCPMarker{
		mapping: make(map[int]uint8),
	}

	// Default mapping
	marker.mapping[PriorityP0] = DSCP_EF   // Expedited Forwarding
	marker.mapping[PriorityP1] = DSCP_AF41 // Assured Forwarding 41
	marker.mapping[PriorityP2] = DSCP_AF21 // Assured Forwarding 21
	marker.mapping[PriorityP3] = DSCP_CS0  // Best Effort

	// Apply custom mapping if provided
	if customMapping != nil {
		if dscp, ok := customMapping["P0"]; ok {
			marker.mapping[PriorityP0] = dscp
		}
		if dscp, ok := customMapping["P1"]; ok {
			marker.mapping[PriorityP1] = dscp
		}
		if dscp, ok := customMapping["P2"]; ok {
			marker.mapping[PriorityP2] = dscp
		}
		if dscp, ok := customMapping["P3"]; ok {
			marker.mapping[PriorityP3] = dscp
		}
	}

	return marker
}

// Mark marks a packet with appropriate DSCP value
func (dm *DSCPMarker) Mark(packet *Packet) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	dscp, ok := dm.mapping[packet.Priority]
	if !ok {
		return fmt.Errorf("no DSCP mapping for priority %d", packet.Priority)
	}

	packet.DSCP = dscp
	return nil
}

// UpdateMapping updates the DSCP mapping for a priority
func (dm *DSCPMarker) UpdateMapping(priority int, dscp uint8) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if priority < PriorityP0 || priority > PriorityP3 {
		return fmt.Errorf("invalid priority: %d", priority)
	}

	if dscp > 63 {
		return fmt.Errorf("invalid DSCP value: %d (must be 0-63)", dscp)
	}

	dm.mapping[priority] = dscp
	return nil
}

// GetMapping returns the current DSCP mapping
func (dm *DSCPMarker) GetMapping() map[int]uint8 {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	mapping := make(map[int]uint8)
	for k, v := range dm.mapping {
		mapping[k] = v
	}
	return mapping
}

// DSCPToString converts a DSCP value to its name
func DSCPToString(dscp uint8) string {
	switch dscp {
	case DSCP_CS0:
		return "CS0 (Best Effort)"
	case DSCP_AF11:
		return "AF11"
	case DSCP_AF12:
		return "AF12"
	case DSCP_AF13:
		return "AF13"
	case DSCP_AF21:
		return "AF21"
	case DSCP_AF22:
		return "AF22"
	case DSCP_AF23:
		return "AF23"
	case DSCP_AF31:
		return "AF31"
	case DSCP_AF32:
		return "AF32"
	case DSCP_AF33:
		return "AF33"
	case DSCP_AF41:
		return "AF41"
	case DSCP_AF42:
		return "AF42"
	case DSCP_AF43:
		return "AF43"
	case DSCP_EF:
		return "EF (Expedited Forwarding)"
	case DSCP_CS6:
		return "CS6"
	case DSCP_CS7:
		return "CS7"
	default:
		return fmt.Sprintf("Unknown (%d)", dscp)
	}
}
