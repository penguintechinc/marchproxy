package nlb

import (
	"bytes"
	"errors"
)

// Protocol represents supported protocols for detection
type Protocol int

const (
	ProtocolUnknown Protocol = iota
	ProtocolHTTP
	ProtocolMySQL
	ProtocolPostgreSQL
	ProtocolMongoDB
	ProtocolRedis
	ProtocolRTMP
)

// String returns the string representation of the protocol
func (p Protocol) String() string {
	switch p {
	case ProtocolHTTP:
		return "HTTP"
	case ProtocolMySQL:
		return "MySQL"
	case ProtocolPostgreSQL:
		return "PostgreSQL"
	case ProtocolMongoDB:
		return "MongoDB"
	case ProtocolRedis:
		return "Redis"
	case ProtocolRTMP:
		return "RTMP"
	default:
		return "Unknown"
	}
}

// ProtocolInspector provides protocol detection capabilities
type ProtocolInspector struct {
	minBytesRequired int
}

// NewProtocolInspector creates a new protocol inspector
func NewProtocolInspector() *ProtocolInspector {
	return &ProtocolInspector{
		minBytesRequired: 16, // Minimum bytes needed for reliable detection
	}
}

// InspectProtocol detects the protocol from the first packet data
func (pi *ProtocolInspector) InspectProtocol(data []byte) (Protocol, error) {
	if len(data) < 3 {
		return ProtocolUnknown, errors.New("insufficient data for protocol detection")
	}

	// HTTP detection - check for common HTTP methods and version
	if pi.isHTTP(data) {
		return ProtocolHTTP, nil
	}

	// MySQL detection - check for greeting packet
	if pi.isMySQL(data) {
		return ProtocolMySQL, nil
	}

	// PostgreSQL detection - check for startup or query message
	if pi.isPostgreSQL(data) {
		return ProtocolPostgreSQL, nil
	}

	// MongoDB detection - check for OP_MSG or OP_QUERY headers
	if pi.isMongoDB(data) {
		return ProtocolMongoDB, nil
	}

	// Redis detection - check for RESP protocol
	if pi.isRedis(data) {
		return ProtocolRedis, nil
	}

	// RTMP detection - check for handshake
	if pi.isRTMP(data) {
		return ProtocolRTMP, nil
	}

	return ProtocolUnknown, nil
}

// isHTTP checks if data contains HTTP protocol signatures
func (pi *ProtocolInspector) isHTTP(data []byte) bool {
	httpMethods := [][]byte{
		[]byte("GET "),
		[]byte("POST "),
		[]byte("PUT "),
		[]byte("DELETE "),
		[]byte("HEAD "),
		[]byte("OPTIONS "),
		[]byte("PATCH "),
		[]byte("CONNECT "),
		[]byte("TRACE "),
		[]byte("HTTP/"),
	}

	for _, method := range httpMethods {
		if bytes.HasPrefix(data, method) {
			return true
		}
	}

	return false
}

// isMySQL checks if data contains MySQL protocol signatures
// MySQL greeting packet starts with protocol version (0x0a for MySQL 5.x+)
func (pi *ProtocolInspector) isMySQL(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// Check for MySQL protocol version 10 (0x0a)
	// Format: packet_length (3 bytes) + sequence_id (1 byte) + protocol_version (1 byte)
	if data[4] == 0x0a {
		return true
	}

	// Also check for common MySQL commands (COM_QUIT, COM_QUERY, etc.)
	if len(data) >= 5 && data[4] >= 0x00 && data[4] <= 0x1f {
		// Additional validation: check packet structure
		packetLen := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
		if packetLen > 0 && packetLen < 16777215 { // Max MySQL packet size
			return true
		}
	}

	return false
}

// isPostgreSQL checks if data contains PostgreSQL protocol signatures
// PostgreSQL startup message or simple query protocol
func (pi *ProtocolInspector) isPostgreSQL(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	// Startup message: length (4 bytes) + protocol version (4 bytes)
	// Protocol version is 196608 (0x00030000) for PostgreSQL 3.0
	if len(data) >= 8 {
		version := uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
		if version == 196608 {
			return true
		}
	}

	// Simple query: starts with 'Q'
	if data[0] == 'Q' {
		return true
	}

	// Startup packet: starts with 'S', 'P', 'X', or 'p'
	if data[0] == 'S' || data[0] == 'P' || data[0] == 'X' || data[0] == 'p' {
		return true
	}

	return false
}

// isMongoDB checks if data contains MongoDB protocol signatures
// MongoDB wire protocol with OP_MSG (2013) or OP_QUERY (2004) opcodes
func (pi *ProtocolInspector) isMongoDB(data []byte) bool {
	if len(data) < 16 {
		return false
	}

	// MongoDB message structure:
	// messageLength (4 bytes) + requestID (4 bytes) + responseTo (4 bytes) + opCode (4 bytes)

	// Extract opCode (little-endian)
	opCode := uint32(data[12]) | uint32(data[13])<<8 | uint32(data[14])<<16 | uint32(data[15])<<24

	// Check for known MongoDB opcodes
	switch opCode {
	case 2004: // OP_QUERY (deprecated but still used)
		return true
	case 2013: // OP_MSG (current standard)
		return true
	case 2001: // OP_UPDATE
		return true
	case 2002: // OP_INSERT
		return true
	case 2005: // OP_GET_MORE
		return true
	case 2006: // OP_DELETE
		return true
	case 2007: // OP_KILL_CURSORS
		return true
	}

	return false
}

// isRedis checks if data contains Redis RESP protocol signatures
// RESP (Redis Serialization Protocol) uses specific first-byte indicators
func (pi *ProtocolInspector) isRedis(data []byte) bool {
	if len(data) < 3 {
		return false
	}

	// RESP protocol type indicators:
	// '+' - Simple String
	// '-' - Error
	// ':' - Integer
	// '$' - Bulk String
	// '*' - Array
	switch data[0] {
	case '+', '-', ':', '$', '*':
		// Verify RESP format with \r\n line endings
		if len(data) >= 3 {
			// Look for \r\n in the first few bytes
			for i := 1; i < len(data)-1 && i < 20; i++ {
				if data[i] == '\r' && data[i+1] == '\n' {
					return true
				}
			}
		}
	}

	// Common Redis commands as fallback
	redisCommands := [][]byte{
		[]byte("PING"),
		[]byte("GET "),
		[]byte("SET "),
		[]byte("DEL "),
		[]byte("AUTH "),
		[]byte("SELECT "),
	}

	dataUpper := bytes.ToUpper(data[:min(len(data), 10)])
	for _, cmd := range redisCommands {
		if bytes.HasPrefix(dataUpper, cmd) {
			return true
		}
	}

	return false
}

// isRTMP checks if data contains RTMP protocol signatures
// RTMP handshake starts with 0x03
func (pi *ProtocolInspector) isRTMP(data []byte) bool {
	if len(data) < 1 {
		return false
	}

	// RTMP handshake C0/S0 packet starts with version byte (0x03)
	if data[0] == 0x03 {
		// Additional validation for handshake structure
		// C0 (1 byte) + C1 (1536 bytes) = 1537 bytes total for client handshake
		// We should see 0x03 followed by timestamp and random bytes
		if len(data) >= 9 {
			// Basic validation: not all zeros after version byte
			hasNonZero := false
			for i := 1; i < min(len(data), 9); i++ {
				if data[i] != 0x00 {
					hasNonZero = true
					break
				}
			}
			return hasNonZero
		}
		return true
	}

	return false
}

// GetMinBytesRequired returns minimum bytes needed for detection
func (pi *ProtocolInspector) GetMinBytesRequired() int {
	return pi.minBytesRequired
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
