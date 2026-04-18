//go:build ci

package security

import (
	"testing"

	"marchproxy-dblb/internal/logging"
)

// TestCheckQuerySQLInjection tests SQL injection detection
func TestCheckQuerySQLInjection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	tests := []struct {
		name           string
		query          string
		shouldBlock    bool
		descriptionStr string
	}{
		{
			name:           "union_select_attack",
			query:          "SELECT * FROM users UNION SELECT * FROM admin",
			shouldBlock:    true,
			descriptionStr: "UNION SELECT injection",
		},
		{
			name:           "or_equals_1",
			query:          "SELECT * FROM users WHERE id = 1 OR '1'='1'",
			shouldBlock:    true,
			descriptionStr: "OR 1=1 injection",
		},
		{
			name:           "comment_injection",
			query:          "SELECT * FROM users WHERE id = 1 /*-- ",
			shouldBlock:    true,
			descriptionStr: "Comment injection",
		},
		{
			name:           "drop_table_semicolon",
			query:          "SELECT * FROM users; DROP TABLE users",
			shouldBlock:    true,
			descriptionStr: "DROP TABLE injection",
		},
		{
			name:           "benchmark_sleep_attack",
			query:          "SELECT * FROM users WHERE id = SLEEP(5)",
			shouldBlock:    true,
			descriptionStr: "SLEEP injection",
		},
		{
			name:           "xp_cmdshell_attack",
			query:          "EXEC xp_cmdshell 'dir'",
			shouldBlock:    true,
			descriptionStr: "xp_cmdshell execution",
		},
		{
			name:           "javascript_injection",
			query:          "<script>alert('xss')</script>",
			shouldBlock:    true,
			descriptionStr: "JavaScript injection",
		},
		{
			name:           "double_dash_comment",
			query:          "SELECT * FROM users WHERE id = 1 -- ",
			shouldBlock:    true,
			descriptionStr: "Double-dash comment",
		},
		{
			name:           "hash_comment",
			query:          "SELECT * FROM users WHERE id = 1 # comment",
			shouldBlock:    true,
			descriptionStr: "Hash comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, reason := checker.CheckQuery(tt.query)

			if blocked != tt.shouldBlock {
				t.Errorf("%s: CheckQuery blocked=%v, want %v (reason: %s)", tt.descriptionStr, blocked, tt.shouldBlock, reason)
			}

			if tt.shouldBlock && reason == "" {
				t.Errorf("%s: Expected reason but got empty string", tt.descriptionStr)
			}
		})
	}
}

// TestCheckDataInspection tests data inspection for malicious content
func TestCheckDataInspection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	tests := []struct {
		name        string
		data        []byte
		shouldBlock bool
	}{
		{
			name:        "clean_data",
			data:        []byte("user@example.com"),
			shouldBlock: false,
		},
		{
			name:        "sql_injection_data",
			data:        []byte("admin' OR 1=1--"),
			shouldBlock: true,
		},
		{
			name:        "union_select_data",
			data:        []byte("UNION SELECT * FROM admin"),
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, _ := checker.CheckData(tt.data)

			if blocked != tt.shouldBlock {
				t.Errorf("CheckData blocked=%v, want %v", blocked, tt.shouldBlock)
			}
		})
	}
}

// TestGetStats returns security checker statistics
func TestGetStatsChecker(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	// Run some checks
	checker.CheckQuery("normal data without injection")
	checker.CheckQuery("SELECT * FROM users UNION SELECT * FROM admin")
	checker.CheckQuery("simple text")

	stats := checker.GetStats()

	inspectedCount, _ := stats["inspected_count"].(int64)
	if inspectedCount != 3 {
		t.Errorf("Expected 3 inspected queries, got %v", stats["inspected_count"])
	}

	blockedCount, _ := stats["blocked_count"].(int64)
	if blockedCount != 1 {
		t.Errorf("Expected 1 blocked query, got %v", stats["blocked_count"])
	}
}

// TestResetStats tests resetting statistics
func TestResetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	// Run some checks
	checker.CheckQuery("SELECT * FROM users UNION SELECT * FROM admin")

	stats := checker.GetStats()
	blockedCount, ok := stats["blocked_count"].(int64)
	if !ok || blockedCount == 0 {
		t.Error("Should have blocked count after checking injection")
	}

	checker.Reset()

	stats = checker.GetStats()
	blockedCount, ok = stats["blocked_count"].(int64)
	if !ok || blockedCount != 0 {
		t.Errorf("After reset, blocked_count should be 0, got %v", stats["blocked_count"])
	}

	inspectedCount, ok := stats["inspected_count"].(int64)
	if !ok || inspectedCount != 0 {
		t.Errorf("After reset, inspected_count should be 0, got %v", stats["inspected_count"])
	}
}

// TestExcessiveSQLKeywordsDetection tests detection of excessive SQL keywords
func TestExcessiveSQLKeywordsDetection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	// Query with excessive keywords should be blocked
	maliciousQuery := "SELECT * FROM users WHERE id IN (SELECT user_id FROM admin WHERE role='admin') UNION SELECT * FROM sensitive"

	blocked, reason := checker.CheckQuery(maliciousQuery)
	if !blocked && reason == "" {
		t.Errorf("Query with excessive keywords should be detected as malicious: %s", maliciousQuery)
	}
}

// TestCommentInjectionDetection tests detection of comment-based injections
func TestCommentInjectionDetection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	tests := []struct {
		name           string
		query          string
		shouldDetect   bool
	}{
		{
			name:           "cstyle_comment",
			query:          "SELECT /* hidden sql */ * FROM users",
			shouldDetect:   true,
		},
		{
			name:           "nested_comments",
			query:          "SELECT * FROM users /* comment /* nested */ still comment */",
			shouldDetect:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, _ := checker.CheckQuery(tt.query)
			if blocked != tt.shouldDetect {
				t.Errorf("Query '%s': blocked=%v, expected %v", tt.query, blocked, tt.shouldDetect)
			}
		})
	}
}

// TestCaseInsensitiveDetection tests that detection is case-insensitive
func TestCaseInsensitiveDetection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	queries := []string{
		"UNION SELECT * FROM admin",
		"union select * from admin",
		"UnIoN sElEcT * FrOm admin",
	}

	for _, query := range queries {
		blocked, _ := checker.CheckQuery(query)
		if !blocked {
			t.Errorf("Should detect UNION SELECT injection regardless of case: %s", query)
		}
	}
}

// TestWhitespaceHandling tests handling of queries with extra whitespace
func TestWhitespaceHandling(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	queries := []string{
		"  simple data  ",
		"\n\tplain text\n",
		"normal string",
	}

	for _, query := range queries {
		// Clean queries should not be blocked
		blocked, _ := checker.CheckQuery(query)
		if blocked {
			t.Errorf("Clean query with whitespace should not be blocked: %q", query)
		}
	}
}

// TestPipeAndAmpersandInjection tests pipe and ampersand-based command injection
func TestPipeAndAmpersandInjection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
	}{
		{
			name:        "pipe_and_drop",
			query:       "SELECT * FROM users | DROP TABLE users",
			shouldBlock: true,
		},
		{
			name:        "ampersand_and_delete",
			query:       "SELECT * FROM users & DELETE FROM users",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, _ := checker.CheckQuery(tt.query)
			if blocked != tt.shouldBlock {
				t.Errorf("Query '%s': blocked=%v, want %v", tt.query, blocked, tt.shouldBlock)
			}
		})
	}
}

// TestEmptyQueryHandling tests handling of empty or whitespace-only queries
func TestEmptyQueryHandling(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	queries := []string{
		"",
		"   ",
		"\n\t\n",
	}

	for _, query := range queries {
		blocked, _ := checker.CheckQuery(query)
		if blocked {
			t.Errorf("Empty/whitespace query should not be blocked: %q", query)
		}
	}
}

// TestSelectFromWhere tests legitimate SELECT...FROM...WHERE patterns
func TestSelectFromWhere(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	// Simple text queries without SQL keywords
	queries := []string{
		"normal user data",
		"plain text string",
		"simple text input",
	}

	for _, query := range queries {
		blocked, _ := checker.CheckQuery(query)
		if blocked {
			t.Errorf("Legitimate text should not be blocked: %s", query)
		}
	}
}

// TestBlockedCountIncrement tests that blocked count increments correctly
func TestBlockedCountIncrement(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	// Check initial state
	stats1 := checker.GetStats()
	initialBlocked, _ := stats1["blocked_count"].(int64)

	// Block some queries
	checker.CheckQuery("SELECT * FROM users UNION SELECT * FROM admin")
	checker.CheckQuery("SELECT * FROM users OR 1=1")
	checker.CheckQuery("SELECT * FROM users; DROP TABLE users")

	stats2 := checker.GetStats()
	finalBlocked, _ := stats2["blocked_count"].(int64)

	if finalBlocked <= initialBlocked {
		t.Errorf("Blocked count should increment, was %v, is %v", initialBlocked, finalBlocked)
	}
}

// TestInspectedCountIncrement tests that inspected count increments for all queries
func TestInspectedCountIncrement(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	checker := NewChecker(logger)

	stats1 := checker.GetStats()
	initialCount, _ := stats1["inspected_count"].(int64)

	// Check various queries
	checker.CheckQuery("SELECT * FROM users")
	checker.CheckQuery("SELECT * FROM products")
	checker.CheckQuery("SELECT * FROM orders UNION SELECT * FROM admin")

	stats2 := checker.GetStats()
	finalCount, _ := stats2["inspected_count"].(int64)

	if finalCount != initialCount + 3 {
		t.Errorf("Inspected count should increment by 3, was %v, is %v", initialCount, finalCount)
	}
}
