// Package keys provides virtual API key management for the AILB service.
package keys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	keyPrefix       = "sk-mp"
	keyIDLength     = 16
	keySecretLength = 32
)

// VirtualKey represents a virtual API key and its constraints.
type VirtualKey struct {
	ID            string
	KeyHash       string
	Name          string
	UserID        string
	TeamID        string
	IsActive      bool
	CreatedAt     time.Time
	ExpiresAt     *time.Time
	LastUsed      *time.Time
	AllowedModels []string
	MaxBudget     float64
	Spent         float64
	TPMLimit      int
	RPMLimit      int
	TotalRequests int64
}

// IsExpired returns true if the key has passed its expiration date.
func (k *VirtualKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsBudgetExceeded returns true if spending has reached the budget limit.
func (k *VirtualKey) IsBudgetExceeded() bool {
	if k.MaxBudget <= 0 {
		return false
	}
	return k.Spent >= k.MaxBudget
}

// ValidationResult holds the result of key validation.
type ValidationResult struct {
	Valid  bool
	KeyID  string
	Key    *VirtualKey
	Error  string
}

// Manager manages virtual API keys with in-memory storage.
type Manager struct {
	mu           sync.RWMutex
	keys         map[string]*VirtualKey   // keyID -> key
	hashIndex    map[string]string        // keyHash -> keyID
}

// NewManager creates a new key manager.
func NewManager() *Manager {
	return &Manager{
		keys:      make(map[string]*VirtualKey),
		hashIndex: make(map[string]string),
	}
}

// GenerateKey creates a new virtual API key and returns the raw key string.
func (m *Manager) GenerateKey(name, userID, teamID string, allowedModels []string, maxBudget float64, tpmLimit, rpmLimit int, expiresDays int) (string, *VirtualKey, error) {
	keyID, err := randomHex(keyIDLength)
	if err != nil {
		return "", nil, fmt.Errorf("generate key ID: %w", err)
	}

	secret, err := randomHex(keySecretLength)
	if err != nil {
		return "", nil, fmt.Errorf("generate secret: %w", err)
	}

	rawKey := fmt.Sprintf("%s-%s-%s", keyPrefix, keyID, secret)
	hash := hashKey(rawKey)

	var expiresAt *time.Time
	if expiresDays > 0 {
		t := time.Now().Add(time.Duration(expiresDays) * 24 * time.Hour)
		expiresAt = &t
	}

	if len(allowedModels) == 0 {
		allowedModels = []string{"*"}
	}

	vk := &VirtualKey{
		ID:            keyID,
		KeyHash:       hash,
		Name:          name,
		UserID:        userID,
		TeamID:        teamID,
		IsActive:      true,
		CreatedAt:     time.Now(),
		ExpiresAt:     expiresAt,
		AllowedModels: allowedModels,
		MaxBudget:     maxBudget,
		TPMLimit:      tpmLimit,
		RPMLimit:      rpmLimit,
	}

	m.mu.Lock()
	m.keys[keyID] = vk
	m.hashIndex[hash] = keyID
	m.mu.Unlock()

	slog.Info("generated virtual key", "key_id", keyID, "user_id", userID, "name", name)
	return rawKey, vk, nil
}

// ValidateKey validates a raw API key string and returns the result.
func (m *Manager) ValidateKey(rawKey string) ValidationResult {
	hash := hashKey(rawKey)

	m.mu.RLock()
	keyID, exists := m.hashIndex[hash]
	if !exists {
		m.mu.RUnlock()
		return ValidationResult{Valid: false, Error: "key not found"}
	}

	vk, exists := m.keys[keyID]
	if !exists {
		m.mu.RUnlock()
		return ValidationResult{Valid: false, KeyID: keyID, Error: "key data not found"}
	}
	m.mu.RUnlock()

	if !vk.IsActive {
		return ValidationResult{Valid: false, KeyID: keyID, Error: "key is inactive"}
	}
	if vk.IsExpired() {
		return ValidationResult{Valid: false, KeyID: keyID, Error: "key has expired"}
	}
	if vk.IsBudgetExceeded() {
		return ValidationResult{Valid: false, KeyID: keyID, Error: "budget exceeded"}
	}

	return ValidationResult{Valid: true, KeyID: keyID, Key: vk}
}

// RecordUsage records token and cost usage against a key.
func (m *Manager) RecordUsage(keyID string, tokens int, cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	vk, exists := m.keys[keyID]
	if !exists {
		return
	}

	vk.Spent += cost
	vk.TotalRequests++
	now := time.Now()
	vk.LastUsed = &now
}

// DeactivateKey soft-deletes a key by marking it inactive.
func (m *Manager) DeactivateKey(keyID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	vk, exists := m.keys[keyID]
	if !exists {
		return false
	}
	vk.IsActive = false
	return true
}

func hashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
