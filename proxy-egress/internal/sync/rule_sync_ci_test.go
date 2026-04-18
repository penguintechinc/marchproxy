//go:build ci

package sync

import (
	"fmt"
	"net"
	"testing"
	"time"

	"marchproxy-egress/internal/acceleration/xdp"
	"marchproxy-egress/internal/manager"
)

// TestNewRuleSynchronizer tests RuleSynchronizer creation
func TestNewRuleSynchronizer(t *testing.T) {
	mockClient := &manager.Client{}
	mockXDP := &xdp.XDPManager{}

	rs := NewRuleSynchronizer(mockClient, mockXDP)

	if rs == nil {
		t.Fatal("Expected non-nil RuleSynchronizer")
	}
	if rs.managerClient != mockClient {
		t.Error("managerClient not set correctly")
	}
	if rs.xdpManager != mockXDP {
		t.Error("xdpManager not set correctly")
	}
	if rs.syncInterval != 30*time.Second {
		t.Errorf("Expected sync interval 30s, got %v", rs.syncInterval)
	}
	if len(rs.fastPathRules) != 0 {
		t.Error("Expected empty fastPathRules")
	}
	if len(rs.slowPathRules) != 0 {
		t.Error("Expected empty slowPathRules")
	}
}

// TestServiceInMapping tests service-to-mapping lookup
func TestServiceInMapping(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name    string
		service *manager.Service
		mapping *manager.Mapping
		want    bool
	}{
		{
			name: "service in source",
			service: &manager.Service{ID: 1},
			mapping: &manager.Mapping{SourceServices: []int{1, 2}},
			want:    true,
		},
		{
			name: "service in destination",
			service: &manager.Service{ID: 2},
			mapping: &manager.Mapping{DestServices: []int{2, 3}},
			want:    true,
		},
		{
			name: "service not in mapping",
			service: &manager.Service{ID: 5},
			mapping: &manager.Mapping{SourceServices: []int{1, 2}, DestServices: []int{3, 4}},
			want:    false,
		},
		{
			name: "empty mapping",
			service: &manager.Service{ID: 1},
			mapping: &manager.Mapping{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.serviceInMapping(tt.service, tt.mapping)
			if got != tt.want {
				t.Errorf("serviceInMapping() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetProtocolNumber tests protocol string to number conversion
func TestGetProtocolNumber(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name      string
		protocols []string
		want      uint8
	}{
		{
			name:      "tcp",
			protocols: []string{"tcp"},
			want:      PROTOCOL_TCP,
		},
		{
			name:      "udp",
			protocols: []string{"udp"},
			want:      PROTOCOL_UDP,
		},
		{
			name:      "icmp",
			protocols: []string{"icmp"},
			want:      PROTOCOL_ICMP,
		},
		{
			name:      "multiple protocols (returns first)",
			protocols: []string{"udp", "tcp"},
			want:      PROTOCOL_UDP,
		},
		{
			name:      "empty protocols (defaults to tcp)",
			protocols: []string{},
			want:      PROTOCOL_TCP,
		},
		{
			name:      "unknown protocol (defaults to tcp)",
			protocols: []string{"unknown"},
			want:      PROTOCOL_TCP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.getProtocolNumber(tt.protocols)
			if got != tt.want {
				t.Errorf("getProtocolNumber() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestIsHTTPProtocol tests HTTP protocol detection
func TestIsHTTPProtocol(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name      string
		protocols []string
		want      bool
	}{
		{
			name:      "http protocol",
			protocols: []string{"http"},
			want:      true,
		},
		{
			name:      "https protocol",
			protocols: []string{"https"},
			want:      true,
		},
		{
			name:      "mixed with http",
			protocols: []string{"tcp", "http"},
			want:      true,
		},
		{
			name:      "no http protocol",
			protocols: []string{"tcp", "udp"},
			want:      false,
		},
		{
			name:      "empty protocols",
			protocols: []string{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.isHTTPProtocol(tt.protocols)
			if got != tt.want {
				t.Errorf("isHTTPProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCanUseFastPath tests fast-path eligibility determination
func TestCanUseFastPath(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name    string
		service *manager.Service
		mapping *manager.Mapping
		want    bool
	}{
		{
			name: "simple tcp service",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
				PortRange: "",
			},
			mapping: &manager.Mapping{
				Protocols:          []string{"tcp"},
				DestinationServices: []manager.Service{{ID: 1}},
				LoadBalancing:      "none",
				Ports:              "",
				DynamicPorts:       false,
			},
			want: true,
		},
		{
			name: "jwt authentication blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "jwt",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols: []string{"tcp"},
			},
			want: false,
		},
		{
			name: "base64 authentication blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "base64",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols: []string{"tcp"},
			},
			want: false,
		},
		{
			name: "tls enabled blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: true,
			},
			mapping: &manager.Mapping{
				Protocols: []string{"tcp"},
			},
			want: false,
		},
		{
			name: "http protocol blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols: []string{"http"},
			},
			want: false,
		},
		{
			name: "https protocol blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols: []string{"https"},
			},
			want: false,
		},
		{
			name: "websocket blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols:       []string{"tcp"},
				SupportsWebSocket: true,
			},
			want: false,
		},
		{
			name: "complex routing blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols:          []string{"tcp"},
				DestinationServices: []manager.Service{{ID: 1}, {ID: 2}},
			},
			want: false,
		},
		{
			name: "complex ports blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
				PortRange: "8000-9000",
			},
			mapping: &manager.Mapping{
				Protocols: []string{"tcp"},
			},
			want: false,
		},
		{
			name: "multiple ports blocks fast-path",
			service: &manager.Service{
				ID:        1,
				AuthType:  "none",
				TLSEnabled: false,
			},
			mapping: &manager.Mapping{
				Protocols: []string{"tcp"},
				Ports:     "80,443,8080",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.canUseFastPath(tt.service, tt.mapping)
			if got != tt.want {
				t.Errorf("canUseFastPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasWebSocketSupport tests websocket detection
func TestHasWebSocketSupport(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name    string
		mapping *manager.Mapping
		want    bool
	}{
		{
			name:    "websocket enabled",
			mapping: &manager.Mapping{SupportsWebSocket: true},
			want:    true,
		},
		{
			name:    "websocket disabled",
			mapping: &manager.Mapping{SupportsWebSocket: false},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.hasWebSocketSupport(tt.mapping)
			if got != tt.want {
				t.Errorf("hasWebSocketSupport() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasComplexRouting tests complex routing detection
func TestHasComplexRouting(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name    string
		mapping *manager.Mapping
		want    bool
	}{
		{
			name:    "multiple destination services",
			mapping: &manager.Mapping{DestinationServices: []manager.Service{{ID: 1}, {ID: 2}}},
			want:    true,
		},
		{
			name:    "load balancing enabled",
			mapping: &manager.Mapping{LoadBalancing: "round-robin"},
			want:    true,
		},
		{
			name:    "routing rules present",
			mapping: &manager.Mapping{RoutingRules: []manager.RoutingRule{{ID: 1}}},
			want:    true,
		},
		{
			name: "simple routing",
			mapping: &manager.Mapping{
				DestinationServices: []manager.Service{{ID: 1}},
				LoadBalancing:       "none",
				RoutingRules:        []manager.RoutingRule{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.hasComplexRouting(tt.mapping)
			if got != tt.want {
				t.Errorf("hasComplexRouting() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasComplexPorts tests complex port detection
func TestHasComplexPorts(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name    string
		service *manager.Service
		mapping *manager.Mapping
		want    bool
	}{
		{
			name:    "port range in service",
			service: &manager.Service{ID: 1, PortRange: "8000-9000"},
			mapping: &manager.Mapping{},
			want:    true,
		},
		{
			name:    "multiple ports in mapping",
			service: &manager.Service{ID: 1},
			mapping: &manager.Mapping{Ports: "80,443,8080"},
			want:    true,
		},
		{
			name:    "dynamic ports enabled",
			service: &manager.Service{ID: 1},
			mapping: &manager.Mapping{DynamicPorts: true},
			want:    true,
		},
		{
			name:    "any non-empty ports string is complex (implementation checks len > 1)",
			service: &manager.Service{ID: 1, Port: 8080, PortRange: ""},
			mapping: &manager.Mapping{Ports: "8080", DynamicPorts: false},
			want:    true,
		},
		{
			name:    "empty ports and range are simple",
			service: &manager.Service{ID: 1, PortRange: ""},
			mapping: &manager.Mapping{Ports: "", DynamicPorts: false},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.hasComplexPorts(tt.service, tt.mapping)
			if got != tt.want {
				t.Errorf("hasComplexPorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIPToUint32 tests IP address to uint32 conversion
func TestIPToUint32(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	tests := []struct {
		name string
		ip   net.IP
		want uint32
	}{
		{
			name: "valid ipv4",
			ip:   net.ParseIP("192.168.1.1"),
			want: (192 << 24) | (168 << 16) | (1 << 8) | 1,
		},
		{
			name: "localhost",
			ip:   net.ParseIP("127.0.0.1"),
			want: (127 << 24) | 1,
		},
		{
			name: "zero ip",
			ip:   net.ParseIP("0.0.0.0"),
			want: 0,
		},
		{
			name: "nil ip",
			ip:   nil,
			want: 0,
		},
		{
			name: "ipv6 ip",
			ip:   net.ParseIP("::1"),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rs.ipToUint32(tt.ip)
			if got != tt.want {
				t.Errorf("ipToUint32() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestClassifyAndAddRule tests rule classification and addition
func TestClassifyAndAddRule(t *testing.T) {
	tests := []struct {
		name          string
		service       *manager.Service
		mapping       *manager.Mapping
		expectFastPath bool
	}{
		{
			name: "simple service creates fast-path rule",
			service: &manager.Service{
				ID:         1,
				IPAddress:  "192.168.1.1",
				Port:       80,
				AuthType:   "none",
				TLSEnabled: false,
				PortRange:  "",
			},
			mapping: &manager.Mapping{
				ID:                  1,
				Protocols:           []string{"tcp"},
				DestinationServices: []manager.Service{{ID: 1}},
				LoadBalancing:       "none",
				Ports:               "",
				DynamicPorts:        false,
			},
			expectFastPath: true,
		},
		{
			name: "authenticated service creates slow-path rule",
			service: &manager.Service{
				ID:         2,
				IPAddress:  "192.168.1.2",
				Port:       81,
				AuthType:   "jwt",
				TLSEnabled: false,
				PortRange:  "",
			},
			mapping: &manager.Mapping{
				ID:                  2,
				Protocols:           []string{"tcp"},
				DestinationServices: []manager.Service{{ID: 2}},
				LoadBalancing:       "none",
				Ports:               "",
				DynamicPorts:        false,
			},
			expectFastPath: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})
			rs.classifyAndAddRule(tt.service, tt.mapping)

			ruleKey := fmt.Sprintf("%d-%d", tt.service.ID, tt.mapping.ID)

			if tt.expectFastPath {
				if _, exists := rs.fastPathRules[ruleKey]; !exists {
					t.Error("Expected fast-path rule, but rule not found")
				}
				if _, exists := rs.slowPathRules[ruleKey]; exists {
					t.Error("Expected fast-path rule, but slow-path rule found")
				}
			} else {
				if _, exists := rs.slowPathRules[ruleKey]; !exists {
					t.Error("Expected slow-path rule, but rule not found")
				}
				if _, exists := rs.fastPathRules[ruleKey]; exists {
					t.Error("Expected slow-path rule, but fast-path rule found")
				}
			}
		})
	}
}

// TestGetFastPathRules tests rule retrieval
func TestGetFastPathRules(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	// Add a test rule
	testRule := &FastPathRule{
		ServiceID: 1,
		Port:      80,
		Protocol:  PROTOCOL_TCP,
		Action:    ACTION_PASS,
	}
	rs.fastPathRules["1-1"] = testRule

	rules := rs.GetFastPathRules()

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rule, exists := rules["1-1"]; !exists {
		t.Error("Expected rule not found")
	} else if rule.ServiceID != 1 {
		t.Error("Rule data mismatch")
	}

	// Verify isolation (modifying returned map shouldn't affect internal state)
	rules["test"] = &FastPathRule{}
	if len(rs.fastPathRules) != 1 {
		t.Error("Internal rules should not be modified")
	}
}

// TestGetSlowPathRules tests slow-path rule retrieval
func TestGetSlowPathRules(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	testRule := &SlowPathRule{
		ServiceID: 1,
		RequiresAuth: true,
		AuthType: "jwt",
	}
	rs.slowPathRules["1-1"] = testRule

	rules := rs.GetSlowPathRules()

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rule, exists := rules["1-1"]; !exists {
		t.Error("Expected rule not found")
	} else if rule.ServiceID != 1 {
		t.Error("Rule data mismatch")
	}

	// Verify isolation
	rules["test"] = &SlowPathRule{}
	if len(rs.slowPathRules) != 1 {
		t.Error("Internal rules should not be modified")
	}
}

// TestGetSyncStats tests statistics retrieval
func TestGetSyncStats(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	rs.lastSyncTime = time.Now()
	rs.lastConfigHash = "test-hash-v1"
	rs.fastPathRules["1-1"] = &FastPathRule{}
	rs.slowPathRules["2-2"] = &SlowPathRule{}

	stats := rs.GetSyncStats()

	if stats["fast_path_rules"] != 1 {
		t.Errorf("Expected 1 fast-path rule in stats, got %d", stats["fast_path_rules"])
	}

	if stats["slow_path_rules"] != 1 {
		t.Errorf("Expected 1 slow-path rule in stats, got %d", stats["slow_path_rules"])
	}

	if stats["last_config_hash"] != "test-hash-v1" {
		t.Error("Config hash mismatch in stats")
	}

	if stats["sync_interval"] != 30*time.Second {
		t.Error("Sync interval mismatch in stats")
	}
}

// TestStop tests rule synchronizer shutdown
func TestStop(t *testing.T) {
	rs := NewRuleSynchronizer(&manager.Client{}, &xdp.XDPManager{})

	// Verify stop channel is open
	select {
	case <-rs.stopChan:
		t.Fatal("Stop channel should not be closed initially")
	default:
		// Expected
	}

	// Stop and verify channel is closed
	rs.Stop()

	select {
	case <-rs.stopChan:
		// Expected - channel is closed
	default:
		t.Fatal("Stop channel should be closed after Stop()")
	}
}
