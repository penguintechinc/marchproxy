//go:build ci
// +build ci

package ratelimit

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// Test checkQuota function - quota checking logic
func TestRateLimiter_CheckQuota_NoManager(t *testing.T) {
	config := RateLimiterConfig{
		EnableQuotas:    false,
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	// checkQuota should return nil when manager is nil
	err := rl.checkQuota("test-client")
	if err != nil {
		t.Errorf("checkQuota with nil manager should return nil, got %v", err)
	}
}

// Test checkQuota with quota exceeded
func TestRateLimiter_CheckQuota_Exceeded(t *testing.T) {
	config := RateLimiterConfig{
		EnableQuotas:    true,
		QuotaPeriod:     1 * time.Hour,
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	clientID := "test-client"
	quota := rl.quotaManager.CreateQuota(clientID, 5)
	quota.Use(5) // Exhaust quota

	err := rl.checkQuota(clientID)
	if err != ErrQuotaExceeded {
		t.Errorf("checkQuota should return ErrQuotaExceeded, got %v", err)
	}
}

// Test checkQuota creates quota automatically
func TestRateLimiter_CheckQuota_AutoCreate(t *testing.T) {
	config := RateLimiterConfig{
		EnableQuotas:    true,
		QuotaPeriod:     1 * time.Hour,
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	clientID := "new-client"
	err := rl.checkQuota(clientID)
	if err != nil {
		t.Errorf("checkQuota should auto-create quota, got error %v", err)
	}

	quota := rl.quotaManager.GetQuota(clientID)
	if quota == nil {
		t.Error("quota should have been auto-created")
	}
}

// Test getCustomLimiter creates new limiter
func TestRateLimiter_GetCustomLimiter_Creates(t *testing.T) {
	config := RateLimiterConfig{
		CustomLimits: map[string]LimitConfig{
			"/api/heavy": {
				Limit: rate.Limit(10),
				Burst: 5,
			},
		},
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	clientLimiter := &ClientLimiter{
		limiter:      rate.NewLimiter(rate.Limit(100), 10),
		customLimits: make(map[string]*rate.Limiter),
	}

	// First call creates limiter
	limiter1 := rl.getCustomLimiter(clientLimiter, "/api/heavy")
	if limiter1 == nil {
		t.Error("getCustomLimiter should create and return limiter")
	}

	// Second call reuses
	limiter2 := rl.getCustomLimiter(clientLimiter, "/api/heavy")
	if limiter1 != limiter2 {
		t.Error("getCustomLimiter should reuse existing limiter")
	}
}

// Test getCustomLimiter returns nil for non-custom endpoints
func TestRateLimiter_GetCustomLimiter_NoCustom(t *testing.T) {
	config := RateLimiterConfig{
		CustomLimits:    map[string]LimitConfig{},
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	clientLimiter := &ClientLimiter{
		limiter:      rate.NewLimiter(rate.Limit(100), 10),
		customLimits: make(map[string]*rate.Limiter),
	}

	limiter := rl.getCustomLimiter(clientLimiter, "/api/unknown")
	if limiter != nil {
		t.Error("getCustomLimiter should return nil for non-custom endpoint")
	}
}

// Test recordViolation with blocking
func TestRateLimiter_RecordViolation_TriggerBlock(t *testing.T) {
	eb := &ExponentialBackoff{
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		Multiplier:     2.0,
		BlockThreshold: 3,
	}
	config := RateLimiterConfig{
		BackoffStrategy: eb,
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	clientID := "ip:192.168.1.1"
	limiter := rl.getOrCreateLimiter(clientID)

	// Record violations
	for i := 0; i < 4; i++ {
		rl.recordViolation(clientID)
	}

	// Check if blocked
	if !limiter.blocked {
		t.Error("recordViolation should block after threshold")
	}
	if limiter.blockedUntil.Before(time.Now()) {
		t.Error("blockedUntil should be in future")
	}
}

// Test recordViolation adds to blocklist for IP clients
func TestRateLimiter_RecordViolation_AddToBlocklist(t *testing.T) {
	eb := &ExponentialBackoff{
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		Multiplier:     2.0,
		BlockThreshold: 2,
	}
	config := RateLimiterConfig{
		BlocklistEnabled: true,
		BlockDuration:    1 * time.Hour,
		BackoffStrategy:  eb,
		PerIPLimit:       rate.Limit(100),
		PerIPBurst:       10,
		CleanupInterval:  1 * time.Second,
	}
	rl := NewRateLimiter(config)

	ip := "192.168.1.1"
	clientID := "ip:" + ip

	// Record violations
	for i := 0; i < 3; i++ {
		rl.recordViolation(clientID)
	}

	// Check blocklist
	if !rl.ipBlocklist.IsBlocked(ip) {
		t.Error("IP should be added to blocklist after violations")
	}
}

// Test Allow with blocked client
func TestRateLimiter_Allow_BlockedClient(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:8080"

	clientID := rl.extractClientID(req)
	limiter := rl.getOrCreateLimiter(clientID)

	// Manually block
	limiter.blocked = true
	limiter.blockedUntil = time.Now().Add(1 * time.Hour)

	err := rl.Allow(req)
	if err != ErrBlocked {
		t.Errorf("Allow should return ErrBlocked for blocked client, got %v", err)
	}
}

// Test Allow with DDoS protection triggering
func TestRateLimiter_Allow_DDoSDetection(t *testing.T) {
	config := RateLimiterConfig{
		EnableDDoSProtection: true,
		DDoSThreshold:        5,
		DDoSWindow:           1 * time.Minute,
		BlocklistEnabled:     true,
		BlockDuration:        1 * time.Hour,
		PerIPLimit:           rate.Limit(1000),
		PerIPBurst:           100,
		CleanupInterval:      1 * time.Second,
	}
	rl := NewRateLimiter(config)

	ip := "192.168.1.1"

	// Trigger high-rate detection
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip + ":8080"
		_ = rl.Allow(req)
	}

	// Check if metrics recorded DDoS detection
	if rl.metrics.DDoSAttacksDetected == 0 {
		t.Error("DDoS attacks should have been detected")
	}
}

// Test setupDefaultMitigations creates rules
func TestRateLimiter_SetupDefaultMitigations(t *testing.T) {
	config := RateLimiterConfig{
		EnableDDoSProtection: true,
		DDoSThreshold:        10,
		DDoSWindow:           1 * time.Minute,
		CleanupInterval:      1 * time.Second,
	}
	rl := NewRateLimiter(config)

	if len(rl.ddosDetector.mitigations) == 0 {
		t.Error("setupDefaultMitigations should create mitigation rules")
	}
}

// Test Allow with custom limits
func TestRateLimiter_Allow_CustomLimits(t *testing.T) {
	config := RateLimiterConfig{
		CustomLimits: map[string]LimitConfig{
			"/api/expensive": {
				Limit: rate.Limit(1),
				Burst: 1,
			},
		},
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      100,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	// First request succeeds
	req1 := httptest.NewRequest("GET", "/api/expensive", nil)
	req1.RemoteAddr = "127.0.0.1:8080"
	err1 := rl.Allow(req1)
	if err1 != nil {
		t.Fatalf("First request should succeed, got %v", err1)
	}

	// Second request on custom endpoint hits limit
	req2 := httptest.NewRequest("GET", "/api/expensive", nil)
	req2.RemoteAddr = "127.0.0.1:8080"
	err2 := rl.Allow(req2)
	if err2 != ErrRateLimitExceeded {
		t.Errorf("Second request should be rate limited, got %v", err2)
	}
}

// Test Allow with Authorization header parsing
func TestRateLimiter_Allow_AuthHeaderClientID(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(1000),
		PerIPBurst:      100,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("Authorization", "Bearer verylongtoken123456789")

	clientID := rl.extractClientID(req)
	if !strings.HasPrefix(clientID, "auth:") {
		t.Errorf("Should extract auth header, got %s", clientID)
	}
	// Token should be truncated to 16 chars
	if len(clientID) > 20 {
		t.Errorf("Auth token should be truncated, got %d chars", len(clientID))
	}
}

// Test extractIP with RemoteAddr without port
func TestRateLimiter_ExtractIP_RemoteAddrNoParse(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1" // No port

	ip := rl.extractIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Should return RemoteAddr directly, got %s", ip)
	}
}

// Test IPBlocklist permanent block
func TestIPBlocklist_PermanentBlock(t *testing.T) {
	bl := NewIPBlocklist(1 * time.Hour)

	ip := "192.168.1.1"
	entry := BlockedEntry{
		IP:           ip,
		Reason:       "permanent",
		BlockedAt:    time.Now(),
		BlockedUntil: time.Now().Add(-1 * time.Hour), // Already expired
		Permanent:    true,
	}

	// Directly set permanent block
	bl.mutex.Lock()
	bl.blocked[ip] = entry
	bl.mutex.Unlock()

	// Should still be blocked
	if !bl.IsBlocked(ip) {
		t.Error("permanent block should not expire")
	}
}

// Test DDoSDetector.ShouldMitigate with action
func TestDDoSDetector_ShouldMitigate_ExecuteAction(t *testing.T) {
	dd := NewDDoSDetector(10, 1*time.Minute)

	pattern := &RequestPattern{
		IP:           "192.168.1.1",
		Count:        50,
		FirstRequest: time.Now().Add(-30 * time.Second),
		LastRequest:  time.Now(),
		Endpoints:    make(map[string]int),
		UserAgents:   make(map[string]int),
	}

	dd.mutex.Lock()
	dd.requests["192.168.1.1"] = pattern
	dd.mutex.Unlock()

	actionCalled := false
	rule := MitigationRule{
		Name:     "test-rule",
		Priority: 1,
		Enabled:  true,
		Condition: func(p *RequestPattern) bool {
			return p.Count > 40
		},
		Action: func(ip string) error {
			actionCalled = true
			return nil
		},
	}

	dd.AddMitigation(rule)
	result := dd.ShouldMitigate("192.168.1.1")

	if !actionCalled {
		t.Error("mitigation action should have been called")
	}
	if !result {
		t.Error("ShouldMitigate should return true when action executed")
	}
}

// Test DDoSDetector with multiple endpoints
func TestDDoSDetector_MultipleEndpoints(t *testing.T) {
	dd := NewDDoSDetector(10, 1*time.Minute)

	ip := "192.168.1.1"

	// Create requests to multiple endpoints
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/api/endpoint", nil)
		req.RemoteAddr = ip + ":8080"
		dd.IsSuspicious(ip, req)
	}

	time.Sleep(50 * time.Millisecond)

	req := httptest.NewRequest("GET", "/other", nil)
	req.RemoteAddr = ip + ":8080"
	suspicious := dd.IsSuspicious(ip, req)

	pattern := dd.requests[ip]
	if pattern == nil || !pattern.Suspicious {
		t.Error("should detect suspicious pattern with high rate")
	}
	if !suspicious {
		t.Error("should return true for suspicious pattern")
	}
}

// Test DDoSDetector endpoint limit detection (>100 endpoints)
func TestDDoSDetector_EndpointLimitDetection(t *testing.T) {
	dd := NewDDoSDetector(100, 1*time.Minute)

	ip := "192.168.1.1"

	// Create requests to many endpoints
	for i := 0; i < 105; i++ {
		req := httptest.NewRequest("GET", "/api/endpoint"+string(rune(i)), nil)
		req.RemoteAddr = ip + ":8080"
		dd.IsSuspicious(ip, req)
	}

	pattern := dd.requests[ip]
	if pattern == nil || !pattern.Suspicious {
		t.Error("should detect suspicious pattern with >100 endpoints")
	}
}

// Test DDoSDetector user-agent limit detection (>10 user agents)
func TestDDoSDetector_UserAgentLimitDetection(t *testing.T) {
	dd := NewDDoSDetector(100, 1*time.Minute)

	ip := "192.168.1.1"

	// Create requests with many user agents
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = ip + ":8080"
		req.Header.Set("User-Agent", "UserAgent-"+string(rune(i)))
		dd.IsSuspicious(ip, req)
	}

	pattern := dd.requests[ip]
	if pattern == nil || !pattern.Suspicious {
		t.Error("should detect suspicious pattern with >10 user agents")
	}
}

// Test DDoSDetector alert severity levels
func TestDDoSDetector_AlertSeverity(t *testing.T) {
	dd := NewDDoSDetector(10, 1*time.Minute)

	tests := []struct {
		count    int
		expected AlertSeverity
	}{
		{15, SeverityLow},      // > threshold (10), < 2x
		{25, SeverityMedium},   // > 2x threshold (20), < 3x
		{35, SeverityHigh},     // > 3x threshold (30), < 5x
		{60, SeverityCritical}, // > 5x threshold (50)
	}

	for _, tc := range tests {
		pattern := &RequestPattern{
			IP:    "192.168.1.1",
			Count: tc.count,
		}

		dd.createAlert(pattern)

		if len(dd.alerts) > 0 {
			lastAlert := dd.alerts[len(dd.alerts)-1]
			if lastAlert.Severity != tc.expected {
				t.Errorf("count=%d: expected severity %d, got %d", tc.count, tc.expected, lastAlert.Severity)
			}
		}
	}
}

// Test QuotaManager quota reset
func TestQuotaManager_QuotaReset(t *testing.T) {
	qm := NewQuotaManager(100 * time.Millisecond)

	clientID := "user:123"
	quota := qm.CreateQuota(clientID, 100)
	quota.Use(50)

	if quota.Used != 50 {
		t.Errorf("quota should be at 50, got %d", quota.Used)
	}

	// Wait for reset period
	time.Sleep(150 * time.Millisecond)

	// Get quota again - should reset
	retrieved := qm.GetQuota(clientID)
	if retrieved.Used != 0 {
		t.Errorf("quota should reset, got %d used", retrieved.Used)
	}
}

// Test Quota IsWarning at different levels
func TestQuota_WarningLevels(t *testing.T) {
	quota := &Quota{
		Limit:        100,
		Used:         0,
		ResetAt:      time.Now().Add(1 * time.Hour),
		WarningLevel: 0.8,
	}

	quota.Use(79)
	if quota.IsWarning() {
		t.Error("79/100 (79%) should not trigger warning at 80%")
	}

	quota.Use(1)
	if !quota.IsWarning() {
		t.Error("80/100 (80%) should trigger warning")
	}
}

// Test metrics operations concurrently
func TestRateLimitMetrics_Concurrent(t *testing.T) {
	metrics := &RateLimitMetrics{}

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			metrics.recordRequest()
			metrics.recordAllowed()
			metrics.recordBlocked()
			metrics.recordQuotaExceeded()
			metrics.recordDDoSDetected()
			metrics.recordDDoSMitigated()
			metrics.incrementUniqueClients()
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	if metrics.TotalRequests != 50 {
		t.Errorf("expected 50 requests, got %d", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 50 {
		t.Errorf("expected 50 allowed, got %d", metrics.AllowedRequests)
	}
	if metrics.BlockedRequests != 50 {
		t.Errorf("expected 50 blocked, got %d", metrics.BlockedRequests)
	}
	if metrics.QuotaExceeded != 50 {
		t.Errorf("expected 50 quota exceeded, got %d", metrics.QuotaExceeded)
	}
	if metrics.DDoSAttacksDetected != 50 {
		t.Errorf("expected 50 DDoS detected, got %d", metrics.DDoSAttacksDetected)
	}
	if metrics.DDoSAttacksMitigated != 50 {
		t.Errorf("expected 50 DDoS mitigated, got %d", metrics.DDoSAttacksMitigated)
	}
	if metrics.UniqueClients != 50 {
		t.Errorf("expected 50 unique clients, got %d", metrics.UniqueClients)
	}
}

// Test min function
func TestMinFunction(t *testing.T) {
	if min(5, 10) != 5 {
		t.Error("min(5, 10) should be 5")
	}
	if min(10, 5) != 5 {
		t.Error("min(10, 5) should be 5")
	}
	if min(5, 5) != 5 {
		t.Error("min(5, 5) should be 5")
	}
}

// Test XDP Fallback implementation
func TestXDPRateLimiter_Fallback(t *testing.T) {
	// These tests verify the fallback implementation returns expected errors
	// since we're building with !xdp tag

	// Note: XDP tests would need proper logging/metrics mocks
	// For now, we verify the interface implementations exist
	tests := []struct {
		name string
		fn   func() error
	}{
		{"LoadProgram", func() error {
			return NewXDPRateLimiter(nil, nil).LoadProgram("test")
		}},
		{"AttachToInterface", func() error {
			return NewXDPRateLimiter(nil, nil).AttachToInterface("eth0")
		}},
		{"DetachFromInterface", func() error {
			return NewXDPRateLimiter(nil, nil).DetachFromInterface("eth0")
		}},
		{"UpdateConfig", func() error {
			return NewXDPRateLimiter(nil, nil).UpdateConfig(nil)
		}},
		{"SetEnterpriseLicense", func() error {
			return NewXDPRateLimiter(nil, nil).SetEnterpriseLicense(true)
		}},
		{"GetIPState", func() error {
			_, err := NewXDPRateLimiter(nil, nil).GetIPState(nil)
			return err
		}},
		{"ClearIPState", func() error {
			return NewXDPRateLimiter(nil, nil).ClearIPState(nil)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Errorf("%s should return error in fallback mode", tc.name)
			}
		})
	}
}

// Test GetRateLimitHeaders with blocked state
func TestRateLimiter_GetRateLimitHeaders_Blocked(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:       rate.Limit(100),
		PerIPBurst:       10,
		RateLimitHeaders: true,
		CleanupInterval:  1 * time.Second,
	}
	rl := NewRateLimiter(config)

	// Create request and allow it first
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	_ = rl.Allow(req)

	clientID := rl.extractClientID(req)
	limiter := rl.getOrCreateLimiter(clientID)

	// Manually block
	limiter.blocked = true
	limiter.blockedUntil = time.Now().Add(10 * time.Second)

	headers := rl.GetRateLimitHeaders(clientID)

	if _, ok := headers["X-RateLimit-Reset"]; !ok {
		t.Error("should have X-RateLimit-Reset header when blocked")
	}
	if _, ok := headers["Retry-After"]; !ok {
		t.Error("should have Retry-After header when blocked")
	}
}

// Test extraction with malformed Authorization header
func TestRateLimiter_ExtractClientID_MalformedAuth(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		CleanupInterval: 1 * time.Second,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("Authorization", "Bearer")      // No token after Bearer
	req.Header.Set("X-Forwarded-For", "10.0.0.1") // Should still use this

	clientID := rl.extractClientID(req)
	if clientID != "ip:10.0.0.1" {
		t.Errorf("should fall back to X-Forwarded-For, got %s", clientID)
	}
}

// Test cleanup goroutine
func TestRateLimiter_Cleanup(t *testing.T) {
	config := RateLimiterConfig{
		PerIPLimit:      rate.Limit(100),
		PerIPBurst:      10,
		WindowSize:      100 * time.Millisecond,
		CleanupInterval: 100 * time.Millisecond,
	}
	rl := NewRateLimiter(config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	_ = rl.Allow(req)

	clientID := rl.extractClientID(req)
	if _, exists := rl.limiters[clientID]; !exists {
		t.Fatal("limiter should exist after Allow")
	}

	// Wait for cleanup to run
	time.Sleep(300 * time.Millisecond)

	// The limiter should be cleaned up
	rl.mutex.RLock()
	_, exists := rl.limiters[clientID]
	rl.mutex.RUnlock()

	if exists {
		t.Error("limiter should have been cleaned up after WindowSize elapsed")
	}
}

// Test IPBlocklist cleanup goroutine
func TestIPBlocklist_Cleanup(t *testing.T) {
	bl := NewIPBlocklist(100 * time.Millisecond)

	ip := "192.168.1.1"
	bl.Block(ip, "test", 100*time.Millisecond)

	if !bl.IsBlocked(ip) {
		t.Error("IP should be blocked initially")
	}

	// Wait for cleanup
	time.Sleep(250 * time.Millisecond)

	if bl.IsBlocked(ip) {
		t.Error("IP should have been cleaned up after expiry")
	}
}

// Test DDoSDetector cleanup of alerts
func TestDDoSDetector_AlertCleanup(t *testing.T) {
	dd := NewDDoSDetector(10, 1*time.Minute)

	// Create many alerts
	for i := 0; i < 1100; i++ {
		pattern := &RequestPattern{
			IP:    "192.168.1." + string(rune(i%255)),
			Count: 50,
		}
		dd.createAlert(pattern)
	}

	if len(dd.alerts) > 1000 {
		t.Errorf("alerts should be capped at 1000, got %d", len(dd.alerts))
	}

	// Cleanup should cap at 1000
	dd.cleanup()

	if len(dd.alerts) > 1000 {
		t.Errorf("after cleanup, alerts should be capped at 1000, got %d", len(dd.alerts))
	}
}

// Test DDoSDetector cleanup of old requests
func TestDDoSDetector_RequestCleanup(t *testing.T) {
	dd := NewDDoSDetector(10, 100*time.Millisecond)

	ip := "192.168.1.1"
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip + ":8080"

	// Create a request
	dd.IsSuspicious(ip, req)

	if _, exists := dd.requests[ip]; !exists {
		t.Fatal("request pattern should exist")
	}

	// Wait for cleanup window to pass
	time.Sleep(200 * time.Millisecond)

	// Cleanup should remove old patterns
	dd.cleanup()

	if _, exists := dd.requests[ip]; exists {
		t.Error("old request pattern should have been cleaned up")
	}
}
