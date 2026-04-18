package ratelimit_test

import (
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/ratelimit"
)

func TestDefaultConfig(t *testing.T) {
	cfg := ratelimit.DefaultConfig()
	if cfg.TPMLimit != 10000 {
		t.Errorf("expected TPMLimit 10000, got %d", cfg.TPMLimit)
	}
	if cfg.RPMLimit != 60 {
		t.Errorf("expected RPMLimit 60, got %d", cfg.RPMLimit)
	}
	if cfg.WindowSeconds != 60 {
		t.Errorf("expected WindowSeconds 60, got %d", cfg.WindowSeconds)
	}
	if !cfg.Enabled {
		t.Error("expected Enabled true")
	}
}

func TestNewLimiter(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	limiter := ratelimit.NewLimiter(defaults)
	if limiter == nil {
		t.Fatal("expected limiter to be created")
	}
}

func TestSetAndGetConfig(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	limiter := ratelimit.NewLimiter(defaults)

	customCfg := ratelimit.Config{
		TPMLimit:      5000,
		RPMLimit:      30,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter.SetConfig("key-123", customCfg)

	retrieved := limiter.GetConfig("key-123")
	if retrieved.TPMLimit != 5000 {
		t.Errorf("expected TPMLimit 5000, got %d", retrieved.TPMLimit)
	}
	if retrieved.RPMLimit != 30 {
		t.Errorf("expected RPMLimit 30, got %d", retrieved.RPMLimit)
	}
}

func TestGetConfigDefaultsForUnknownKey(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	limiter := ratelimit.NewLimiter(defaults)

	retrieved := limiter.GetConfig("unknown-key")
	if retrieved.TPMLimit != defaults.TPMLimit {
		t.Errorf("expected default TPMLimit %d, got %d", defaults.TPMLimit, retrieved.TPMLimit)
	}
}

func TestCheckLimitDisabled(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	defaults.Enabled = false
	limiter := ratelimit.NewLimiter(defaults)

	allowed, status := limiter.CheckLimit("key-123", 1000)
	if !allowed {
		t.Error("expected limit check to pass when disabled")
	}
	if status.IsLimited {
		t.Error("expected IsLimited to be false when disabled")
	}
}

func TestCheckLimitTPMExceeded(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      100,
		RPMLimit:      100,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	allowed, status := limiter.CheckLimit("key-123", 150)
	if allowed {
		t.Error("expected limit check to fail for TPM exceeded")
	}
	if !status.IsLimited {
		t.Error("expected IsLimited to be true")
	}
	if status.LimitReason != "tpm_exceeded" {
		t.Errorf("expected LimitReason tpm_exceeded, got %s", status.LimitReason)
	}
	_ = allowed // Suppress unused variable warning
}

func TestCheckLimitRPMExceeded(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      100000,
		RPMLimit:      3,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-123", 10)
	limiter.RecordRequest("key-123", 10)
	limiter.RecordRequest("key-123", 10)

	allowed, status := limiter.CheckLimit("key-123", 10)
	if allowed {
		t.Error("expected limit check to fail for RPM exceeded")
	}
	if status.LimitReason != "rpm_exceeded" {
		t.Errorf("expected LimitReason rpm_exceeded, got %s", status.LimitReason)
	}
}

func TestCheckLimitAllowed(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      1000,
		RPMLimit:      60,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	allowed, status := limiter.CheckLimit("key-123", 100)
	if !allowed {
		t.Errorf("expected limit check to pass, got IsLimited=%v", status.IsLimited)
	}
	if status.IsLimited {
		t.Error("expected IsLimited to be false")
	}
}

func TestRecordRequest(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-123", 100)

	// Second request should see increased counts
	allowed, status := limiter.CheckLimit("key-123", 100)
	if !allowed {
		t.Error("expected second request to pass")
	}
	if status.CurrentTPM != 100 {
		t.Errorf("expected CurrentTPM 100, got %d", status.CurrentTPM)
	}
	if status.CurrentRPM != 1 {
		t.Errorf("expected CurrentRPM 1, got %d", status.CurrentRPM)
	}
}

func TestRecordRequestDisabled(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      100,
		RPMLimit:      10,
		WindowSeconds: 60,
		Enabled:       false,
	}
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-123", 500)
	// Should not update counters

	allowed, status := limiter.CheckLimit("key-123", 100)
	if !allowed {
		t.Error("expected check to pass when disabled")
	}
	if status.CurrentTPM != 0 {
		t.Errorf("expected CurrentTPM 0 when disabled, got %d", status.CurrentTPM)
	}
}

func TestReset(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-123", 1000)
	limiter.Reset("key-123")

	allowed, status := limiter.CheckLimit("key-123", 100)
	if !allowed {
		t.Error("expected check to pass after reset")
	}
	if status.CurrentTPM != 0 {
		t.Errorf("expected CurrentTPM 0 after reset, got %d", status.CurrentTPM)
	}
}

func TestRemaining(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      1000,
		RPMLimit:      60,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-123", 300)
	allowed, status := limiter.CheckLimit("key-123", 100)
	if !allowed {
		t.Error("expected check to pass")
	}
	if status.RemainingTPM != 700 {
		t.Errorf("expected RemainingTPM 700, got %d", status.RemainingTPM)
	}
	if status.RemainingRPM != 59 {
		t.Errorf("expected RemainingRPM 59, got %d", status.RemainingRPM)
	}
}

func TestRemainingWithUnlimitedLimits(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      0,
		RPMLimit:      0,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	allowed, status := limiter.CheckLimit("key-123", 1000)
	if !allowed {
		t.Error("expected check to pass with unlimited limits")
	}
	if status.RemainingTPM != 0 {
		t.Errorf("expected RemainingTPM 0 for unlimited, got %d", status.RemainingTPM)
	}
	if status.RemainingRPM != 0 {
		t.Errorf("expected RemainingRPM 0 for unlimited, got %d", status.RemainingRPM)
	}
}

func TestCleanupExpired(t *testing.T) {
	defaults := ratelimit.DefaultConfig()
	limiter := ratelimit.NewLimiter(defaults)

	// Record some usage
	limiter.RecordRequest("key-1", 100)
	limiter.RecordRequest("key-2", 200)

	// Check that they were recorded
	_, status1Before := limiter.CheckLimit("key-1", 10)
	if status1Before.CurrentTPM == 0 {
		t.Fatal("expected CurrentTPM > 0 before cleanup")
	}

	// Sleep briefly to allow time for cleanup detection
	time.Sleep(10 * time.Millisecond)

	// Cleanup with very short idle time - should remove entries that haven't been used recently
	limiter.CleanupExpired(5 * time.Millisecond)

	// After cleanup, window should be reset
	_, status1After := limiter.CheckLimit("key-1", 10)
	if status1After.CurrentTPM != 0 {
		t.Errorf("expected CurrentTPM 0 after cleanup, got %d", status1After.CurrentTPM)
	}
}

func TestMultipleKeysIndependent(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      100,
		RPMLimit:      10,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-1", 50)
	limiter.RecordRequest("key-2", 70)

	allowed1, status1 := limiter.CheckLimit("key-1", 30)
	allowed2, status2 := limiter.CheckLimit("key-2", 20)

	if !allowed1 {
		t.Error("expected key-1 check to pass")
	}
	if !allowed2 {
		t.Error("expected key-2 check to pass")
	}

	if status1.CurrentTPM != 50 {
		t.Errorf("expected key-1 CurrentTPM 50, got %d", status1.CurrentTPM)
	}
	if status2.CurrentTPM != 70 {
		t.Errorf("expected key-2 CurrentTPM 70, got %d", status2.CurrentTPM)
	}
}

func TestPerKeyConfig(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      1000,
		RPMLimit:      60,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	customCfg := ratelimit.Config{
		TPMLimit:      100,
		RPMLimit:      10,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter.SetConfig("key-premium", customCfg)

	limiter.RecordRequest("key-premium", 90)
	allowed, status := limiter.CheckLimit("key-premium", 20)
	if allowed {
		t.Error("expected premium key to hit its lower limit")
	}
	if status.LimitReason != "tpm_exceeded" {
		t.Errorf("expected tpm_exceeded, got %s", status.LimitReason)
	}
}

func TestStatusValues(t *testing.T) {
	defaults := ratelimit.Config{
		TPMLimit:      500,
		RPMLimit:      30,
		WindowSeconds: 60,
		Enabled:       true,
	}
	limiter := ratelimit.NewLimiter(defaults)

	limiter.RecordRequest("key-123", 200)
	_, status := limiter.CheckLimit("key-123", 100)

	if status.CurrentTPM != 200 {
		t.Errorf("expected CurrentTPM 200, got %d", status.CurrentTPM)
	}
	if status.CurrentRPM != 1 {
		t.Errorf("expected CurrentRPM 1, got %d", status.CurrentRPM)
	}
	if status.RemainingTPM != 300 {
		t.Errorf("expected RemainingTPM 300, got %d", status.RemainingTPM)
	}
	if status.RemainingRPM != 29 {
		t.Errorf("expected RemainingRPM 29, got %d", status.RemainingRPM)
	}
}
