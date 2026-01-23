// Package threat provides DNS resolution with caching for the egress proxy
package threat

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DNSCacheEntry represents a cached DNS resolution result
type DNSCacheEntry struct {
	IPs       []string      `json:"ips"`
	TTL       time.Duration `json:"ttl"`
	CachedAt  time.Time     `json:"cached_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	NXDOMAIN  bool          `json:"nxdomain"` // Domain doesn't exist
}

// DNSResolverConfig holds configuration for the DNS resolver
type DNSResolverConfig struct {
	CacheSize   int
	PositiveTTL time.Duration
	NegativeTTL time.Duration
	Upstream    []string // DNS server addresses (e.g., "8.8.8.8:53")
}

// DNSResolver provides DNS resolution with caching
type DNSResolver struct {
	cache       map[string]*DNSCacheEntry
	positiveTTL time.Duration
	negativeTTL time.Duration
	upstream    []string
	cacheSize   int
	resolver    *net.Resolver

	mu     sync.RWMutex
	logger *logrus.Logger

	// Statistics
	stats struct {
		Hits       int64
		Misses     int64
		NXDOMAIN   int64
		Errors     int64
		Evictions  int64
	}
}

// NewDNSResolver creates a new DNS resolver with caching
func NewDNSResolver(cfg DNSResolverConfig, logger *logrus.Logger) *DNSResolver {
	if logger == nil {
		logger = logrus.New()
	}

	// Default values
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 50000
	}
	if cfg.PositiveTTL == 0 {
		cfg.PositiveTTL = 5 * time.Minute
	}
	if cfg.NegativeTTL == 0 {
		cfg.NegativeTTL = 1 * time.Minute
	}
	if len(cfg.Upstream) == 0 {
		cfg.Upstream = []string{"8.8.8.8:53", "1.1.1.1:53"}
	}

	// Create a custom resolver if upstream DNS servers are specified
	var resolver *net.Resolver
	if len(cfg.Upstream) > 0 {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				// Use the first upstream server
				return d.DialContext(ctx, "udp", cfg.Upstream[0])
			},
		}
	} else {
		resolver = net.DefaultResolver
	}

	return &DNSResolver{
		cache:       make(map[string]*DNSCacheEntry),
		positiveTTL: cfg.PositiveTTL,
		negativeTTL: cfg.NegativeTTL,
		upstream:    cfg.Upstream,
		cacheSize:   cfg.CacheSize,
		resolver:    resolver,
		logger:      logger,
	}
}

// Resolve resolves a domain name to IP addresses
func (r *DNSResolver) Resolve(ctx context.Context, domain string) ([]string, error) {
	// Check cache first
	r.mu.RLock()
	entry, found := r.cache[domain]
	r.mu.RUnlock()

	if found && time.Now().Before(entry.ExpiresAt) {
		r.mu.Lock()
		r.stats.Hits++
		r.mu.Unlock()

		if entry.NXDOMAIN {
			return nil, fmt.Errorf("NXDOMAIN: %s", domain)
		}
		return entry.IPs, nil
	}

	// Cache miss - perform actual resolution
	r.mu.Lock()
	r.stats.Misses++
	r.mu.Unlock()

	ips, err := r.resolver.LookupHost(ctx, domain)
	if err != nil {
		// Check if it's NXDOMAIN
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			r.mu.Lock()
			r.stats.NXDOMAIN++
			r.mu.Unlock()

			// Cache negative result
			r.cacheNegative(domain)
			return nil, fmt.Errorf("NXDOMAIN: %s", domain)
		}

		r.mu.Lock()
		r.stats.Errors++
		r.mu.Unlock()

		return nil, fmt.Errorf("DNS resolution failed for %s: %w", domain, err)
	}

	// Cache the result
	r.cachePositive(domain, ips)

	return ips, nil
}

// cachePositive caches a successful DNS resolution
func (r *DNSResolver) cachePositive(domain string, ips []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache size and evict if necessary
	if len(r.cache) >= r.cacheSize {
		r.evictOldest()
	}

	r.cache[domain] = &DNSCacheEntry{
		IPs:       ips,
		TTL:       r.positiveTTL,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(r.positiveTTL),
		NXDOMAIN:  false,
	}

	r.logger.WithFields(logrus.Fields{
		"domain": domain,
		"ips":    ips,
		"ttl":    r.positiveTTL,
	}).Debug("Cached DNS resolution")
}

// cacheNegative caches a negative DNS result (NXDOMAIN)
func (r *DNSResolver) cacheNegative(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache size and evict if necessary
	if len(r.cache) >= r.cacheSize {
		r.evictOldest()
	}

	r.cache[domain] = &DNSCacheEntry{
		IPs:       nil,
		TTL:       r.negativeTTL,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(r.negativeTTL),
		NXDOMAIN:  true,
	}

	r.logger.WithFields(logrus.Fields{
		"domain": domain,
		"ttl":    r.negativeTTL,
	}).Debug("Cached NXDOMAIN")
}

// evictOldest evicts the oldest cache entry (simple LRU-like behavior)
func (r *DNSResolver) evictOldest() {
	var oldestDomain string
	var oldestTime time.Time

	for domain, entry := range r.cache {
		if oldestDomain == "" || entry.CachedAt.Before(oldestTime) {
			oldestDomain = domain
			oldestTime = entry.CachedAt
		}
	}

	if oldestDomain != "" {
		delete(r.cache, oldestDomain)
		r.stats.Evictions++
	}
}

// Clear clears the DNS cache
func (r *DNSResolver) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache = make(map[string]*DNSCacheEntry)
	r.logger.Info("Cleared DNS cache")
}

// Count returns the number of cached entries
func (r *DNSResolver) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.cache)
}

// GetStats returns current statistics
func (r *DNSResolver) GetStats() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]int64{
		"hits":      r.stats.Hits,
		"misses":    r.stats.Misses,
		"nxdomain":  r.stats.NXDOMAIN,
		"errors":    r.stats.Errors,
		"evictions": r.stats.Evictions,
		"size":      int64(len(r.cache)),
	}
}

// CleanExpired removes expired cache entries
func (r *DNSResolver) CleanExpired() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	removed := 0

	for domain, entry := range r.cache {
		if now.After(entry.ExpiresAt) {
			delete(r.cache, domain)
			removed++
		}
	}

	if removed > 0 {
		r.logger.WithField("count", removed).Debug("Cleaned expired DNS cache entries")
	}

	return removed
}

// GetCachedEntry returns a cached entry for a domain (if it exists and is valid)
func (r *DNSResolver) GetCachedEntry(domain string) (*DNSCacheEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, found := r.cache[domain]
	if !found || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry, true
}

// Prefetch resolves and caches a domain without returning the result
func (r *DNSResolver) Prefetch(ctx context.Context, domain string) {
	go func() {
		_, _ = r.Resolve(ctx, domain)
	}()
}

// PrefetchBatch resolves and caches multiple domains
func (r *DNSResolver) PrefetchBatch(ctx context.Context, domains []string) {
	for _, domain := range domains {
		go func(d string) {
			_, _ = r.Resolve(ctx, d)
		}(domain)
	}
}
