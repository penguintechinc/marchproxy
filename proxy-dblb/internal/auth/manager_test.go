package auth

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewManager(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.users == nil {
		t.Error("users map not initialized")
	}

	if m.permissions == nil {
		t.Error("permissions map not initialized")
	}
}

func TestAddAndGetUser(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	user := &User{
		Username:     "testuser",
		PasswordHash: m.HashPassword("testpass"),
		Enabled:      true,
		RateLimit:    100,
		AllowedIPs:   []string{"192.168.1.0/24", "10.0.0.1"},
		CreatedAt:    time.Now(),
	}

	m.AddUser(user)

	retrieved, ok := m.GetUser("testuser")
	if !ok {
		t.Fatal("GetUser returned false for existing user")
	}

	if retrieved.Username != "testuser" {
		t.Errorf("Username mismatch: got %s, want testuser", retrieved.Username)
	}

	if retrieved.RateLimit != 100 {
		t.Errorf("RateLimit mismatch: got %d, want 100", retrieved.RateLimit)
	}

	// Test non-existent user
	_, ok = m.GetUser("nonexistent")
	if ok {
		t.Error("GetUser returned true for non-existent user")
	}
}

func TestAddAndGetPermission(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	perm := &Permission{
		Username: "testuser",
		Database: "testdb",
		Table:    "*",
		Actions:  []string{"read", "write"},
	}

	m.AddPermission(perm)

	retrieved, ok := m.GetPermission("testuser")
	if !ok {
		t.Fatal("GetPermission returned false for existing permission")
	}

	if retrieved.Database != "testdb" {
		t.Errorf("Database mismatch: got %s, want testdb", retrieved.Database)
	}

	if len(retrieved.Actions) != 2 {
		t.Errorf("Actions count mismatch: got %d, want 2", len(retrieved.Actions))
	}
}

func TestAuthenticate(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)
	ctx := context.Background()

	// Add user and permission
	user := &User{
		Username:     "authuser",
		PasswordHash: m.HashPassword("pass"),
		Enabled:      true,
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "authuser",
		Database: "*",
		Table:    "*",
		Actions:  []string{"*"},
	}
	m.AddPermission(perm)

	// Test successful auth
	if !m.Authenticate(ctx, "authuser", "anydb", "mysql") {
		t.Error("Authenticate returned false for valid user")
	}

	// Test disabled user
	disabledUser := &User{
		Username: "disabled",
		Enabled:  false,
	}
	m.AddUser(disabledUser)

	if m.Authenticate(ctx, "disabled", "anydb", "mysql") {
		t.Error("Authenticate returned true for disabled user")
	}

	// Test non-existent user
	if m.Authenticate(ctx, "nonexistent", "anydb", "mysql") {
		t.Error("Authenticate returned true for non-existent user")
	}
}

func TestAuthenticateWithIP(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)
	ctx := context.Background()

	user := &User{
		Username:   "ipuser",
		Enabled:    true,
		AllowedIPs: []string{"192.168.1.0/24", "10.0.0.5"},
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "ipuser",
		Database: "*",
		Actions:  []string{"*"},
	}
	m.AddPermission(perm)

	// Test allowed IP (CIDR)
	if !m.AuthenticateWithIP(ctx, "ipuser", "db", "mysql", "192.168.1.50") {
		t.Error("AuthenticateWithIP failed for IP in allowed CIDR")
	}

	// Test allowed IP (exact)
	if !m.AuthenticateWithIP(ctx, "ipuser", "db", "mysql", "10.0.0.5") {
		t.Error("AuthenticateWithIP failed for exact allowed IP")
	}

	// Test disallowed IP
	if m.AuthenticateWithIP(ctx, "ipuser", "db", "mysql", "172.16.0.1") {
		t.Error("AuthenticateWithIP succeeded for disallowed IP")
	}
}

func TestAuthorize(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)
	ctx := context.Background()

	// User with read-only access
	readPerm := &Permission{
		Username: "reader",
		Database: "proddb",
		Table:    "*",
		Actions:  []string{"read"},
	}
	m.AddPermission(readPerm)

	// Test read access
	if !m.Authorize(ctx, "reader", "proddb", "users", false) {
		t.Error("Authorize failed for read access")
	}

	// Test write access (should fail)
	if m.Authorize(ctx, "reader", "proddb", "users", true) {
		t.Error("Authorize succeeded for unauthorized write")
	}

	// User with full access
	fullPerm := &Permission{
		Username: "admin",
		Database: "*",
		Table:    "*",
		Actions:  []string{"*"},
	}
	m.AddPermission(fullPerm)

	if !m.Authorize(ctx, "admin", "anydb", "anytable", true) {
		t.Error("Authorize failed for admin with full access")
	}
}

func TestHashPassword(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	hash1 := m.HashPassword("password123")
	hash2 := m.HashPassword("password123")

	if hash1 != hash2 {
		t.Error("Same password produced different hashes")
	}

	hash3 := m.HashPassword("differentpass")
	if hash1 == hash3 {
		t.Error("Different passwords produced same hash")
	}

	if len(hash1) != 64 { // SHA256 produces 64 hex chars
		t.Errorf("Hash length wrong: got %d, want 64", len(hash1))
	}
}

func TestValidatePassword(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	user := &User{
		Username:     "passuser",
		PasswordHash: m.HashPassword("secretpass"),
		Enabled:      true,
	}
	m.AddUser(user)

	if !m.ValidatePassword("passuser", "secretpass") {
		t.Error("ValidatePassword failed for correct password")
	}

	if m.ValidatePassword("passuser", "wrongpass") {
		t.Error("ValidatePassword succeeded for wrong password")
	}

	if m.ValidatePassword("nonexistent", "anypass") {
		t.Error("ValidatePassword succeeded for non-existent user")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	key1 := m.GenerateAPIKey()
	key2 := m.GenerateAPIKey()

	if key1 == key2 {
		t.Error("GenerateAPIKey produced duplicate keys")
	}

	if len(key1) < 32 {
		t.Errorf("API key too short: got %d chars", len(key1))
	}
}

func TestValidateAPIKey(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)
	ctx := context.Background()

	apiKey := m.GenerateAPIKey()

	user := &User{
		Username: "apiuser",
		APIKey:   apiKey,
		Enabled:  true,
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "apiuser",
		Database: "*",
		Actions:  []string{"*"},
	}
	m.AddPermission(perm)

	username, ok := m.ValidateAPIKey(ctx, apiKey, "testdb", "mysql")
	if !ok {
		t.Error("ValidateAPIKey failed for valid key")
	}
	if username != "apiuser" {
		t.Errorf("Username mismatch: got %s, want apiuser", username)
	}

	// Test invalid key
	_, ok = m.ValidateAPIKey(ctx, "invalidkey", "testdb", "mysql")
	if ok {
		t.Error("ValidateAPIKey succeeded for invalid key")
	}

	// Test empty key
	_, ok = m.ValidateAPIKey(ctx, "", "testdb", "mysql")
	if ok {
		t.Error("ValidateAPIKey succeeded for empty key")
	}
}

func TestIsIPAllowed(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	allowedIPs := []string{
		"192.168.1.0/24",
		"10.0.0.5",
		"172.16.0.0/16",
	}

	tests := []struct {
		ip      string
		allowed bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.254", true},
		{"192.168.2.1", false},
		{"10.0.0.5", true},
		{"10.0.0.6", false},
		{"172.16.50.100", true},
		{"172.17.0.1", false},
		{"8.8.8.8", false},
	}

	for _, tt := range tests {
		result := m.isIPAllowed(tt.ip, allowedIPs)
		if result != tt.allowed {
			t.Errorf("isIPAllowed(%s) = %v, want %v", tt.ip, result, tt.allowed)
		}
	}
}

func TestCheckTLSRequired(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	tlsUser := &User{
		Username:   "tlsuser",
		RequireTLS: true,
	}
	m.AddUser(tlsUser)

	noTLSUser := &User{
		Username:   "notlsuser",
		RequireTLS: false,
	}
	m.AddUser(noTLSUser)

	if !m.CheckTLSRequired("tlsuser") {
		t.Error("CheckTLSRequired returned false for TLS user")
	}

	if m.CheckTLSRequired("notlsuser") {
		t.Error("CheckTLSRequired returned true for non-TLS user")
	}

	if m.CheckTLSRequired("nonexistent") {
		t.Error("CheckTLSRequired returned true for non-existent user")
	}
}

func TestGetUserRateLimit(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	user := &User{
		Username:  "ratelimited",
		RateLimit: 500,
	}
	m.AddUser(user)

	limit := m.GetUserRateLimit("ratelimited")
	if limit != 500 {
		t.Errorf("GetUserRateLimit = %d, want 500", limit)
	}

	limit = m.GetUserRateLimit("nonexistent")
	if limit != 0 {
		t.Errorf("GetUserRateLimit for non-existent = %d, want 0", limit)
	}
}

func TestGetStats(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)

	m.AddUser(&User{Username: "user1"})
	m.AddUser(&User{Username: "user2"})
	m.AddPermission(&Permission{Username: "user1"})

	stats := m.GetStats()

	if stats["users_count"] != 2 {
		t.Errorf("users_count = %v, want 2", stats["users_count"])
	}

	if stats["permissions_count"] != 1 {
		t.Errorf("permissions_count = %v, want 1", stats["permissions_count"])
	}
}

func TestAccountExpiration(t *testing.T) {
	logger := logrus.New()
	m := NewManager(nil, logger)
	ctx := context.Background()

	// Expired user
	expiredTime := time.Now().Add(-24 * time.Hour)
	expiredUser := &User{
		Username:  "expired",
		Enabled:   true,
		ExpiresAt: &expiredTime,
	}
	m.AddUser(expiredUser)
	m.AddPermission(&Permission{Username: "expired", Database: "*", Actions: []string{"*"}})

	if m.Authenticate(ctx, "expired", "db", "mysql") {
		t.Error("Authenticate succeeded for expired user")
	}

	// Non-expired user
	futureTime := time.Now().Add(24 * time.Hour)
	validUser := &User{
		Username:  "valid",
		Enabled:   true,
		ExpiresAt: &futureTime,
	}
	m.AddUser(validUser)
	m.AddPermission(&Permission{Username: "valid", Database: "*", Actions: []string{"*"}})

	if !m.Authenticate(ctx, "valid", "db", "mysql") {
		t.Error("Authenticate failed for non-expired user")
	}
}
