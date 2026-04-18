//go:build ci
// +build ci

package ratelimit

import (
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimiterConfig
		checkFn func(*RateLimiter) error
	}{
		{
			name: "default config",
			config: RateLimiterConfig{
				CleanupInterval: 1 * time.Second,
			},
			checkFn: func(rl *RateLimiter) error {
				if rl.globalLimiter != nil {
					t.Error("global limiter should be nil")
				}
				return nil
			},
		},
		{
			name: "global limit enabled",
			config: RateLimiterConfig{
				GlobalLimit:     rate.Limit(100),
				GlobalBurst:     10,
				CleanupInterval: 1 * time.Second,
			},
			checkFn: func(rl *RateLimiter) error {
				if rl.globalLimiter == nil {
					t.Error("global limiter should not be nil")
				}
				return nil
			},
		},
		{
			name: "blocklist enabled",
			config: RateLimiterConfig{
				BlocklistEnabled: true,
				BlockDuration:    5 * time.Second,
				CleanupInterval:  1 * time.Second,
			},
			checkFn: func(rl *RateLimiter) error {
				if rl.ipBlocklist == nil {
					t.Error("ipBlocklist should not be nil")
				}
				return nil
			},
		},
		{
			name: "ddos protection enabled",
			config: RateLimiterConfig{
				EnableDDoSProtection: true,
				DDoSThreshold:        10,
				DDoSWindow:           1 * time.Minute,
				CleanupInterval:      1 * time.Second,
			},
			checkFn: func(rl *RateLimiter) error {
				if rl.ddosDetector == nil {
					t.Error("ddosDetector should not be nil")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.config)
			if rl == nil {
				t.Fatal("expected non-nil rate limiter")
			}
			if tt.checkFn != nil {
				tt.checkFn(rl)
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(10),
		PerIPBurst:      5,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"

	err := rl.Allow(req)
	if err != nil {
		t.Errorf("first request should be allowed, got error: %v", err)
	}
}

func TestRateLimiter_Allow_GlobalLimit(t *testing.T) {
	config := RateLimiterConfig{
		GlobalLimit:     rate.Limit(1), // 1 per second
		GlobalBurst:     1,
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      50,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"

	// First request should be allowed
	err := rl.Allow(req)
	if err != nil {
		t.Fatalf("first request should be allowed, got error: %v", err)
	}

	// Second request should be rate limited
	err = rl.Allow(req)
	if err != ErrRateLimitExceeded {
		t.Errorf("second request should be rate limited, got error: %v", err)
	}
}

func TestRateLimiter_ExtractClientID(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		wantID   string
	}{
		{
			name:     "api key header",
			headers:  map[string]string{"X-API-Key": "key123"},
			remoteIP: "127.0.0.1",
			wantID:   "apikey:key123",
		},
		{
			name:     "user id header",
			headers:  map[string]string{"X-User-ID": "user456"},
			remoteIP: "127.0.0.1",
			wantID:   "user:user456",
		},
		{
			name:     "bearer token",
			headers:  map[string]string{"Authorization": "Bearer token123token456"},
			remoteIP: "127.0.0.1",
			wantID:   "auth:token123token456",
		},
		{
			name:     "fallback to ip",
			headers:  map[string]string{},
			remoteIP: "192.168.1.1",
			wantID:   "ip:192.168.1.1",
		},
		{
			name:     "x-forwarded-for",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1, 127.0.0.1"},
			remoteIP: "127.0.0.1",
			wantID:   "ip:10.0.0.1",
		},
	}

	rl := NewRateLimiter(RateLimiterConfig{CleanupInterval: 1 * time.Second})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteIP + ":8080"
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			clientID := rl.extractClientID(req)
			if clientID != tt.wantID {
				t.Errorf("got client ID %q, want %q", clientID, tt.wantID)
			}
		})
	}
}

func TestRateLimiter_ExtractIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		wantIP   string
	}{
		{
			name:     "x-forwarded-for",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1, 192.168.1.1"},
			remoteIP: "127.0.0.1:8080",
			wantIP:   "10.0.0.1",
		},
		{
			name:     "x-real-ip",
			headers:  map[string]string{"X-Real-IP": "10.0.0.2"},
			remoteIP: "127.0.0.1:8080",
			wantIP:   "10.0.0.2",
		},
		{
			name:     "remote addr",
			headers:  map[string]string{},
			remoteIP: "192.168.1.100:8080",
			wantIP:   "192.168.1.100",
		},
	}

	rl := NewRateLimiter(RateLimiterConfig{CleanupInterval: 1 * time.Second})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteIP
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := rl.extractIP(req)
			if ip != tt.wantIP {
				t.Errorf("got IP %q, want %q", ip, tt.wantIP)
			}
		})
	}
}

func TestIPBlocklist_Block_IsBlocked(t *testing.T) {
	bl := NewIPBlocklist(5 * time.Second)

	ip := "192.168.1.1"

	// IP should not be blocked initially
	if bl.IsBlocked(ip) {
		t.Error("IP should not be blocked initially")
	}

	// Block the IP
	bl.Block(ip, "test reason", 5*time.Second)

	// IP should now be blocked
	if !bl.IsBlocked(ip) {
		t.Error("IP should be blocked after Block()")
	}
}

func TestIPBlocklist_Whitelist(t *testing.T) {
	bl := NewIPBlocklist(5 * time.Second)

	ip := "192.168.1.1"

	// Block then whitelist
	bl.Block(ip, "test", 5*time.Second)
	bl.Whitelist(ip)

	// Should not be blocked anymore
	if bl.IsBlocked(ip) {
		t.Error("whitelisted IP should not be blocked")
	}
}

func TestDDoSDetector_IsSuspicious(t *testing.T) {
	dd := NewDDoSDetector(10, 1*time.Minute)

	ip := "192.168.1.1"

	// Create requests with high rate
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/api/endpoint", nil)
		req.RemoteAddr = ip + ":8080"
		dd.IsSuspicious(ip, req)
	}

	// Wait a bit to allow rate calculation
	time.Sleep(100 * time.Millisecond)

	// Check again
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip + ":8080"
	suspicious := dd.IsSuspicious(ip, req)

	if !suspicious {
		t.Error("high-rate IP should be suspicious")
	}
}

func TestQuotaManager_CreateQuota(t *testing.T) {
	qm := NewQuotaManager(1 * time.Hour)

	clientID := "user:123"
	limit := int64(1000)

	quota := qm.CreateQuota(clientID, limit)

	if quota.Limit != limit {
		t.Errorf("limit: got %d, want %d", quota.Limit, limit)
	}
	if quota.Used != 0 {
		t.Errorf("used: got %d, want 0", quota.Used)
	}
}

func TestQuotaManager_GetQuota(t *testing.T) {
	qm := NewQuotaManager(100 * time.Millisecond)

	clientID := "user:123"
	_ = qm.CreateQuota(clientID, 100)

	// Get immediately
	retrieved := qm.GetQuota(clientID)
	if retrieved == nil {
		t.Fatal("GetQuota returned nil")
	}
	if retrieved.Limit != 100 {
		t.Errorf("limit: got %d, want 100", retrieved.Limit)
	}
}

func TestQuota_Operations(t *testing.T) {
	quota := &Quota{
		Limit:        100,
		Used:         0,
		ResetAt:      time.Now().Add(1 * time.Hour),
		WarningLevel: 0.8,
	}

	if !quota.HasRemaining() {
		t.Error("should have remaining quota")
	}

	quota.Use(50)
	if quota.Used != 50 {
		t.Errorf("used: got %d, want 50", quota.Used)
	}

	remaining := quota.GetRemaining()
	if remaining != 50 {
		t.Errorf("remaining: got %d, want 50", remaining)
	}

	if quota.IsWarning() {
		t.Error("should not be warning at 50% usage")
	}

	quota.Use(30)
	if !quota.IsWarning() {
		t.Error("should be warning at 80% usage")
	}
}

func TestExponentialBackoff_CalculateDelay(t *testing.T) {
	eb := &ExponentialBackoff{
		BaseDelay:      1 * time.Second,
		MaxDelay:       1 * time.Minute,
		Multiplier:     2.0,
		BlockThreshold: 10,
	}

	tests := []struct {
		violations int
		minDelay   time.Duration
		maxDelay   time.Duration
	}{
		{1, 1 * time.Second, 1 * time.Second},
		{2, 2 * time.Second, 2 * time.Second},
		{3, 4 * time.Second, 4 * time.Second},
		{11, 1 * time.Minute, 1 * time.Minute}, // Capped at max
	}

	for _, tt := range tests {
		delay := eb.CalculateDelay(tt.violations)
		if delay < tt.minDelay || delay > tt.maxDelay {
			t.Errorf("violations %d: got delay %v, want [%v, %v]", tt.violations, delay, tt.minDelay, tt.maxDelay)
		}
	}
}

func TestExponentialBackoff_ShouldBlock(t *testing.T) {
	eb := &ExponentialBackoff{
		BlockThreshold: 5,
	}

	if eb.ShouldBlock(4) {
		t.Error("should not block at 4 violations")
	}

	if !eb.ShouldBlock(5) {
		t.Error("should block at 5 violations")
	}

	if !eb.ShouldBlock(10) {
		t.Error("should block at 10 violations")
	}
}

func TestRateLimitMetrics_Operations(t *testing.T) {
	metrics := &RateLimitMetrics{}

	metrics.recordRequest()
	if metrics.TotalRequests != 1 {
		t.Errorf("TotalRequests: got %d, want 1", metrics.TotalRequests)
	}

	metrics.recordAllowed()
	if metrics.AllowedRequests != 1 {
		t.Errorf("AllowedRequests: got %d, want 1", metrics.AllowedRequests)
	}

	metrics.recordBlocked()
	if metrics.BlockedRequests != 1 {
		t.Errorf("BlockedRequests: got %d, want 1", metrics.BlockedRequests)
	}

	metrics.incrementUniqueClients()
	metrics.incrementUniqueClients()
	if metrics.UniqueClients != 2 {
		t.Errorf("UniqueClients: got %d, want 2", metrics.UniqueClients)
	}
}

func TestRateLimiter_GetRateLimitHeaders(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:       rate.Limit(100),
		PerIPBurst:       10,
		RateLimitHeaders: true,
		CleanupInterval:  1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"

	// Allow a request first
	_ = rl.Allow(req)

	clientID := rl.extractClientID(req)
	headers := rl.GetRateLimitHeaders(clientID)

	if len(headers) == 0 {
		t.Error("expected rate limit headers")
	}

	if _, exists := headers["X-RateLimit-Limit"]; !exists {
		t.Error("X-RateLimit-Limit header missing")
	}
	if _, exists := headers["X-RateLimit-Remaining"]; !exists {
		t.Error("X-RateLimit-Remaining header missing")
	}
}

func TestRateLimiter_BlocklistIntegration(t *testing.T) {
	config := RateLimiterConfig{
		BlocklistEnabled: true,
		BlockDuration:    5 * time.Second,
		PerIPLimit:       rate.Limit(100),
		PerIPBurst:       10,
		CleanupInterval:  1 * time.Second,
	}
	rl := NewRateLimiter(config)

	ip := "192.168.1.1"
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip + ":8080"

	// Block the IP
	rl.ipBlocklist.Block(ip, "test block", 5*time.Second)

	// Request should be blocked
	err := rl.Allow(req)
	if err != ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(1000),
		PerIPBurst:      100,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "127.0.0.1:8080"
			_ = rl.Allow(req)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	metrics := rl.metrics
	if metrics.TotalRequests != 100 {
		t.Errorf("TotalRequests: got %d, want 100", metrics.TotalRequests)
	}
}
