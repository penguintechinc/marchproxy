//go:build ci

package security

import (
	"marchproxy-dblb/internal/logging"
	"testing"
)

func TestNewChecker(t *testing.T) {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	checker := NewChecker(logger)
	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}

	if checker.logger == nil {
		t.Error("logger not set")
	}

	if len(checker.patterns) == 0 {
		t.Error("patterns should be initialized")
	}

	stats := checker.GetStats()
	if stats["inspected_count"].(int64) != 0 {
		t.Error("initial inspected_count should be 0")
	}

	if stats["blocked_count"].(int64) != 0 {
		t.Error("initial blocked_count should be 0")
	}

	if stats["patterns_loaded"].(int) == 0 {
		t.Error("patterns should be loaded")
	}
}

func TestCheckQuery_CleanQueries(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"numeric only", "12345", false},
		{"normal text", "hello world", false},
		{"simple name", "john doe", false},
		{"email format", "user@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSQL, _ := checker.CheckQuery(tt.query)
			if isSQL != tt.want {
				t.Errorf("CheckQuery(%q) = %v, want %v", tt.query, isSQL, tt.want)
			}
		})
	}
}

func TestCheckQuery_SQLInjection_UnionSelect(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"union select", "SELECT id FROM users UNION SELECT password FROM admin", true},
		{"union select from", "SELECT * FROM users UNION SELECT * FROM passwords", true},
		{"union with where", "SELECT id FROM users WHERE id=1 UNION SELECT id FROM admin", true},
		{"multiple unions", "SELECT 1 UNION SELECT 2 UNION SELECT 3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSQL, _ := checker.CheckQuery(tt.query)
			if isSQL != tt.want {
				t.Errorf("CheckQuery(%q) = %v, want %v", tt.query, isSQL, tt.want)
			}
		})
	}
}

func TestCheckQuery_SQLInjection_DropTable(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"drop table", "DROP TABLE users", true},
		{"drop with semicolon", "SELECT * FROM users; DROP TABLE users;", true},
		{"drop with pipe", "SELECT id FROM users | DROP TABLE admin", true},
		{"drop with ampersand", "SELECT * FROM users & DROP TABLE users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSQL, _ := checker.CheckQuery(tt.query)
			if isSQL != tt.want {
				t.Errorf("CheckQuery(%q) = %v, want %v", tt.query, isSQL, tt.want)
			}
		})
	}
}

func TestCheckQuery_SQLInjection_CommentInjection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"double dash comment", "SELECT * FROM users --", true},
		{"block comment", "SELECT * FROM users /* comment */", true},
		{"hash comment", "SELECT * FROM users #comment", true},
		{"block comment unclosed", "SELECT * FROM users /*", true},
		{";-- injection", "SELECT * FROM users; --", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSQL, _ := checker.CheckQuery(tt.query)
			if isSQL != tt.want {
				t.Errorf("CheckQuery(%q) = %v, want %v", tt.query, isSQL, tt.want)
			}
		})
	}
}

func TestCheckQuery_CaseInsensitivity(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"uppercase UNION", "SELECT id FROM users UNION SELECT password FROM admin", true},
		{"lowercase union", "select id from users union select password from admin", true},
		{"mixed case Union", "SeLeCt id FrOm users UnIoN SeLeCt password FrOm admin", true},
		{"uppercase DROP", "DROP TABLE users", true},
		{"lowercase drop", "drop table users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSQL, _ := checker.CheckQuery(tt.query)
			if isSQL != tt.want {
				t.Errorf("CheckQuery(%q) = %v, want %v", tt.query, isSQL, tt.want)
			}
		})
	}
}

func TestCheckData(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		data  []byte
		want  bool
	}{
		{"clean data", []byte("hello world"), false},
		{"sql injection in bytes", []byte("SELECT * FROM users UNION SELECT password"), true},
		{"comment injection", []byte("SELECT * FROM users --"), true},
		{"empty bytes", []byte(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSQL, _ := checker.CheckData(tt.data)
			if isSQL != tt.want {
				t.Errorf("CheckData(%q) = %v, want %v", string(tt.data), isSQL, tt.want)
			}
		})
	}
}

func TestHasExcessiveSQLKeywords(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"no keywords", "hello world", false},
		{"one keyword", "select name", false},
		{"two keywords", "select insert", false},
		{"three keywords", "select insert update", false},
		{"four keywords", "select insert update delete", true},
		{"five keywords", "select union insert update delete", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExcessive := checker.hasExcessiveSQLKeywords(tt.query)
			if isExcessive != tt.want {
				t.Errorf("hasExcessiveSQLKeywords(%q) = %v, want %v", tt.query, isExcessive, tt.want)
			}
		})
	}
}

func TestHasCommentInjection(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"no comment", "hello world test", false},
		{"double dash", "SELECT * --", true},
		{"block start", "SELECT /* comment */", true},
		{"block end", "SELECT */", true},
		{"hash comment", "SELECT #", true},
		{"semicolon dash", "SELECT; --", true},
		{"multiple comments", "/* start */ SELECT --", true},
		{"dash in hyphen", "my-name-is-test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasComment := checker.hasCommentInjection(tt.query)
			if hasComment != tt.want {
				t.Errorf("hasCommentInjection(%q) = %v, want %v", tt.query, hasComment, tt.want)
			}
		})
	}
}

func TestCheckerGetStatsFull(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	checker.CheckQuery("hello world")
	checker.CheckQuery("DROP TABLE users --")

	stats := checker.GetStats()
	if stats["inspected_count"].(int64) != 2 {
		t.Errorf("inspected_count = %d, want 2", stats["inspected_count"].(int64))
	}
	if stats["blocked_count"].(int64) != 1 {
		t.Errorf("blocked_count = %d, want 1", stats["blocked_count"].(int64))
	}
}

func TestReset(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	checker.CheckQuery("hello world")
	checker.CheckQuery("DROP TABLE users")

	stats := checker.GetStats()
	if stats["inspected_count"].(int64) != 2 {
		t.Errorf("inspected_count before reset = %d, want 2", stats["inspected_count"].(int64))
	}

	checker.Reset()

	stats = checker.GetStats()
	if stats["inspected_count"].(int64) != 0 {
		t.Error("inspected_count should be 0 after reset")
	}
	if stats["blocked_count"].(int64) != 0 {
		t.Error("blocked_count should be 0 after reset")
	}
}

func TestAddPattern(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	initialCount := len(checker.patterns)

	err := checker.AddPattern(`(?i)custom_injection`)
	if err != nil {
		t.Errorf("AddPattern with valid regex failed: %v", err)
	}

	if len(checker.patterns) != initialCount+1 {
		t.Error("pattern count should increase by 1")
	}

	err = checker.AddPattern(`(?i)(unclosed[regex`)
	if err == nil {
		t.Error("AddPattern with invalid regex should fail")
	}
}

func TestRemovePattern(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	initialCount := len(checker.patterns)

	err := checker.RemovePattern(0)
	if err != nil {
		t.Errorf("RemovePattern with valid index failed: %v", err)
	}

	if len(checker.patterns) != initialCount-1 {
		t.Error("pattern count should decrease by 1")
	}
}

func TestIsWhitelisted(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	query := "DROP TABLE users"
	whitelist := []string{"DROP TABLE users"}

	if !checker.IsWhitelisted(query, whitelist) {
		t.Error("IsWhitelisted should match query in whitelist")
	}

	whitelist = []string{"DELETE FROM users"}
	if checker.IsWhitelisted(query, whitelist) {
		t.Error("IsWhitelisted should not match query not in whitelist")
	}
}

func TestConcurrentCheckQuery(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	checker := NewChecker(logger)

	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 10; i++ {
			checker.CheckQuery("hello world test")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			checker.CheckQuery("SELECT id FROM users UNION SELECT password FROM admin")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			checker.GetStats()
		}
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}

	stats := checker.GetStats()
	if stats["inspected_count"].(int64) != 20 {
		t.Errorf("inspected_count = %d, want 20", stats["inspected_count"].(int64))
	}
	if stats["blocked_count"].(int64) != 10 {
		t.Errorf("blocked_count = %d, want 10", stats["blocked_count"].(int64))
	}
}
