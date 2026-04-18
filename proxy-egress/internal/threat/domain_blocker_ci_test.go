//go:build ci

package threat

import (
	"strings"
	"testing"
	"time"

	"marchproxy-egress/internal/logging"
)

func TestDomainBlockerCheck(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		wildcards     bool
		rules         []BlockRule
		domain        string
		expectedBlock bool
	}{
		{
			name:      "exact domain match",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "malicious.com",
				},
			},
			domain:        "malicious.com",
			expectedBlock: true,
		},
		{
			name:      "exact domain no match",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "malicious.com",
				},
			},
			domain:        "safe.com",
			expectedBlock: false,
		},
		{
			name:      "case insensitive match",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "MALICIOUS.COM",
				},
			},
			domain:        "malicious.com",
			expectedBlock: true,
		},
		{
			name:      "empty domain",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "example.com",
				},
			},
			domain:        "",
			expectedBlock: false,
		},
		{
			name:      "domain with port",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "example.com",
				},
			},
			domain:        "example.com:8080",
			expectedBlock: true,
		},
		{
			name:      "wildcard match - single level",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "*.example.com",
				},
			},
			domain:        "sub.example.com",
			expectedBlock: true,
		},
		{
			name:      "wildcard match - multiple levels",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "*.example.com",
				},
			},
			domain:        "deep.sub.example.com",
			expectedBlock: true,
		},
		{
			name:      "wildcard no match - different domain",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "*.example.com",
				},
			},
			domain:        "other.com",
			expectedBlock: false,
		},
		{
			name:      "wildcard not supported - exact match only",
			wildcards: false,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "sub.example.com",
				},
			},
			domain:        "sub.example.com",
			expectedBlock: true,
		},
		{
			name:      "multiple rules - first matches",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "bad.com",
				},
				{
					ID:      "rule2",
					Pattern: "*.evil.com",
				},
			},
			domain:        "bad.com",
			expectedBlock: true,
		},
		{
			name:      "multiple rules - second matches",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "bad.com",
				},
				{
					ID:      "rule2",
					Pattern: "*.evil.com",
				},
			},
			domain:        "phish.evil.com",
			expectedBlock: true,
		},
		{
			name:      "whitespace trimmed",
			wildcards: true,
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "example.com",
				},
			},
			domain:        "  example.com  ",
			expectedBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker := NewDomainBlocker(tt.wildcards, logger)

			for _, rule := range tt.rules {
				if err := blocker.AddRule(rule); err != nil {
					t.Fatalf("failed to add rule: %v", err)
				}
			}

			decision := blocker.Check(tt.domain)

			if decision.Blocked != tt.expectedBlock {
				t.Errorf("expected blocked=%v, got %v (reason: %s)", tt.expectedBlock, decision.Blocked, decision.Reason)
			}
		})
	}
}

func TestDomainBlockerAddRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		wildcards     bool
		pattern       string
		shouldSucceed bool
	}{
		{
			name:          "valid exact domain",
			wildcards:     true,
			pattern:       "example.com",
			shouldSucceed: true,
		},
		{
			name:          "valid wildcard",
			wildcards:     true,
			pattern:       "*.example.com",
			shouldSucceed: true,
		},
		{
			name:          "wildcard when disabled",
			wildcards:     false,
			pattern:       "*.example.com",
			shouldSucceed: false,
		},
		{
			name:          "empty pattern",
			wildcards:     true,
			pattern:       "",
			shouldSucceed: false,
		},
		{
			name:          "only whitespace",
			wildcards:     true,
			pattern:       "   ",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker := NewDomainBlocker(tt.wildcards, logger)
			rule := BlockRule{
				ID:      "test",
				Pattern: tt.pattern,
			}

			err := blocker.AddRule(rule)

			if (err == nil) != tt.shouldSucceed {
				t.Errorf("expected success=%v, got error=%v", tt.shouldSucceed, err)
			}
		})
	}
}

func TestDomainBlockerRemoveRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:      "rule1",
		Pattern: "example.com",
	}

	blocker.AddRule(rule)

	if blocker.Count() != 1 {
		t.Errorf("expected 1 rule after add, got %d", blocker.Count())
	}

	if err := blocker.RemoveRule("rule1"); err != nil {
		t.Errorf("failed to remove rule: %v", err)
	}

	if blocker.Count() != 0 {
		t.Errorf("expected 0 rules after remove, got %d", blocker.Count())
	}

	if err := blocker.RemoveRule("nonexistent"); err == nil {
		t.Error("expected error when removing nonexistent rule")
	}
}

func TestDomainBlockerClear(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	domains := []string{"bad1.com", "bad2.com", "*.evil.com", "*.malicious.net"}

	for i, domain := range domains {
		rule := BlockRule{
			ID:      "rule" + string(rune(i)),
			Pattern: domain,
		}
		blocker.AddRule(rule)
	}

	if blocker.Count() != 4 {
		t.Errorf("expected 4 rules, got %d", blocker.Count())
	}

	blocker.Clear()

	if blocker.Count() != 0 {
		t.Errorf("expected 0 rules after clear, got %d", blocker.Count())
	}
}

func TestDomainBlockerExpiration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	// Add expired rule
	expiredRule := BlockRule{
		ID:        "expired",
		Pattern:   "expired.com",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	blocker.AddRule(expiredRule)

	decision := blocker.Check("expired.com")
	if decision.Blocked {
		t.Error("expected expired rule to not block")
	}

	// Add active rule
	activeRule := BlockRule{
		ID:        "active",
		Pattern:   "active.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	blocker.AddRule(activeRule)

	decision = blocker.Check("active.com")
	if !decision.Blocked {
		t.Error("expected active rule to block")
	}
}

func TestDomainBlockerCleanExpired(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	// Add expired exact domain
	blocker.AddRule(BlockRule{
		ID:        "expired1",
		Pattern:   "expired1.com",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	// Add expired wildcard
	blocker.AddRule(BlockRule{
		ID:        "expired2",
		Pattern:   "*.expired.com",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	// Add active rules
	blocker.AddRule(BlockRule{
		ID:        "active1",
		Pattern:   "active.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	blocker.AddRule(BlockRule{
		ID:      "permanent",
		Pattern: "forever.com",
	})

	if blocker.Count() != 4 {
		t.Errorf("expected 4 rules before clean, got %d", blocker.Count())
	}

	removed := blocker.CleanExpired()

	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	if blocker.Count() != 2 {
		t.Errorf("expected 2 rules after clean, got %d", blocker.Count())
	}
}

func TestDomainBlockerGetRules(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	rules := []BlockRule{
		{ID: "r1", Pattern: "bad.com"},
		{ID: "r2", Pattern: "evil.org"},
		{ID: "r3", Pattern: "*.phish.net"},
		{ID: "r4", Pattern: "*.malware.com"},
	}

	for _, rule := range rules {
		blocker.AddRule(rule)
	}

	retrieved := blocker.GetRules()

	if len(retrieved) != 4 {
		t.Errorf("expected 4 rules, got %d", len(retrieved))
	}

	// Verify all patterns are present
	patternMap := make(map[string]bool)
	for _, rule := range retrieved {
		patternMap[rule.Pattern] = true
	}

	for _, rule := range rules {
		if !patternMap[rule.Pattern] {
			t.Errorf("pattern %s not found in retrieved rules", rule.Pattern)
		}
	}
}

func TestDomainBlockerWildcardMatching(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	// Add wildcard rule
	blocker.AddRule(BlockRule{
		ID:      "rule1",
		Pattern: "*.example.com",
	})

	tests := []struct {
		domain        string
		expectedBlock bool
	}{
		{"example.com", false},        // Base domain doesn't match wildcard
		{"sub.example.com", true},     // Single level subdomain
		{"deep.sub.example.com", true}, // Multiple levels
		{"a.b.c.example.com", true},   // Many levels
		{"example.org", false},        // Different TLD
		{"notexample.com", false},     // Similar but different
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.domain)
		if decision.Blocked != tt.expectedBlock {
			t.Errorf("domain %s: expected blocked=%v, got %v", tt.domain, tt.expectedBlock, decision.Blocked)
		}
	}
}

func TestDomainBlockerNormalization(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	// Add rule with uppercase
	blocker.AddRule(BlockRule{
		ID:      "rule1",
		Pattern: "MALICIOUS.COM",
	})

	tests := []struct {
		domain string
		block  bool
	}{
		{"malicious.com", true},
		{"MALICIOUS.COM", true},
		{"Malicious.Com", true},
		{"MaLiCiOuS.cOm", true},
		{"safe.com", false},
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.domain)
		if decision.Blocked != tt.block {
			t.Errorf("domain %s: expected blocked=%v, got %v", tt.domain, tt.block, decision.Blocked)
		}
	}
}

func TestDomainBlockerIsWildcardSupported(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blockerWithWildcards := NewDomainBlocker(true, logger)
	if !blockerWithWildcards.IsWildcardSupported() {
		t.Error("expected wildcard support to be enabled")
	}

	blockerWithoutWildcards := NewDomainBlocker(false, logger)
	if blockerWithoutWildcards.IsWildcardSupported() {
		t.Error("expected wildcard support to be disabled")
	}
}

func TestDomainBlockerCount(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	if blocker.Count() != 0 {
		t.Errorf("expected count=0 initially, got %d", blocker.Count())
	}

	for i := 0; i < 10; i++ {
		pattern := "domain" + string(rune(i)) + ".com"
		if i%3 == 0 {
			pattern = "*." + pattern
		}
		blocker.AddRule(BlockRule{
			ID:      "rule" + string(rune(i)),
			Pattern: pattern,
		})
	}

	if blocker.Count() != 10 {
		t.Errorf("expected count=10, got %d", blocker.Count())
	}
}

func TestDomainBlockerMixedExactAndWildcard(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	blocker.AddRule(BlockRule{ID: "r1", Pattern: "exact.com"})
	blocker.AddRule(BlockRule{ID: "r2", Pattern: "*.wildcard.com"})

	tests := []struct {
		domain string
		block  bool
	}{
		{"exact.com", true},
		{"other.com", false},
		{"sub.wildcard.com", true},
		{"deep.sub.wildcard.com", true},
		{"wildcard.com", false},
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.domain)
		if decision.Blocked != tt.block {
			t.Errorf("domain %s: expected blocked=%v, got %v", tt.domain, tt.block, decision.Blocked)
		}
	}
}

func TestDomainBlockerPortHandling(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	blocker.AddRule(BlockRule{
		ID:      "rule1",
		Pattern: "example.com",
	})

	tests := []struct {
		domain string
		block  bool
	}{
		{"example.com", true},
		{"example.com:80", true},
		{"example.com:443", true},
		{"example.com:8080", true},
		{"example.com:9999", true},
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.domain)
		if decision.Blocked != tt.block {
			t.Errorf("domain %s: expected blocked=%v, got %v", tt.domain, tt.block, decision.Blocked)
		}
	}
}

func TestDomainBlockerMatchedRuleInfo(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewDomainBlocker(true, logger)

	rule := BlockRule{
		ID:       "test-rule-123",
		Pattern:  "malicious.com",
		Category: "phishing",
	}

	blocker.AddRule(rule)

	decision := blocker.Check("malicious.com")

	if !decision.Blocked {
		t.Fatal("expected domain to be blocked")
	}

	if decision.MatchedRule != "test-rule-123" {
		t.Errorf("expected matched rule ID 'test-rule-123', got %s", decision.MatchedRule)
	}

	if decision.Category != "domain" {
		t.Errorf("expected category 'domain', got %s", decision.Category)
	}

	if !strings.Contains(decision.Reason, "malicious.com") {
		t.Errorf("expected reason to contain domain name, got: %s", decision.Reason)
	}
}
