package keys_test

import (
	"strings"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/keys"
)

func TestNewManager(t *testing.T) {
	m := keys.NewManager()
	if m == nil {
		t.Fatal("expected manager to be created, got nil")
	}
}

func TestGenerateKey(t *testing.T) {
	m := keys.NewManager()
	rawKey, vk, err := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if rawKey == "" {
		t.Fatal("expected non-empty raw key")
		return
	}
	if !strings.HasPrefix(rawKey, "sk-mp-") {
		t.Errorf("expected key to start with sk-mp-, got %s", rawKey)
	}

	if vk == nil {
		t.Fatal("expected virtual key to be created")
	}
	if vk.Name != "test-key" {
		t.Errorf("expected name test-key, got %s", vk.Name)
	}
	if vk.UserID != "user123" {
		t.Errorf("expected UserID user123, got %s", vk.UserID)
	}
	if vk.TeamID != "team456" {
		t.Errorf("expected TeamID team456, got %s", vk.TeamID)
	}
	if !vk.IsActive {
		t.Error("expected key to be active")
	}
}

func TestGenerateKeyWithoutExpiration(t *testing.T) {
	m := keys.NewManager()
	_, vk, err := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if vk.ExpiresAt != nil {
		t.Errorf("expected ExpiresAt to be nil, got %v", vk.ExpiresAt)
	}
}

func TestGenerateKeyWithExpiration(t *testing.T) {
	m := keys.NewManager()
	_, vk, err := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 7)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if vk.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}

	timeUntilExpiry := vk.ExpiresAt.Sub(vk.CreatedAt)
	expectedDays := 7 * 24 * time.Hour
	// Allow 1 minute tolerance
	if timeUntilExpiry < expectedDays-time.Minute || timeUntilExpiry > expectedDays+time.Minute {
		t.Errorf("expected expiration in ~7 days, got %v", timeUntilExpiry)
	}
}

func TestGenerateKeyDefaultModels(t *testing.T) {
	m := keys.NewManager()
	_, vk, err := m.GenerateKey("test-key", "user123", "team456", []string{}, 10.0, 1000, 60, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vk.AllowedModels) != 1 || vk.AllowedModels[0] != "*" {
		t.Errorf("expected default wildcard model, got %v", vk.AllowedModels)
	}
}

func TestGenerateKeyMultipleModels(t *testing.T) {
	m := keys.NewManager()
	models := []string{"gpt-4", "gpt-3.5-turbo", "claude-3"}
	_, vk, err := m.GenerateKey("test-key", "user123", "team456", models, 10.0, 1000, 60, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vk.AllowedModels) != 3 {
		t.Errorf("expected 3 models, got %d", len(vk.AllowedModels))
	}
}

func TestValidateKeyValid(t *testing.T) {
	m := keys.NewManager()
	rawKey, _, err := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result := m.ValidateKey(rawKey)
	if !result.Valid {
		t.Errorf("expected valid key, got %s", result.Error)
	}
	if result.Key == nil {
		t.Fatal("expected key data in result")
	}
}

func TestValidateKeyNotFound(t *testing.T) {
	m := keys.NewManager()
	result := m.ValidateKey("sk-mp-nonexistent-key")
	if result.Valid {
		t.Error("expected invalid key")
	}
	if result.Error != "key not found" {
		t.Errorf("expected 'key not found', got %s", result.Error)
	}
}

func TestVirtualKeyIsExpired(t *testing.T) {
	now := time.Now()
	expiredTime := now.AddDate(0, 0, -1)
	notExpiredTime := now.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		expiresAt *time.Time
		expected bool
	}{
		{"no expiration", nil, false},
		{"expired", &expiredTime, true},
		{"not expired", &notExpiredTime, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vk := &keys.VirtualKey{ExpiresAt: tt.expiresAt}
			result := vk.IsExpired()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestVirtualKeyIsBudgetExceeded(t *testing.T) {
	tests := []struct {
		name       string
		maxBudget  float64
		spent      float64
		expected   bool
	}{
		{"no budget limit", 0, 0, false},
		{"budget not exceeded", 10.0, 5.0, false},
		{"budget exactly met", 10.0, 10.0, true},
		{"budget exceeded", 10.0, 15.0, true},
		{"negative budget", -1, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vk := &keys.VirtualKey{MaxBudget: tt.maxBudget, Spent: tt.spent}
			result := vk.IsBudgetExceeded()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateKeyExpired(t *testing.T) {
	m := keys.NewManager()
	rawKey, _, _ := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 0)

	result := m.ValidateKey(rawKey)
	if !result.Valid {
		t.Fatal("expected valid key initially")
	}

	// Manually expire the key
	expiredTime := time.Now().Add(-time.Hour)
	result.Key.ExpiresAt = &expiredTime

	// Re-validate
	result2 := m.ValidateKey(rawKey)
	if result2.Valid {
		t.Error("expected expired key to be invalid")
	}
	if result2.Error != "key has expired" {
		t.Errorf("expected 'key has expired', got %s", result2.Error)
	}
}

func TestValidateKeyInactive(t *testing.T) {
	m := keys.NewManager()
	rawKey, _, _ := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)

	result := m.ValidateKey(rawKey)
	if !result.Valid {
		t.Fatal("expected valid key initially")
	}

	// Deactivate the key
	m.DeactivateKey(result.KeyID)

	// Re-validate
	result2 := m.ValidateKey(rawKey)
	if result2.Valid {
		t.Error("expected inactive key to be invalid")
	}
	if result2.Error != "key is inactive" {
		t.Errorf("expected 'key is inactive', got %s", result2.Error)
	}
}

func TestValidateKeyBudgetExceeded(t *testing.T) {
	m := keys.NewManager()
	rawKey, vk, _ := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)

	// Set spending to exceed budget
	vk.Spent = 11.0

	result := m.ValidateKey(rawKey)
	if result.Valid {
		t.Error("expected budget exceeded key to be invalid")
	}
	if result.Error != "budget exceeded" {
		t.Errorf("expected 'budget exceeded', got %s", result.Error)
	}
}

func TestRecordUsage(t *testing.T) {
	m := keys.NewManager()
	rawKey, _, _ := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)

	result := m.ValidateKey(rawKey)
	keyID := result.KeyID

	m.RecordUsage(keyID, 100, 0.5)

	result = m.ValidateKey(rawKey)
	if result.Key.Spent != 0.5 {
		t.Errorf("expected spent 0.5, got %f", result.Key.Spent)
	}
	if result.Key.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", result.Key.TotalRequests)
	}
}

func TestRecordUsageMultipleTimes(t *testing.T) {
	m := keys.NewManager()
	rawKey, _, _ := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)

	result := m.ValidateKey(rawKey)
	keyID := result.KeyID

	m.RecordUsage(keyID, 100, 0.5)
	m.RecordUsage(keyID, 200, 1.0)
	m.RecordUsage(keyID, 150, 0.75)

	result = m.ValidateKey(rawKey)
	if result.Key.Spent != 2.25 {
		t.Errorf("expected spent 2.25, got %f", result.Key.Spent)
	}
	if result.Key.TotalRequests != 3 {
		t.Errorf("expected 3 requests, got %d", result.Key.TotalRequests)
	}
}

func TestRecordUsageNonexistentKey(t *testing.T) {
	m := keys.NewManager()
	m.RecordUsage("nonexistent-key", 100, 0.5)
	// Should not panic
}

func TestDeactivateKey(t *testing.T) {
	m := keys.NewManager()
	rawKey, _, _ := m.GenerateKey("test-key", "user123", "team456", []string{"gpt-4"}, 10.0, 1000, 60, 30)

	result := m.ValidateKey(rawKey)
	keyID := result.KeyID

	success := m.DeactivateKey(keyID)
	if !success {
		t.Error("expected deactivation to succeed")
	}

	result = m.ValidateKey(rawKey)
	if result.Valid {
		t.Error("expected deactivated key to be invalid")
	}
}

func TestDeactivateKeyNonexistent(t *testing.T) {
	m := keys.NewManager()
	success := m.DeactivateKey("nonexistent-key-id")
	if success {
		t.Error("expected deactivation to fail for nonexistent key")
	}
}

func TestGenerateMultipleKeys(t *testing.T) {
	m := keys.NewManager()
	rawKey1, vk1, _ := m.GenerateKey("key1", "user1", "team1", []string{"gpt-4"}, 10.0, 1000, 60, 0)
	rawKey2, vk2, _ := m.GenerateKey("key2", "user2", "team2", []string{"gpt-3.5"}, 5.0, 500, 30, 0)

	if rawKey1 == rawKey2 {
		t.Error("expected different raw keys")
	}
	if vk1.ID == vk2.ID {
		t.Error("expected different key IDs")
	}

	result1 := m.ValidateKey(rawKey1)
	result2 := m.ValidateKey(rawKey2)

	if !result1.Valid || !result2.Valid {
		t.Error("expected both keys to be valid")
	}
}
