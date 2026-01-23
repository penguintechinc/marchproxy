// Package threat provides IP blocking functionality for the egress proxy
package threat

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BlockRule represents a blocking rule
type BlockRule struct {
	ID        string    `json:"id"`
	Pattern   string    `json:"pattern"`   // IP address or CIDR
	Category  string    `json:"category"`  // "malware", "botnet", "tor", etc.
	Source    string    `json:"source"`    // "manager", "feed:name"
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// IPBlocker handles IP and CIDR blocking
type IPBlocker struct {
	// Exact IP matches
	exactIPv4 map[string]BlockRule
	exactIPv6 map[string]BlockRule

	// CIDR ranges - stored as parsed networks
	cidrIPv4 []*cidrEntry
	cidrIPv6 []*cidrEntry

	maxSize int
	mu      sync.RWMutex
	logger  *logrus.Logger
}

// cidrEntry represents a CIDR range with its associated rule
type cidrEntry struct {
	network *net.IPNet
	rule    BlockRule
}

// NewIPBlocker creates a new IP blocker
func NewIPBlocker(maxSize int, logger *logrus.Logger) *IPBlocker {
	if logger == nil {
		logger = logrus.New()
	}

	return &IPBlocker{
		exactIPv4: make(map[string]BlockRule),
		exactIPv6: make(map[string]BlockRule),
		cidrIPv4:  make([]*cidrEntry, 0),
		cidrIPv6:  make([]*cidrEntry, 0),
		maxSize:   maxSize,
		logger:    logger,
	}
}

// Check checks if an IP address is blocked
func (b *IPBlocker) Check(ip string) *BlockDecision {
	b.mu.RLock()
	defer b.mu.RUnlock()

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return &BlockDecision{
			Blocked:   false,
			Reason:    "invalid IP address",
			Timestamp: time.Now(),
		}
	}

	// Check exact matches first (faster)
	isIPv4 := parsedIP.To4() != nil
	var exactMap map[string]BlockRule
	var cidrList []*cidrEntry

	if isIPv4 {
		exactMap = b.exactIPv4
		cidrList = b.cidrIPv4
	} else {
		exactMap = b.exactIPv6
		cidrList = b.cidrIPv6
	}

	// Check exact IP match
	if rule, ok := exactMap[ip]; ok {
		// Check expiration
		if !rule.ExpiresAt.IsZero() && time.Now().After(rule.ExpiresAt) {
			return &BlockDecision{
				Blocked:   false,
				Timestamp: time.Now(),
			}
		}

		return &BlockDecision{
			Blocked:     true,
			Reason:      fmt.Sprintf("IP %s is blocked", ip),
			Category:    "ip",
			MatchedRule: rule.ID,
			Timestamp:   time.Now(),
		}
	}

	// Check CIDR ranges
	for _, entry := range cidrList {
		if entry.network.Contains(parsedIP) {
			// Check expiration
			if !entry.rule.ExpiresAt.IsZero() && time.Now().After(entry.rule.ExpiresAt) {
				continue
			}

			return &BlockDecision{
				Blocked:     true,
				Reason:      fmt.Sprintf("IP %s is in blocked CIDR %s", ip, entry.rule.Pattern),
				Category:    "ip",
				MatchedRule: entry.rule.ID,
				Timestamp:   time.Now(),
			}
		}
	}

	return &BlockDecision{
		Blocked:   false,
		Timestamp: time.Now(),
	}
}

// AddRule adds a blocking rule
func (b *IPBlocker) AddRule(rule BlockRule) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if it's a CIDR or single IP
	if _, ipNet, err := net.ParseCIDR(rule.Pattern); err == nil {
		// It's a CIDR
		return b.addCIDR(ipNet, rule)
	}

	// Try parsing as a single IP
	ip := net.ParseIP(rule.Pattern)
	if ip == nil {
		return fmt.Errorf("invalid IP or CIDR: %s", rule.Pattern)
	}

	// Add as single IP
	return b.addIP(ip, rule)
}

// addIP adds a single IP address to the blocklist
func (b *IPBlocker) addIP(ip net.IP, rule BlockRule) error {
	ipStr := ip.String()

	if ip.To4() != nil {
		// Check capacity
		if len(b.exactIPv4)+len(b.cidrIPv4) >= b.maxSize {
			return fmt.Errorf("blocklist capacity exceeded (%d)", b.maxSize)
		}
		b.exactIPv4[ipStr] = rule
		b.logger.WithFields(logrus.Fields{
			"ip":       ipStr,
			"rule_id":  rule.ID,
			"category": rule.Category,
		}).Debug("Added IPv4 to blocklist")
	} else {
		// Check capacity
		if len(b.exactIPv6)+len(b.cidrIPv6) >= b.maxSize {
			return fmt.Errorf("blocklist capacity exceeded (%d)", b.maxSize)
		}
		b.exactIPv6[ipStr] = rule
		b.logger.WithFields(logrus.Fields{
			"ip":       ipStr,
			"rule_id":  rule.ID,
			"category": rule.Category,
		}).Debug("Added IPv6 to blocklist")
	}

	return nil
}

// addCIDR adds a CIDR range to the blocklist
func (b *IPBlocker) addCIDR(ipNet *net.IPNet, rule BlockRule) error {
	entry := &cidrEntry{
		network: ipNet,
		rule:    rule,
	}

	// Determine if it's IPv4 or IPv6
	if ipNet.IP.To4() != nil {
		// Check capacity
		if len(b.exactIPv4)+len(b.cidrIPv4) >= b.maxSize {
			return fmt.Errorf("blocklist capacity exceeded (%d)", b.maxSize)
		}
		b.cidrIPv4 = append(b.cidrIPv4, entry)
		b.logger.WithFields(logrus.Fields{
			"cidr":     rule.Pattern,
			"rule_id":  rule.ID,
			"category": rule.Category,
		}).Debug("Added IPv4 CIDR to blocklist")
	} else {
		// Check capacity
		if len(b.exactIPv6)+len(b.cidrIPv6) >= b.maxSize {
			return fmt.Errorf("blocklist capacity exceeded (%d)", b.maxSize)
		}
		b.cidrIPv6 = append(b.cidrIPv6, entry)
		b.logger.WithFields(logrus.Fields{
			"cidr":     rule.Pattern,
			"rule_id":  rule.ID,
			"category": rule.Category,
		}).Debug("Added IPv6 CIDR to blocklist")
	}

	return nil
}

// RemoveRule removes a blocking rule by ID
func (b *IPBlocker) RemoveRule(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check exact IPv4
	for ip, rule := range b.exactIPv4 {
		if rule.ID == id {
			delete(b.exactIPv4, ip)
			b.logger.WithField("rule_id", id).Debug("Removed IPv4 from blocklist")
			return nil
		}
	}

	// Check exact IPv6
	for ip, rule := range b.exactIPv6 {
		if rule.ID == id {
			delete(b.exactIPv6, ip)
			b.logger.WithField("rule_id", id).Debug("Removed IPv6 from blocklist")
			return nil
		}
	}

	// Check CIDR IPv4
	for i, entry := range b.cidrIPv4 {
		if entry.rule.ID == id {
			b.cidrIPv4 = append(b.cidrIPv4[:i], b.cidrIPv4[i+1:]...)
			b.logger.WithField("rule_id", id).Debug("Removed IPv4 CIDR from blocklist")
			return nil
		}
	}

	// Check CIDR IPv6
	for i, entry := range b.cidrIPv6 {
		if entry.rule.ID == id {
			b.cidrIPv6 = append(b.cidrIPv6[:i], b.cidrIPv6[i+1:]...)
			b.logger.WithField("rule_id", id).Debug("Removed IPv6 CIDR from blocklist")
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", id)
}

// Clear removes all rules
func (b *IPBlocker) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.exactIPv4 = make(map[string]BlockRule)
	b.exactIPv6 = make(map[string]BlockRule)
	b.cidrIPv4 = make([]*cidrEntry, 0)
	b.cidrIPv6 = make([]*cidrEntry, 0)

	b.logger.Info("Cleared all IP blocklist entries")
}

// Count returns the number of rules
func (b *IPBlocker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.exactIPv4) + len(b.exactIPv6) + len(b.cidrIPv4) + len(b.cidrIPv6)
}

// GetRules returns all current rules
func (b *IPBlocker) GetRules() []BlockRule {
	b.mu.RLock()
	defer b.mu.RUnlock()

	rules := make([]BlockRule, 0, b.Count())

	for _, rule := range b.exactIPv4 {
		rules = append(rules, rule)
	}
	for _, rule := range b.exactIPv6 {
		rules = append(rules, rule)
	}
	for _, entry := range b.cidrIPv4 {
		rules = append(rules, entry.rule)
	}
	for _, entry := range b.cidrIPv6 {
		rules = append(rules, entry.rule)
	}

	return rules
}

// CleanExpired removes expired rules
func (b *IPBlocker) CleanExpired() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	removed := 0

	// Clean exact IPv4
	for ip, rule := range b.exactIPv4 {
		if !rule.ExpiresAt.IsZero() && now.After(rule.ExpiresAt) {
			delete(b.exactIPv4, ip)
			removed++
		}
	}

	// Clean exact IPv6
	for ip, rule := range b.exactIPv6 {
		if !rule.ExpiresAt.IsZero() && now.After(rule.ExpiresAt) {
			delete(b.exactIPv6, ip)
			removed++
		}
	}

	// Clean CIDR IPv4
	newCidrIPv4 := make([]*cidrEntry, 0, len(b.cidrIPv4))
	for _, entry := range b.cidrIPv4 {
		if entry.rule.ExpiresAt.IsZero() || now.Before(entry.rule.ExpiresAt) {
			newCidrIPv4 = append(newCidrIPv4, entry)
		} else {
			removed++
		}
	}
	b.cidrIPv4 = newCidrIPv4

	// Clean CIDR IPv6
	newCidrIPv6 := make([]*cidrEntry, 0, len(b.cidrIPv6))
	for _, entry := range b.cidrIPv6 {
		if entry.rule.ExpiresAt.IsZero() || now.Before(entry.rule.ExpiresAt) {
			newCidrIPv6 = append(newCidrIPv6, entry)
		} else {
			removed++
		}
	}
	b.cidrIPv6 = newCidrIPv6

	if removed > 0 {
		b.logger.WithField("count", removed).Debug("Cleaned expired IP blocklist entries")
	}

	return removed
}
