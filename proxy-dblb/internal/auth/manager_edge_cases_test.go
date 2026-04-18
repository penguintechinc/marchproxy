//go:build ci

package auth

import (
	"context"
	"testing"
	"time"

	"marchproxy-dblb/internal/logging"
)

func TestAuthenticateDatabaseAccessDenied(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	user := &User{
		Username: "user",
		Enabled:  true,
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "user",
		Database: "alloweddb",
		Actions:  []string{"read"},
	}
	m.AddPermission(perm)

	ctx := context.Background()
	result := m.Authenticate(ctx, "user", "denieddb", "postgresql")
	if result {
		t.Error("Expected authentication to fail for denied database")
	}
}

func TestAuthenticatePermissionExpired(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	user := &User{
		Username: "user",
		Enabled:  true,
	}
	m.AddUser(user)

	expiredTime := time.Now().Add(-1 * time.Hour)
	perm := &Permission{
		Username: "user",
		Database: "*",
		Actions:  []string{"read"},
		TimeLimit: &expiredTime,
	}
	m.AddPermission(perm)

	ctx := context.Background()
	result := m.Authenticate(ctx, "user", "testdb", "postgresql")
	if result {
		t.Error("Expected authentication to fail for expired permission")
	}
}

func TestAuthorizeTableRestriction(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	perm := &Permission{
		Username: "user",
		Database: "testdb",
		Table:    "allowed_table",
		Actions:  []string{"read", "write"},
	}
	m.AddPermission(perm)

	ctx := context.Background()

	result := m.Authorize(ctx, "user", "testdb", "allowed_table", false)
	if !result {
		t.Error("Expected authorization to succeed for allowed table")
	}

	result = m.Authorize(ctx, "user", "testdb", "denied_table", false)
	if result {
		t.Error("Expected authorization to fail for denied table")
	}
}

func TestAuthorizeNoPermission(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	ctx := context.Background()
	result := m.Authorize(ctx, "unknownuser", "testdb", "table", false)
	if result {
		t.Error("Expected authorization to fail for user without permissions")
	}
}

func TestIPWhitelistMultiple(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	user := &User{
		Username: "user",
		Enabled:  true,
		AllowedIPs: []string{
			"192.168.1.100",
			"10.0.0.0/24",
			"172.16.0.50",
		},
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "user",
		Database: "*",
	}
	m.AddPermission(perm)

	tests := []struct {
		ip     string
		expect bool
	}{
		{"192.168.1.100", true},
		{"10.0.0.5", true},
		{"10.0.1.1", false},
		{"172.16.0.50", true},
		{"172.16.0.51", false},
	}

	for _, tt := range tests {
		result := m.AuthenticateWithIP(context.Background(), "user", "db", "pg", tt.ip)
		if result != tt.expect {
			t.Errorf("IP %s: expected %v, got %v", tt.ip, tt.expect, result)
		}
	}
}

func TestCacheAuthResultWithoutRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger) // nil redis client

	user := &User{
		Username: "user",
		Enabled:  true,
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "user",
		Database: "*",
	}
	m.AddPermission(perm)

	ctx := context.Background()
	// Should succeed without redis (cache skipped)
	result := m.Authenticate(ctx, "user", "testdb", "postgresql")
	if !result {
		t.Error("Expected authentication to succeed without redis")
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	password := "test-pass"
	hash1 := m.HashPassword(password)
	hash2 := m.HashPassword(password)

	if hash1 != hash2 {
		t.Error("Same password should produce same hash")
	}

	user := &User{
		Username:     "user",
		PasswordHash: hash1,
		Enabled:      true,
	}
	m.AddUser(user)

	if !m.ValidatePassword("user", password) {
		t.Error("ValidatePassword should succeed with correct hash")
	}
}

func TestValidatePasswordNonexistentUser(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	result := m.ValidatePassword("nonexistent", "password")
	if result {
		t.Error("ValidatePassword should fail for nonexistent user")
	}
}

func TestValidateAPIKeyDisabledUser(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	m := NewManager(nil, logger)

	key := m.GenerateAPIKey()
	user := &User{
		Username: "disabled",
		APIKey:   key,
		Enabled:  false, // disabled
	}
	m.AddUser(user)

	ctx := context.Background()
	_, ok := m.ValidateAPIKey(ctx, key, "testdb", "postgresql")
	if ok {
		t.Error("ValidateAPIKey should fail for disabled user")
	}
}
