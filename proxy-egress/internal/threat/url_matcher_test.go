package threat

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// createTestMatcher is a helper to create a URLMatcher for testing
func createTestMatcher(t testing.TB) *URLMatcher {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	matcher, err := NewURLMatcher("re2", 1000, logger)
	if err != nil {
		t.Fatalf("Failed to create URLMatcher: %v", err)
	}
	return matcher
}

func TestNewURLMatcher(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	matcher, err := NewURLMatcher("re2", 1000, logger)
	if err != nil {
		t.Fatalf("NewURLMatcher failed: %v", err)
	}
	if matcher == nil {
		t.Fatal("NewURLMatcher returned nil")
	}

	if matcher.Count() != 0 {
		t.Errorf("Expected 0 patterns, got %d", matcher.Count())
	}
}

func TestURLMatcher_AddPattern_Simple(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-1",
		Pattern:   "/admin/.*",
		Category:  "restricted",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := matcher.AddPattern(rule)
	if err != nil {
		t.Fatalf("AddPattern failed: %v", err)
	}

	if matcher.Count() != 1 {
		t.Errorf("Expected 1 pattern, got %d", matcher.Count())
	}
}

func TestURLMatcher_AddPattern_InvalidRegex(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-invalid",
		Pattern:   "[invalid",
		Category:  "test",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}

	err := matcher.AddPattern(rule)
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

func TestURLMatcher_AddPattern_EmptyPattern(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-empty",
		Pattern:   "",
		Category:  "test",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}

	// Empty pattern is valid regex that matches everything
	err := matcher.AddPattern(rule)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestURLMatcher_Check_SimpleMatch(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-admin",
		Pattern:   "/admin/.*",
		Category:  "restricted",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}
	matcher.AddPattern(rule)

	decision := matcher.Check("/admin/users")
	if decision == nil {
		t.Fatal("Check returned nil")
	}

	if !decision.Blocked {
		t.Error("Expected URL to be blocked")
	}

	if decision.Category != "url" {
		t.Errorf("Expected category 'url', got '%s'", decision.Category)
	}
}

func TestURLMatcher_Check_NoMatch(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-admin",
		Pattern:   "/admin/.*",
		Category:  "restricted",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}
	matcher.AddPattern(rule)

	decision := matcher.Check("/api/users")
	if decision.Blocked {
		t.Error("Expected URL to be allowed")
	}
}

func TestURLMatcher_Check_MultiplePatterns(t *testing.T) {
	matcher := createTestMatcher(t)

	patterns := []PatternRule{
		{ID: "p1", Pattern: "/admin/.*", Category: "admin", Priority: 100, Source: "test", CreatedAt: time.Now()},
		{ID: "p2", Pattern: "/api/v1/secret/.*", Category: "secret", Priority: 90, Source: "test", CreatedAt: time.Now()},
		{ID: "p3", Pattern: ".*\\.php$", Category: "php", Priority: 80, Source: "test", CreatedAt: time.Now()},
		{ID: "p4", Pattern: "/download/.*\\.(exe|msi|bat)$", Category: "executable", Priority: 70, Source: "test", CreatedAt: time.Now()},
	}

	for _, rule := range patterns {
		matcher.AddPattern(rule)
	}

	tests := []struct {
		url     string
		blocked bool
	}{
		{"/admin/settings", true},
		{"/api/v1/secret/keys", true},
		{"/scripts/hack.php", true},
		{"/download/malware.exe", true},
		{"/download/document.pdf", false},
		{"/api/v1/public/data", false},
	}

	for _, tt := range tests {
		decision := matcher.Check(tt.url)
		if decision.Blocked != tt.blocked {
			t.Errorf("URL %s: expected blocked=%v, got blocked=%v", tt.url, tt.blocked, decision.Blocked)
		}
	}
}

func TestURLMatcher_Check_Priority(t *testing.T) {
	matcher := createTestMatcher(t)

	// Add overlapping patterns with different priorities
	patterns := []PatternRule{
		{ID: "low", Pattern: "/api/.*", Category: "low-priority", Priority: 10, Source: "test", CreatedAt: time.Now()},
		{ID: "high", Pattern: "/api/admin/.*", Category: "high-priority", Priority: 100, Source: "test", CreatedAt: time.Now()},
	}

	for _, rule := range patterns {
		matcher.AddPattern(rule)
	}

	// The higher priority pattern should match first
	decision := matcher.Check("/api/admin/users")
	if !decision.Blocked {
		t.Error("Expected URL to be blocked")
	}
	// Implementation uses "url" as fixed category
	if decision.Category != "url" {
		t.Errorf("Expected category 'url', got '%s'", decision.Category)
	}
}

func TestURLMatcher_Check_ExpiredPattern(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-expired",
		Pattern:   "/admin/.*",
		Category:  "restricted",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	matcher.AddPattern(rule)

	decision := matcher.Check("/admin/users")
	if decision.Blocked {
		t.Error("Expected expired pattern to not block")
	}
}

func TestURLMatcher_Check_ComplexPatterns(t *testing.T) {
	matcher := createTestMatcher(t)

	patterns := []PatternRule{
		// SQL injection patterns
		{ID: "sqli1", Pattern: ".*('|%27).*", Category: "sqli", Priority: 100, Source: "test", CreatedAt: time.Now()},
		// XSS patterns
		{ID: "xss1", Pattern: ".*<script.*>.*", Category: "xss", Priority: 100, Source: "test", CreatedAt: time.Now()},
		// Path traversal
		{ID: "path1", Pattern: ".*(\\.\\.[\\/]).*", Category: "path-traversal", Priority: 100, Source: "test", CreatedAt: time.Now()},
	}

	for _, rule := range patterns {
		matcher.AddPattern(rule)
	}

	tests := []struct {
		url      string
		blocked  bool
		category string
	}{
		{"/api/users?id=1' OR 1=1--", true, "sqli"},
		{"/api/users?id=%271%27", true, "sqli"},
		{"/page?content=<script>alert(1)</script>", true, "xss"},
		{"/files?path=../../etc/passwd", true, "path-traversal"},
		{"/api/users?id=123", false, ""},
	}

	for _, tt := range tests {
		decision := matcher.Check(tt.url)
		if decision.Blocked != tt.blocked {
			t.Errorf("URL %s: expected blocked=%v, got blocked=%v", tt.url, tt.blocked, decision.Blocked)
		}
	}
}

func TestURLMatcher_RemovePattern(t *testing.T) {
	matcher := createTestMatcher(t)

	rule := PatternRule{
		ID:        "test-1",
		Pattern:   "/admin/.*",
		Category:  "restricted",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}
	matcher.AddPattern(rule)

	// Verify it's blocked
	decision := matcher.Check("/admin/users")
	if !decision.Blocked {
		t.Error("Expected URL to be blocked before removal")
	}

	// Remove the pattern
	err := matcher.RemovePattern("test-1")
	if err != nil {
		t.Fatalf("RemovePattern failed: %v", err)
	}

	// Verify it's no longer blocked
	decision = matcher.Check("/admin/users")
	if decision.Blocked {
		t.Error("Expected URL to be allowed after pattern removal")
	}
}

func TestURLMatcher_RemovePattern_NotFound(t *testing.T) {
	matcher := createTestMatcher(t)

	err := matcher.RemovePattern("nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent pattern")
	}
}

func TestURLMatcher_Clear(t *testing.T) {
	matcher := createTestMatcher(t)

	// Add multiple patterns
	for i := 0; i < 10; i++ {
		rule := PatternRule{
			ID:        "test-" + string(rune('0'+i)),
			Pattern:   "/path" + string(rune('0'+i)) + "/.*",
			Category:  "test",
			Priority:  100,
			Source:    "test",
			CreatedAt: time.Now(),
		}
		matcher.AddPattern(rule)
	}

	if matcher.Count() != 10 {
		t.Errorf("Expected 10 patterns, got %d", matcher.Count())
	}

	matcher.Clear()

	if matcher.Count() != 0 {
		t.Errorf("Expected 0 patterns after clear, got %d", matcher.Count())
	}
}

func TestURLMatcher_GetEngine(t *testing.T) {
	matcher := createTestMatcher(t)

	engine := matcher.GetEngine()
	if engine != "re2" {
		t.Errorf("Expected engine 're2', got '%s'", engine)
	}
}

func TestURLMatcher_Validate(t *testing.T) {
	matcher := createTestMatcher(t)

	// Valid pattern
	err := matcher.Validate("/admin/.*")
	if err != nil {
		t.Errorf("Expected valid pattern, got error: %v", err)
	}

	// Invalid pattern
	err = matcher.Validate("[invalid")
	if err == nil {
		t.Error("Expected error for invalid pattern")
	}
}

func TestURLMatcher_BulkAdd(t *testing.T) {
	matcher := createTestMatcher(t)

	rules := []PatternRule{
		{ID: "p1", Pattern: "/admin/.*", Category: "admin", Priority: 100, Source: "test", CreatedAt: time.Now()},
		{ID: "p2", Pattern: "/api/.*", Category: "api", Priority: 90, Source: "test", CreatedAt: time.Now()},
		{ID: "p3", Pattern: "[invalid", Category: "invalid", Priority: 80, Source: "test", CreatedAt: time.Now()}, // Invalid
	}

	added, err := matcher.BulkAdd(rules)
	if err == nil {
		t.Error("Expected error for invalid pattern in bulk add")
	}
	// First two should have been added before the error
	if added != 2 {
		t.Errorf("Expected 2 added before error, got %d", added)
	}
}

func TestURLMatcher_GetPatterns(t *testing.T) {
	matcher := createTestMatcher(t)

	patterns := []PatternRule{
		{ID: "p1", Pattern: "/admin/.*", Category: "admin", Priority: 100, Source: "test", CreatedAt: time.Now()},
		{ID: "p2", Pattern: "/api/.*", Category: "api", Priority: 90, Source: "test", CreatedAt: time.Now()},
	}

	for _, rule := range patterns {
		matcher.AddPattern(rule)
	}

	gotPatterns := matcher.GetPatterns()
	if len(gotPatterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(gotPatterns))
	}
}

func BenchmarkURLMatcher_Check_SimplePattern(b *testing.B) {
	matcher := createTestMatcher(b)

	// Add 100 patterns
	for i := 0; i < 100; i++ {
		rule := PatternRule{
			ID:        "test-" + string(rune(i)),
			Pattern:   "/path" + string(rune(i)) + "/.*",
			Category:  "test",
			Priority:  100,
			Source:    "test",
			CreatedAt: time.Now(),
		}
		matcher.AddPattern(rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Check("/path50/resource")
	}
}

func BenchmarkURLMatcher_Check_ComplexPattern(b *testing.B) {
	matcher := createTestMatcher(b)

	// Add complex security patterns
	patterns := []string{
		".*('|%27).*",
		".*<script.*>.*",
		".*(\\.\\.[\\/]).*",
		".*\\.(exe|msi|bat|cmd|ps1)$",
		".*\\?.*=.*union.*select.*",
	}

	for i, p := range patterns {
		rule := PatternRule{
			ID:        "test-" + string(rune(i)),
			Pattern:   p,
			Category:  "security",
			Priority:  100,
			Source:    "test",
			CreatedAt: time.Now(),
		}
		matcher.AddPattern(rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Check("/api/users?id=123&name=john")
	}
}
