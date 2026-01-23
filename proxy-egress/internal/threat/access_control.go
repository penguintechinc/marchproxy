// Package threat provides access control based on authentication tokens
// This allows restricting access to destinations based on authenticated services
package threat

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AccessControlMode defines how access control is applied
type AccessControlMode string

const (
	// AccessControlModeAllow means the rule allows access (whitelist)
	AccessControlModeAllow AccessControlMode = "allow"
	// AccessControlModeDeny means the rule denies access (blacklist)
	AccessControlModeDeny AccessControlMode = "deny"
)

// AccessControlRule defines access control based on authenticated tokens/services
type AccessControlRule struct {
	ID              string            `json:"id"`
	TargetType      string            `json:"target_type"`    // "domain", "ip", "url_pattern"
	TargetPattern   string            `json:"target_pattern"` // The target (domain, IP, or URL pattern)
	Mode            AccessControlMode `json:"mode"`           // "allow" or "deny"
	AllowedServices []string          `json:"allowed_services"` // Service IDs or names that can access
	AllowedTokens   []string          `json:"allowed_tokens"`   // Specific token identifiers
	RequireAuth     bool              `json:"require_auth"`   // Whether authentication is required
	Category        string            `json:"category"`       // Category for logging
	Source          string            `json:"source"`         // Where the rule came from
	CreatedAt       time.Time         `json:"created_at"`
	ExpiresAt       time.Time         `json:"expires_at,omitempty"`
}

// AccessControlDecision represents the result of an access control check
type AccessControlDecision struct {
	Allowed       bool      `json:"allowed"`
	Reason        string    `json:"reason,omitempty"`
	RequiresAuth  bool      `json:"requires_auth"`
	MatchedRule   string    `json:"matched_rule,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// ServiceContext contains information about the authenticated service
type ServiceContext struct {
	ServiceID    string `json:"service_id"`
	ServiceName  string `json:"service_name"`
	TokenID      string `json:"token_id"`
	Authenticated bool  `json:"authenticated"`
}

// AccessController manages access control rules
type AccessController struct {
	// Rules indexed by target pattern for fast lookup
	domainRules map[string]*AccessControlRule
	ipRules     map[string]*AccessControlRule
	urlRules    []*accessControlURLRule // URL patterns need regex matching

	// Global default
	defaultRequireAuth bool
	defaultAllow       bool

	mu     sync.RWMutex
	logger *logrus.Logger
}

// accessControlURLRule combines an access control rule with compiled regex
type accessControlURLRule struct {
	rule *AccessControlRule
}

// NewAccessController creates a new access controller
func NewAccessController(defaultRequireAuth bool, logger *logrus.Logger) *AccessController {
	if logger == nil {
		logger = logrus.New()
	}

	return &AccessController{
		domainRules:        make(map[string]*AccessControlRule),
		ipRules:            make(map[string]*AccessControlRule),
		urlRules:           make([]*accessControlURLRule, 0),
		defaultRequireAuth: defaultRequireAuth,
		defaultAllow:       true, // Default to allowing if no rules match
		logger:             logger,
	}
}

// Check evaluates access control for a request
func (c *AccessController) Check(target string, targetType string, svc *ServiceContext) *AccessControlDecision {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var rule *AccessControlRule

	// Find matching rule based on target type
	switch targetType {
	case "domain":
		rule = c.findDomainRule(target)
	case "ip":
		rule = c.findIPRule(target)
	case "url":
		rule = c.findURLRule(target)
	default:
		return &AccessControlDecision{
			Allowed:   c.defaultAllow,
			Reason:    "unknown target type",
			Timestamp: time.Now(),
		}
	}

	// No matching rule - apply defaults
	if rule == nil {
		if c.defaultRequireAuth && (svc == nil || !svc.Authenticated) {
			return &AccessControlDecision{
				Allowed:      false,
				Reason:       "authentication required (default policy)",
				RequiresAuth: true,
				Timestamp:    time.Now(),
			}
		}
		return &AccessControlDecision{
			Allowed:   c.defaultAllow,
			Reason:    "no matching rule",
			Timestamp: time.Now(),
		}
	}

	// Check if rule has expired
	if !rule.ExpiresAt.IsZero() && time.Now().After(rule.ExpiresAt) {
		return &AccessControlDecision{
			Allowed:   c.defaultAllow,
			Reason:    "matching rule expired",
			Timestamp: time.Now(),
		}
	}

	// Check if authentication is required
	if rule.RequireAuth && (svc == nil || !svc.Authenticated) {
		return &AccessControlDecision{
			Allowed:      false,
			Reason:       fmt.Sprintf("authentication required for %s", target),
			RequiresAuth: true,
			MatchedRule:  rule.ID,
			Timestamp:    time.Now(),
		}
	}

	// Check service/token allowlist
	if svc != nil && svc.Authenticated {
		// Check allowed services
		if len(rule.AllowedServices) > 0 {
			serviceAllowed := false
			for _, allowed := range rule.AllowedServices {
				if allowed == svc.ServiceID || allowed == svc.ServiceName {
					serviceAllowed = true
					break
				}
			}
			if !serviceAllowed {
				return &AccessControlDecision{
					Allowed:     false,
					Reason:      fmt.Sprintf("service %s not authorized for %s", svc.ServiceName, target),
					MatchedRule: rule.ID,
					Timestamp:   time.Now(),
				}
			}
		}

		// Check allowed tokens
		if len(rule.AllowedTokens) > 0 {
			tokenAllowed := false
			for _, allowed := range rule.AllowedTokens {
				if allowed == svc.TokenID {
					tokenAllowed = true
					break
				}
			}
			if !tokenAllowed {
				return &AccessControlDecision{
					Allowed:     false,
					Reason:      fmt.Sprintf("token not authorized for %s", target),
					MatchedRule: rule.ID,
					Timestamp:   time.Now(),
				}
			}
		}
	}

	// Apply the rule mode
	if rule.Mode == AccessControlModeDeny {
		return &AccessControlDecision{
			Allowed:     false,
			Reason:      fmt.Sprintf("access denied by rule for %s", target),
			MatchedRule: rule.ID,
			Timestamp:   time.Now(),
		}
	}

	return &AccessControlDecision{
		Allowed:     true,
		Reason:      "access allowed",
		MatchedRule: rule.ID,
		Timestamp:   time.Now(),
	}
}

// findDomainRule finds a matching domain rule
func (c *AccessController) findDomainRule(domain string) *AccessControlRule {
	// Exact match
	if rule, ok := c.domainRules[domain]; ok {
		return rule
	}

	// Check for wildcard matches (*.example.com)
	// Implementation similar to DomainBlocker
	return nil
}

// findIPRule finds a matching IP rule
func (c *AccessController) findIPRule(ip string) *AccessControlRule {
	if rule, ok := c.ipRules[ip]; ok {
		return rule
	}
	// CIDR matching could be added here
	return nil
}

// findURLRule finds a matching URL rule
func (c *AccessController) findURLRule(url string) *AccessControlRule {
	// URL rules would need regex matching
	// For now, simple prefix matching
	for _, aclRule := range c.urlRules {
		// Simple implementation - could use regex
		if aclRule.rule.TargetPattern == url {
			return aclRule.rule
		}
	}
	return nil
}

// AddRule adds an access control rule
func (c *AccessController) AddRule(rule *AccessControlRule) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	switch rule.TargetType {
	case "domain":
		c.domainRules[rule.TargetPattern] = rule
	case "ip":
		c.ipRules[rule.TargetPattern] = rule
	case "url", "url_pattern":
		c.urlRules = append(c.urlRules, &accessControlURLRule{rule: rule})
	default:
		return fmt.Errorf("unknown target type: %s", rule.TargetType)
	}

	c.logger.WithFields(logrus.Fields{
		"rule_id":        rule.ID,
		"target_type":    rule.TargetType,
		"target_pattern": rule.TargetPattern,
		"mode":           rule.Mode,
		"require_auth":   rule.RequireAuth,
	}).Debug("Added access control rule")

	return nil
}

// RemoveRule removes an access control rule by ID
func (c *AccessController) RemoveRule(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check domain rules
	for pattern, rule := range c.domainRules {
		if rule.ID == id {
			delete(c.domainRules, pattern)
			c.logger.WithField("rule_id", id).Debug("Removed access control rule")
			return nil
		}
	}

	// Check IP rules
	for pattern, rule := range c.ipRules {
		if rule.ID == id {
			delete(c.ipRules, pattern)
			c.logger.WithField("rule_id", id).Debug("Removed access control rule")
			return nil
		}
	}

	// Check URL rules
	for i, aclRule := range c.urlRules {
		if aclRule.rule.ID == id {
			c.urlRules = append(c.urlRules[:i], c.urlRules[i+1:]...)
			c.logger.WithField("rule_id", id).Debug("Removed access control rule")
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", id)
}

// Clear removes all rules
func (c *AccessController) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.domainRules = make(map[string]*AccessControlRule)
	c.ipRules = make(map[string]*AccessControlRule)
	c.urlRules = make([]*accessControlURLRule, 0)

	c.logger.Info("Cleared all access control rules")
}

// Count returns the total number of rules
func (c *AccessController) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.domainRules) + len(c.ipRules) + len(c.urlRules)
}

// GetRules returns all rules
func (c *AccessController) GetRules() []*AccessControlRule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rules := make([]*AccessControlRule, 0, c.Count())

	for _, rule := range c.domainRules {
		rules = append(rules, rule)
	}
	for _, rule := range c.ipRules {
		rules = append(rules, rule)
	}
	for _, aclRule := range c.urlRules {
		rules = append(rules, aclRule.rule)
	}

	return rules
}

// SetDefaultRequireAuth sets the default authentication requirement
func (c *AccessController) SetDefaultRequireAuth(require bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultRequireAuth = require
}

// SetDefaultAllow sets the default allow/deny behavior
func (c *AccessController) SetDefaultAllow(allow bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultAllow = allow
}
