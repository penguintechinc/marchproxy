package threat

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// createTestIPBlocker is a helper to create an IPBlocker for testing
func createTestIPBlocker(t testing.TB) *IPBlocker {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return NewIPBlocker(10000, logger)
}

func TestNewIPBlocker(t *testing.T) {
	blocker := createTestIPBlocker(t)
	if blocker == nil {
		t.Fatal("NewIPBlocker returned nil")
	}

	if blocker.Count() != 0 {
		t.Errorf("Expected 0 rules, got %d", blocker.Count())
	}
}

func TestIPBlocker_AddRule_SingleIP(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := blocker.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if blocker.Count() != 1 {
		t.Errorf("Expected 1 rule, got %d", blocker.Count())
	}
}

func TestIPBlocker_AddRule_CIDR(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-cidr",
		Pattern:   "10.0.0.0/8",
		Category:  "internal",
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := blocker.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if blocker.Count() != 1 {
		t.Errorf("Expected 1 rule, got %d", blocker.Count())
	}
}

func TestIPBlocker_AddRule_InvalidPattern(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-invalid",
		Pattern:   "not-an-ip",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := blocker.AddRule(rule)
	if err == nil {
		t.Error("Expected error for invalid IP pattern")
	}
}

func TestIPBlocker_Check_BlockedIP(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	decision := blocker.Check("192.168.1.100")
	if decision == nil {
		t.Fatal("Check returned nil")
	}

	if !decision.Blocked {
		t.Error("Expected IP to be blocked")
	}

	if decision.Category != "ip" {
		t.Errorf("Expected category 'ip', got '%s'", decision.Category)
	}

	if decision.MatchedRule != "test-1" {
		t.Errorf("Expected matched rule 'test-1', got '%s'", decision.MatchedRule)
	}
}

func TestIPBlocker_Check_AllowedIP(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	decision := blocker.Check("192.168.1.50")
	if decision == nil {
		t.Fatal("Check returned nil")
	}

	if decision.Blocked {
		t.Error("Expected IP to be allowed")
	}
}

func TestIPBlocker_Check_CIDRMatch(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-cidr",
		Pattern:   "10.0.0.0/8",
		Category:  "internal",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	tests := []struct {
		ip      string
		blocked bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"10.100.50.25", true},
		{"192.168.1.1", false},
		{"11.0.0.1", false},
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.ip)
		if decision.Blocked != tt.blocked {
			t.Errorf("IP %s: expected blocked=%v, got blocked=%v", tt.ip, tt.blocked, decision.Blocked)
		}
	}
}

func TestIPBlocker_Check_ExpiredRule(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-expired",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	blocker.AddRule(rule)

	decision := blocker.Check("192.168.1.100")
	if decision.Blocked {
		t.Error("Expected expired rule to not block")
	}
}

func TestIPBlocker_Check_IPv6(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-ipv6",
		Pattern:   "2001:db8::1",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	decision := blocker.Check("2001:db8::1")
	if !decision.Blocked {
		t.Error("Expected IPv6 address to be blocked")
	}

	decision = blocker.Check("2001:db8::2")
	if decision.Blocked {
		t.Error("Expected different IPv6 address to be allowed")
	}
}

func TestIPBlocker_Check_IPv6CIDR(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-ipv6-cidr",
		Pattern:   "2001:db8::/32",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	tests := []struct {
		ip      string
		blocked bool
	}{
		{"2001:db8::1", true},
		{"2001:db8:1234::1", true},
		{"2001:db9::1", false},
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.ip)
		if decision.Blocked != tt.blocked {
			t.Errorf("IPv6 %s: expected blocked=%v, got blocked=%v", tt.ip, tt.blocked, decision.Blocked)
		}
	}
}

func TestIPBlocker_RemoveRule(t *testing.T) {
	blocker := createTestIPBlocker(t)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	// Verify it's blocked
	decision := blocker.Check("192.168.1.100")
	if !decision.Blocked {
		t.Error("Expected IP to be blocked before removal")
	}

	// Remove the rule
	err := blocker.RemoveRule("test-1")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// Verify it's no longer blocked
	decision = blocker.Check("192.168.1.100")
	if decision.Blocked {
		t.Error("Expected IP to be allowed after rule removal")
	}
}

func TestIPBlocker_RemoveRule_NotFound(t *testing.T) {
	blocker := createTestIPBlocker(t)

	err := blocker.RemoveRule("nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent rule")
	}
}

func TestIPBlocker_Clear(t *testing.T) {
	blocker := createTestIPBlocker(t)

	// Add multiple rules
	for i := 0; i < 10; i++ {
		rule := BlockRule{
			ID:        "test-" + string(rune('0'+i)),
			Pattern:   "192.168.1." + string(rune('0'+i)),
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		}
		blocker.AddRule(rule)
	}

	if blocker.Count() == 0 {
		t.Error("Expected rules to be added")
	}

	blocker.Clear()

	if blocker.Count() != 0 {
		t.Errorf("Expected 0 rules after clear, got %d", blocker.Count())
	}
}

func TestIPBlocker_CleanExpired(t *testing.T) {
	blocker := createTestIPBlocker(t)

	// Add an expired rule
	rule := BlockRule{
		ID:        "test-expired",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	blocker.AddRule(rule)

	// Add a valid rule
	validRule := BlockRule{
		ID:        "test-valid",
		Pattern:   "192.168.1.200",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(validRule)

	if blocker.Count() != 2 {
		t.Errorf("Expected 2 rules before clean, got %d", blocker.Count())
	}

	cleaned := blocker.CleanExpired()
	if cleaned != 1 {
		t.Errorf("Expected 1 rule cleaned, got %d", cleaned)
	}

	if blocker.Count() != 1 {
		t.Errorf("Expected 1 rule after clean, got %d", blocker.Count())
	}
}

func BenchmarkIPBlocker_Check(b *testing.B) {
	blocker := createTestIPBlocker(b)

	// Add 1000 rules
	for i := 0; i < 1000; i++ {
		ip := "192.168." + string(rune(i/256)) + "." + string(rune(i%256))
		rule := BlockRule{
			ID:        "test-" + string(rune(i)),
			Pattern:   ip,
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		}
		blocker.AddRule(rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blocker.Check("192.168.1.100")
	}
}

func BenchmarkIPBlocker_Check_CIDR(b *testing.B) {
	blocker := createTestIPBlocker(b)

	// Add 100 CIDR rules
	for i := 0; i < 100; i++ {
		rule := BlockRule{
			ID:        "test-cidr-" + string(rune(i)),
			Pattern:   "10." + string(rune(i)) + ".0.0/16",
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		}
		blocker.AddRule(rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blocker.Check("10.50.100.200")
	}
}
