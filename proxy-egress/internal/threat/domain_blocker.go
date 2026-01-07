// Package threat provides domain blocking functionality for the egress proxy
package threat

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DomainBlocker handles domain blocking including wildcard support
type DomainBlocker struct {
	// Exact domain matches
	exactDomains map[string]BlockRule

	// Wildcard domains (*.example.com)
	wildcardDomains map[string]BlockRule

	wildcardSupport bool
	mu              sync.RWMutex
	logger          *logrus.Logger
}

// NewDomainBlocker creates a new domain blocker
func NewDomainBlocker(wildcardSupport bool, logger *logrus.Logger) *DomainBlocker {
	if logger == nil {
		logger = logrus.New()
	}

	return &DomainBlocker{
		exactDomains:    make(map[string]BlockRule),
		wildcardDomains: make(map[string]BlockRule),
		wildcardSupport: wildcardSupport,
		logger:          logger,
	}
}

// Check checks if a domain is blocked
func (b *DomainBlocker) Check(domain string) *BlockDecision {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Normalize domain to lowercase
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return &BlockDecision{
			Blocked:   false,
			Timestamp: time.Now(),
		}
	}

	// Remove port if present
	if idx := strings.LastIndex(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Check exact match first
	if rule, ok := b.exactDomains[domain]; ok {
		// Check expiration
		if !rule.ExpiresAt.IsZero() && time.Now().After(rule.ExpiresAt) {
			return &BlockDecision{
				Blocked:   false,
				Timestamp: time.Now(),
			}
		}

		return &BlockDecision{
			Blocked:     true,
			Reason:      fmt.Sprintf("domain %s is blocked", domain),
			Category:    "domain",
			MatchedRule: rule.ID,
			Timestamp:   time.Now(),
		}
	}

	// Check wildcard matches if enabled
	if b.wildcardSupport {
		if decision := b.checkWildcard(domain); decision.Blocked {
			return decision
		}
	}

	return &BlockDecision{
		Blocked:   false,
		Timestamp: time.Now(),
	}
}

// checkWildcard checks if a domain matches any wildcard pattern
func (b *DomainBlocker) checkWildcard(domain string) *BlockDecision {
	// Split domain into parts
	parts := strings.Split(domain, ".")

	// Try progressively shorter suffixes
	// e.g., for "sub.example.com", try "*.example.com", "*.com"
	for i := 1; i < len(parts); i++ {
		suffix := strings.Join(parts[i:], ".")
		wildcardPattern := "*." + suffix

		if rule, ok := b.wildcardDomains[wildcardPattern]; ok {
			// Check expiration
			if !rule.ExpiresAt.IsZero() && time.Now().After(rule.ExpiresAt) {
				continue
			}

			return &BlockDecision{
				Blocked:     true,
				Reason:      fmt.Sprintf("domain %s matches wildcard %s", domain, wildcardPattern),
				Category:    "domain",
				MatchedRule: rule.ID,
				Timestamp:   time.Now(),
			}
		}
	}

	return &BlockDecision{
		Blocked:   false,
		Timestamp: time.Now(),
	}
}

// AddRule adds a domain blocking rule
func (b *DomainBlocker) AddRule(rule BlockRule) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	pattern := strings.ToLower(strings.TrimSpace(rule.Pattern))
	if pattern == "" {
		return fmt.Errorf("empty domain pattern")
	}

	// Check if it's a wildcard pattern
	if strings.HasPrefix(pattern, "*.") {
		if !b.wildcardSupport {
			return fmt.Errorf("wildcard patterns not supported")
		}
		b.wildcardDomains[pattern] = rule
		b.logger.WithFields(logrus.Fields{
			"pattern":  pattern,
			"rule_id":  rule.ID,
			"category": rule.Category,
		}).Debug("Added wildcard domain to blocklist")
	} else {
		b.exactDomains[pattern] = rule
		b.logger.WithFields(logrus.Fields{
			"domain":   pattern,
			"rule_id":  rule.ID,
			"category": rule.Category,
		}).Debug("Added domain to blocklist")
	}

	return nil
}

// RemoveRule removes a domain blocking rule by ID
func (b *DomainBlocker) RemoveRule(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check exact domains
	for domain, rule := range b.exactDomains {
		if rule.ID == id {
			delete(b.exactDomains, domain)
			b.logger.WithField("rule_id", id).Debug("Removed domain from blocklist")
			return nil
		}
	}

	// Check wildcard domains
	for pattern, rule := range b.wildcardDomains {
		if rule.ID == id {
			delete(b.wildcardDomains, pattern)
			b.logger.WithField("rule_id", id).Debug("Removed wildcard domain from blocklist")
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", id)
}

// Clear removes all rules
func (b *DomainBlocker) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.exactDomains = make(map[string]BlockRule)
	b.wildcardDomains = make(map[string]BlockRule)

	b.logger.Info("Cleared all domain blocklist entries")
}

// Count returns the number of rules
func (b *DomainBlocker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.exactDomains) + len(b.wildcardDomains)
}

// GetRules returns all current rules
func (b *DomainBlocker) GetRules() []BlockRule {
	b.mu.RLock()
	defer b.mu.RUnlock()

	rules := make([]BlockRule, 0, b.Count())

	for _, rule := range b.exactDomains {
		rules = append(rules, rule)
	}
	for _, rule := range b.wildcardDomains {
		rules = append(rules, rule)
	}

	return rules
}

// CleanExpired removes expired rules
func (b *DomainBlocker) CleanExpired() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	removed := 0

	// Clean exact domains
	for domain, rule := range b.exactDomains {
		if !rule.ExpiresAt.IsZero() && now.After(rule.ExpiresAt) {
			delete(b.exactDomains, domain)
			removed++
		}
	}

	// Clean wildcard domains
	for pattern, rule := range b.wildcardDomains {
		if !rule.ExpiresAt.IsZero() && now.After(rule.ExpiresAt) {
			delete(b.wildcardDomains, pattern)
			removed++
		}
	}

	if removed > 0 {
		b.logger.WithField("count", removed).Debug("Cleaned expired domain blocklist entries")
	}

	return removed
}

// IsWildcardSupported returns whether wildcard patterns are supported
func (b *DomainBlocker) IsWildcardSupported() bool {
	return b.wildcardSupport
}
