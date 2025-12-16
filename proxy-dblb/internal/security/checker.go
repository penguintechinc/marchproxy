package security

import (
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Checker implements SQL injection detection
type Checker struct {
	patterns       []*regexp.Regexp
	blockedCount   int64
	inspectedCount int64
	logger         *logrus.Logger
	mu             sync.RWMutex
}

// NewChecker creates a new security checker
func NewChecker(logger *logrus.Logger) *Checker {
	checker := &Checker{
		logger: logger,
	}

	// Compile common SQL injection patterns
	checker.patterns = []*regexp.Regexp{
		// SQL injection patterns
		regexp.MustCompile(`(?i)(\b(union|select|insert|update|delete|drop|create|alter|exec|execute)\b.*\b(from|into|where|table|database)\b)`),
		regexp.MustCompile(`(?i)('|\")(\s)*(or|and)(\s)*('|\")?(\s)*=(\s)*('|\")?`),
		regexp.MustCompile(`(?i)(;|\||&)(\s)*(drop|delete|update|insert|create|alter|exec|execute)`),
		regexp.MustCompile(`(?i)(/\*|\*/|--|\#|xp_cmdshell|sp_executesql)`),
		regexp.MustCompile(`(?i)(script|javascript|onerror|onload|eval|expression|vbscript)`),
		regexp.MustCompile(`(?i)(\bor\b|\band\b)(\s)+[\d\w]+(\s)*=(\s)*[\d\w]+`),
		regexp.MustCompile(`(?i)(union.*select|select.*from.*where)`),
		regexp.MustCompile(`(?i)(benchmark|sleep|waitfor|delay)\s*\(`),
	}

	return checker
}

// CheckQuery inspects a query for potential SQL injection
func (c *Checker) CheckQuery(query string) (bool, string) {
	c.mu.Lock()
	c.inspectedCount++
	c.mu.Unlock()

	// Normalize query for inspection
	normalized := strings.TrimSpace(strings.ToLower(query))

	// Check against patterns
	for _, pattern := range c.patterns {
		if pattern.MatchString(normalized) {
			c.mu.Lock()
			c.blockedCount++
			c.mu.Unlock()

			reason := "Potential SQL injection detected: pattern match"
			c.logger.WithFields(logrus.Fields{
				"query":   query,
				"pattern": pattern.String(),
			}).Warn(reason)

			return true, reason
		}
	}

	// Additional heuristic checks
	if c.hasExcessiveSQLKeywords(normalized) {
		c.mu.Lock()
		c.blockedCount++
		c.mu.Unlock()

		reason := "Potential SQL injection detected: excessive SQL keywords"
		c.logger.WithField("query", query).Warn(reason)
		return true, reason
	}

	if c.hasCommentInjection(normalized) {
		c.mu.Lock()
		c.blockedCount++
		c.mu.Unlock()

		reason := "Potential SQL injection detected: comment injection"
		c.logger.WithField("query", query).Warn(reason)
		return true, reason
	}

	return false, ""
}

// CheckData inspects data for malicious content
func (c *Checker) CheckData(data []byte) (bool, string) {
	// Convert to string and check
	return c.CheckQuery(string(data))
}

// hasExcessiveSQLKeywords checks for excessive SQL keywords in query
func (c *Checker) hasExcessiveSQLKeywords(query string) bool {
	keywords := []string{
		"select", "union", "insert", "update", "delete",
		"drop", "create", "alter", "exec", "execute",
		"declare", "cast", "convert", "concat",
	}

	count := 0
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			count++
		}
	}

	// If more than 3 different SQL keywords, flag as suspicious
	return count > 3
}

// hasCommentInjection checks for comment-based injection attempts
func (c *Checker) hasCommentInjection(query string) bool {
	commentPatterns := []string{
		"--",
		"/*",
		"*/",
		"#",
		";--",
	}

	for _, pattern := range commentPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	return false
}

// GetStats returns security checker statistics
func (c *Checker) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"inspected_count": c.inspectedCount,
		"blocked_count":   c.blockedCount,
		"patterns_loaded": len(c.patterns),
	}
}

// Reset resets the counters
func (c *Checker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.blockedCount = 0
	c.inspectedCount = 0

	c.logger.Info("Security checker counters reset")
}

// AddPattern adds a custom pattern to the checker
func (c *Checker) AddPattern(pattern string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.patterns = append(c.patterns, compiled)

	c.logger.WithField("pattern", pattern).Info("Custom pattern added")
	return nil
}

// RemovePattern removes a pattern by index
func (c *Checker) RemovePattern(index int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index < 0 || index >= len(c.patterns) {
		return nil
	}

	c.patterns = append(c.patterns[:index], c.patterns[index+1:]...)

	c.logger.WithField("index", index).Info("Pattern removed")
	return nil
}

// IsWhitelisted checks if a query matches a whitelist pattern
func (c *Checker) IsWhitelisted(query string, whitelist []string) bool {
	normalized := strings.TrimSpace(strings.ToLower(query))

	for _, pattern := range whitelist {
		if strings.Contains(normalized, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}
