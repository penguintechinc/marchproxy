//go:build ci

package threat

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"marchproxy-egress/internal/logging"
)

// TestFeedSynchronization tests request checking with various patterns
func TestFeedSynchronization(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = true
	cfg.DomainBlockingEnabled = true
	cfg.URLMatchingEnabled = true

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test checking multiple request patterns
	for i := 0; i < 5; i++ {
		sourceIP := fmt.Sprintf("10.0.%d.1", i)
		req := &RequestContext{
			SourceIP:      sourceIP,
			DestinationIP: "192.168.1.1",
			Host:          "example.com",
			Path:          "/test",
		}

		result := manager.Check(context.Background(), req)
		if result == nil {
			t.Errorf("iteration %d: expected non-nil result", i)
		}
	}
}

// TestExpiryCleanup tests time-based request checking
func TestExpiryCleanup(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = true

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test time-based request checking
	requests := []*RequestContext{
		{
			SourceIP: "10.0.1.1",
			Host:     "example.com",
			Path:     "/api",
		},
		{
			SourceIP: "10.0.2.1",
			Host:     "example.com",
			Path:     "/api/v2",
		},
	}

	for _, req := range requests {
		result := manager.Check(context.Background(), req)
		if result == nil {
			t.Error("expected non-nil result")
		}
	}
}

// TestDomainBlockingRules tests domain-specific checking
func TestDomainBlockingRules(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = false
	cfg.DomainBlockingEnabled = true
	cfg.URLMatchingEnabled = false

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	tests := []struct {
		host     string
		expected bool
	}{
		{"example.com", false},
		{"safe.com", false},
		{"google.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req := &RequestContext{
				SourceIP: "192.168.1.1",
				Host:     tt.host,
				Path:     "/",
			}

			result := manager.Check(context.Background(), req)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// TestWildcardDomainMatching tests wildcard domain patterns
func TestWildcardDomainMatching(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = false
	cfg.DomainBlockingEnabled = true
	cfg.WildcardSupport = true
	cfg.URLMatchingEnabled = false

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	tests := []struct {
		host string
	}{
		{"api.example.com"},
		{"v2.api.example.com"},
		{"safe.com"},
		{"example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req := &RequestContext{
				SourceIP: "192.168.1.1",
				Host:     tt.host,
				Path:     "/",
			}

			result := manager.Check(context.Background(), req)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// TestURLPatternMatching tests URL pattern matching in paths
func TestURLPatternMatching(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = false
	cfg.DomainBlockingEnabled = false
	cfg.URLMatchingEnabled = true
	cfg.URLEngine = "re2"

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	tests := []struct {
		path string
	}{
		{"/admin/users"},
		{"/api/users"},
		{"/login"},
		{"/user/login"},
		{"/config.php"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := &RequestContext{
				SourceIP: "192.168.1.1",
				Host:     "example.com",
				Path:     tt.path,
			}

			result := manager.Check(context.Background(), req)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// TestStatistics tests threat manager statistics
func TestStatistics(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Perform several checks to generate stats
	for i := 0; i < 10; i++ {
		req := &RequestContext{
			SourceIP:      fmt.Sprintf("10.0.1.%d", i),
			DestinationIP: "192.168.1.1",
			Host:          "example.com",
			Path:          "/api",
		}
		manager.Check(context.Background(), req)
	}
}

// TestConcurrentChecks tests concurrent threat checks
func TestConcurrentChecks(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = true
	cfg.DomainBlockingEnabled = true
	cfg.URLMatchingEnabled = true

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	var wg sync.WaitGroup
	checkCount := 50

	// Concurrent checks from multiple goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			for j := 0; j < checkCount; j++ {
				ip := fmt.Sprintf("192.168.%d.%d", index, j%256)
				host := fmt.Sprintf("domain%d.com", j)
				path := fmt.Sprintf("/path/%d", j)

				req := &RequestContext{
					SourceIP:      ip,
					DestinationIP: "10.0.0.1",
					Host:          host,
					Path:          path,
				}

				result := manager.Check(context.Background(), req)
				if result == nil {
					t.Errorf("goroutine %d iteration %d: nil result", index, j)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestMultipleRequestTypes tests various request context combinations
func TestMultipleRequestTypes(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = true
	cfg.DomainBlockingEnabled = true
	cfg.URLMatchingEnabled = true

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	tests := []struct {
		name string
		req  *RequestContext
	}{
		{
			name: "minimal request",
			req: &RequestContext{
				SourceIP: "192.168.1.1",
			},
		},
		{
			name: "full request",
			req: &RequestContext{
				SourceIP:      "192.168.1.1",
				DestinationIP: "10.0.0.1",
				Host:          "example.com",
				Path:          "/api/v1/users",
				Method:        "GET",
				TLS:           true,
				Protocol:      "HTTP/2",
				Headers: map[string]string{
					"User-Agent": "test-agent",
				},
			},
		},
		{
			name: "POST request",
			req: &RequestContext{
				SourceIP:      "10.0.0.1",
				DestinationIP: "192.168.1.1",
				Host:          "api.example.com",
				Path:          "/api/v1/submit",
				Method:        "POST",
				TLS:           true,
				Protocol:      "HTTP/1.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.Check(context.Background(), tt.req)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// TestContextCancellation tests behavior with cancelled context
func TestContextCancellation(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &RequestContext{
		SourceIP: "192.168.1.1",
		Host:     "example.com",
	}

	result := manager.Check(ctx, req)
	if result == nil {
		t.Error("expected result even with cancelled context")
	}
}

// TestContextTimeout tests behavior with timeout context
func TestContextTimeout(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &RequestContext{
		SourceIP: "192.168.1.1",
		Host:     "example.com",
		Path:     "/test",
	}

	result := manager.Check(ctx, req)
	if result == nil {
		t.Error("expected result before timeout")
	}
}

// TestManagerConfiguration tests different configuration options
func TestManagerConfiguration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	configs := []struct {
		name string
		cfg  ManagerConfig
	}{
		{
			name: "minimal config",
			cfg: ManagerConfig{
				IPBlockingEnabled: true,
			},
		},
		{
			name: "full config",
			cfg:  DefaultManagerConfig(),
		},
		{
			name: "IP and URL only",
			cfg: ManagerConfig{
				IPBlockingEnabled:  true,
				URLMatchingEnabled: true,
				URLEngine:          "re2",
				MaxPatterns:        1000,
			},
		},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.cfg, logger)
			if err != nil {
				t.Fatalf("failed to create manager: %v", err)
			}

			req := &RequestContext{
				SourceIP: "192.168.1.1",
				Host:     "example.com",
			}

			result := manager.Check(context.Background(), req)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// TestBlockDecisionConsistency tests consistent decision making
func TestBlockDecisionConsistency(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	req := &RequestContext{
		SourceIP: "192.168.1.1",
		Host:     "example.com",
		Path:     "/api",
	}

	// Check multiple times with same request
	results := make([]*BlockDecision, 3)
	for i := 0; i < 3; i++ {
		results[i] = manager.Check(context.Background(), req)
	}

	// Verify consistency
	for i := 1; i < len(results); i++ {
		if results[i].Blocked != results[0].Blocked {
			t.Errorf("inconsistent results: check %d blocked=%v, check 0 blocked=%v",
				i, results[i].Blocked, results[0].Blocked)
		}
	}
}

// TestEmptyRequestContext tests behavior with empty request context
func TestEmptyRequestContext(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	req := &RequestContext{}

	result := manager.Check(context.Background(), req)
	if result == nil {
		t.Error("expected result for empty request")
	}
}

// TestRapidSequentialChecks tests rapid sequential checks
func TestRapidSequentialChecks(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	cfg.IPBlockingEnabled = true

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Rapid sequential checks
	for i := 0; i < 100; i++ {
		req := &RequestContext{
			SourceIP: fmt.Sprintf("10.0.0.%d", (i % 254) + 1),
			Host:     fmt.Sprintf("host%d.com", i%10),
			Path:     fmt.Sprintf("/path%d", i%5),
		}

		result := manager.Check(context.Background(), req)
		if result == nil {
			t.Errorf("iteration %d: nil result", i)
		}
	}
}

// TestVariousHTTPMethods tests request context with different HTTP methods
func TestVariousHTTPMethods(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := &RequestContext{
				SourceIP: "192.168.1.1",
				Host:     "example.com",
				Path:     "/api",
				Method:   method,
			}

			result := manager.Check(context.Background(), req)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// TestRequestContextWithHeaders tests request context with various headers
func TestRequestContextWithHeaders(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := DefaultManagerConfig()
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	req := &RequestContext{
		SourceIP: "192.168.1.1",
		Host:     "example.com",
		Path:     "/api",
		Headers: map[string]string{
			"User-Agent":      "Test-Client/1.0",
			"Content-Type":    "application/json",
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer token123",
		},
	}

	result := manager.Check(context.Background(), req)
	if result == nil {
		t.Error("expected non-nil result with headers")
	}
}
