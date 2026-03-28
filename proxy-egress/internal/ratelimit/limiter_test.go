package ratelimit_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
	"marchproxy-egress/internal/ratelimit"
)

func TestErrorVarsDefined(t *testing.T) {
	if ratelimit.ErrRateLimitExceeded == nil {
		t.Error("ErrRateLimitExceeded should be defined and non-nil")
	}
	if ratelimit.ErrQuotaExceeded == nil {
		t.Error("ErrQuotaExceeded should be defined and non-nil")
	}
	if ratelimit.ErrTooManyRequests == nil {
		t.Error("ErrTooManyRequests should be defined and non-nil")
	}
	if ratelimit.ErrBlocked == nil {
		t.Error("ErrBlocked should be defined and non-nil")
	}
}

func TestErrorVarsDistinct(t *testing.T) {
	if ratelimit.ErrRateLimitExceeded == ratelimit.ErrQuotaExceeded {
		t.Error("ErrRateLimitExceeded and ErrQuotaExceeded must be distinct")
	}
	if ratelimit.ErrRateLimitExceeded == ratelimit.ErrTooManyRequests {
		t.Error("ErrRateLimitExceeded and ErrTooManyRequests must be distinct")
	}
	if ratelimit.ErrRateLimitExceeded == ratelimit.ErrBlocked {
		t.Error("ErrRateLimitExceeded and ErrBlocked must be distinct")
	}
	if ratelimit.ErrQuotaExceeded == ratelimit.ErrTooManyRequests {
		t.Error("ErrQuotaExceeded and ErrTooManyRequests must be distinct")
	}
}

func TestNewRateLimiterNotNil(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerIPLimit:  rate.Limit(100),
		PerIPBurst:  100,
		WindowSize:  time.Minute,
		CleanupInterval: time.Minute,
	}
	limiter := ratelimit.NewRateLimiter(cfg)
	if limiter == nil {
		t.Fatal("NewRateLimiter should return non-nil limiter")
	}
}

func TestRateLimiterAllowsRequest(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerIPLimit:      rate.Limit(1000),
		PerIPBurst:      1000,
		WindowSize:      time.Minute,
		CleanupInterval: time.Hour,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	err := limiter.Allow(req)
	if err != nil {
		t.Errorf("Allow should return nil for first request within limits, got %v", err)
	}
}

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerIPLimit:      rate.Limit(1),
		PerIPBurst:      1,
		WindowSize:      time.Minute,
		CleanupInterval: time.Hour,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:9999"

	// Consume the burst of 1
	_ = limiter.Allow(req)

	// Next request should be rate-limited
	err := limiter.Allow(req)
	if err == nil {
		t.Error("expected rate limit error after burst exhausted")
	}
	if err != ratelimit.ErrRateLimitExceeded {
		t.Errorf("expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestRateLimiterAPIKeyLimit(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerAPIKeyLimit:  rate.Limit(1000),
		PerAPIKeyBurst:  1000,
		PerIPLimit:      rate.Limit(10),
		PerIPBurst:      10,
		WindowSize:      time.Minute,
		CleanupInterval: time.Hour,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "test-api-key-123")
	req.RemoteAddr = "192.168.1.1:5000"

	err := limiter.Allow(req)
	if err != nil {
		t.Errorf("expected nil error for API key request, got %v", err)
	}
}

func TestRateLimiterUserIDHeader(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerUserLimit:    rate.Limit(1000),
		PerUserBurst:    1000,
		PerIPLimit:      rate.Limit(10),
		PerIPBurst:      10,
		WindowSize:      time.Minute,
		CleanupInterval: time.Hour,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-ID", "user-456")
	req.RemoteAddr = "192.168.1.2:5001"

	err := limiter.Allow(req)
	if err != nil {
		t.Errorf("expected nil error for user ID request, got %v", err)
	}
}

func TestRateLimiterGlobalLimit(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		GlobalLimit:     rate.Limit(1),
		GlobalBurst:     1,
		PerIPLimit:      rate.Limit(1000),
		PerIPBurst:      1000,
		WindowSize:      time.Minute,
		CleanupInterval: time.Hour,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "10.0.0.1:1111"
	_ = limiter.Allow(req1)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.0.0.2:2222"
	err := limiter.Allow(req2)
	if err == nil {
		t.Error("expected global rate limit to be hit after burst=1")
	}
}

func TestRateLimiterRateLimitHeadersDisabled(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerIPLimit:       rate.Limit(100),
		PerIPBurst:       100,
		WindowSize:       time.Minute,
		CleanupInterval:  time.Hour,
		RateLimitHeaders: false,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	headers := limiter.GetRateLimitHeaders("ip:127.0.0.1")
	if headers == nil {
		t.Fatal("GetRateLimitHeaders should return non-nil map")
	}
	if len(headers) != 0 {
		t.Errorf("expected empty headers when RateLimitHeaders disabled, got %d", len(headers))
	}
}

func TestRateLimiterRateLimitHeadersEnabled(t *testing.T) {
	cfg := ratelimit.RateLimiterConfig{
		PerIPLimit:       rate.Limit(100),
		PerIPBurst:       100,
		WindowSize:       time.Minute,
		CleanupInterval:  time.Hour,
		RateLimitHeaders: true,
	}
	limiter := ratelimit.NewRateLimiter(cfg)

	// First, make a request to create the limiter entry
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:80"
	_ = limiter.Allow(req)

	// Now headers should be available
	headers := limiter.GetRateLimitHeaders("ip:1.2.3.4")
	if headers == nil {
		t.Fatal("GetRateLimitHeaders should return non-nil map")
	}
	// Headers may be populated now
}

func TestIPBlocklistNotNil(t *testing.T) {
	bl := ratelimit.NewIPBlocklist(time.Hour)
	if bl == nil {
		t.Fatal("NewIPBlocklist should return non-nil")
	}
}

func TestIPBlocklistBlockAndCheck(t *testing.T) {
	bl := ratelimit.NewIPBlocklist(time.Hour)

	if bl.IsBlocked("192.168.1.100") {
		t.Error("IP should not be blocked before being added")
	}

	bl.Block("192.168.1.100", "test block", time.Hour)

	if !bl.IsBlocked("192.168.1.100") {
		t.Error("IP should be blocked after Block() call")
	}
}

func TestIPBlocklistExpiry(t *testing.T) {
	bl := ratelimit.NewIPBlocklist(time.Hour)

	// Block for a very short duration (already expired)
	bl.Block("10.0.0.1", "short block", -1*time.Second)

	// Should not be blocked since duration is in the past
	if bl.IsBlocked("10.0.0.1") {
		t.Error("IP should not be blocked after expiry")
	}
}

func TestIPBlocklistWhitelist(t *testing.T) {
	bl := ratelimit.NewIPBlocklist(time.Hour)

	bl.Block("10.0.0.2", "blocked", time.Hour)
	bl.Whitelist("10.0.0.2")

	if bl.IsBlocked("10.0.0.2") {
		t.Error("whitelisted IP should not be blocked")
	}
}

func TestNewIPBlocklistNotBlocked(t *testing.T) {
	bl := ratelimit.NewIPBlocklist(time.Hour)
	if bl.IsBlocked("8.8.8.8") {
		t.Error("new blocklist should not block unknown IPs")
	}
}

func TestAlertSeverityConstants(t *testing.T) {
	severities := []ratelimit.AlertSeverity{
		ratelimit.SeverityLow,
		ratelimit.SeverityMedium,
		ratelimit.SeverityHigh,
		ratelimit.SeverityCritical,
	}
	seen := make(map[ratelimit.AlertSeverity]bool)
	for _, s := range severities {
		if seen[s] {
			t.Errorf("duplicate AlertSeverity constant: %d", s)
		}
		seen[s] = true
	}
}

func TestExponentialBackoffCalculateDelay(t *testing.T) {
	eb := &ratelimit.ExponentialBackoff{
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		Multiplier:     2.0,
		BlockThreshold: 5,
	}

	d1 := eb.CalculateDelay(1)
	if d1 <= 0 {
		t.Errorf("CalculateDelay(1) should return positive duration, got %v", d1)
	}

	d2 := eb.CalculateDelay(2)
	if d2 < d1 {
		t.Errorf("CalculateDelay should increase with violations: d1=%v d2=%v", d1, d2)
	}
}

func TestExponentialBackoffMaxDelay(t *testing.T) {
	eb := &ratelimit.ExponentialBackoff{
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		Multiplier:     10.0,
		BlockThreshold: 5,
	}

	// Large violation count should be capped at MaxDelay
	d := eb.CalculateDelay(100)
	if d > eb.MaxDelay {
		t.Errorf("CalculateDelay should not exceed MaxDelay: got %v, max %v", d, eb.MaxDelay)
	}
}

func TestExponentialBackoffShouldBlock(t *testing.T) {
	eb := &ratelimit.ExponentialBackoff{
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		Multiplier:     2.0,
		BlockThreshold: 5,
	}

	if eb.ShouldBlock(4) {
		t.Error("ShouldBlock should return false below threshold")
	}
	if !eb.ShouldBlock(5) {
		t.Error("ShouldBlock should return true at threshold")
	}
	if !eb.ShouldBlock(10) {
		t.Error("ShouldBlock should return true above threshold")
	}
}

func TestNewDDoSDetectorNotNil(t *testing.T) {
	dd := ratelimit.NewDDoSDetector(100, time.Minute)
	if dd == nil {
		t.Fatal("NewDDoSDetector should return non-nil")
	}
}

func TestNewQuotaManagerNotNil(t *testing.T) {
	qm := ratelimit.NewQuotaManager(time.Hour)
	if qm == nil {
		t.Fatal("NewQuotaManager should return non-nil")
	}
}

func TestQuotaCreateAndUse(t *testing.T) {
	qm := ratelimit.NewQuotaManager(time.Hour)

	quota := qm.CreateQuota("client-1", 10)
	if quota == nil {
		t.Fatal("CreateQuota should return non-nil")
	}

	if !quota.HasRemaining() {
		t.Error("new quota should have remaining capacity")
	}

	remaining := quota.GetRemaining()
	if remaining != 10 {
		t.Errorf("expected remaining = 10, got %d", remaining)
	}

	quota.Use(3)
	if quota.GetRemaining() != 7 {
		t.Errorf("expected remaining = 7 after Use(3), got %d", quota.GetRemaining())
	}
}

func TestQuotaExhausted(t *testing.T) {
	qm := ratelimit.NewQuotaManager(time.Hour)
	quota := qm.CreateQuota("client-2", 2)

	quota.Use(2)
	if quota.HasRemaining() {
		t.Error("quota should be exhausted after using all capacity")
	}
}

func TestQuotaWarning(t *testing.T) {
	qm := ratelimit.NewQuotaManager(time.Hour)
	quota := qm.CreateQuota("client-3", 10)

	// Use 8 out of 10 = 80% which meets the default warning threshold
	quota.Use(8)
	if !quota.IsWarning() {
		t.Error("quota should trigger warning at >=80% usage (8/10 = 80%)")
	}
}

func TestQuotaGetReturnsExisting(t *testing.T) {
	qm := ratelimit.NewQuotaManager(time.Hour)
	_ = qm.CreateQuota("client-4", 100)

	got := qm.GetQuota("client-4")
	if got == nil {
		t.Fatal("GetQuota should return the created quota")
	}
}

func TestQuotaGetReturnsNilForMissing(t *testing.T) {
	qm := ratelimit.NewQuotaManager(time.Hour)
	got := qm.GetQuota("nonexistent")
	if got != nil {
		t.Error("GetQuota should return nil for unknown client")
	}
}

func TestLimitConfigFields(t *testing.T) {
	cfg := ratelimit.LimitConfig{
		Limit:  rate.Limit(50),
		Burst:  50,
		Quota:  10000,
		Window: time.Hour,
	}
	if cfg.Burst != 50 {
		t.Errorf("unexpected Burst: %d", cfg.Burst)
	}
	if cfg.Quota != 10000 {
		t.Errorf("unexpected Quota: %d", cfg.Quota)
	}
}
