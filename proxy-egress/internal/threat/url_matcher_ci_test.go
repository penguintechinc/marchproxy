//go:build ci

package threat

import (
	"testing"
	"time"

	"marchproxy-egress/internal/logging"
)

func TestURLMatcherCheck(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		patterns      []PatternRule
		url           string
		expectedBlock bool
	}{
		{
			name: "simple literal match",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  "malware\\.com/download",
					Priority: 1,
				},
			},
			url:           "malware.com/download",
			expectedBlock: true,
		},
		{
			name: "no match",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  "malware\\.com/download",
					Priority: 1,
				},
			},
			url:           "safe.com/download",
			expectedBlock: false,
		},
		{
			name: "regex pattern match",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  ".*\\.com/(admin|config)",
					Priority: 1,
				},
			},
			url:           "example.com/admin",
			expectedBlock: true,
		},
		{
			name: "regex pattern no match",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  ".*\\.com/(admin|config)",
					Priority: 1,
				},
			},
			url:           "example.com/user",
			expectedBlock: false,
		},
		{
			name: "empty URL",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  ".*",
					Priority: 1,
				},
			},
			url:           "",
			expectedBlock: false,
		},
		{
			name: "multiple patterns - first matches",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  "malware.*",
					Priority: 10,
				},
				{
					ID:       "rule2",
					Pattern:  "phishing.*",
					Priority: 5,
				},
			},
			url:           "malware.com/payload",
			expectedBlock: true,
		},
		{
			name: "multiple patterns - second matches",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  "malware.*",
					Priority: 10,
				},
				{
					ID:       "rule2",
					Pattern:  "phishing.*",
					Priority: 5,
				},
			},
			url:           "phishing.io/form",
			expectedBlock: true,
		},
		{
			name: "priority ordering - high priority checked first",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  ".*malicious.*",
					Priority: 5,
				},
				{
					ID:       "rule2",
					Pattern:  ".*evil.*",
					Priority: 10,
				},
			},
			url:           "http://evil.com/malicious",
			expectedBlock: true,
		},
		{
			name: "case-sensitive matching",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  "Malware",
					Priority: 1,
				},
			},
			url:           "malware",
			expectedBlock: false, // Case doesn't match
		},
		{
			name: "pattern with anchors",
			patterns: []PatternRule{
				{
					ID:       "rule1",
					Pattern:  "^http://.*\\.com/admin$",
					Priority: 1,
				},
			},
			url:           "http://example.com/admin",
			expectedBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewURLMatcher("re2", 1000, logger)
			if err != nil {
				t.Fatalf("failed to create matcher: %v", err)
			}

			for _, pattern := range tt.patterns {
				if err := matcher.AddPattern(pattern); err != nil {
					t.Fatalf("failed to add pattern: %v", err)
				}
			}

			decision := matcher.Check(tt.url)

			if decision.Blocked != tt.expectedBlock {
				t.Errorf("expected blocked=%v, got %v (reason: %s)", tt.expectedBlock, decision.Blocked, decision.Reason)
			}
		})
	}
}

func TestURLMatcherAddPattern(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		pattern       string
		shouldSucceed bool
	}{
		{
			name:          "valid simple pattern",
			pattern:       "malware.*",
			shouldSucceed: true,
		},
		{
			name:          "valid regex with groups",
			pattern:       "(admin|config|settings)",
			shouldSucceed: true,
		},
		{
			name:          "valid anchored pattern",
			pattern:       "^https://.*\\.com/api$",
			shouldSucceed: true,
		},
		{
			name:          "invalid regex - unmatched bracket",
			pattern:       "[malware",
			shouldSucceed: false,
		},
		{
			name:          "invalid regex - bad group",
			pattern:       "(?P<invalid",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, _ := NewURLMatcher("re2", 1000, logger)
			rule := PatternRule{
				ID:       "test",
				Pattern:  tt.pattern,
				Priority: 1,
			}

			err := matcher.AddPattern(rule)

			if (err == nil) != tt.shouldSucceed {
				t.Errorf("expected success=%v, got error=%v", tt.shouldSucceed, err)
			}
		})
	}
}

func TestURLMatcherCapacity(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 3, logger)

	rules := []PatternRule{
		{ID: "r1", Pattern: "malware1.*", Priority: 1},
		{ID: "r2", Pattern: "malware2.*", Priority: 1},
		{ID: "r3", Pattern: "malware3.*", Priority: 1},
		{ID: "r4", Pattern: "malware4.*", Priority: 1}, // Should exceed capacity
	}

	for i, rule := range rules {
		err := matcher.AddPattern(rule)
		if i < 3 {
			if err != nil {
				t.Errorf("failed to add pattern %d: %v", i, err)
			}
		} else {
			if err == nil {
				t.Errorf("expected capacity exceeded error for pattern %d", i)
			}
		}
	}
}

func TestURLMatcherRemovePattern(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	pattern := PatternRule{
		ID:       "rule1",
		Pattern:  "malware.*",
		Priority: 1,
	}

	matcher.AddPattern(pattern)

	if matcher.Count() != 1 {
		t.Errorf("expected 1 pattern after add, got %d", matcher.Count())
	}

	if err := matcher.RemovePattern("rule1"); err != nil {
		t.Errorf("failed to remove pattern: %v", err)
	}

	if matcher.Count() != 0 {
		t.Errorf("expected 0 patterns after remove, got %d", matcher.Count())
	}

	if err := matcher.RemovePattern("nonexistent"); err == nil {
		t.Error("expected error when removing nonexistent pattern")
	}
}

func TestURLMatcherClear(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	for i := 0; i < 5; i++ {
		pattern := PatternRule{
			ID:       "rule" + string(rune(i)),
			Pattern:  "malware" + string(rune(i)) + ".*",
			Priority: i,
		}
		matcher.AddPattern(pattern)
	}

	if matcher.Count() != 5 {
		t.Errorf("expected 5 patterns, got %d", matcher.Count())
	}

	matcher.Clear()

	if matcher.Count() != 0 {
		t.Errorf("expected 0 patterns after clear, got %d", matcher.Count())
	}
}

func TestURLMatcherExpiration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	// Add expired pattern
	expiredPattern := PatternRule{
		ID:        "expired",
		Pattern:   "malware.*",
		Priority:  1,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	matcher.AddPattern(expiredPattern)

	decision := matcher.Check("malware.com/payload")
	if decision.Blocked {
		t.Error("expected expired pattern to not block")
	}

	// Add active pattern
	activePattern := PatternRule{
		ID:        "active",
		Pattern:   "phishing.*",
		Priority:  1,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	matcher.AddPattern(activePattern)

	decision = matcher.Check("phishing.io/form")
	if !decision.Blocked {
		t.Error("expected active pattern to block")
	}
}

func TestURLMatcherCleanExpired(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	// Add expired patterns
	matcher.AddPattern(PatternRule{
		ID:        "expired1",
		Pattern:   "malware.*",
		Priority:  1,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	matcher.AddPattern(PatternRule{
		ID:        "expired2",
		Pattern:   "phishing.*",
		Priority:  1,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	// Add active patterns
	matcher.AddPattern(PatternRule{
		ID:        "active",
		Pattern:   "active.*",
		Priority:  1,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	matcher.AddPattern(PatternRule{
		ID:       "permanent",
		Pattern:  "permanent.*",
		Priority: 1,
	})

	if matcher.Count() != 4 {
		t.Errorf("expected 4 patterns before clean, got %d", matcher.Count())
	}

	removed := matcher.CleanExpired()

	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	if matcher.Count() != 2 {
		t.Errorf("expected 2 patterns after clean, got %d", matcher.Count())
	}
}

func TestURLMatcherGetPatterns(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	patterns := []PatternRule{
		{ID: "r1", Pattern: "malware.*", Priority: 1},
		{ID: "r2", Pattern: "phishing.*", Priority: 2},
		{ID: "r3", Pattern: "trojan.*", Priority: 3},
	}

	for _, p := range patterns {
		matcher.AddPattern(p)
	}

	retrieved := matcher.GetPatterns()

	if len(retrieved) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(retrieved))
	}

	// Verify all IDs are present
	idMap := make(map[string]bool)
	for _, p := range retrieved {
		idMap[p.ID] = true
	}

	for _, p := range patterns {
		if !idMap[p.ID] {
			t.Errorf("pattern %s not found in retrieved patterns", p.ID)
		}
	}
}

func TestURLMatcherValidate(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	tests := []struct {
		pattern       string
		shouldSucceed bool
	}{
		{
			pattern:       "malware.*",
			shouldSucceed: true,
		},
		{
			pattern:       "^https://.*\\.com/api$",
			shouldSucceed: true,
		},
		{
			pattern:       "[invalid",
			shouldSucceed: false,
		},
		{
			pattern:       "(?P<bad",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		err := matcher.Validate(tt.pattern)
		if (err == nil) != tt.shouldSucceed {
			t.Errorf("validate %s: expected success=%v, got error=%v", tt.pattern, tt.shouldSucceed, err)
		}
	}
}

func TestURLMatcherBulkAdd(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 100, logger)

	patterns := []PatternRule{
		{ID: "r1", Pattern: "malware1.*", Priority: 1},
		{ID: "r2", Pattern: "malware2.*", Priority: 2},
		{ID: "r3", Pattern: "malware3.*", Priority: 3},
		{ID: "r4", Pattern: "malware4.*", Priority: 4},
		{ID: "r5", Pattern: "malware5.*", Priority: 5},
	}

	added, err := matcher.BulkAdd(patterns)

	if added != 5 {
		t.Errorf("expected 5 added, got %d", added)
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if matcher.Count() != 5 {
		t.Errorf("expected 5 patterns after bulk add, got %d", matcher.Count())
	}
}

func TestURLMatcherBulkAddWithErrors(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 100, logger)

	patterns := []PatternRule{
		{ID: "r1", Pattern: "malware1.*", Priority: 1},
		{ID: "r2", Pattern: "[invalid", Priority: 2},
		{ID: "r3", Pattern: "malware3.*", Priority: 3},
	}

	added, err := matcher.BulkAdd(patterns)

	if added != 2 {
		t.Errorf("expected 2 added despite error, got %d", added)
	}

	if err == nil {
		t.Error("expected error for invalid pattern")
	}
}

func TestURLMatcherGetEngine(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	if matcher.GetEngine() != "re2" {
		t.Errorf("expected engine 're2', got %s", matcher.GetEngine())
	}
}

func TestURLMatcherGetMaxPatterns(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 500, logger)

	if matcher.GetMaxPatterns() != 500 {
		t.Errorf("expected max patterns 500, got %d", matcher.GetMaxPatterns())
	}
}

func TestURLMatcherPriorityOrdering(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	// Add patterns in non-priority order
	patterns := []PatternRule{
		{ID: "r1", Pattern: ".*pattern1.*", Priority: 1},
		{ID: "r5", Pattern: ".*pattern5.*", Priority: 5},
		{ID: "r3", Pattern: ".*pattern3.*", Priority: 3},
		{ID: "r2", Pattern: ".*pattern2.*", Priority: 2},
		{ID: "r4", Pattern: ".*pattern4.*", Priority: 4},
	}

	for _, p := range patterns {
		matcher.AddPattern(p)
	}

	retrieved := matcher.GetPatterns()

	// Patterns should be returned in priority order (highest first)
	if len(retrieved) != 5 {
		t.Fatalf("expected 5 patterns, got %d", len(retrieved))
	}

	for i := 0; i < len(retrieved)-1; i++ {
		if retrieved[i].Priority < retrieved[i+1].Priority {
			t.Errorf("patterns not in priority order: %d >= %d", retrieved[i].Priority, retrieved[i+1].Priority)
		}
	}
}

func TestURLMatcherComplexRegex(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	matcher, _ := NewURLMatcher("re2", 1000, logger)

	// Add complex regex pattern for SQL injection detection
	matcher.AddPattern(PatternRule{
		ID:       "sqli",
		Pattern:  "(?i)(union.*select|select.*from|insert.*into|delete.*from|drop.*table)",
		Priority: 10,
	})

	tests := []struct {
		url    string
		block  bool
	}{
		{"http://example.com/?id=1", false},
		{"http://example.com/?id=1 UNION SELECT 1,2,3", true},
		{"http://example.com/?id=1 union select * from users", true},
		{"http://example.com/insert-data", false},
		{"http://example.com/?sql=INSERT INTO table VALUES (1)", true},
	}

	for _, tt := range tests {
		decision := matcher.Check(tt.url)
		if decision.Blocked != tt.block {
			t.Errorf("URL %s: expected blocked=%v, got %v", tt.url, tt.block, decision.Blocked)
		}
	}
}
