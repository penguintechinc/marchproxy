//go:build ci

package threat

import (
	"testing"
	"time"

	"marchproxy-egress/internal/logging"
)

func TestIPBlockerCheck(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		rules         []BlockRule
		ip            string
		expectedBlock bool
	}{
		{
			name: "exact IPv4 match",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.1",
					Category: "malware",
				},
			},
			ip:            "192.168.1.1",
			expectedBlock: true,
		},
		{
			name: "IPv4 no match",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.1",
					Category: "malware",
				},
			},
			ip:            "192.168.1.2",
			expectedBlock: false,
		},
		{
			name: "CIDR IPv4 match",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.0/24",
					Category: "malware",
				},
			},
			ip:            "192.168.1.50",
			expectedBlock: true,
		},
		{
			name: "CIDR IPv4 boundary - network address",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.0/24",
					Category: "malware",
				},
			},
			ip:            "192.168.1.0",
			expectedBlock: true,
		},
		{
			name: "CIDR IPv4 boundary - broadcast address",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.0/24",
					Category: "malware",
				},
			},
			ip:            "192.168.1.255",
			expectedBlock: true,
		},
		{
			name: "CIDR IPv4 outside range",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.0/24",
					Category: "malware",
				},
			},
			ip:            "192.168.2.1",
			expectedBlock: false,
		},
		{
			name: "invalid IP address",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.1",
					Category: "malware",
				},
			},
			ip:            "invalid-ip",
			expectedBlock: false,
		},
		{
			name: "IPv6 exact match",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "2001:db8::1",
					Category: "malware",
				},
			},
			ip:            "2001:db8::1",
			expectedBlock: true,
		},
		{
			name: "IPv6 CIDR match",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "2001:db8::/32",
					Category: "malware",
				},
			},
			ip:            "2001:db8::50",
			expectedBlock: true,
		},
		{
			name: "multiple rules - first matches",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.1",
					Category: "malware",
				},
				{
					ID:      "rule2",
					Pattern: "192.168.2.0/24",
					Category: "botnet",
				},
			},
			ip:            "192.168.1.1",
			expectedBlock: true,
		},
		{
			name: "multiple rules - second matches",
			rules: []BlockRule{
				{
					ID:      "rule1",
					Pattern: "192.168.1.1",
					Category: "malware",
				},
				{
					ID:      "rule2",
					Pattern: "192.168.2.0/24",
					Category: "botnet",
				},
			},
			ip:            "192.168.2.100",
			expectedBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker := NewIPBlocker(10000, logger)

			for _, rule := range tt.rules {
				if err := blocker.AddRule(rule); err != nil {
					t.Fatalf("failed to add rule: %v", err)
				}
			}

			decision := blocker.Check(tt.ip)

			if decision.Blocked != tt.expectedBlock {
				t.Errorf("expected blocked=%v, got %v (reason: %s)", tt.expectedBlock, decision.Blocked, decision.Reason)
			}
		})
	}
}

func TestIPBlockerExpiration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(10000, logger)

	// Add an IP that expires in the past
	expiredRule := BlockRule{
		ID:        "expired",
		Pattern:   "192.168.1.1",
		Category:  "malware",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	blocker.AddRule(expiredRule)

	// Should not be blocked (rule expired)
	decision := blocker.Check("192.168.1.1")
	if decision.Blocked {
		t.Error("expected expired rule to not block")
	}

	// Add a rule that expires in the future
	futureRule := BlockRule{
		ID:        "future",
		Pattern:   "192.168.2.1",
		Category:  "malware",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	blocker.AddRule(futureRule)

	decision = blocker.Check("192.168.2.1")
	if !decision.Blocked {
		t.Error("expected active rule to block")
	}
}

func TestIPBlockerAddRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		pattern       string
		shouldSucceed bool
	}{
		{
			name:          "valid IPv4",
			pattern:       "192.168.1.1",
			shouldSucceed: true,
		},
		{
			name:          "valid IPv4 CIDR",
			pattern:       "192.168.1.0/24",
			shouldSucceed: true,
		},
		{
			name:          "valid IPv6",
			pattern:       "2001:db8::1",
			shouldSucceed: true,
		},
		{
			name:          "valid IPv6 CIDR",
			pattern:       "2001:db8::/32",
			shouldSucceed: true,
		},
		{
			name:          "invalid IP",
			pattern:       "invalid",
			shouldSucceed: false,
		},
		{
			name:          "invalid CIDR",
			pattern:       "192.168.1.0/33",
			shouldSucceed: false,
		},
		{
			name:          "hostname",
			pattern:       "example.com",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocker := NewIPBlocker(10000, logger)
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

func TestIPBlockerCapacity(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(3, logger) // Very small capacity

	rules := []BlockRule{
		{ID: "r1", Pattern: "192.168.1.1"},
		{ID: "r2", Pattern: "192.168.1.2"},
		{ID: "r3", Pattern: "192.168.1.3"},
		{ID: "r4", Pattern: "192.168.1.4"}, // Should exceed capacity
	}

	for i, rule := range rules {
		err := blocker.AddRule(rule)
		if i < 3 {
			if err != nil {
				t.Errorf("failed to add rule %d: %v", i, err)
			}
		} else {
			if err == nil {
				t.Errorf("expected capacity exceeded error for rule %d", i)
			}
		}
	}
}

func TestIPBlockerRemoveRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(10000, logger)

	rule := BlockRule{
		ID:      "rule1",
		Pattern: "192.168.1.1",
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

func TestIPBlockerClear(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(10000, logger)

	for i := 0; i < 5; i++ {
		rule := BlockRule{
			ID:      "rule" + string(rune(i+48)), // Convert to ASCII: 0=48, 1=49, etc.
			Pattern: "192.168.1." + string(rune(i+48)),
		}
		blocker.AddRule(rule)
	}

	if blocker.Count() != 5 {
		t.Errorf("expected 5 rules, got %d", blocker.Count())
	}

	blocker.Clear()

	if blocker.Count() != 0 {
		t.Errorf("expected 0 rules after clear, got %d", blocker.Count())
	}
}

func TestIPBlockerCleanExpired(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(10000, logger)

	// Add expired rule
	expiredRule := BlockRule{
		ID:        "expired",
		Pattern:   "192.168.1.1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	blocker.AddRule(expiredRule)

	// Add active rule
	activeRule := BlockRule{
		ID:        "active",
		Pattern:   "192.168.1.2",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	blocker.AddRule(activeRule)

	// Add rule with no expiration
	permanentRule := BlockRule{
		ID:      "permanent",
		Pattern: "192.168.1.3",
	}
	blocker.AddRule(permanentRule)

	if blocker.Count() != 3 {
		t.Errorf("expected 3 rules before clean, got %d", blocker.Count())
	}

	removed := blocker.CleanExpired()

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	if blocker.Count() != 2 {
		t.Errorf("expected 2 rules after clean, got %d", blocker.Count())
	}
}

func TestIPBlockerGetRules(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(10000, logger)

	rules := []BlockRule{
		{ID: "r1", Pattern: "192.168.1.1"},
		{ID: "r2", Pattern: "192.168.1.0/24"},
		{ID: "r3", Pattern: "2001:db8::1"},
		{ID: "r4", Pattern: "2001:db8::/32"},
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

func TestIPBlockerMixedIPv4IPv6(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	blocker := NewIPBlocker(10000, logger)

	// Add mix of IPv4 and IPv6
	blocker.AddRule(BlockRule{ID: "r1", Pattern: "192.168.1.1"})
	blocker.AddRule(BlockRule{ID: "r2", Pattern: "2001:db8::1"})
	blocker.AddRule(BlockRule{ID: "r3", Pattern: "10.0.0.0/8"})
	blocker.AddRule(BlockRule{ID: "r4", Pattern: "fc00::/7"}) // Unique local addresses

	tests := []struct {
		ip    string
		block bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.2", false},
		{"10.0.0.1", true},
		{"10.1.1.1", true},
		{"11.0.0.1", false},
		{"2001:db8::1", true},
		{"2001:db8::2", false},
		{"fc00::1", true},
		{"fd00::1", true},
		{"fe80::1", false}, // Link-local, not in blocked range
	}

	for _, tt := range tests {
		decision := blocker.Check(tt.ip)
		if decision.Blocked != tt.block {
			t.Errorf("IP %s: expected blocked=%v, got %v", tt.ip, tt.block, decision.Blocked)
		}
	}
}
