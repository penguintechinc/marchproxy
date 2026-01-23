package threat

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewDomainBlocker(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	blocker := NewDomainBlocker(true, logger)
	if blocker == nil {
		t.Fatal("NewDomainBlocker returned nil")
	}

	if blocker.Count() != 0 {
		t.Errorf("Expected 0 rules, got %d", blocker.Count())
	}
}

func TestDomainBlocker_AddRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "malware.com",
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

func TestDomainBlocker_AddRule_WildCard(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-wildcard",
		Pattern:   "*.malware.com",
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

func TestDomainBlocker_AddRule_EmptyPattern(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-empty",
		Pattern:   "",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := blocker.AddRule(rule)
	if err == nil {
		t.Error("Expected error for empty pattern")
	}
}

func TestDomainBlocker_Check_ExactMatch(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	decision := blocker.Check("malware.com")
	if decision == nil {
		t.Fatal("Check returned nil")
	}

	if !decision.Blocked {
		t.Error("Expected domain to be blocked")
	}

	if decision.Category != "domain" {
		t.Errorf("Expected category 'domain', got '%s'", decision.Category)
	}
}

func TestDomainBlocker_Check_WildcardMatch(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-wildcard",
		Pattern:   "*.malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	tests := []struct {
		domain  string
		blocked bool
	}{
		{"sub.malware.com", true},
		{"deep.sub.malware.com", true},
		{"www.malware.com", true},
		{"malware.com", false},      // Wildcard doesn't match root
		{"notmalware.com", false},   // Different domain
		{"malware.com.evil", false}, // Domain suffix attack
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.domain)
		if decision.Blocked != tt.blocked {
			t.Errorf("Domain %s: expected blocked=%v, got blocked=%v", tt.domain, tt.blocked, decision.Blocked)
		}
	}
}

func TestDomainBlocker_Check_CaseInsensitive(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	tests := []string{
		"malware.com",
		"MALWARE.COM",
		"Malware.Com",
		"MaLwArE.cOm",
	}

	for _, domain := range tests {
		decision := blocker.Check(domain)
		if !decision.Blocked {
			t.Errorf("Expected %s to be blocked (case insensitive)", domain)
		}
	}
}

func TestDomainBlocker_Check_AllowedDomain(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	decision := blocker.Check("safe.com")
	if decision.Blocked {
		t.Error("Expected domain to be allowed")
	}
}

func TestDomainBlocker_Check_ExpiredRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-expired",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	blocker.AddRule(rule)

	decision := blocker.Check("malware.com")
	if decision.Blocked {
		t.Error("Expected expired rule to not block")
	}
}

func TestDomainBlocker_Check_MultipleWildcards(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	// Add multiple wildcard rules
	rules := []BlockRule{
		{ID: "w1", Pattern: "*.google.com", Category: "ads", Source: "test", CreatedAt: time.Now()},
		{ID: "w2", Pattern: "*.facebook.com", Category: "social", Source: "test", CreatedAt: time.Now()},
		{ID: "w3", Pattern: "*.tracker.net", Category: "tracking", Source: "test", CreatedAt: time.Now()},
	}

	for _, rule := range rules {
		blocker.AddRule(rule)
	}

	tests := []struct {
		domain  string
		blocked bool
	}{
		{"ads.google.com", true},
		{"www.facebook.com", true},
		{"pixel.tracker.net", true},
		{"safe.com", false},
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.domain)
		if decision.Blocked != tt.blocked {
			t.Errorf("Domain %s: expected blocked=%v, got blocked=%v", tt.domain, tt.blocked, decision.Blocked)
		}
	}
}

func TestDomainBlocker_RemoveRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-1",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	// Verify it's blocked
	decision := blocker.Check("malware.com")
	if !decision.Blocked {
		t.Error("Expected domain to be blocked before removal")
	}

	// Remove the rule
	err := blocker.RemoveRule("test-1")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// Verify it's no longer blocked
	decision = blocker.Check("malware.com")
	if decision.Blocked {
		t.Error("Expected domain to be allowed after rule removal")
	}
}

func TestDomainBlocker_RemoveRule_Wildcard(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:        "test-wildcard",
		Pattern:   "*.malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	blocker.AddRule(rule)

	// Verify it's blocked
	decision := blocker.Check("sub.malware.com")
	if !decision.Blocked {
		t.Error("Expected subdomain to be blocked before removal")
	}

	// Remove the rule
	err := blocker.RemoveRule("test-wildcard")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// Verify it's no longer blocked
	decision = blocker.Check("sub.malware.com")
	if decision.Blocked {
		t.Error("Expected subdomain to be allowed after rule removal")
	}
}

func TestDomainBlocker_Clear(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	// Add multiple rules
	domains := []string{"malware1.com", "malware2.com", "*.evil.com"}
	for i, domain := range domains {
		rule := BlockRule{
			ID:        "test-" + string(rune('0'+i)),
			Pattern:   domain,
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		}
		blocker.AddRule(rule)
	}

	if blocker.Count() != 3 {
		t.Errorf("Expected 3 rules, got %d", blocker.Count())
	}

	blocker.Clear()

	if blocker.Count() != 0 {
		t.Errorf("Expected 0 rules after clear, got %d", blocker.Count())
	}
}

func TestDomainBlocker_WildcardDisabled(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(false, logger)

	// Without wildcard support, wildcards shouldn't work
	if blocker.IsWildcardSupported() {
		t.Error("Expected wildcard to be disabled")
	}
}

func TestDomainBlocker_GetRules(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	rules := []BlockRule{
		{ID: "r1", Pattern: "malware1.com", Category: "cat1", Source: "test", CreatedAt: time.Now()},
		{ID: "r2", Pattern: "malware2.com", Category: "cat2", Source: "test", CreatedAt: time.Now()},
		{ID: "r3", Pattern: "*.evil.com", Category: "cat3", Source: "test", CreatedAt: time.Now()},
	}

	for _, rule := range rules {
		blocker.AddRule(rule)
	}

	gotRules := blocker.GetRules()
	if len(gotRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(gotRules))
	}
}

func BenchmarkDomainBlocker_Check_ExactMatch(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	// Add 1000 domain rules
	for i := 0; i < 1000; i++ {
		rule := BlockRule{
			ID:        "test-" + string(rune(i)),
			Pattern:   "domain" + string(rune(i)) + ".com",
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		}
		blocker.AddRule(rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blocker.Check("domain500.com")
	}
}

func BenchmarkDomainBlocker_Check_WildcardMatch(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	blocker := NewDomainBlocker(true, logger)

	// Add 100 wildcard rules
	for i := 0; i < 100; i++ {
		rule := BlockRule{
			ID:        "test-wildcard-" + string(rune(i)),
			Pattern:   "*.domain" + string(rune(i)) + ".com",
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		}
		blocker.AddRule(rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blocker.Check("sub.domain50.com")
	}
}
