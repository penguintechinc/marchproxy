// Package threat provides threat intelligence management for the egress proxy
package threat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BlockDecision represents the result of a threat check
type BlockDecision struct {
	Blocked     bool      `json:"blocked"`
	Reason      string    `json:"reason,omitempty"`
	Category    string    `json:"category,omitempty"` // "ip", "domain", "url", "dns"
	MatchedRule string    `json:"matched_rule,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// RequestContext contains information about the request being checked
type RequestContext struct {
	SourceIP      string            `json:"source_ip"`
	DestinationIP string            `json:"destination_ip"`
	Host          string            `json:"host"`         // Host header
	Path          string            `json:"path"`         // URL path
	Method        string            `json:"method"`       // HTTP method
	Headers       map[string]string `json:"headers"`      // Additional headers
	TLS           bool              `json:"tls"`          // Is TLS connection
	Protocol      string            `json:"protocol"`     // HTTP/1.1, HTTP/2, HTTP/3
}

// Manager orchestrates all threat intelligence components
type Manager struct {
	ipBlocker     *IPBlocker
	domainBlocker *DomainBlocker
	urlMatcher    *URLMatcher
	dnsResolver   *DNSResolver
	feedSync      *FeedSync

	enabled bool
	mu      sync.RWMutex
	logger  *logrus.Logger

	// Statistics
	stats struct {
		TotalChecks    int64
		BlockedByIP    int64
		BlockedByDomain int64
		BlockedByURL   int64
		BlockedByDNS   int64
		Allowed        int64
	}
}

// ManagerConfig holds configuration for the threat manager
type ManagerConfig struct {
	// IP blocking
	IPBlockingEnabled bool
	IPCacheSize       int

	// Domain blocking
	DomainBlockingEnabled bool
	WildcardSupport       bool

	// URL matching
	URLMatchingEnabled bool
	URLEngine          string // "re2" or "hyperscan"
	MaxPatterns        int

	// DNS cache
	DNSCacheEnabled bool
	DNSCacheSize    int
	DNSPositiveTTL  time.Duration
	DNSNegativeTTL  time.Duration
	DNSUpstream     []string

	// Feed synchronization
	FeedSyncMode     string        // "grpc", "poll", or "both"
	PollInterval     time.Duration
	GRPCEndpoint     string
	ManagerAPIURL    string
	ClusterAPIKey    string
}

// DefaultManagerConfig returns a ManagerConfig with sensible defaults
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           100000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           10000,
		DNSCacheEnabled:       true,
		DNSCacheSize:          50000,
		DNSPositiveTTL:        5 * time.Minute,
		DNSNegativeTTL:        1 * time.Minute,
		DNSUpstream:           []string{"8.8.8.8:53", "1.1.1.1:53"},
		FeedSyncMode:          "both",
		PollInterval:          60 * time.Second,
	}
}

// NewManager creates a new threat intelligence manager
func NewManager(cfg ManagerConfig, logger *logrus.Logger) (*Manager, error) {
	if logger == nil {
		logger = logrus.New()
	}

	m := &Manager{
		enabled: true,
		logger:  logger,
	}

	// Initialize IP blocker
	if cfg.IPBlockingEnabled {
		m.ipBlocker = NewIPBlocker(cfg.IPCacheSize, logger)
		logger.Info("IP blocking enabled")
	}

	// Initialize domain blocker
	if cfg.DomainBlockingEnabled {
		m.domainBlocker = NewDomainBlocker(cfg.WildcardSupport, logger)
		logger.Info("Domain blocking enabled")
	}

	// Initialize URL matcher
	if cfg.URLMatchingEnabled {
		var err error
		m.urlMatcher, err = NewURLMatcher(cfg.URLEngine, cfg.MaxPatterns, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create URL matcher: %w", err)
		}
		logger.Info("URL matching enabled")
	}

	// Initialize DNS resolver
	if cfg.DNSCacheEnabled {
		m.dnsResolver = NewDNSResolver(DNSResolverConfig{
			CacheSize:   cfg.DNSCacheSize,
			PositiveTTL: cfg.DNSPositiveTTL,
			NegativeTTL: cfg.DNSNegativeTTL,
			Upstream:    cfg.DNSUpstream,
		}, logger)
		logger.Info("DNS cache enabled")
	}

	// Initialize feed synchronization
	m.feedSync = NewFeedSync(FeedSyncConfig{
		Mode:          cfg.FeedSyncMode,
		PollInterval:  cfg.PollInterval,
		GRPCEndpoint:  cfg.GRPCEndpoint,
		ManagerAPIURL: cfg.ManagerAPIURL,
		ClusterAPIKey: cfg.ClusterAPIKey,
	}, m, logger)

	return m, nil
}

// Check evaluates a request against all threat intelligence sources
// Returns a BlockDecision indicating whether the request should be blocked
func (m *Manager) Check(ctx context.Context, req *RequestContext) *BlockDecision {
	m.mu.RLock()
	enabled := m.enabled
	m.mu.RUnlock()

	if !enabled {
		return &BlockDecision{
			Blocked:   false,
			Timestamp: time.Now(),
		}
	}

	// Update statistics
	m.mu.Lock()
	m.stats.TotalChecks++
	m.mu.Unlock()

	// Check IP blocklist (destination IP)
	if m.ipBlocker != nil && req.DestinationIP != "" {
		if decision := m.ipBlocker.Check(req.DestinationIP); decision.Blocked {
			m.mu.Lock()
			m.stats.BlockedByIP++
			m.mu.Unlock()
			return decision
		}
	}

	// Check domain blocklist (Host header)
	if m.domainBlocker != nil && req.Host != "" {
		if decision := m.domainBlocker.Check(req.Host); decision.Blocked {
			m.mu.Lock()
			m.stats.BlockedByDomain++
			m.mu.Unlock()
			return decision
		}
	}

	// Check URL patterns
	if m.urlMatcher != nil && req.Path != "" {
		fullURL := req.Path
		if req.Host != "" {
			fullURL = req.Host + req.Path
		}
		if decision := m.urlMatcher.Check(fullURL); decision.Blocked {
			m.mu.Lock()
			m.stats.BlockedByURL++
			m.mu.Unlock()
			return decision
		}
	}

	// Check resolved domain (DNS cache) - resolve Host header and check against IP blocklist
	if m.dnsResolver != nil && m.ipBlocker != nil && req.Host != "" {
		ips, err := m.dnsResolver.Resolve(ctx, req.Host)
		if err == nil {
			for _, ip := range ips {
				if decision := m.ipBlocker.Check(ip); decision.Blocked {
					decision.Category = "dns"
					decision.Reason = fmt.Sprintf("resolved IP %s blocked: %s", ip, decision.Reason)
					m.mu.Lock()
					m.stats.BlockedByDNS++
					m.mu.Unlock()
					return decision
				}
			}
		}
	}

	// All checks passed
	m.mu.Lock()
	m.stats.Allowed++
	m.mu.Unlock()

	return &BlockDecision{
		Blocked:   false,
		Timestamp: time.Now(),
	}
}

// Start begins the threat intelligence management (feed sync, etc.)
func (m *Manager) Start(ctx context.Context) error {
	if m.feedSync != nil {
		return m.feedSync.Start(ctx)
	}
	return nil
}

// Stop gracefully stops the threat intelligence manager
func (m *Manager) Stop() error {
	if m.feedSync != nil {
		return m.feedSync.Stop()
	}
	return nil
}

// Enable enables threat checking
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
	m.logger.Info("Threat intelligence enabled")
}

// Disable disables threat checking
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	m.logger.Info("Threat intelligence disabled")
}

// IsEnabled returns whether threat checking is enabled
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// GetIPBlocker returns the IP blocker component
func (m *Manager) GetIPBlocker() *IPBlocker {
	return m.ipBlocker
}

// GetDomainBlocker returns the domain blocker component
func (m *Manager) GetDomainBlocker() *DomainBlocker {
	return m.domainBlocker
}

// GetURLMatcher returns the URL matcher component
func (m *Manager) GetURLMatcher() *URLMatcher {
	return m.urlMatcher
}

// GetDNSResolver returns the DNS resolver component
func (m *Manager) GetDNSResolver() *DNSResolver {
	return m.dnsResolver
}

// GetStats returns current statistics
func (m *Manager) GetStats() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]int64{
		"total_checks":     m.stats.TotalChecks,
		"blocked_by_ip":    m.stats.BlockedByIP,
		"blocked_by_domain": m.stats.BlockedByDomain,
		"blocked_by_url":   m.stats.BlockedByURL,
		"blocked_by_dns":   m.stats.BlockedByDNS,
		"allowed":          m.stats.Allowed,
	}
}

// ResetStats resets all statistics to zero
func (m *Manager) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalChecks = 0
	m.stats.BlockedByIP = 0
	m.stats.BlockedByDomain = 0
	m.stats.BlockedByURL = 0
	m.stats.BlockedByDNS = 0
	m.stats.Allowed = 0
}

// AddIPRule adds an IP blocking rule
func (m *Manager) AddIPRule(rule BlockRule) error {
	if m.ipBlocker == nil {
		return fmt.Errorf("IP blocking is not enabled")
	}
	return m.ipBlocker.AddRule(rule)
}

// RemoveIPRule removes an IP blocking rule
func (m *Manager) RemoveIPRule(id string) error {
	if m.ipBlocker == nil {
		return fmt.Errorf("IP blocking is not enabled")
	}
	return m.ipBlocker.RemoveRule(id)
}

// AddDomainRule adds a domain blocking rule
func (m *Manager) AddDomainRule(rule BlockRule) error {
	if m.domainBlocker == nil {
		return fmt.Errorf("domain blocking is not enabled")
	}
	return m.domainBlocker.AddRule(rule)
}

// RemoveDomainRule removes a domain blocking rule
func (m *Manager) RemoveDomainRule(id string) error {
	if m.domainBlocker == nil {
		return fmt.Errorf("domain blocking is not enabled")
	}
	return m.domainBlocker.RemoveRule(id)
}

// AddURLPattern adds a URL pattern rule
func (m *Manager) AddURLPattern(rule PatternRule) error {
	if m.urlMatcher == nil {
		return fmt.Errorf("URL matching is not enabled")
	}
	return m.urlMatcher.AddPattern(rule)
}

// RemoveURLPattern removes a URL pattern rule
func (m *Manager) RemoveURLPattern(id string) error {
	if m.urlMatcher == nil {
		return fmt.Errorf("URL matching is not enabled")
	}
	return m.urlMatcher.RemovePattern(id)
}
