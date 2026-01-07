package threat

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		DomainBlockingEnabled: true,
		URLMatchingEnabled:    true,
	}

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManager_Check_IPBlocked(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled: true,
		IPCacheSize:       10000,
	}

	manager, _ := NewManager(cfg, logger)

	// Add IP block rule
	ipBlocker := manager.GetIPBlocker()
	rule := BlockRule{
		ID:        "test-ip",
		Pattern:   "192.168.1.100",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	ipBlocker.AddRule(rule)

	ctx := context.Background()
	// Manager checks DestinationIP, not SourceIP
	reqCtx := &RequestContext{
		SourceIP:      "10.0.0.1",
		DestinationIP: "192.168.1.100",
		Host:          "example.com",
		Path:          "/api/test",
	}

	decision := manager.Check(ctx, reqCtx)
	if decision == nil {
		t.Fatal("Check returned nil")
	}

	if !decision.Blocked {
		t.Error("Expected request to be blocked by IP")
	}

	// Implementation uses fixed "ip" category
	if decision.Category != "ip" {
		t.Errorf("Expected category 'ip', got '%s'", decision.Category)
	}
}

func TestManager_Check_DomainBlocked(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
	}

	manager, _ := NewManager(cfg, logger)

	// Add domain block rule
	domainBlocker := manager.GetDomainBlocker()
	rule := BlockRule{
		ID:        "test-domain",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	domainBlocker.AddRule(rule)

	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP: "192.168.1.50",
		Host:     "malware.com",
		Path:     "/api/test",
	}

	decision := manager.Check(ctx, reqCtx)
	if !decision.Blocked {
		t.Error("Expected request to be blocked by domain")
	}
}

func TestManager_Check_URLBlocked(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		URLMatchingEnabled: true,
		URLEngine:          "re2",
		MaxPatterns:        1000,
	}

	manager, _ := NewManager(cfg, logger)

	// Add URL pattern rule
	urlMatcher := manager.GetURLMatcher()
	rule := PatternRule{
		ID:        "test-url",
		Pattern:   "/admin/.*",
		Category:  "restricted",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}
	urlMatcher.AddPattern(rule)

	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP: "192.168.1.50",
		Host:     "example.com",
		Path:     "/admin/users",
	}

	decision := manager.Check(ctx, reqCtx)
	if !decision.Blocked {
		t.Error("Expected request to be blocked by URL pattern")
	}
}

func TestManager_Check_AllAllowed(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           10000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           1000,
	}

	manager, _ := NewManager(cfg, logger)

	// Add some rules that won't match
	ipBlocker := manager.GetIPBlocker()
	ipBlocker.AddRule(BlockRule{
		ID:        "test-ip",
		Pattern:   "10.0.0.1",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	domainBlocker := manager.GetDomainBlocker()
	domainBlocker.AddRule(BlockRule{
		ID:        "test-domain",
		Pattern:   "blocked.com",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP: "192.168.1.50",
		Host:     "allowed.com",
		Path:     "/api/test",
	}

	decision := manager.Check(ctx, reqCtx)
	if decision.Blocked {
		t.Error("Expected request to be allowed")
	}
}

func TestManager_Check_DestinationIPBlocked(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled: true,
		IPCacheSize:       10000,
	}

	manager, _ := NewManager(cfg, logger)

	// Add destination IP block rule
	ipBlocker := manager.GetIPBlocker()
	rule := BlockRule{
		ID:        "test-dest-ip",
		Pattern:   "10.0.0.100",
		Category:  "internal",
		Source:    "test",
		CreatedAt: time.Now(),
	}
	ipBlocker.AddRule(rule)

	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP:      "192.168.1.50",
		DestinationIP: "10.0.0.100",
		Host:          "example.com",
		Path:          "/api/test",
	}

	decision := manager.Check(ctx, reqCtx)
	if !decision.Blocked {
		t.Error("Expected request to be blocked by destination IP")
	}
}

func TestManager_Check_FullURL(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		URLMatchingEnabled: true,
		URLEngine:          "re2",
		MaxPatterns:        1000,
	}

	manager, _ := NewManager(cfg, logger)

	urlMatcher := manager.GetURLMatcher()
	rule := PatternRule{
		ID:        "test-full-url",
		Pattern:   ".*example\\.com/secret.*",
		Category:  "secret",
		Priority:  100,
		Source:    "test",
		CreatedAt: time.Now(),
	}
	urlMatcher.AddPattern(rule)

	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP: "192.168.1.50",
		Host:     "example.com",
		Path:     "/secret/data",
	}

	decision := manager.Check(ctx, reqCtx)
	if !decision.Blocked {
		t.Error("Expected request to be blocked by full URL pattern")
	}
}

func TestManager_GetStats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           10000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           1000,
	}

	manager, _ := NewManager(cfg, logger)

	// Add rules and make some checks
	ipBlocker := manager.GetIPBlocker()
	ipBlocker.AddRule(BlockRule{
		ID:        "test-ip",
		Pattern:   "192.168.1.100",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	ctx := context.Background()
	manager.Check(ctx, &RequestContext{SourceIP: "192.168.1.50", DestinationIP: "192.168.1.100", Host: "test.com", Path: "/"})
	manager.Check(ctx, &RequestContext{SourceIP: "192.168.1.50", Host: "test.com", Path: "/"})

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if stats["total_checks"] != 2 {
		t.Errorf("Expected 2 total checks, got %d", stats["total_checks"])
	}
}

func TestManager_GetBlockers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           10000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           1000,
	}

	manager, _ := NewManager(cfg, logger)

	if manager.GetIPBlocker() == nil {
		t.Error("Expected IP blocker to be initialized")
	}

	if manager.GetDomainBlocker() == nil {
		t.Error("Expected domain blocker to be initialized")
	}

	if manager.GetURLMatcher() == nil {
		t.Error("Expected URL matcher to be initialized")
	}
}

func TestManager_DisabledBlockers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled:     false,
		DomainBlockingEnabled: false,
		URLMatchingEnabled:    false,
	}

	manager, _ := NewManager(cfg, logger)

	// Even with disabled blockers, Check should work
	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP: "192.168.1.100",
		Host:     "example.com",
		Path:     "/api/test",
	}

	decision := manager.Check(ctx, reqCtx)
	if decision.Blocked {
		t.Error("Expected request to be allowed when all blockers disabled")
	}
}

func BenchmarkManager_Check_AllEnabled(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           100000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           10000,
	}

	manager, _ := NewManager(cfg, logger)

	// Add some rules
	ipBlocker := manager.GetIPBlocker()
	for i := 0; i < 100; i++ {
		ipBlocker.AddRule(BlockRule{
			ID:        "ip-" + string(rune(i)),
			Pattern:   "10.0." + string(rune(i/256)) + "." + string(rune(i%256)),
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		})
	}

	domainBlocker := manager.GetDomainBlocker()
	for i := 0; i < 100; i++ {
		domainBlocker.AddRule(BlockRule{
			ID:        "domain-" + string(rune(i)),
			Pattern:   "blocked" + string(rune(i)) + ".com",
			Category:  "test",
			Source:    "test",
			CreatedAt: time.Now(),
		})
	}

	urlMatcher := manager.GetURLMatcher()
	for i := 0; i < 50; i++ {
		urlMatcher.AddPattern(PatternRule{
			ID:        "url-" + string(rune(i)),
			Pattern:   "/path" + string(rune(i)) + "/.*",
			Category:  "test",
			Priority:  100,
			Source:    "test",
			CreatedAt: time.Now(),
		})
	}

	ctx := context.Background()
	reqCtx := &RequestContext{
		SourceIP: "192.168.1.50",
		Host:     "allowed.com",
		Path:     "/api/test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Check(ctx, reqCtx)
	}
}
