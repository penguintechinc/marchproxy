package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewMemoryStore(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:          1024 * 1024, // 1MB
		MaxKeys:          1000,
		EvictionPolicy:   EvictionLRU,
		CleanupInterval:  time.Minute,
		TTLCheckInterval: time.Second * 30,
	}

	store := NewMemoryStore(config)
	defer store.Close()

	if store == nil {
		t.Fatal("Expected store to be created, got nil")
	}

	if store.config.MaxSize != config.MaxSize {
		t.Errorf("Expected MaxSize %d, got %d", config.MaxSize, store.config.MaxSize)
	}

	if store.config.EvictionPolicy != EvictionLRU {
		t.Errorf("Expected EvictionPolicy LRU, got %s", store.config.EvictionPolicy)
	}
}

func TestMemoryStoreBasicOperations(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Test Set and Get
	key := "test_key"
	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  10,
	}
	ttl := time.Hour

	err := store.Set(ctx, key, entry, ttl)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if retrieved == nil {
		t.Error("Expected entry to be found")
	}

	if string(retrieved.Value) != string(entry.Value) {
		t.Errorf("Expected value %s, got %s", string(entry.Value), string(retrieved.Value))
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	key := "test_key"
	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  10,
	}

	// Set value
	err := store.Set(ctx, key, entry, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Verify it exists
	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected key to be found before deletion")
	}

	// Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Verify it's gone
	retrieved, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value after deletion: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected key to be not found after deletion")
	}
}

func TestMemoryStoreTTL(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	key := "test_key"
	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  10,
	}
	ttl := time.Millisecond * 100 // Very short TTL

	// Set value with short TTL
	err := store.Set(ctx, key, entry, ttl)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Should be found immediately
	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected key to be found immediately after setting")
	}

	// Wait for TTL to expire
	time.Sleep(ttl + time.Millisecond*50)

	// Should be expired
	retrieved, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value after TTL: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected key to be expired after TTL")
	}
}

func TestMemoryStoreExists(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	key := "test_key"
	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  10,
	}

	// Check non-existent key
	exists, err := store.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}

	// Set value
	err = store.Set(ctx, key, entry, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Check existing key
	exists, err = store.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}
}

func TestMemoryStoreClear(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte(fmt.Sprintf("value_%d", i)),
			Size:  10,
		}
		err := store.Set(ctx, key, entry, time.Hour)
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Verify they exist
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.KeyCount != 10 {
		t.Errorf("Expected 10 keys, got %d", stats.KeyCount)
	}

	// Clear all
	err = store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear store: %v", err)
	}

	// Verify they're gone
	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.KeyCount != 0 {
		t.Errorf("Expected 0 keys after clear, got %d", stats.KeyCount)
	}
}

func TestMemoryStoreStats(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Initial stats
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.KeyCount != 0 {
		t.Errorf("Expected 0 initial keys, got %d", stats.KeyCount)
	}

	// Set a value
	key := "test_key"
	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  10,
	}
	err = store.Set(ctx, key, entry, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Check stats after set
	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.KeyCount != 1 {
		t.Errorf("Expected 1 key after set, got %d", stats.KeyCount)
	}

	// Get the value (hit)
	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected key to be found")
	}

	// Get non-existent value (miss)
	retrieved, err = store.Get(ctx, "non_existent")
	if err != nil {
		t.Fatalf("Failed to get non-existent value: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected key to not be found")
	}
}

func TestMemoryStoreKeys(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Set multiple values
	expectedKeys := make(map[string]bool)
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte(fmt.Sprintf("value_%d", i)),
			Size:  10,
		}
		expectedKeys[key] = true
		err := store.Set(ctx, key, entry, time.Hour)
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Get all keys using ".*" pattern (regex)
	keys, err := store.Keys(ctx, ".*")
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	// Verify all expected keys are present
	for _, key := range keys {
		if !expectedKeys[key] {
			t.Errorf("Unexpected key found: %s", key)
		}
		delete(expectedKeys, key)
	}

	if len(expectedKeys) > 0 {
		t.Errorf("Missing keys: %v", expectedKeys)
	}
}

func TestMemoryStoreEviction(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        100, // Very small size to trigger eviction
		MaxKeys:        3,   // Very small key limit
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Fill cache beyond capacity
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte(fmt.Sprintf("large_value_to_trigger_eviction_%d", i)),
			Size:  50,
		}
		err := store.Set(ctx, key, entry, time.Hour)
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Check that some keys were evicted
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.KeyCount > int64(config.MaxKeys) {
		t.Errorf("Expected keys to be limited to %d, got %d", config.MaxKeys, stats.KeyCount)
	}
}

func TestMemoryStorePatternMatching(t *testing.T) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Set values with different patterns
	testData := map[string]string{
		"user:123":    "user_data_123",
		"user:456":    "user_data_456",
		"session:abc": "session_data_abc",
		"session:def": "session_data_def",
		"config:main": "config_data",
	}

	for key, value := range testData {
		entry := &CacheEntry{
			Value: []byte(value),
			Size:  int64(len(value)),
		}
		err := store.Set(ctx, key, entry, time.Hour)
		if err != nil {
			t.Fatalf("Failed to set value for key %s: %v", key, err)
		}
	}

	// Test pattern matching (using regex)
	userKeys, err := store.Keys(ctx, "^user:.*")
	if err != nil {
		t.Fatalf("Failed to get user keys: %v", err)
	}

	if len(userKeys) != 2 {
		t.Errorf("Expected 2 user keys, got %d", len(userKeys))
	}

	sessionKeys, err := store.Keys(ctx, "^session:.*")
	if err != nil {
		t.Fatalf("Failed to get session keys: %v", err)
	}

	if len(sessionKeys) != 2 {
		t.Errorf("Expected 2 session keys, got %d", len(sessionKeys))
	}

	configKeys, err := store.Keys(ctx, "^config:.*")
	if err != nil {
		t.Fatalf("Failed to get config keys: %v", err)
	}

	if len(configKeys) != 1 {
		t.Errorf("Expected 1 config key, got %d", len(configKeys))
	}
}

func BenchmarkMemoryStoreSet(b *testing.B) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024 * 10, // 10MB
		MaxKeys:        10000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("benchmark_value"),
			Size:  15,
		}
		store.Set(ctx, key, entry, time.Hour)
	}
}

func BenchmarkMemoryStoreGet(b *testing.B) {
	config := MemoryStoreConfig{
		MaxSize:        1024 * 1024 * 10, // 10MB
		MaxKeys:        10000,
		EvictionPolicy: EvictionLRU,
	}

	store := NewMemoryStore(config)
	defer store.Close()
	ctx := context.Background()

	// Pre-populate with test data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("benchmark_value"),
			Size:  15,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		store.Get(ctx, key)
	}
}
