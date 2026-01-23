// Package threat provides URL pattern matching functionality for the egress proxy
package threat

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PatternRule represents a URL pattern blocking rule
type PatternRule struct {
	ID        string    `json:"id"`
	Pattern   string    `json:"pattern"`  // Regex pattern
	Category  string    `json:"category"` // "malware", "phishing", "restricted", etc.
	Priority  int       `json:"priority"` // Higher priority = checked first
	Source    string    `json:"source"`   // "manager", "feed:name"
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// compiledPattern holds a compiled regex pattern with its rule
type compiledPattern struct {
	rule    PatternRule
	regex   *regexp.Regexp
	matched int64 // Match count for statistics
}

// URLMatcher handles URL pattern matching using regex
type URLMatcher struct {
	patterns    []*compiledPattern
	engine      string // "re2" (Go's default)
	maxPatterns int
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// NewURLMatcher creates a new URL pattern matcher
func NewURLMatcher(engine string, maxPatterns int, logger *logrus.Logger) (*URLMatcher, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Currently only RE2 (Go's regexp) is supported
	if engine != "re2" && engine != "" {
		logger.WithField("engine", engine).Warn("Requested engine not supported, using re2")
	}

	return &URLMatcher{
		patterns:    make([]*compiledPattern, 0),
		engine:      "re2",
		maxPatterns: maxPatterns,
		logger:      logger,
	}, nil
}

// Check checks if a URL matches any blocked pattern
func (m *URLMatcher) Check(url string) *BlockDecision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if url == "" {
		return &BlockDecision{
			Blocked:   false,
			Timestamp: time.Now(),
		}
	}

	// Check patterns in priority order (highest first)
	for _, cp := range m.patterns {
		// Check expiration
		if !cp.rule.ExpiresAt.IsZero() && time.Now().After(cp.rule.ExpiresAt) {
			continue
		}

		if cp.regex.MatchString(url) {
			return &BlockDecision{
				Blocked:     true,
				Reason:      fmt.Sprintf("URL matches pattern %s", cp.rule.Pattern),
				Category:    "url",
				MatchedRule: cp.rule.ID,
				Timestamp:   time.Now(),
			}
		}
	}

	return &BlockDecision{
		Blocked:   false,
		Timestamp: time.Now(),
	}
}

// AddPattern adds a URL pattern rule
func (m *URLMatcher) AddPattern(rule PatternRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check capacity
	if len(m.patterns) >= m.maxPatterns {
		return fmt.Errorf("pattern capacity exceeded (%d)", m.maxPatterns)
	}

	// Compile the pattern
	regex, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", rule.Pattern, err)
	}

	cp := &compiledPattern{
		rule:  rule,
		regex: regex,
	}

	// Insert in priority order (highest first)
	inserted := false
	for i, existing := range m.patterns {
		if rule.Priority > existing.rule.Priority {
			m.patterns = append(m.patterns[:i], append([]*compiledPattern{cp}, m.patterns[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		m.patterns = append(m.patterns, cp)
	}

	m.logger.WithFields(logrus.Fields{
		"pattern":  rule.Pattern,
		"rule_id":  rule.ID,
		"category": rule.Category,
		"priority": rule.Priority,
	}).Debug("Added URL pattern to blocklist")

	return nil
}

// RemovePattern removes a URL pattern by ID
func (m *URLMatcher) RemovePattern(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, cp := range m.patterns {
		if cp.rule.ID == id {
			m.patterns = append(m.patterns[:i], m.patterns[i+1:]...)
			m.logger.WithField("rule_id", id).Debug("Removed URL pattern from blocklist")
			return nil
		}
	}

	return fmt.Errorf("pattern not found: %s", id)
}

// Clear removes all patterns
func (m *URLMatcher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.patterns = make([]*compiledPattern, 0)
	m.logger.Info("Cleared all URL patterns")
}

// Count returns the number of patterns
func (m *URLMatcher) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.patterns)
}

// GetPatterns returns all current patterns
func (m *URLMatcher) GetPatterns() []PatternRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]PatternRule, len(m.patterns))
	for i, cp := range m.patterns {
		rules[i] = cp.rule
	}

	return rules
}

// CleanExpired removes expired patterns
func (m *URLMatcher) CleanExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0

	newPatterns := make([]*compiledPattern, 0, len(m.patterns))
	for _, cp := range m.patterns {
		if cp.rule.ExpiresAt.IsZero() || now.Before(cp.rule.ExpiresAt) {
			newPatterns = append(newPatterns, cp)
		} else {
			removed++
		}
	}
	m.patterns = newPatterns

	if removed > 0 {
		m.logger.WithField("count", removed).Debug("Cleaned expired URL patterns")
	}

	return removed
}

// GetEngine returns the regex engine being used
func (m *URLMatcher) GetEngine() string {
	return m.engine
}

// GetMaxPatterns returns the maximum number of patterns allowed
func (m *URLMatcher) GetMaxPatterns() int {
	return m.maxPatterns
}

// Validate validates a pattern without adding it
func (m *URLMatcher) Validate(pattern string) error {
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	return nil
}

// BulkAdd adds multiple patterns at once (more efficient than individual adds)
func (m *URLMatcher) BulkAdd(rules []PatternRule) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	added := 0
	var lastErr error

	for _, rule := range rules {
		// Check capacity
		if len(m.patterns) >= m.maxPatterns {
			return added, fmt.Errorf("pattern capacity exceeded (%d) after adding %d patterns", m.maxPatterns, added)
		}

		// Compile the pattern
		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			lastErr = fmt.Errorf("invalid regex pattern '%s': %w", rule.Pattern, err)
			continue
		}

		cp := &compiledPattern{
			rule:  rule,
			regex: regex,
		}
		m.patterns = append(m.patterns, cp)
		added++
	}

	// Sort by priority after bulk add
	m.sortByPriority()

	m.logger.WithField("count", added).Debug("Bulk added URL patterns")

	return added, lastErr
}

// sortByPriority sorts patterns by priority (highest first)
func (m *URLMatcher) sortByPriority() {
	// Simple insertion sort (patterns list is typically small)
	for i := 1; i < len(m.patterns); i++ {
		key := m.patterns[i]
		j := i - 1
		for j >= 0 && m.patterns[j].rule.Priority < key.rule.Priority {
			m.patterns[j+1] = m.patterns[j]
			j--
		}
		m.patterns[j+1] = key
	}
}
