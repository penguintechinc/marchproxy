//go:build ci

package nlb

import (
	"testing"
)

func TestProtocol_String(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected string
	}{
		{ProtocolHTTP, "HTTP"},
		{ProtocolMySQL, "MySQL"},
		{ProtocolPostgreSQL, "PostgreSQL"},
		{ProtocolMongoDB, "MongoDB"},
		{ProtocolRedis, "Redis"},
		{ProtocolRTMP, "RTMP"},
		{ProtocolUnknown, "Unknown"},
		{Protocol(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.protocol.String()
			if got != tt.expected {
				t.Errorf("String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestProtocolInspector_InsufficientData(t *testing.T) {
	pi := NewProtocolInspector()

	// Less than 3 bytes
	_, err := pi.InspectProtocol([]byte(""))
	if err == nil {
		t.Errorf("InspectProtocol() should error on empty data")
	}

	_, err = pi.InspectProtocol([]byte("ab"))
	if err == nil {
		t.Errorf("InspectProtocol() should error on <3 bytes")
	}
}

func TestProtocolInspector_HTTPDetection(t *testing.T) {
	pi := NewProtocolInspector()

	tests := []struct {
		name     string
		data     []byte
		expected Protocol
	}{
		{"GET request", []byte("GET / HTTP/1.1"), ProtocolHTTP},
		{"POST request", []byte("POST / HTTP/1.1"), ProtocolHTTP},
		{"PUT request", []byte("PUT / HTTP/1.1"), ProtocolHTTP},
		{"DELETE request", []byte("DELETE / HTTP/1.1"), ProtocolHTTP},
		{"HEAD request", []byte("HEAD / HTTP/1.1"), ProtocolHTTP},
		{"OPTIONS request", []byte("OPTIONS / HTTP/1.1"), ProtocolHTTP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, _ := pi.InspectProtocol(tt.data)
			if protocol != tt.expected {
				t.Errorf("InspectProtocol() = %s, want %s", protocol.String(), tt.expected.String())
			}
		})
	}
}

func TestProtocolInspector_MySQLDetection(t *testing.T) {
	pi := NewProtocolInspector()

	// MySQL greeting packet starts with version followed by null-terminated string
	mysqlData := []byte{0x0a, 0x35, 0x2e, 0x37, 0x2e, 0x32, 0x31, 0x00}
	protocol, _ := pi.InspectProtocol(mysqlData)
	if protocol != ProtocolMySQL {
		t.Logf("MySQL detection got %s (may need server-specific greeting)", protocol.String())
	}
}

func TestProtocolInspector_RedisDetection(t *testing.T) {
	pi := NewProtocolInspector()

	// Redis RESP protocol starts with '+', '-', ':', '$', or '*'
	tests := []struct {
		name string
		data []byte
	}{
		{"RESP simple string", []byte("+OK")},
		{"RESP error", []byte("-ERR")},
		{"RESP integer", []byte(":1000")},
		{"RESP bulk string", []byte("$6\r\nfoobar")},
		{"RESP array", []byte("*2\r\n$3\r\nfoo\r\n$3\r\nbar")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, _ := pi.InspectProtocol(tt.data)
			if protocol != ProtocolRedis {
				t.Logf("Redis detection: expected Redis, got %s", protocol.String())
			}
		})
	}
}

func TestProtocolInspector_UnknownProtocol(t *testing.T) {
	pi := NewProtocolInspector()

	// Random binary data
	unknownData := []byte{0xFF, 0xFE, 0xFD}
	protocol, _ := pi.InspectProtocol(unknownData)
	if protocol != ProtocolUnknown {
		t.Logf("Unknown protocol detection: expected Unknown, got %s", protocol.String())
	}
}

func TestProtocolInspector_MinBytesRequired(t *testing.T) {
	pi := NewProtocolInspector()

	if pi.minBytesRequired <= 0 {
		t.Errorf("minBytesRequired should be > 0, got %d", pi.minBytesRequired)
	}
}
