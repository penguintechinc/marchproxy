// Package threat provides threat feed synchronization for the egress proxy
package threat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// FeedSyncConfig holds configuration for feed synchronization
type FeedSyncConfig struct {
	Mode          string        // "grpc", "poll", or "both"
	PollInterval  time.Duration
	GRPCEndpoint  string
	ManagerAPIURL string
	ClusterAPIKey string
}

// ThreatFeedResponse represents the response from the Manager API
type ThreatFeedResponse struct {
	IPBlocklist     []IPBlocklistEntry     `json:"ip_blocklist"`
	DomainBlocklist []DomainBlocklistEntry `json:"domain_blocklist"`
	URLPatterns     []URLPatternEntry      `json:"url_patterns"`
	Version         string                 `json:"version"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// IPBlocklistEntry represents an IP blocklist entry from the API
type IPBlocklistEntry struct {
	IP        string    `json:"ip,omitempty"`
	CIDR      string    `json:"cidr,omitempty"`
	Category  string    `json:"category"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// DomainBlocklistEntry represents a domain blocklist entry from the API
type DomainBlocklistEntry struct {
	Domain    string    `json:"domain"`
	Category  string    `json:"category"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// URLPatternEntry represents a URL pattern entry from the API
type URLPatternEntry struct {
	Pattern   string    `json:"pattern"`
	Category  string    `json:"category"`
	Priority  int       `json:"priority"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// FeedSync handles synchronization of threat feeds from the Manager API
type FeedSync struct {
	config  FeedSyncConfig
	manager *Manager
	logger  *logrus.Logger

	client  *http.Client
	version string // Current feed version

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Statistics
	stats struct {
		LastSync       time.Time
		SyncCount      int64
		SyncErrors     int64
		RulesProcessed int64
	}
}

// NewFeedSync creates a new feed synchronization manager
func NewFeedSync(cfg FeedSyncConfig, manager *Manager, logger *logrus.Logger) *FeedSync {
	if logger == nil {
		logger = logrus.New()
	}

	// Default values
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 60 * time.Second
	}
	if cfg.Mode == "" {
		cfg.Mode = "both"
	}

	return &FeedSync{
		config:  cfg,
		manager: manager,
		logger:  logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins the feed synchronization process
func (f *FeedSync) Start(ctx context.Context) error {
	f.mu.Lock()
	f.ctx, f.cancel = context.WithCancel(ctx)
	f.mu.Unlock()

	// Initial sync
	if err := f.sync(); err != nil {
		f.logger.WithError(err).Warn("Initial threat feed sync failed")
	}

	// Start polling if enabled
	if f.config.Mode == "poll" || f.config.Mode == "both" {
		f.wg.Add(1)
		go f.pollLoop()
	}

	// Start gRPC streaming if enabled
	if f.config.Mode == "grpc" || f.config.Mode == "both" {
		f.wg.Add(1)
		go f.grpcStreamLoop()
	}

	f.logger.WithFields(logrus.Fields{
		"mode":          f.config.Mode,
		"poll_interval": f.config.PollInterval,
	}).Info("Started threat feed synchronization")

	return nil
}

// Stop gracefully stops the feed synchronization
func (f *FeedSync) Stop() error {
	f.mu.Lock()
	if f.cancel != nil {
		f.cancel()
	}
	f.mu.Unlock()

	f.wg.Wait()
	f.logger.Info("Stopped threat feed synchronization")
	return nil
}

// pollLoop periodically polls the Manager API for threat feed updates
func (f *FeedSync) pollLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			if err := f.sync(); err != nil {
				f.logger.WithError(err).Warn("Threat feed sync failed")
				f.mu.Lock()
				f.stats.SyncErrors++
				f.mu.Unlock()
			}
		}
	}
}

// grpcStreamLoop handles gRPC streaming updates (placeholder for future implementation)
func (f *FeedSync) grpcStreamLoop() {
	defer f.wg.Done()

	// For now, this is a placeholder that falls back to polling
	// Full gRPC streaming implementation would go here
	f.logger.Warn("gRPC streaming not yet implemented, using polling only")

	// Wait for context cancellation
	<-f.ctx.Done()
}

// sync fetches and applies the latest threat feed
func (f *FeedSync) sync() error {
	if f.config.ManagerAPIURL == "" {
		return fmt.Errorf("manager API URL not configured")
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/threat-feeds", f.config.ManagerAPIURL)

	req, err := http.NewRequestWithContext(f.ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	if f.config.ClusterAPIKey != "" {
		req.Header.Set("X-Cluster-API-Key", f.config.ClusterAPIKey)
	}

	// Add version header for conditional fetching
	f.mu.RLock()
	currentVersion := f.version
	f.mu.RUnlock()
	if currentVersion != "" {
		req.Header.Set("If-None-Match", currentVersion)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch threat feed: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		f.logger.Debug("Threat feed unchanged")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("threat feed API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var feed ThreatFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return fmt.Errorf("failed to decode threat feed: %w", err)
	}

	// Apply the feed
	if err := f.applyFeed(&feed); err != nil {
		return fmt.Errorf("failed to apply threat feed: %w", err)
	}

	// Update version
	f.mu.Lock()
	f.version = feed.Version
	f.stats.LastSync = time.Now()
	f.stats.SyncCount++
	f.mu.Unlock()

	f.logger.WithFields(logrus.Fields{
		"version":      feed.Version,
		"ip_rules":     len(feed.IPBlocklist),
		"domain_rules": len(feed.DomainBlocklist),
		"url_patterns": len(feed.URLPatterns),
	}).Info("Applied threat feed update")

	return nil
}

// applyFeed applies a threat feed to the manager's blockers
func (f *FeedSync) applyFeed(feed *ThreatFeedResponse) error {
	rulesProcessed := 0

	// Apply IP blocklist
	if ipBlocker := f.manager.GetIPBlocker(); ipBlocker != nil {
		// Clear existing rules from this source and re-add
		// (In production, you'd want incremental updates)
		for i, entry := range feed.IPBlocklist {
			pattern := entry.IP
			if pattern == "" {
				pattern = entry.CIDR
			}

			rule := BlockRule{
				ID:        fmt.Sprintf("feed-ip-%d", i),
				Pattern:   pattern,
				Category:  entry.Category,
				Source:    "manager-feed",
				CreatedAt: time.Now(),
				ExpiresAt: entry.ExpiresAt,
			}

			if err := ipBlocker.AddRule(rule); err != nil {
				f.logger.WithError(err).WithField("pattern", pattern).Warn("Failed to add IP rule")
				continue
			}
			rulesProcessed++
		}
	}

	// Apply domain blocklist
	if domainBlocker := f.manager.GetDomainBlocker(); domainBlocker != nil {
		for i, entry := range feed.DomainBlocklist {
			rule := BlockRule{
				ID:        fmt.Sprintf("feed-domain-%d", i),
				Pattern:   entry.Domain,
				Category:  entry.Category,
				Source:    "manager-feed",
				CreatedAt: time.Now(),
				ExpiresAt: entry.ExpiresAt,
			}

			if err := domainBlocker.AddRule(rule); err != nil {
				f.logger.WithError(err).WithField("domain", entry.Domain).Warn("Failed to add domain rule")
				continue
			}
			rulesProcessed++
		}
	}

	// Apply URL patterns
	if urlMatcher := f.manager.GetURLMatcher(); urlMatcher != nil {
		for i, entry := range feed.URLPatterns {
			rule := PatternRule{
				ID:        fmt.Sprintf("feed-url-%d", i),
				Pattern:   entry.Pattern,
				Category:  entry.Category,
				Priority:  entry.Priority,
				Source:    "manager-feed",
				CreatedAt: time.Now(),
				ExpiresAt: entry.ExpiresAt,
			}

			if err := urlMatcher.AddPattern(rule); err != nil {
				f.logger.WithError(err).WithField("pattern", entry.Pattern).Warn("Failed to add URL pattern")
				continue
			}
			rulesProcessed++
		}
	}

	f.mu.Lock()
	f.stats.RulesProcessed += int64(rulesProcessed)
	f.mu.Unlock()

	return nil
}

// GetStats returns synchronization statistics
func (f *FeedSync) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return map[string]interface{}{
		"last_sync":       f.stats.LastSync,
		"sync_count":      f.stats.SyncCount,
		"sync_errors":     f.stats.SyncErrors,
		"rules_processed": f.stats.RulesProcessed,
		"current_version": f.version,
	}
}

// GetCurrentVersion returns the current feed version
func (f *FeedSync) GetCurrentVersion() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.version
}

// ForceSync forces an immediate synchronization
func (f *FeedSync) ForceSync() error {
	return f.sync()
}
