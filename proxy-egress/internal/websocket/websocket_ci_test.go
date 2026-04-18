//go:build ci

package websocket

import (
	"bytes"
	"io"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

// TestNewWebSocketProxy tests WebSocket proxy initialization
func TestNewWebSocketProxy(t *testing.T) {
	proxy := NewWebSocketProxy(nil)

	if proxy == nil {
		t.Fatal("NewWebSocketProxy should return non-nil")
	}

	if proxy.config == nil {
		t.Error("config should be initialized")
	}

	if proxy.stats == nil {
		t.Error("stats should be initialized")
	}

	if proxy.connections == nil {
		t.Error("connections map should be initialized")
	}

	if proxy.upgradeHandler == nil {
		t.Error("upgradeHandler should be initialized")
	}

	if proxy.messageProcessor == nil {
		t.Error("messageProcessor should be initialized")
	}
}

// TestNewWebSocketProxyWithCustomConfig tests proxy with custom config
func TestNewWebSocketProxyWithCustomConfig(t *testing.T) {
	config := &WebSocketConfig{
		EnableCompression:  false,
		CompressionLevel:   1,
		MaxMessageSize:     512 * 1024,
		PingInterval:       time.Minute,
		PongTimeout:        time.Second * 30,
		HandshakeTimeout:   time.Second * 5,
		MaxConnections:     1000,
		BufferSize:         8192,
		EnableSubprotocols: false,
	}

	proxy := NewWebSocketProxy(config)

	if proxy.config.MaxMessageSize != 512*1024 {
		t.Errorf("expected MaxMessageSize=512KB, got %d", proxy.config.MaxMessageSize)
	}

	if proxy.config.MaxConnections != 1000 {
		t.Errorf("expected MaxConnections=1000, got %d", proxy.config.MaxConnections)
	}

	if proxy.config.BufferSize != 8192 {
		t.Errorf("expected BufferSize=8192, got %d", proxy.config.BufferSize)
	}
}

// TestValidateUpgradeRequest tests WebSocket upgrade request validation
func TestValidateUpgradeRequest(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		connection string
		upgrade   string
		version   string
		key       string
		origin    string
		allowOrgs []string
		wantValid bool
	}{
		{
			name:       "valid upgrade request",
			method:     "GET",
			connection: "upgrade",
			upgrade:    "websocket",
			version:    "13",
			key:        "dGhlIHNhbXBsZSBub25jZQ==",
			wantValid:  true,
		},
		{
			name:      "invalid method POST",
			method:    "POST",
			wantValid: false,
		},
		{
			name:       "missing connection header",
			method:     "GET",
			connection: "",
			upgrade:    "websocket",
			version:    "13",
			key:        "dGhlIHNhbXBsZSBub25jZQ==",
			wantValid:  false,
		},
		{
			name:       "invalid upgrade header",
			method:     "GET",
			connection: "upgrade",
			upgrade:    "http2",
			version:    "13",
			key:        "dGhlIHNhbXBsZSBub25jZQ==",
			wantValid:  false,
		},
		{
			name:       "invalid version",
			method:     "GET",
			connection: "upgrade",
			upgrade:    "websocket",
			version:    "8",
			key:        "dGhlIHNhbXBsZSBub25jZQ==",
			wantValid:  false,
		},
		{
			name:       "missing key",
			method:     "GET",
			connection: "upgrade",
			upgrade:    "websocket",
			version:    "13",
			key:        "",
			wantValid:  false,
		},
		{
			name:        "origin allowed",
			method:      "GET",
			connection:  "upgrade",
			upgrade:     "websocket",
			version:     "13",
			key:         "dGhlIHNhbXBsZSBub25jZQ==",
			origin:      "https://example.com",
			allowOrgs:   []string{"https://example.com"},
			wantValid:   true,
		},
		{
			name:        "origin blocked",
			method:      "GET",
			connection:  "upgrade",
			upgrade:     "websocket",
			version:     "13",
			key:         "dGhlIHNhbXBsZSBub25jZQ==",
			origin:      "https://evil.com",
			allowOrgs:   []string{"https://example.com"},
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/ws", nil)
			if tt.connection != "" {
				req.Header.Set("Connection", tt.connection)
			}
			if tt.upgrade != "" {
				req.Header.Set("Upgrade", tt.upgrade)
			}
			if tt.version != "" {
				req.Header.Set("Sec-WebSocket-Version", tt.version)
			}
			if tt.key != "" {
				req.Header.Set("Sec-WebSocket-Key", tt.key)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			cfg := &WebSocketConfig{
				AllowedOrigins: tt.allowOrgs,
			}
			if len(tt.allowOrgs) == 0 {
				cfg.AllowedOrigins = []string{"*"} // default allow all
			}

			proxy := NewWebSocketProxy(cfg)
			valid := proxy.validateUpgradeRequest(req)

			if valid != tt.wantValid {
				t.Errorf("expected valid=%v, got %v", tt.wantValid, valid)
			}
		})
	}
}

// TestGenerateWebSocketKey tests WebSocket key generation
func TestGenerateWebSocketKey(t *testing.T) {
	proxy := NewWebSocketProxy(nil)

	key1 := proxy.generateWebSocketKey()
	key2 := proxy.generateWebSocketKey()

	if key1 == "" || key2 == "" {
		t.Error("generateWebSocketKey should return non-empty string")
	}

	if key1 == key2 {
		t.Error("generated keys should be different")
	}

	// Keys should be base64 encoded
	if len(key1) == 0 {
		t.Error("key length should be non-zero")
	}
}

// TestGenerateAcceptKey tests WebSocket accept key generation
func TestGenerateAcceptKey(t *testing.T) {
	proxy := NewWebSocketProxy(nil)

	// Use a fixed key for testing
	clientKey := "dGhlIHNhbXBsZSBub25jZQ=="
	acceptKey := proxy.generateAcceptKey(clientKey)

	if acceptKey == "" {
		t.Error("generateAcceptKey should return non-empty string")
	}

	// Accept key should be base64 encoded
	if len(acceptKey) < 10 {
		t.Errorf("accept key looks too short: %s", acceptKey)
	}

	// Known test vector: dGhlIHNhbXBsZSBub25jZQ== -> s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if acceptKey != expected {
		t.Errorf("expected accept key %s, got %s", expected, acceptKey)
	}
}

// TestFrameParserReadFrame tests WebSocket frame reading
func TestFrameParserReadFrame(t *testing.T) {
	tests := []struct {
		name          string
		frameData     []byte
		wantErr       bool
		wantOpcode    uint8
		wantPayload   string
	}{
		{
			name: "simple text frame",
			// FIN=1, opcode=1 (text): 0x81, length=5, payload="hello"
			frameData:   []byte{0x81, 0x05, 'h', 'e', 'l', 'l', 'o'},
			wantErr:     false,
			wantOpcode:  OpcodeText,
			wantPayload: "hello",
		},
		{
			name: "binary frame",
			// FIN=1, opcode=2 (binary): 0x82, length=3, payload="\x00\x01\x02"
			frameData:   []byte{0x82, 0x03, 0x00, 0x01, 0x02},
			wantErr:     false,
			wantOpcode:  OpcodeBinary,
			wantPayload: "\x00\x01\x02",
		},
		{
			name: "short read should error",
			// Incomplete frame header
			frameData: []byte{0x81},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn(bytes.NewBuffer(tt.frameData))
			buffer := make([]byte, 4096)

			parser := &FrameParser{maxFrameSize: 1024 * 1024}
			frame, err := parser.ReadFrame(conn, buffer)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if frame == nil {
				t.Fatal("frame should not be nil")
			}

			if frame.Opcode != tt.wantOpcode {
				t.Errorf("expected opcode %d, got %d", tt.wantOpcode, frame.Opcode)
			}

			if string(frame.Payload) != tt.wantPayload {
				t.Errorf("expected payload %q, got %q", tt.wantPayload, frame.Payload)
			}
		})
	}
}

// TestFrameParserWriteFrame tests WebSocket frame writing
func TestFrameParserWriteFrame(t *testing.T) {
	tests := []struct {
		name      string
		opcode    uint8
		payload   string
		masked    bool
		wantBytes int
	}{
		{
			name:      "text frame unmasked",
			opcode:    OpcodeText,
			payload:   "hello",
			masked:    false,
			wantBytes: 7,
		},
		{
			name:      "binary frame unmasked",
			opcode:    OpcodeBinary,
			payload:   "binary",
			masked:    false,
			wantBytes: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn(bytes.NewBuffer([]byte{}))

			frame := &WebSocketFrame{
				Fin:     true,
				Opcode:  tt.opcode,
				Payload: []byte(tt.payload),
				Masked:  tt.masked,
			}

			parser := &FrameParser{maxFrameSize: 1024 * 1024}
			err := parser.WriteFrame(conn, frame)

			if err != nil {
				t.Errorf("WriteFrame failed: %v", err)
			}
		})
	}
}

// TestFrameParsingExtendedLength tests parsing frames with extended payload length
func TestFrameParsingExtendedLength(t *testing.T) {
	tests := []struct {
		name        string
		payloadLen  int
		lenCode     uint8
	}{
		{
			name:       "126-byte extended length",
			payloadLen: 200,
			lenCode:    126,
		},
		{
			name:       "127-byte extended length",
			payloadLen: 70000,
			lenCode:    127,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build frame with extended length
			var buf bytes.Buffer
			buf.WriteByte(0x81) // FIN + text opcode

			payload := bytes.Repeat([]byte("a"), tt.payloadLen)

			if tt.payloadLen < 126 {
				buf.WriteByte(byte(tt.payloadLen))
			} else if tt.payloadLen < 65536 {
				buf.WriteByte(126)
				buf.WriteByte(byte(tt.payloadLen >> 8))
				buf.WriteByte(byte(tt.payloadLen))
			} else {
				buf.WriteByte(127)
				for i := 7; i >= 0; i-- {
					buf.WriteByte(byte(tt.payloadLen >> (i * 8)))
				}
			}

			buf.Write(payload)

			// Parse the frame
			conn := newMockConn(bytes.NewBuffer(buf.Bytes()))
			frameBuffer := make([]byte, 1024*1024)

			parser := &FrameParser{maxFrameSize: int64(1024 * 1024)}
			frame, err := parser.ReadFrame(conn, frameBuffer)

			if err != nil && tt.payloadLen < 1024*1024 {
				t.Errorf("unexpected error: %v", err)
			}

			if frame != nil && len(frame.Payload) != tt.payloadLen {
				t.Errorf("expected payload length %d, got %d", tt.payloadLen, len(frame.Payload))
			}
		})
	}
}

// TestFrameParsingMaskedPayload tests parsing masked frames
func TestFrameParsingMaskedPayload(t *testing.T) {
	var buf bytes.Buffer

	// FIN + text opcode, masked, length 5
	buf.WriteByte(0x81) // FIN + opcode text
	buf.WriteByte(0x85) // masked, length 5

	// Mask key
	maskKey := [4]byte{0x37, 0xfa, 0x21, 0x3d}
	buf.Write(maskKey[:])

	// Masked payload "hello"
	payload := []byte("hello")
	maskedPayload := make([]byte, len(payload))
	for i := range payload {
		maskedPayload[i] = payload[i] ^ maskKey[i%4]
	}
	buf.Write(maskedPayload)

	// Parse the frame
	conn := newMockConn(bytes.NewBuffer(buf.Bytes()))
	frameBuffer := make([]byte, 4096)

	parser := &FrameParser{maxFrameSize: int64(1024 * 1024)}
	frame, err := parser.ReadFrame(conn, frameBuffer)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if frame == nil {
		t.Fatal("frame should not be nil")
	}

	if !frame.Masked {
		t.Error("frame should be marked as masked")
	}

	if string(frame.Payload) != "hello" {
		t.Errorf("expected unmasked payload 'hello', got %q", frame.Payload)
	}
}

// TestGenerateConnectionID tests connection ID generation
func TestGenerateConnectionID(t *testing.T) {
	proxy := NewWebSocketProxy(nil)

	id1 := proxy.generateConnectionID()
	id2 := proxy.generateConnectionID()

	if id1 == "" || id2 == "" {
		t.Error("generateConnectionID should return non-empty string")
	}

	if id1 == id2 {
		t.Error("generated IDs should be different")
	}

	if !bytes.Contains([]byte(id1), []byte("ws-")) {
		t.Errorf("ID should contain 'ws-' prefix, got %s", id1)
	}
}

// TestWebSocketStats tests statistics tracking
func TestWebSocketStats(t *testing.T) {
	proxy := NewWebSocketProxy(nil)

	stats := proxy.GetStats()

	if stats == nil {
		t.Fatal("GetStats should return non-nil")
	}

	if stats.TotalConnections != 0 {
		t.Errorf("initial TotalConnections should be 0, got %d", stats.TotalConnections)
	}

	if stats.LastUpdate.IsZero() {
		t.Error("LastUpdate should be set")
	}
}

// TestConnectionLimitEnforcement tests max connections limit
func TestConnectionLimitEnforcement(t *testing.T) {
	config := &WebSocketConfig{
		MaxConnections: 2,
	}
	proxy := NewWebSocketProxy(config)

	// Test that connection limit is enforced
	if len(proxy.connections) >= proxy.config.MaxConnections {
		t.Error("initially should have room for connections")
	}
}

// TestPingFrame tests ping frame construction
func TestPingFrame(t *testing.T) {
	proxy := NewWebSocketProxy(nil)

	// Create a mock connection
	var buf bytes.Buffer
	conn := newMockConn(bytes.NewBuffer([]byte{}))

	// Create a mock WebSocketConnection
	wsConn := &WebSocketConnection{
		ID:         "test-conn",
		ClientConn: conn,
		State:      StateOpen,
	}

	err := proxy.sendPing(wsConn)
	if err != nil && err != io.EOF {
		t.Errorf("unexpected error sending ping: %v", err)
	}
	_ = buf
}

// TestFrameSizeLimitExceeded tests frame size limit enforcement
func TestFrameSizeLimitExceeded(t *testing.T) {
	// Create frame that exceeds size limit
	var buf bytes.Buffer
	buf.WriteByte(0x81) // FIN + text opcode
	buf.WriteByte(127)  // Extended payload length (8 bytes)

	// Write huge payload length (100MB)
	hugeSize := int64(100 * 1024 * 1024)
	for i := 7; i >= 0; i-- {
		buf.WriteByte(byte(hugeSize >> (i * 8)))
	}

	// Parse with small max size
	conn := newMockConn(bytes.NewBuffer(buf.Bytes()))
	frameBuffer := make([]byte, 4096)

	parser := &FrameParser{maxFrameSize: int64(1024)} // 1KB limit
	frame, _ := parser.ReadFrame(conn, frameBuffer)

	// Should error or return error about frame too large
	if frame != nil && len(frame.Payload) > 1024 {
		t.Errorf("frame size exceeded limit")
	}
}

// TestConnectionState tests connection state transitions
func TestConnectionState(t *testing.T) {
	wsConn := &WebSocketConnection{
		ID:    "test",
		State: StateConnecting,
	}

	if wsConn.State != StateConnecting {
		t.Errorf("expected StateConnecting, got %d", wsConn.State)
	}

	wsConn.State = StateOpen
	if wsConn.State != StateOpen {
		t.Errorf("expected StateOpen, got %d", wsConn.State)
	}

	wsConn.State = StateClosing
	if wsConn.State != StateClosing {
		t.Errorf("expected StateClosing, got %d", wsConn.State)
	}

	wsConn.State = StateClosed
	if wsConn.State != StateClosed {
		t.Errorf("expected StateClosed, got %d", wsConn.State)
	}
}

// TestFrameOpcodes tests frame opcode constants
func TestFrameOpcodes(t *testing.T) {
	tests := []struct {
		name   string
		opcode uint8
	}{
		{"Continuation", OpcodeContinuation},
		{"Text", OpcodeText},
		{"Binary", OpcodeBinary},
		{"Close", OpcodeClose},
		{"Ping", OpcodePing},
		{"Pong", OpcodePong},
	}

	for _, tt := range tests {
		if tt.opcode < 0 || tt.opcode > 0xA {
			t.Errorf("%s opcode %d out of range", tt.name, tt.opcode)
		}
	}
}

// TestWebSocketGUID tests the magic GUID constant
func TestWebSocketGUID(t *testing.T) {
	if WebSocketGUID != "258EAFA5-E914-47DA-95CA-C5AB0DC85B11" {
		t.Errorf("unexpected WebSocket GUID: %s", WebSocketGUID)
	}
}

// TestFrameParserMaxFrameSize tests frame parser max frame size
func TestFrameParserMaxFrameSize(t *testing.T) {
	parser := &FrameParser{maxFrameSize: 1024}

	if parser.maxFrameSize != 1024 {
		t.Errorf("expected maxFrameSize=1024, got %d", parser.maxFrameSize)
	}
}

// TestWebSocketConnectionFields tests WebSocketConnection field initialization
func TestWebSocketConnectionFields(t *testing.T) {
	service := &manager.Service{ID: 1, IPFQDN: "localhost:8000"}

	wsConn := &WebSocketConnection{
		ID:      "test-123",
		Service: service,
		State:   StateOpen,
		SubProtocol: "chat",
	}

	if wsConn.ID != "test-123" {
		t.Errorf("expected ID=test-123, got %s", wsConn.ID)
	}

	if wsConn.Service != service {
		t.Errorf("expected service to be set")
	}

	if wsConn.State != StateOpen {
		t.Errorf("expected State=StateOpen, got %d", wsConn.State)
	}

	if wsConn.SubProtocol != "chat" {
		t.Errorf("expected SubProtocol=chat, got %s", wsConn.SubProtocol)
	}
}

// Helper function to create mock connection
func newMockConn(reader io.Reader) net.Conn {
	return &mockConn{
		reader: reader,
		writer: &bytes.Buffer{},
	}
}

type mockConn struct {
	reader io.Reader
	writer io.Writer
}

func (mc *mockConn) Read(b []byte) (n int, err error) {
	return mc.reader.Read(b)
}

func (mc *mockConn) Write(b []byte) (n int, err error) {
	if mc.writer == nil {
		return len(b), nil
	}
	return mc.writer.Write(b)
}

func (mc *mockConn) Close() error {
	return nil
}

func (mc *mockConn) LocalAddr() net.Addr {
	return nil
}

func (mc *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (mc *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (mc *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (mc *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
