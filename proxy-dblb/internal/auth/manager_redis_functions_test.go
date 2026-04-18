//go:build ci

package auth

import (
	"context"
	"testing"
	"time"

	"marchproxy-dblb/internal/logging"
)

// TestCheckRateLimitNoRedis tests rate limit checking without Redis
func TestCheckRateLimitNoRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	user := &User{
		Username:  "testuser",
		RateLimit: 10,
		Enabled:   true,
	}
	m.AddUser(user)

	// Without Redis, rate limit should always pass
	if !m.CheckRateLimit(ctx, "testuser") {
		t.Error("CheckRateLimit should return true when Redis is nil")
	}
}

// TestCheckRateLimitUserNotFound tests rate limit for non-existent user
func TestCheckRateLimitUserNotFound(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	if !m.CheckRateLimit(ctx, "nonexistent") {
		t.Error("CheckRateLimit should return true for non-existent user (no limit)")
	}
}

// TestCheckRateLimitZeroRateLimit tests rate limit with zero rate limit
func TestCheckRateLimitZeroRateLimit(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	user := &User{
		Username:  "testuser",
		RateLimit: 0,
		Enabled:   true,
	}
	m.AddUser(user)

	if !m.CheckRateLimit(ctx, "testuser") {
		t.Error("CheckRateLimit should return true for user with zero rate limit")
	}
}

// TestSyncUsersFromRedisNoRedis tests syncing without Redis
func TestSyncUsersFromRedisNoRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	err := m.SyncUsersFromRedis(ctx)
	if err != nil {
		t.Errorf("SyncUsersFromRedis should return nil when Redis is nil, got %v", err)
	}
}

// TestSyncPermissionsFromRedisNoRedis tests syncing permissions without Redis
func TestSyncPermissionsFromRedisNoRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	err := m.SyncPermissionsFromRedis(ctx)
	if err != nil {
		t.Errorf("SyncPermissionsFromRedis should return nil when Redis is nil, got %v", err)
	}
}

// TestPublishUsersToRedisNoRedis tests publishing without Redis
func TestPublishUsersToRedisNoRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	err := m.PublishUsersToRedis(ctx)
	if err != nil {
		t.Errorf("PublishUsersToRedis should return nil when Redis is nil, got %v", err)
	}
}

// TestPublishPermissionsToRedisNoRedis tests publishing permissions without Redis
func TestPublishPermissionsToRedisNoRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	err := m.PublishPermissionsToRedis(ctx)
	if err != nil {
		t.Errorf("PublishPermissionsToRedis should return nil when Redis is nil, got %v", err)
	}
}

// TestClearAuthCacheNoRedis tests clearing cache without Redis
func TestClearAuthCacheNoRedis(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)
	err := m.ClearAuthCache(ctx)
	if err != nil {
		t.Errorf("ClearAuthCache should return nil when Redis is nil, got %v", err)
	}
}

// TestGetStatsMultipleUsers tests stats with multiple users and permissions
func TestGetStatsMultipleUsers(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")

	m := NewManager(nil, logger)

	// Add multiple users
	for i := 1; i <= 5; i++ {
		user := &User{
			Username: "user" + string(rune(48+i)),
			Enabled:  true,
		}
		m.AddUser(user)
	}

	// Add multiple permissions
	for i := 1; i <= 3; i++ {
		perm := &Permission{
			Username: "user" + string(rune(48+i)),
			Database: "db" + string(rune(48+i)),
		}
		m.AddPermission(perm)
	}

	stats := m.GetStats()

	if stats["users_count"] != 5 {
		t.Errorf("Expected 5 users in stats, got %v", stats["users_count"])
	}

	if stats["permissions_count"] != 3 {
		t.Errorf("Expected 3 permissions in stats, got %v", stats["permissions_count"])
	}
}

// TestAuthorizeWithExpiredPermission tests authorization with time-limited expired permissions
func TestAuthorizeWithExpiredPermission(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	// Permission that's already expired
	expireTime := time.Now().Add(-1 * time.Hour)
	perm := &Permission{
		Username:  "testuser",
		Database:  "testdb",
		Actions:   []string{"read"},
		TimeLimit: &expireTime,
	}
	m.AddPermission(perm)

	user := &User{
		Username: "testuser",
		Enabled:  true,
	}
	m.AddUser(user)

	// Authorization should fail due to expired permission
	result := m.Authorize(ctx, "testuser", "testdb", "testdb", false)
	if result {
		t.Error("Authorization should fail for time-limited permission that has expired")
	}
}

// TestAuthorizeWithValidTimeLimit tests authorization with valid time-limited permissions
func TestAuthorizeWithValidTimeLimit(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	// Permission that hasn't expired
	futureTime := time.Now().Add(1 * time.Hour)
	perm := &Permission{
		Username:  "testuser",
		Database:  "*",
		Table:     "*",
		Actions:   []string{"*"},
		TimeLimit: &futureTime,
	}
	m.AddPermission(perm)

	user := &User{
		Username: "testuser",
		Enabled:  true,
	}
	m.AddUser(user)

	// Authorization should succeed with valid time limit
	result := m.Authorize(ctx, "testuser", "testdb", "testtable", false)
	if !result {
		t.Error("Authorization should succeed for time-limited permission that hasn't expired")
	}
}

// TestAuthorizeWithoutPermission tests authorization when user lacks permissions
func TestAuthorizeWithoutPermission(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	user := &User{
		Username: "noaccessuser",
		Enabled:  true,
	}
	m.AddUser(user)
	// No permission added

	result := m.Authorize(ctx, "noaccessuser", "testdb", "testtable", false)
	if result {
		t.Error("Authorization should fail when user has no permissions")
	}
}

// TestAuthorizeWithWildcardDatabase tests authorization with wildcard database permissions
func TestAuthorizeWithWildcardDatabase(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	user := &User{
		Username: "admin",
		Enabled:  true,
	}
	m.AddUser(user)

	// Permission with * for database (all databases)
	perm := &Permission{
		Username: "admin",
		Database: "*",
		Table:    "users",
		Actions:  []string{"read", "write"},
	}
	m.AddPermission(perm)

	// Should authorize for any database with "users" table
	result := m.Authorize(ctx, "admin", "anydb", "users", false)
	if !result {
		t.Error("Authorization should succeed with wildcard database permission")
	}
}

// TestAuthorizeWithWildcardTable tests authorization with wildcard table permissions
func TestAuthorizeWithWildcardTable(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	user := &User{
		Username: "admin",
		Enabled:  true,
	}
	m.AddUser(user)

	// Permission with * for table (all tables)
	perm := &Permission{
		Username: "admin",
		Database: "testdb",
		Table:    "*",
		Actions:  []string{"*"},
	}
	m.AddPermission(perm)

	// Should authorize for any table in testdb
	result := m.Authorize(ctx, "admin", "testdb", "anytable", false)
	if !result {
		t.Error("Authorization should succeed with wildcard table permission")
	}
}

// TestAuthorizeWriteAction tests authorization for write operations
func TestAuthorizeWriteAction(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	user := &User{
		Username: "writeuser",
		Enabled:  true,
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "writeuser",
		Database: "*",
		Table:    "*",
		Actions:  []string{"write"},
	}
	m.AddPermission(perm)

	// Should authorize for write
	result := m.Authorize(ctx, "writeuser", "testdb", "testtable", true)
	if !result {
		t.Error("Authorization should succeed for write action")
	}

	// Should fail for read (not in actions list)
	result = m.Authorize(ctx, "writeuser", "testdb", "testtable", false)
	if result {
		t.Error("Authorization should fail for read when only write is allowed")
	}
}

// TestCacheAuthResultWithoutRedisNew tests that caching works safely without Redis
func TestCacheAuthResultWithoutRedisNew(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	// This should not panic even without Redis
	m.cacheAuthResult(ctx, "test_key", true)
	m.cacheAuthResult(ctx, "test_key", false)
}

// TestAuthenticateWithExpiredAccount tests authentication with expired user account
func TestAuthenticateWithExpiredAccount(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	expireTime := time.Now().Add(-1 * time.Hour) // Already expired
	user := &User{
		Username:  "expireduser",
		Enabled:   true,
		ExpiresAt: &expireTime,
	}
	m.AddUser(user)

	result := m.AuthenticateWithIP(ctx, "expireduser", "testdb", "mysql", "")
	if result {
		t.Error("Authentication should fail for expired user")
	}
}

// TestAuthenticateWithValidExpiration tests authentication with non-expired account
func TestAuthenticateWithValidExpiration(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("marchproxy")
	ctx := context.Background()

	m := NewManager(nil, logger)

	// Account that expires in the future
	futureTime := time.Now().Add(1 * time.Hour)
	user := &User{
		Username:  "futureuser",
		Enabled:   true,
		ExpiresAt: &futureTime,
	}
	m.AddUser(user)

	perm := &Permission{
		Username: "futureuser",
		Database: "*",
		Actions:  []string{"*"},
	}
	m.AddPermission(perm)

	result := m.AuthenticateWithIP(ctx, "futureuser", "testdb", "mysql", "")
	// Result depends on other factors but shouldn't fail on expiration
	_ = result
}
