//go:build ci

package threat

import (
	"context"
	"testing"
	"time"

	"marchproxy-egress/internal/logging"
)

func TestManagerCheck(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           10000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           1000,
		DNSCacheEnabled:       false,
	}

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	_ = manager

	tests := []struct {
		name                string
		ip                  string
		domain              string
		path                string
		expectedBlock       bool
		expectedBlockReason string
		setupRules          func(*Manager)
	}{
		{
			name:          "no rules allows all",
			ip:            "1.2.3.4",
			domain:        "example.com",
			path:          "/",
			expectedBlock: false,
			setupRules:    func(m *Manager) {},
		},
		{
			name:          "IP blocked by IP blocker",
			ip:            "192.168.1.1",
			domain:        "example.com",
			path:          "/",
			expectedBlock: true,
			setupRules: func(m *Manager) {
				m.AddIPRule(BlockRule{
					ID:      "ip1",
					Pattern: "192.168.1.1",
				})
			},
		},
		{
			name:          "domain blocked by domain blocker",
			ip:            "1.2.3.4",
			domain:        "malicious.com",
			path:          "/",
			expectedBlock: true,
			setupRules: func(m *Manager) {
				m.AddDomainRule(BlockRule{
					ID:      "domain1",
					Pattern: "malicious.com",
				})
			},
		},
		{
			name:          "URL blocked by URL matcher",
			ip:            "1.2.3.4",
			domain:        "example.com",
			path:          "/admin/config",
			expectedBlock: true,
			setupRules: func(m *Manager) {
				m.AddURLPattern(PatternRule{
					ID:       "url1",
					Pattern:  ".*/admin/.*",
					Priority: 1,
				})
			},
		},
		{
			name:          "multiple checks use same rules",
			ip:            "10.0.0.1",
			domain:        "safe.com",
			path:          "/api/v1/users",
			expectedBlock: false,
			setupRules: func(m *Manager) {
				m.AddIPRule(BlockRule{
					ID:      "blocked-ip",
					Pattern: "192.168.1.0/24",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newMgr, _ := NewManager(cfg, logger)
			tt.setupRules(newMgr)

			req := &RequestContext{
				DestinationIP: tt.ip,
				Host:          tt.domain,
				Path:          tt.path,
			}

			decision := newMgr.Check(context.Background(), req)

			if decision.Blocked != tt.expectedBlock {
				t.Errorf("expected blocked=%v, got %v (reason: %s)", tt.expectedBlock, decision.Blocked, decision.Reason)
			}
		})
	}
}

func TestManagerIPBlockingDisabled(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ManagerConfig{
		IPBlockingEnabled:     false,
		DomainBlockingEnabled: true,
		URLMatchingEnabled:    true,
	}

	manager, _ := NewManager(cfg, logger)

	err := manager.AddIPRule(BlockRule{
		ID:      "ip1",
		Pattern: "192.168.1.1",
	})

	if err == nil {
		t.Error("expected error when adding IP rule with IP blocking disabled")
	}
}

func TestManagerDomainBlockingDisabled(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		DomainBlockingEnabled: false,
		URLMatchingEnabled:    true,
	}

	manager, _ := NewManager(cfg, logger)

	err := manager.AddDomainRule(BlockRule{
		ID:      "domain1",
		Pattern: "malicious.com",
	})

	if err == nil {
		t.Error("expected error when adding domain rule with domain blocking disabled")
	}
}

func TestManagerURLMatchingDisabled(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		DomainBlockingEnabled: true,
		URLMatchingEnabled:    false,
	}

	manager, _ := NewManager(cfg, logger)

	err := manager.AddURLPattern(PatternRule{
		ID:      "url1",
		Pattern: ".*/admin/.*",
	})

	if err == nil {
		t.Error("expected error when adding URL pattern with URL matching disabled")
	}
}

func TestManagerEnable(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	manager.Disable()
	if manager.IsEnabled() {
		t.Error("expected manager to be disabled")
	}

	manager.Enable()
	if !manager.IsEnabled() {
		t.Error("expected manager to be enabled")
	}
}

func TestManagerDisable(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	if !manager.IsEnabled() {
		t.Fatal("expected manager to be enabled initially")
	}

	manager.Disable()

	req := &RequestContext{
		DestinationIP: "192.168.1.1",
		Host:          "malicious.com",
	}

	manager.AddIPRule(BlockRule{
		ID:      "ip1",
		Pattern: "192.168.1.1",
	})

	decision := manager.Check(context.Background(), req)

	if decision.Blocked {
		t.Error("expected disabled manager to allow all traffic")
	}
}

func TestManagerGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           1000,
		DomainBlockingEnabled: true,
		URLMatchingEnabled:    true,
		MaxPatterns:           1000,
		DNSCacheEnabled:       false,
	}

	// Test IP blocking
	ipMgr, _ := NewManager(cfg, logger)
	if ipMgr.GetIPBlocker() == nil {
		t.Fatal("IP blocker should be initialized")
	}

	err := ipMgr.AddIPRule(BlockRule{ID: "ip1", Pattern: "192.168.1.1"})
	if err != nil {
		t.Fatalf("failed to add IP rule: %v", err)
	}

	d1 := ipMgr.Check(context.Background(), &RequestContext{DestinationIP: "192.168.1.1"})
	if !d1.Blocked {
		t.Errorf("IP rule should block 192.168.1.1, got blocked=%v, reason=%s", d1.Blocked, d1.Reason)
	}

	// Test domain blocking
	domainMgr, _ := NewManager(cfg, logger)
	if domainMgr.GetDomainBlocker() == nil {
		t.Fatal("Domain blocker should be initialized")
	}

	err = domainMgr.AddDomainRule(BlockRule{ID: "d1", Pattern: "malicious.com"})
	if err != nil {
		t.Fatalf("failed to add domain rule: %v", err)
	}

	d2 := domainMgr.Check(context.Background(), &RequestContext{Host: "malicious.com"})
	if !d2.Blocked {
		t.Errorf("Domain rule should block malicious.com, got blocked=%v", d2.Blocked)
	}

	// Test URL matching
	urlMgr, _ := NewManager(cfg, logger)
	if urlMgr.GetURLMatcher() == nil {
		t.Fatal("URL matcher should be initialized")
	}

	err = urlMgr.AddURLPattern(PatternRule{ID: "u1", Pattern: ".*admin.*", Priority: 1})
	if err != nil {
		t.Fatalf("failed to add URL pattern: %v", err)
	}

	d3 := urlMgr.Check(context.Background(), &RequestContext{Path: "/admin/test", Host: "example.com"})
	if !d3.Blocked {
		t.Errorf("URL rule should block /admin/test, got blocked=%v", d3.Blocked)
	}
}

func TestManagerResetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	manager.AddIPRule(BlockRule{ID: "ip1", Pattern: "192.168.1.1"})

	// Run a check
	manager.Check(context.Background(), &RequestContext{DestinationIP: "192.168.1.1"})

	stats := manager.GetStats()
	if stats["total_checks"] == 0 {
		t.Fatal("expected at least one check")
	}

	manager.ResetStats()

	stats = manager.GetStats()
	if stats["total_checks"] != 0 {
		t.Errorf("expected 0 total checks after reset, got %d", stats["total_checks"])
	}

	if stats["blocked_by_ip"] != 0 {
		t.Errorf("expected 0 blocked by IP after reset, got %d", stats["blocked_by_ip"])
	}
}

func TestManagerGetters(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ManagerConfig{
		IPBlockingEnabled:     true,
		DomainBlockingEnabled: true,
		URLMatchingEnabled:    true,
		DNSCacheEnabled:       true,
	}

	manager, _ := NewManager(cfg, logger)

	if manager.GetIPBlocker() == nil {
		t.Error("expected IP blocker to be initialized")
	}

	if manager.GetDomainBlocker() == nil {
		t.Error("expected domain blocker to be initialized")
	}

	if manager.GetURLMatcher() == nil {
		t.Error("expected URL matcher to be initialized")
	}

	if manager.GetDNSResolver() == nil {
		t.Error("expected DNS resolver to be initialized")
	}
}

func TestManagerAddRemoveIPRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	rule := BlockRule{
		ID:      "test-ip",
		Pattern: "192.168.1.0/24",
	}

	if err := manager.AddIPRule(rule); err != nil {
		t.Errorf("failed to add IP rule: %v", err)
	}

	req := &RequestContext{DestinationIP: "192.168.1.50"}
	decision := manager.Check(context.Background(), req)
	if !decision.Blocked {
		t.Error("expected IP to be blocked after adding rule")
	}

	if err := manager.RemoveIPRule("test-ip"); err != nil {
		t.Errorf("failed to remove IP rule: %v", err)
	}

	decision = manager.Check(context.Background(), req)
	if decision.Blocked {
		t.Error("expected IP to be allowed after removing rule")
	}
}

func TestManagerAddRemoveDomainRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	rule := BlockRule{
		ID:      "test-domain",
		Pattern: "malicious.com",
	}

	if err := manager.AddDomainRule(rule); err != nil {
		t.Errorf("failed to add domain rule: %v", err)
	}

	req := &RequestContext{Host: "malicious.com"}
	decision := manager.Check(context.Background(), req)
	if !decision.Blocked {
		t.Error("expected domain to be blocked after adding rule")
	}

	if err := manager.RemoveDomainRule("test-domain"); err != nil {
		t.Errorf("failed to remove domain rule: %v", err)
	}

	decision = manager.Check(context.Background(), req)
	if decision.Blocked {
		t.Error("expected domain to be allowed after removing rule")
	}
}

func TestManagerAddRemoveURLPattern(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	rule := PatternRule{
		ID:       "test-url",
		Pattern:  ".*/admin/.*",
		Priority: 1,
	}

	if err := manager.AddURLPattern(rule); err != nil {
		t.Errorf("failed to add URL pattern: %v", err)
	}

	req := &RequestContext{Host: "example.com", Path: "/admin/dashboard"}
	decision := manager.Check(context.Background(), req)
	if !decision.Blocked {
		t.Error("expected URL to be blocked after adding pattern")
	}

	if err := manager.RemoveURLPattern("test-url"); err != nil {
		t.Errorf("failed to remove URL pattern: %v", err)
	}

	decision = manager.Check(context.Background(), req)
	if decision.Blocked {
		t.Error("expected URL to be allowed after removing pattern")
	}
}

func TestManagerDefaultConfig(t *testing.T) {
	cfg := DefaultManagerConfig()

	if !cfg.IPBlockingEnabled {
		t.Error("expected IP blocking enabled in default config")
	}

	if !cfg.DomainBlockingEnabled {
		t.Error("expected domain blocking enabled in default config")
	}

	if !cfg.URLMatchingEnabled {
		t.Error("expected URL matching enabled in default config")
	}

	if !cfg.DNSCacheEnabled {
		t.Error("expected DNS cache enabled in default config")
	}

	if cfg.IPCacheSize == 0 {
		t.Error("expected non-zero IP cache size")
	}

	if cfg.MaxPatterns == 0 {
		t.Error("expected non-zero max patterns")
	}
}

func TestManagerCheckBlocking(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	// Setup: Block by IP, then by domain, then by URL
	manager.AddIPRule(BlockRule{ID: "ip1", Pattern: "192.168.0.0/16"})
	manager.AddDomainRule(BlockRule{ID: "d1", Pattern: "evil.com"})
	manager.AddURLPattern(PatternRule{ID: "u1", Pattern: ".*/backdoor", Priority: 1})

	tests := []struct {
		name         string
		ip           string
		domain       string
		path         string
		expectedBlock bool
		expectedCategory string
	}{
		{
			name:         "blocked by IP",
			ip:           "192.168.1.1",
			domain:       "safe.com",
			path:         "/",
			expectedBlock: true,
			expectedCategory: "ip",
		},
		{
			name:         "blocked by domain",
			ip:           "1.2.3.4",
			domain:       "evil.com",
			path:         "/",
			expectedBlock: true,
			expectedCategory: "domain",
		},
		{
			name:         "blocked by URL",
			ip:           "1.2.3.4",
			domain:       "safe.com",
			path:         "/backdoor",
			expectedBlock: true,
			expectedCategory: "url",
		},
		{
			name:         "allowed all",
			ip:           "1.2.3.4",
			domain:       "safe.com",
			path:         "/api/users",
			expectedBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RequestContext{
				DestinationIP: tt.ip,
				Host:          tt.domain,
				Path:          tt.path,
			}

			decision := manager.Check(context.Background(), req)

			if decision.Blocked != tt.expectedBlock {
				t.Errorf("expected blocked=%v, got %v", tt.expectedBlock, decision.Blocked)
			}

			if tt.expectedBlock && decision.Category != tt.expectedCategory {
				t.Errorf("expected category=%s, got %s", tt.expectedCategory, decision.Category)
			}
		})
	}
}

func TestManagerContextCancellation(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Even with cancelled context, Check should work (it doesn't use context for checking)
	req := &RequestContext{DestinationIP: "1.1.1.1"}
	decision := manager.Check(ctx, req)

	if decision.Blocked {
		t.Error("expected allowed on cancelled context")
	}
}

func TestManagerTimestamp(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, _ := NewManager(cfg, logger)

	before := time.Now()
	decision := manager.Check(context.Background(), &RequestContext{DestinationIP: "1.1.1.1"})
	after := time.Now()

	if decision.Timestamp.Before(before) || decision.Timestamp.After(after) {
		t.Errorf("decision timestamp not within expected range")
	}
}
