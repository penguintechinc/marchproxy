package ebpf

import (
	"net"
	"time"
)

// EBPFManager handles eBPF program lifecycle and map management
type EBPFManager struct {
	enabled     bool
	programPath string
	maps        *EBPFMaps
	stats       *EBPFStats
}

// EBPFMaps represents the eBPF maps for configuration and tracking
type EBPFMaps struct {
	Services    map[uint32]*EBPFService
	Mappings    map[uint32]*EBPFMapping
	Connections map[ConnectionKey]*ConnectionValue
	Stats       *ProxyStats
}

// EBPFService represents a service in eBPF map format
type EBPFService struct {
	ID           uint32
	IPAddr       uint32 // Network byte order
	Port         uint16
	AuthRequired uint8  // 0 = no auth, 1 = auth required
	AuthType     uint8  // 0 = none, 1 = base64, 2 = jwt
	Flags        uint32
}

// EBPFMapping represents a mapping rule in eBPF map format
type EBPFMapping struct {
	ID             uint32
	SourceServices [16]uint32 // Source service IDs
	DestServices   [16]uint32 // Destination service IDs
	Ports          [16]uint16 // Allowed ports
	Protocols      uint8      // Bitmask: 1=TCP, 2=UDP, 4=ICMP
	AuthRequired   uint8      // Authentication requirement
	Priority       uint8      // Routing priority (higher = preferred)
	PortCount      uint8      // Number of valid ports
	SrcCount       uint8      // Number of source services
	DestCount      uint8      // Number of dest services
}

// ConnectionKey represents a connection tracking key
type ConnectionKey struct {
	SrcIP    uint32
	DstIP    uint32
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
}

// ConnectionValue represents connection tracking data
type ConnectionValue struct {
	Packets       uint64
	Bytes         uint64
	Timestamp     uint64
	ServiceID     uint32
	Authenticated uint8
}

// ProxyStats represents eBPF statistics
type ProxyStats struct {
	TotalPackets        uint64
	TotalBytes          uint64
	TCPPackets          uint64
	UDPPackets          uint64
	ICMPPackets         uint64
	DroppedPackets      uint64
	ForwardedPackets    uint64
	AuthRequired        uint64
	FallbackToUserspace uint64
}

// EBPFStats provides runtime statistics and monitoring
type EBPFStats struct {
	ProgramLoaded    bool
	AttachedInterfaces []string
	LastUpdate       time.Time
	MapSyncErrors    uint64
	ProgramErrors    uint64
}

// Constants for eBPF programs
const (
	// Protocol constants
	ProtoTCP  = 1
	ProtoUDP  = 2
	ProtoICMP = 4

	// Action constants
	ActionDrop     = 0
	ActionForward  = 1
	ActionFallback = 2

	// Auth type constants
	AuthTypeNone   = 0
	AuthTypeBase64 = 1
	AuthTypeJWT    = 2

	// Map size limits
	MaxServices    = 1024
	MaxMappings    = 512
	MaxPorts       = 16
	MaxConnections = 65536
)

// Helper functions for network operations
func IPToUint32(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func Uint32ToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// ProtocolToMask converts protocol name to bitmask
func ProtocolToMask(protocol string) uint8 {
	switch protocol {
	case "tcp":
		return ProtoTCP
	case "udp":
		return ProtoUDP
	case "icmp":
		return ProtoICMP
	default:
		return 0
	}
}

// MaskToProtocols converts bitmask to protocol names
func MaskToProtocols(mask uint8) []string {
	var protocols []string
	if mask&ProtoTCP != 0 {
		protocols = append(protocols, "tcp")
	}
	if mask&ProtoUDP != 0 {
		protocols = append(protocols, "udp")
	}
	if mask&ProtoICMP != 0 {
		protocols = append(protocols, "icmp")
	}
	return protocols
}