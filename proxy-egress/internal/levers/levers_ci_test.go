//go:build ci

package levers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockEBPFManager implements EBPFManager for testing
type MockEBPFManager struct {
	blockCIDRs  []string
	allowCIDRs  []string
	rateLimits  map[string]int
	blockErr    error
	allowErr    error
	rateLimitErr error
}

func (m *MockEBPFManager) BlockCIDR(cidr string) error {
	if m.blockErr != nil {
		return m.blockErr
	}
	m.blockCIDRs = append(m.blockCIDRs, cidr)
	return nil
}

func (m *MockEBPFManager) AllowCIDR(cidr string) error {
	if m.allowErr != nil {
		return m.allowErr
	}
	m.allowCIDRs = append(m.allowCIDRs, cidr)
	return nil
}

func (m *MockEBPFManager) SetRateLimit(srcIP string, pps int) error {
	if m.rateLimitErr != nil {
		return m.rateLimitErr
	}
	if m.rateLimits == nil {
		m.rateLimits = make(map[string]int)
	}
	m.rateLimits[srcIP] = pps
	return nil
}

func (m *MockEBPFManager) ClearBlocklist() error {
	m.blockCIDRs = nil
	return nil
}

func (m *MockEBPFManager) ClearAllowlist() error {
	m.allowCIDRs = nil
	return nil
}

// TestNewEnforcer tests enforcer creation
func TestNewEnforcer(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	if enforcer == nil {
		t.Error("NewEnforcer returned nil")
	}
	if enforcer.ebpfMgr != manager {
		t.Error("ebpfMgr not set correctly")
	}
}

// TestNewEnforcerNilManager tests enforcer creation with nil manager
func TestNewEnforcerNilManager(t *testing.T) {
	enforcer := NewEnforcer(nil)

	if enforcer == nil {
		t.Error("NewEnforcer returned nil")
	}
	if enforcer.ebpfMgr != nil {
		t.Error("ebpfMgr should be nil")
	}
}

// TestApplyBasicRules tests applying a basic rule set
func TestApplyBasicRules(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	rules := RuleSet{
		BlockCIDRs: []string{"192.168.1.0/24"},
		AllowCIDRs: []string{"10.0.0.0/8"},
		BlockDomains: []string{"evil.com"},
		RouteClusters: []ClusterDef{
			{Name: "cluster1", Endpoints: []string{"10.0.0.1:8080"}},
		},
		RateLimits: map[string]int{"192.168.1.100": 1000},
	}

	err := enforcer.Apply(rules)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(manager.blockCIDRs) != 1 {
		t.Errorf("expected 1 block CIDR, got %d", len(manager.blockCIDRs))
	}
	if len(manager.allowCIDRs) != 1 {
		t.Errorf("expected 1 allow CIDR, got %d", len(manager.allowCIDRs))
	}
	if len(manager.rateLimits) != 1 {
		t.Errorf("expected 1 rate limit, got %d", len(manager.rateLimits))
	}
}

// TestApplyEmptyRules tests applying empty rules
func TestApplyEmptyRules(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	err := enforcer.Apply(RuleSet{})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(manager.blockCIDRs) != 0 {
		t.Error("expected no block CIDRs")
	}
	if len(manager.allowCIDRs) != 0 {
		t.Error("expected no allow CIDRs")
	}
}

// TestApplyInvalidCIDR tests handling of invalid CIDR notation
func TestApplyInvalidCIDR(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	rules := RuleSet{
		BlockCIDRs: []string{"invalid-cidr"},
		AllowCIDRs: []string{"10.0.0.0/8"},
	}

	err := enforcer.Apply(rules)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Invalid CIDR should be skipped, but valid one should be applied
	if len(manager.allowCIDRs) != 1 {
		t.Error("expected valid CIDR to be applied")
	}
}

// TestApplyWithEBPFError tests error handling from eBPF manager
func TestApplyWithEBPFError(t *testing.T) {
	manager := &MockEBPFManager{
		blockErr: fmt.Errorf("eBPF block failed"),
	}
	enforcer := NewEnforcer(manager)

	rules := RuleSet{
		BlockCIDRs: []string{"192.168.1.0/24"},
	}

	err := enforcer.Apply(rules)
	if err == nil {
		t.Error("expected error from eBPF operation")
	}
}

// TestApplyMultipleCIDRs tests applying multiple CIDRs
func TestApplyMultipleCIDRs(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	rules := RuleSet{
		BlockCIDRs: []string{
			"192.168.0.0/16",
			"172.16.0.0/12",
			"10.0.0.0/8",
		},
	}

	err := enforcer.Apply(rules)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(manager.blockCIDRs) != 3 {
		t.Errorf("expected 3 block CIDRs, got %d", len(manager.blockCIDRs))
	}
}

// TestApplyMultipleRateLimits tests applying multiple rate limits
func TestApplyMultipleRateLimits(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	rules := RuleSet{
		RateLimits: map[string]int{
			"192.168.1.1":   1000,
			"192.168.1.2":   2000,
			"192.168.1.3":   500,
		},
	}

	err := enforcer.Apply(rules)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(manager.rateLimits) != 3 {
		t.Errorf("expected 3 rate limits, got %d", len(manager.rateLimits))
	}
}

// TestApplyWithNilManager tests applying rules with nil eBPF manager
func TestApplyWithNilManager(t *testing.T) {
	enforcer := NewEnforcer(nil)

	rules := RuleSet{
		BlockCIDRs: []string{"192.168.1.0/24"},
		BlockDomains: []string{"evil.com"},
	}

	// Should not panic and should succeed
	err := enforcer.Apply(rules)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
}

// TestApplyClearsExistingRules tests that Apply clears existing rules
func TestApplyClearsExistingRules(t *testing.T) {
	manager := &MockEBPFManager{
		blockCIDRs: []string{"192.168.0.0/16"}, // Pre-existing rule
		allowCIDRs: []string{"10.0.0.0/8"},
	}
	enforcer := NewEnforcer(manager)

	newRules := RuleSet{
		BlockCIDRs: []string{"172.16.0.0/12"},
	}

	err := enforcer.Apply(newRules)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Old rules should be cleared
	if len(manager.blockCIDRs) != 1 || manager.blockCIDRs[0] != "172.16.0.0/12" {
		t.Error("expected old rules to be cleared and replaced")
	}
}

// TestNewServer creates a test server instance
func TestNewServer(t *testing.T) {
	enforcer := NewEnforcer(&MockEBPFManager{})
	server := NewServer(enforcer, nil)

	if server == nil {
		t.Error("NewServer returned nil")
	}
	if server.enforcer != enforcer {
		t.Error("enforcer not set correctly")
	}
}

// TestServerHandleRules tests the /api/v1/levers/rules endpoint
func TestServerHandleRules(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)
	server := NewServer(enforcer, nil)

	rules := RuleSet{
		BlockCIDRs: []string{"192.168.1.0/24"},
		AllowCIDRs: []string{"10.0.0.0/8"},
	}

	body, _ := json.Marshal(rules)
	req := httptest.NewRequest("POST", "/api/v1/levers/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.handleRules(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify rules were applied
	if len(manager.blockCIDRs) != 1 {
		t.Error("expected rules to be applied")
	}
}

// TestServerHandleRulesInvalidJSON tests handleRules with invalid JSON
func TestServerHandleRulesInvalidJSON(t *testing.T) {
	enforcer := NewEnforcer(&MockEBPFManager{})
	server := NewServer(enforcer, nil)

	req := httptest.NewRequest("POST", "/api/v1/levers/rules", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.handleRules(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestServerHandleRulesMethodNotAllowed tests wrong HTTP method
func TestServerHandleRulesMethodNotAllowed(t *testing.T) {
	enforcer := NewEnforcer(&MockEBPFManager{})
	server := NewServer(enforcer, nil)

	req := httptest.NewRequest("GET", "/api/v1/levers/rules", nil)

	w := httptest.NewRecorder()
	server.handleRules(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestServerHandleStatus tests the /api/v1/levers/status endpoint
func TestServerHandleStatus(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)
	server := NewServer(enforcer, nil)

	// Apply some rules first via the server's handleRules
	rules := RuleSet{
		BlockCIDRs: []string{"192.168.1.0/24", "172.16.0.0/12"},
		AllowCIDRs: []string{"10.0.0.0/8"},
	}

	body, _ := json.Marshal(rules)
	req := httptest.NewRequest("POST", "/api/v1/levers/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleRules(w, req)

	// Now check status
	statusReq := httptest.NewRequest("GET", "/api/v1/levers/status", nil)
	statusW := httptest.NewRecorder()
	server.handleStatus(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", statusW.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(statusW.Body).Decode(&response)

	if response["block_cidrs"] != float64(2) {
		t.Errorf("expected block_cidrs count 2, got %v", response["block_cidrs"])
	}
	if response["allow_cidrs"] != float64(1) {
		t.Errorf("expected allow_cidrs count 1, got %v", response["allow_cidrs"])
	}
}

// TestServerHandleStatusEmpty tests status endpoint with no rules
func TestServerHandleStatusEmpty(t *testing.T) {
	enforcer := NewEnforcer(&MockEBPFManager{})
	server := NewServer(enforcer, nil)

	req := httptest.NewRequest("GET", "/api/v1/levers/status", nil)
	w := httptest.NewRecorder()
	server.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err == nil {
		// Check empty counts if JSON parsed successfully
		if blockCIDRs, ok := response["block_cidrs"].(float64); ok {
			if blockCIDRs != float64(0) {
				t.Errorf("expected empty block_cidrs, got %v", blockCIDRs)
			}
		}
	}
}

// TestServerRegisterRoutes tests route registration
func TestServerRegisterRoutes(t *testing.T) {
	enforcer := NewEnforcer(&MockEBPFManager{})
	server := NewServer(enforcer, nil)
	mux := http.NewServeMux()

	server.RegisterRoutes(mux)

	// Test that routes are registered by making requests
	req := httptest.NewRequest("GET", "/api/v1/levers/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("route not registered properly")
	}
}

// TestRuleSetWithOIDC tests rule set with OIDC provider config
func TestRuleSetWithOIDC(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)
	server := NewServer(enforcer, nil)

	rules := RuleSet{
		BlockCIDRs: []string{"192.168.1.0/24"},
		OIDCProvider: &OIDCConfig{
			IssuerURL: "https://auth.example.com",
			ClientID:  "client-123",
			Audience:  "api.example.com",
		},
	}

	body, _ := json.Marshal(rules)
	req := httptest.NewRequest("POST", "/api/v1/levers/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.handleRules(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestClusterDef tests cluster definition structure
func TestClusterDef(t *testing.T) {
	cluster := ClusterDef{
		Name:      "my-cluster",
		Endpoints: []string{"10.0.0.1:8080", "10.0.0.2:8080"},
		LBPolicy:  "ROUND_ROBIN",
	}

	if cluster.Name != "my-cluster" {
		t.Error("cluster name mismatch")
	}
	if len(cluster.Endpoints) != 2 {
		t.Error("endpoint count mismatch")
	}
	if cluster.LBPolicy != "ROUND_ROBIN" {
		t.Error("LB policy mismatch")
	}
}

// TestOIDCConfig tests OIDC config structure
func TestOIDCConfig(t *testing.T) {
	config := &OIDCConfig{
		IssuerURL: "https://auth.example.com",
		ClientID:  "client-123",
		Audience:  "api.example.com",
	}

	if config.IssuerURL == "" {
		t.Error("IssuerURL should not be empty")
	}
	if config.ClientID == "" {
		t.Error("ClientID should not be empty")
	}
	if config.Audience == "" {
		t.Error("Audience should not be empty")
	}
}

// TestRuleSetJSON tests RuleSet serialization
func TestRuleSetJSON(t *testing.T) {
	rules := RuleSet{
		BlockCIDRs:   []string{"192.168.1.0/24"},
		AllowCIDRs:   []string{"10.0.0.0/8"},
		BlockDomains: []string{"evil.com"},
		RateLimits:   map[string]int{"192.168.1.100": 1000},
	}

	data, err := json.Marshal(rules)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var unmarshaled RuleSet
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(unmarshaled.BlockCIDRs) != 1 {
		t.Error("BlockCIDRs not preserved")
	}
	if len(unmarshaled.AllowCIDRs) != 1 {
		t.Error("AllowCIDRs not preserved")
	}
	if len(unmarshaled.RateLimits) != 1 {
		t.Error("RateLimits not preserved")
	}
}

// TestApplyCIDRRanges tests applying various valid CIDR ranges
func TestApplyCIDRRanges(t *testing.T) {
	manager := &MockEBPFManager{}
	enforcer := NewEnforcer(manager)

	tests := []struct {
		name string
		cidr string
	}{
		{"single IP", "192.168.1.1/32"},
		{"class C", "192.168.1.0/24"},
		{"class B", "172.16.0.0/12"},
		{"class A", "10.0.0.0/8"},
		{"IPv6", "2001:db8::/32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.blockCIDRs = nil
			manager.allowCIDRs = nil
			rules := RuleSet{BlockCIDRs: []string{tt.cidr}}
			err := enforcer.Apply(rules)
			if err != nil {
				t.Errorf("Apply failed: %v", err)
			}
			if len(manager.blockCIDRs) != 1 {
				t.Errorf("expected CIDR to be applied")
			}
		})
	}
}
