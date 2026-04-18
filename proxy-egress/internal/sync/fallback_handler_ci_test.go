//go:build ci

package sync

import (
	"context"
	"net"
	"testing"
	"time"

	"marchproxy-egress/internal/auth"
	"marchproxy-egress/internal/manager"
)

// TestNewFallbackHandler tests FallbackHandler creation
func TestNewFallbackHandler(t *testing.T) {
	mockSyncer := &RuleSynchronizer{}
	mockAuth := &auth.Authenticator{}
	mockClient := &manager.Client{}

	fh := NewFallbackHandler(mockSyncer, mockAuth, mockClient)

	if fh == nil {
		t.Fatal("Expected non-nil FallbackHandler")
	}
	if fh.ruleSyncer != mockSyncer {
		t.Error("ruleSyncer not set correctly")
	}
	if fh.authenticator != mockAuth {
		t.Error("authenticator not set correctly")
	}
	if fh.managerClient != mockClient {
		t.Error("managerClient not set correctly")
	}
	if len(fh.activeConnections) != 0 {
		t.Error("Expected empty activeConnections")
	}
	if fh.stats == nil {
		t.Error("Expected non-nil stats")
	}
}

// TestGenerateConnectionID tests connection ID generation
func TestGenerateConnectionID(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	packet := &PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.1"),
		SourcePort: 54321,
		DestIP:     net.ParseIP("10.0.0.1"),
		DestPort:   80,
		Protocol:   6,
	}

	connID := fh.generateConnectionID(packet)

	expectedID := "192.168.1.1:54321-10.0.0.1:80-6"
	if connID != expectedID {
		t.Errorf("generateConnectionID() = %s, want %s", connID, expectedID)
	}
}

// TestGetOrCreateConnection tests connection creation
func TestGetOrCreateConnection(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	packet := &PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.1"),
		SourcePort: 54321,
		DestIP:     net.ParseIP("10.0.0.1"),
		DestPort:   80,
		Protocol:   6,
		ServiceID:  100,
		Size:       1024,
	}

	connID := fh.generateConnectionID(packet)

	// First call should create connection
	conn1 := fh.getOrCreateConnection(connID, packet)
	if conn1 == nil {
		t.Fatal("Expected non-nil connection")
	}

	if conn1.ID != connID {
		t.Errorf("Connection ID mismatch: got %s, want %s", conn1.ID, connID)
	}

	if conn1.ServiceID != 100 {
		t.Errorf("ServiceID mismatch: got %d, want 100", conn1.ServiceID)
	}

	if conn1.PacketCount != 1 {
		t.Errorf("PacketCount mismatch: got %d, want 1", conn1.PacketCount)
	}

	if conn1.ByteCount != 1024 {
		t.Errorf("ByteCount mismatch: got %d, want 1024", conn1.ByteCount)
	}

	// Second call should return existing connection
	conn2 := fh.getOrCreateConnection(connID, packet)
	if conn1 != conn2 {
		t.Error("Expected same connection object on second call")
	}
}

// TestUpdateConnectionActivity tests connection activity updates
func TestUpdateConnectionActivity(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	conn := &SlowPathConnection{
		ID:           "test-conn",
		PacketCount:  5,
		ByteCount:    5000,
		LastActivity: time.Now().Add(-10 * time.Second),
	}

	fh.activeConnections["test-conn"] = conn

	packet := &PacketInfo{Size: 512}

	beforeTime := time.Now()
	fh.updateConnectionActivity(conn, packet)
	afterTime := time.Now()

	if conn.PacketCount != 6 {
		t.Errorf("PacketCount not incremented: got %d, want 6", conn.PacketCount)
	}

	if conn.ByteCount != 5512 {
		t.Errorf("ByteCount not incremented: got %d, want 5512", conn.ByteCount)
	}

	if conn.LastActivity.Before(beforeTime) || conn.LastActivity.After(afterTime) {
		t.Error("LastActivity not updated correctly")
	}
}

// TestFindSlowPathRule tests slow-path rule lookup
func TestFindSlowPathRule(t *testing.T) {
	mockSyncer := &RuleSynchronizer{
		slowPathRules: make(map[string]*SlowPathRule),
	}

	testRule := &SlowPathRule{
		ServiceID: 42,
		RequiresAuth: true,
	}
	mockSyncer.slowPathRules["42-1"] = testRule

	fh := NewFallbackHandler(mockSyncer, &auth.Authenticator{}, &manager.Client{})

	// Test finding existing rule
	rule := fh.findSlowPathRule(42)
	if rule == nil {
		t.Error("Expected rule to be found")
	} else if rule.ServiceID != 42 {
		t.Error("Rule data mismatch")
	}

	// Test not finding rule
	rule = fh.findSlowPathRule(999)
	if rule != nil {
		t.Error("Expected nil for non-existent service")
	}
}

// TestExtractAuthToken tests authentication token extraction
func TestExtractAuthToken(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	tests := []struct {
		name      string
		packet    *PacketInfo
		authType  string
		wantToken string
		wantErr   bool
	}{
		{
			name: "authorization header",
			packet: &PacketInfo{
				Headers: map[string]string{"Authorization": "Bearer token123"},
			},
			authType:  "jwt",
			wantToken: "Bearer token123",
			wantErr:   false,
		},
		{
			name: "x-auth-token header",
			packet: &PacketInfo{
				Headers: map[string]string{"X-Auth-Token": "custom-token-456"},
			},
			authType:  "jwt",
			wantToken: "custom-token-456",
			wantErr:   false,
		},
		{
			name: "authorization preferred over x-auth-token",
			packet: &PacketInfo{
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"X-Auth-Token": "custom-token-456",
				},
			},
			authType:  "jwt",
			wantToken: "Bearer token123",
			wantErr:   false,
		},
		{
			name:     "no headers",
			packet:   &PacketInfo{Headers: nil},
			authType: "jwt",
			wantErr:  true,
		},
		{
			name:     "empty headers",
			packet:   &PacketInfo{Headers: map[string]string{}},
			authType: "jwt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := fh.extractAuthToken(tt.packet, tt.authType)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractAuthToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token != tt.wantToken {
				t.Errorf("extractAuthToken() = %s, want %s", token, tt.wantToken)
			}
		})
	}
}

// TestIsWebSocketUpgrade tests WebSocket upgrade detection
func TestIsWebSocketUpgrade(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	tests := []struct {
		name    string
		packet  *PacketInfo
		wantWS  bool
	}{
		{
			name: "valid websocket upgrade",
			packet: &PacketInfo{
				Headers: map[string]string{
					"Upgrade": "websocket",
					"Connection": "Upgrade",
				},
			},
			wantWS: true,
		},
		{
			name: "missing upgrade header",
			packet: &PacketInfo{
				Headers: map[string]string{
					"Connection": "Upgrade",
				},
			},
			wantWS: false,
		},
		{
			name: "missing connection header",
			packet: &PacketInfo{
				Headers: map[string]string{
					"Upgrade": "websocket",
				},
			},
			wantWS: false,
		},
		{
			name: "wrong upgrade value",
			packet: &PacketInfo{
				Headers: map[string]string{
					"Upgrade": "h2c",
					"Connection": "Upgrade",
				},
			},
			wantWS: false,
		},
		{
			name: "wrong connection value",
			packet: &PacketInfo{
				Headers: map[string]string{
					"Upgrade": "websocket",
					"Connection": "keep-alive",
				},
			},
			wantWS: false,
		},
		{
			name:   "nil headers",
			packet: &PacketInfo{Headers: nil},
			wantWS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fh.isWebSocketUpgrade(tt.packet)
			if got != tt.wantWS {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.wantWS)
			}
		})
	}
}

// TestSelectDestination tests destination selection
func TestSelectDestination(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	tests := []struct {
		name    string
		mapping *manager.Mapping
		wantErr bool
	}{
		{
			name: "with destination services",
			mapping: &manager.Mapping{
				DestServices: []int{1, 2, 3},
			},
			wantErr: false,
		},
		{
			name:    "no destination services",
			mapping: &manager.Mapping{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := fh.selectDestination(tt.mapping)

			if (err != nil) != tt.wantErr {
				t.Errorf("selectDestination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && dest == nil {
				t.Error("Expected non-nil destination")
			}
		})
	}
}

// TestUpdateProcessingTime tests processing time tracking
func TestUpdateProcessingTime(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	if fh.stats.AverageProcessTime != 0 {
		t.Error("Expected initial AverageProcessTime to be 0")
	}

	duration1 := 10 * time.Millisecond
	fh.updateProcessingTime(duration1)

	if fh.stats.AverageProcessTime != 5*time.Millisecond {
		t.Errorf("After first update, AverageProcessTime = %v, want 5ms", fh.stats.AverageProcessTime)
	}

	duration2 := 20 * time.Millisecond
	fh.updateProcessingTime(duration2)

	expected := (5*time.Millisecond + 20*time.Millisecond) / 2
	if fh.stats.AverageProcessTime != expected {
		t.Errorf("After second update, AverageProcessTime = %v, want %v", fh.stats.AverageProcessTime, expected)
	}
}

// TestFallbackStatsIncrements tests statistics increment methods
func TestFallbackStatsIncrements(t *testing.T) {
	stats := &FallbackStats{}

	if stats.TotalConnections != 0 {
		t.Error("Expected initial TotalConnections to be 0")
	}

	stats.incrementTotalConnections()
	if stats.TotalConnections != 1 {
		t.Error("TotalConnections increment failed")
	}

	if stats.AuthenticatedConns != 0 {
		t.Error("Expected initial AuthenticatedConns to be 0")
	}

	stats.incrementAuthenticatedConns()
	if stats.AuthenticatedConns != 1 {
		t.Error("AuthenticatedConns increment failed")
	}

	if stats.FailedAuthentication != 0 {
		t.Error("Expected initial FailedAuthentication to be 0")
	}

	stats.incrementFailedAuth()
	if stats.FailedAuthentication != 1 {
		t.Error("FailedAuthentication increment failed")
	}

	if stats.TLSConnections != 0 {
		t.Error("Expected initial TLSConnections to be 0")
	}

	stats.incrementTLSConns()
	if stats.TLSConnections != 1 {
		t.Error("TLSConnections increment failed")
	}

	if stats.WebSocketUpgrades != 0 {
		t.Error("Expected initial WebSocketUpgrades to be 0")
	}

	stats.incrementWebSocketUpgrades()
	if stats.WebSocketUpgrades != 1 {
		t.Error("WebSocketUpgrades increment failed")
	}

	if stats.ComplexRouting != 0 {
		t.Error("Expected initial ComplexRouting to be 0")
	}

	stats.incrementComplexRouting()
	if stats.ComplexRouting != 1 {
		t.Error("ComplexRouting increment failed")
	}
}

// TestGetStats tests statistics retrieval
func TestGetStats(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	// Add some stats
	fh.stats.incrementTotalConnections()
	fh.stats.incrementAuthenticatedConns()
	fh.stats.incrementFailedAuth()
	fh.stats.incrementTLSConns()
	fh.stats.incrementWebSocketUpgrades()
	fh.stats.incrementComplexRouting()

	stats := fh.GetStats()

	if stats["total_connections"] != uint64(1) {
		t.Errorf("total_connections = %v, want 1", stats["total_connections"])
	}

	if stats["authenticated_conns"] != uint64(1) {
		t.Errorf("authenticated_conns = %v, want 1", stats["authenticated_conns"])
	}

	if stats["failed_authentication"] != uint64(1) {
		t.Errorf("failed_authentication = %v, want 1", stats["failed_authentication"])
	}

	if stats["tls_connections"] != uint64(1) {
		t.Errorf("tls_connections = %v, want 1", stats["tls_connections"])
	}

	if stats["websocket_upgrades"] != uint64(1) {
		t.Errorf("websocket_upgrades = %v, want 1", stats["websocket_upgrades"])
	}

	if stats["complex_routing"] != uint64(1) {
		t.Errorf("complex_routing = %v, want 1", stats["complex_routing"])
	}

	if stats["active_connections"] != 0 {
		t.Errorf("active_connections = %v, want 0", stats["active_connections"])
	}
}

// TestProcessPacketWithNilRule tests packet processing when rule not found
func TestProcessPacketWithNilRule(t *testing.T) {
	// Create RuleSynchronizer with no rules
	ruleSyncer := &RuleSynchronizer{
		slowPathRules: make(map[string]*SlowPathRule),
	}

	fh := NewFallbackHandler(ruleSyncer, &auth.Authenticator{}, &manager.Client{})

	packet := &PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.1"),
		SourcePort: 54321,
		DestIP:     net.ParseIP("10.0.0.1"),
		DestPort:   80,
		Protocol:   6,
		ServiceID:  999,
		Size:       1024,
	}

	ctx := context.Background()
	result, _ := fh.ProcessPacket(ctx, packet)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Action != "drop" {
		t.Errorf("Expected action 'drop', got %s", result.Action)
	}

	if result.Reason != "no rule found" {
		t.Errorf("Expected reason 'no rule found', got %s", result.Reason)
	}
}

// TestProcessPacketCreatesConnection tests that ProcessPacket creates connections
func TestProcessPacketCreatesConnection(t *testing.T) {
	// Create RuleSynchronizer with a test rule
	ruleSyncer := &RuleSynchronizer{
		slowPathRules: make(map[string]*SlowPathRule),
	}

	testRule := &SlowPathRule{
		ServiceID:      100,
		RequiresAuth:   false,
		AuthType:       "none",
		HasTLS:         false,
		HasWebSocket:   false,
		ComplexRouting: false,
	}
	ruleSyncer.slowPathRules["100-1"] = testRule

	fh := NewFallbackHandler(ruleSyncer, &auth.Authenticator{}, &manager.Client{})

	packet := &PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.1"),
		SourcePort: 54321,
		DestIP:     net.ParseIP("10.0.0.1"),
		DestPort:   80,
		Protocol:   6,
		ServiceID:  100,
		Size:       1024,
	}

	ctx := context.Background()
	result, _ := fh.ProcessPacket(ctx, packet)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Action == "drop" {
		t.Fatalf("Unexpected drop action: %s", result.Reason)
	}

	// Verify connection was created
	connID := fh.generateConnectionID(packet)
	if _, exists := fh.activeConnections[connID]; !exists {
		t.Error("Expected connection to be created")
	}
}

// TestProcessTLS tests TLS processing
func TestProcessTLS(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	conn := &SlowPathConnection{
		ID:      "test-conn",
		DestIP:  net.ParseIP("10.0.0.1"),
		DestPort: 443,
	}

	packet := &PacketInfo{}

	result, err := fh.processTLS(conn, packet)

	if err != nil {
		t.Errorf("processTLS() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Action != "forward" {
		t.Errorf("Expected action 'forward', got %s", result.Action)
	}

	if result.Destination == nil {
		t.Fatal("Expected non-nil destination")
	}

	if result.Destination.IP.String() != "10.0.0.1" {
		t.Errorf("Destination IP mismatch: got %s, want 10.0.0.1", result.Destination.IP)
	}

	if result.Destination.Port != 443 {
		t.Errorf("Destination port mismatch: got %d, want 443", result.Destination.Port)
	}
}

// TestProcessWebSocketUpgrade tests WebSocket upgrade processing
func TestProcessWebSocketUpgrade(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	conn := &SlowPathConnection{
		ID:       "test-conn",
		DestIP:   net.ParseIP("10.0.0.1"),
		DestPort: 8080,
	}

	packet := &PacketInfo{}

	result, err := fh.processWebSocketUpgrade(conn, packet)

	if err != nil {
		t.Errorf("processWebSocketUpgrade() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Action != "websocket_upgrade" {
		t.Errorf("Expected action 'websocket_upgrade', got %s", result.Action)
	}

	if result.Destination == nil {
		t.Fatal("Expected non-nil destination")
	}

	if result.Destination.IP.String() != "10.0.0.1" {
		t.Errorf("Destination IP mismatch: got %s, want 10.0.0.1", result.Destination.IP)
	}
}

// TestProcessDefaultForwarding tests default forwarding
func TestProcessDefaultForwarding(t *testing.T) {
	fh := NewFallbackHandler(&RuleSynchronizer{}, &auth.Authenticator{}, &manager.Client{})

	conn := &SlowPathConnection{
		ID:       "test-conn",
		DestIP:   net.ParseIP("10.0.0.1"),
		DestPort: 80,
	}

	rule := &SlowPathRule{}
	packet := &PacketInfo{}

	result, err := fh.processDefaultForwarding(conn, rule, packet)

	if err != nil {
		t.Errorf("processDefaultForwarding() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Action != "forward" {
		t.Errorf("Expected action 'forward', got %s", result.Action)
	}

	if result.Destination == nil {
		t.Fatal("Expected non-nil destination")
	}

	if result.Destination.IP.String() != "10.0.0.1" {
		t.Errorf("Destination IP mismatch: got %s, want 10.0.0.1", result.Destination.IP)
	}

	if result.Destination.Port != 80 {
		t.Errorf("Destination port mismatch: got %d, want 80", result.Destination.Port)
	}
}
