//go:build ci

package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMemoryStoreTouch verifies entry touch updates access count and last accessed time
func TestMemoryStoreTouch(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	key := "test_key"
	entry := &CacheEntry{
		Value:        []byte("test_value"),
		Size:         10,
		AccessCount:  0,
		LastAccessed: time.Now().Add(-time.Hour),
	}

	err := store.Set(ctx, key, entry, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	retrieved, _ := store.Get(ctx, key)
	if retrieved.AccessCount != 1 {
		t.Errorf("Expected AccessCount 1, got %d", retrieved.AccessCount)
	}

	retrieved, _ = store.Get(ctx, key)
	if retrieved.AccessCount != 2 {
		t.Errorf("Expected AccessCount 2, got %d", retrieved.AccessCount)
	}
}

// TestCacheEntryExpiry checks IsExpired function
func TestCacheEntryExpiry(t *testing.T) {
	futureTime := time.Now().Add(time.Hour)
	entry := &CacheEntry{
		ExpiresAt: futureTime,
	}

	if entry.IsExpired() {
		t.Error("Entry should not be expired in the future")
	}

	pastEntry := &CacheEntry{
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	if !pastEntry.IsExpired() {
		t.Error("Entry should be expired in the past")
	}
}

// TestCacheEntryStale checks IsStale function
func TestCacheEntryStale(t *testing.T) {
	entry := &CacheEntry{
		LastAccessed: time.Now().Add(-time.Hour),
	}

	staleTime := 30 * time.Minute
	if !entry.IsStale(staleTime) {
		t.Error("Entry should be stale after 1 hour with 30min threshold")
	}

	entry.LastAccessed = time.Now().Add(-10 * time.Minute)
	if entry.IsStale(staleTime) {
		t.Error("Entry should not be stale after 10min with 30min threshold")
	}
}

// TestMemoryStoreEvictionLRU tests LRU eviction removes least recently used
func TestMemoryStoreEvictionLRU(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        150, // Small to force eviction
		MaxKeys:        5,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	// Insert 5 entries to hit max keys
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("val"),
			Size:  30,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	// Add one more to trigger eviction
	entry := &CacheEntry{
		Value: []byte("val"),
		Size:  30,
	}
	store.Set(ctx, "key_5", entry, time.Hour)

	// Check that eviction happened
	stats, _ := store.Stats(ctx)
	if stats.KeyCount > int64(store.config.MaxKeys) {
		t.Errorf("Expected keys to be limited to %d, got %d", store.config.MaxKeys, stats.KeyCount)
	}
}

// TestMemoryStoreEvictionLFU tests LFU eviction removes least frequently used
func TestMemoryStoreEvictionLFU(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        150,
		MaxKeys:        5,
		EvictionPolicy: EvictionLFU,
	})
	defer store.Close()
	ctx := context.Background()

	// Insert entries
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("val"),
			Size:  50,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	// Access key_0 multiple times to increase frequency
	for i := 0; i < 3; i++ {
		store.Get(ctx, "key_0")
	}

	// Add more entries to trigger eviction
	for i := 3; i < 6; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("val"),
			Size:  50,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	// key_0 should still exist (was accessed frequently)
	exists, _ := store.Exists(ctx, "key_0")
	if !exists {
		t.Error("key_0 should still exist (was frequently accessed)")
	}
}

// TestMemoryStoreEvictionFIFO tests FIFO eviction removes oldest entry
func TestMemoryStoreEvictionFIFO(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        150,
		MaxKeys:        5,
		EvictionPolicy: EvictionFIFO,
	})
	defer store.Close()
	ctx := context.Background()

	// Insert entries in order
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("val"),
			Size:  50,
		}
		store.Set(ctx, key, entry, time.Hour)
		time.Sleep(10 * time.Millisecond)
	}

	// Add more to trigger eviction
	for i := 3; i < 6; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("val"),
			Size:  50,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	// key_0 should be evicted (was first inserted)
	exists, _ := store.Exists(ctx, "key_0")
	if exists {
		t.Error("key_0 should be evicted (was first inserted with FIFO)")
	}
}

// TestMemoryStoreEvictionRandom tests random eviction
func TestMemoryStoreEvictionRandom(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        150,
		MaxKeys:        5,
		EvictionPolicy: EvictionRandom,
	})
	defer store.Close()
	ctx := context.Background()

	// Insert entries
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("val"),
			Size:  50,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	// Add one more to trigger eviction
	entry := &CacheEntry{
		Value: []byte("val"),
		Size:  50,
	}
	store.Set(ctx, "key_5", entry, time.Hour)

	// Some key should be evicted
	stats, _ := store.Stats(ctx)
	if stats.KeyCount > int64(store.config.MaxKeys) {
		t.Errorf("Expected keys <= %d, got %d", store.config.MaxKeys, stats.KeyCount)
	}
}

// TestMemoryStoreSize tests size tracking
func TestMemoryStoreSize(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  11,
	}

	err := store.Set(ctx, "key1", entry, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	size, _ := store.Size(ctx)
	if size != 11 {
		t.Errorf("Expected size 11, got %d", size)
	}

	store.Set(ctx, "key2", entry, time.Hour)
	size, _ = store.Size(ctx)
	if size != 22 {
		t.Errorf("Expected size 22, got %d", size)
	}

	store.Delete(ctx, "key1")
	size, _ = store.Size(ctx)
	if size != 11 {
		t.Errorf("Expected size 11 after delete, got %d", size)
	}
}

// TestMemoryStoreConcurrency tests concurrent access
func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        10 * 1024 * 1024,
		MaxKeys:        10000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10
	opsPerGoroutine := 100

	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := fmt.Sprintf("key_%d_%d", goroutineID, i)
				entry := &CacheEntry{
					Value: []byte("concurrent_value"),
					Size:  16,
				}
				_ = store.Set(ctx, key, entry, time.Hour)
				_, _ = store.Get(ctx, key)
			}
		}(g)
	}

	wg.Wait()

	stats, _ := store.Stats(ctx)
	expectedKeys := int64(numGoroutines * opsPerGoroutine)
	if stats.KeyCount != expectedKeys {
		t.Errorf("Expected %d keys, got %d", expectedKeys, stats.KeyCount)
	}
}

// TestMemoryStoreBackgroundCleaner tests background cleanup
func TestMemoryStoreBackgroundCleaner(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:          1024 * 1024,
		MaxKeys:          1000,
		EvictionPolicy:   EvictionLRU,
		CleanupInterval:  100 * time.Millisecond,
		TTLCheckInterval: 100 * time.Millisecond,
	})
	defer store.Close()
	ctx := context.Background()

	// Add entry with short TTL
	entry := &CacheEntry{
		Value: []byte("short_ttl"),
		Size:  9,
	}
	store.Set(ctx, "short_key", entry, 100*time.Millisecond)

	// Should exist initially
	exists, _ := store.Exists(ctx, "short_key")
	if !exists {
		t.Error("Key should exist initially")
	}

	// Wait for cleanup
	time.Sleep(300 * time.Millisecond)

	// Should be cleaned up
	exists, _ = store.Exists(ctx, "short_key")
	if exists {
		t.Error("Key should be cleaned up after TTL")
	}
}

// TestMemoryStoreReplaceEntry tests overwriting an entry
func TestMemoryStoreReplaceEntry(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	key := "test_key"

	// Set first value
	entry1 := &CacheEntry{
		Value: []byte("value1"),
		Size:  6,
	}
	store.Set(ctx, key, entry1, time.Hour)

	// Replace with second value
	entry2 := &CacheEntry{
		Value: []byte("value2_with_more_data"),
		Size:  21,
	}
	store.Set(ctx, key, entry2, time.Hour)

	// Get should return the new value
	retrieved, _ := store.Get(ctx, key)
	if string(retrieved.Value) != "value2_with_more_data" {
		t.Errorf("Expected value2_with_more_data, got %s", string(retrieved.Value))
	}

	// Size should reflect new value
	size, _ := store.Size(ctx)
	if size != 21 {
		t.Errorf("Expected size 21, got %d", size)
	}
}

// TestMemoryStoreKeyPatternInvalid tests invalid regex pattern
func TestMemoryStoreKeyPatternInvalid(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	_, err := store.Keys(ctx, "[invalid(regex")
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

// TestMemoryStoreEmptyPattern tests empty result for pattern
func TestMemoryStoreEmptyPattern(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	// Add keys that won't match pattern
	entry := &CacheEntry{
		Value: []byte("test"),
		Size:  4,
	}
	store.Set(ctx, "user:123", entry, time.Hour)

	// Search for non-matching pattern
	keys, _ := store.Keys(ctx, "^admin:.*")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for non-matching pattern, got %d", len(keys))
	}
}

// TestMemoryStoreExistsExpired tests Exists with expired entry
func TestMemoryStoreExistsExpired(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024 * 1024,
		MaxKeys:        1000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	key := "test_key"
	entry := &CacheEntry{
		Value: []byte("test_value"),
		Size:  10,
	}

	// Set with short TTL
	store.Set(ctx, key, entry, 50*time.Millisecond)

	// Should exist initially
	exists, _ := store.Exists(ctx, key)
	if !exists {
		t.Error("Key should exist initially")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should not exist after expiry
	exists, _ = store.Exists(ctx, key)
	if exists {
		t.Error("Expired key should not exist")
	}
}

// TestMemoryStoreMaxSizeEnforcement tests that max size is enforced
func TestMemoryStoreMaxSizeEnforcement(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        100, // Very small
		MaxKeys:        100,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	// Try to add entry larger than max size
	entry := &CacheEntry{
		Value: []byte("test"),
		Size:  200,
	}
	store.Set(ctx, "key1", entry, time.Hour)

	stats, _ := store.Stats(ctx)
	if stats.Size > 100 {
		t.Errorf("Size %d exceeds max of 100", stats.Size)
	}
}

// TestMemoryStoreDefaultConfig tests default configuration values
func TestMemoryStoreDefaultConfig(t *testing.T) {
	config := MemoryStoreConfig{}
	store := NewMemoryStore(config)
	defer store.Close()

	if store.config.MaxSize != 50*1024*1024 {
		t.Errorf("Expected default MaxSize 50MB, got %d", store.config.MaxSize)
	}
	if store.config.MaxKeys != 10000 {
		t.Errorf("Expected default MaxKeys 10000, got %d", store.config.MaxKeys)
	}
	if store.config.EvictionPolicy != EvictionLRU {
		t.Errorf("Expected default EvictionPolicy LRU, got %s", store.config.EvictionPolicy)
	}
}

// TestLRUListOperations tests LRU list node operations
func TestLRUListOperations(t *testing.T) {
	lru := NewLRUList()

	// Touch keys in order
	lru.Touch("key1")
	lru.Touch("key2")
	lru.Touch("key3")

	// Get LRU (should be key1)
	lruKey := lru.GetLRU()
	if lruKey != "key1" {
		t.Errorf("Expected LRU key1, got %s", lruKey)
	}

	// Touch key1 again (move to head)
	lru.Touch("key1")

	// Get LRU (should now be key2)
	lruKey = lru.GetLRU()
	if lruKey != "key2" {
		t.Errorf("Expected LRU key2, got %s", lruKey)
	}

	// Remove key2
	lru.Remove("key2")

	// Get LRU (should now be key3)
	lruKey = lru.GetLRU()
	if lruKey != "key3" {
		t.Errorf("Expected LRU key3, got %s", lruKey)
	}
}

// TestLFUTrackerOperations tests LFU tracker operations
func TestLFUTrackerOperations(t *testing.T) {
	lfu := NewLFUTracker()

	// Touch keys different number of times
	lfu.Touch("key1")
	for i := 0; i < 3; i++ {
		lfu.Touch("key2")
	}
	for i := 0; i < 2; i++ {
		lfu.Touch("key3")
	}

	// Get LFU (should be key1 with freq 1)
	lfuKey := lfu.GetLFU()
	if lfuKey != "key1" {
		t.Errorf("Expected LFU key1, got %s", lfuKey)
	}

	// Remove key1
	lfu.Remove("key1")

	// Get LFU (should be key3 with freq 2)
	lfuKey = lfu.GetLFU()
	if lfuKey != "key3" {
		t.Errorf("Expected LFU key3, got %s", lfuKey)
	}
}

// TestFIFOQueueOperations tests FIFO queue operations
func TestFIFOQueueOperations(t *testing.T) {
	fifo := NewFIFOQueue()

	// Add keys in order
	fifo.Touch("key1")
	fifo.Touch("key2")
	fifo.Touch("key3")

	// Get next (should be key1)
	nextKey := fifo.GetNext()
	if nextKey != "key1" {
		t.Errorf("Expected next key1, got %s", nextKey)
	}

	// Remove key1
	fifo.Remove("key1")

	// Get next (should now be key2)
	nextKey = fifo.GetNext()
	if nextKey != "key2" {
		t.Errorf("Expected next key2, got %s", nextKey)
	}

	// Touch key3 again (should not add duplicate)
	fifo.Touch("key3")
	nextKey = fifo.GetNext()
	if nextKey != "key2" {
		t.Errorf("Expected next key2 still, got %s", nextKey)
	}
}

// TestMetricsRecording tests metric recording functions
func TestMetricsRecording(t *testing.T) {
	metrics := &Metrics{}

	metrics.recordHit()
	if metrics.Hits != 1 {
		t.Errorf("Expected Hits=1, got %d", metrics.Hits)
	}

	metrics.recordMiss()
	if metrics.Misses != 1 {
		t.Errorf("Expected Misses=1, got %d", metrics.Misses)
	}

	metrics.recordSet()
	if metrics.Sets != 1 {
		t.Errorf("Expected Sets=1, got %d", metrics.Sets)
	}

	metrics.recordDelete()
	if metrics.Deletes != 1 {
		t.Errorf("Expected Deletes=1, got %d", metrics.Deletes)
	}

	metrics.recordError()
	if metrics.Errors != 1 {
		t.Errorf("Expected Errors=1, got %d", metrics.Errors)
	}

	metrics.recordEviction()
	if metrics.Evictions != 1 {
		t.Errorf("Expected Evictions=1, got %d", metrics.Evictions)
	}
}

// TestMetricsHitRate tests hit rate calculation
func TestMetricsHitRate(t *testing.T) {
	metrics := &Metrics{}

	// No hits/misses
	hitRate := metrics.GetHitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected hit rate 0 for empty metrics, got %f", hitRate)
	}

	// 50% hit rate
	for i := 0; i < 5; i++ {
		metrics.recordHit()
		metrics.recordMiss()
	}

	hitRate = metrics.GetHitRate()
	if hitRate != 50.0 {
		t.Errorf("Expected hit rate 50, got %f", hitRate)
	}

	// 100% hit rate
	metrics = &Metrics{}
	for i := 0; i < 10; i++ {
		metrics.recordHit()
	}

	hitRate = metrics.GetHitRate()
	if hitRate != 100.0 {
		t.Errorf("Expected hit rate 100, got %f", hitRate)
	}
}

// TestMemoryStoreConcurrentEviction tests eviction under concurrent load
func TestMemoryStoreConcurrentEviction(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        500, // Small to trigger eviction
		MaxKeys:        20,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	var evictions int32

	numGoroutines := 5
	opsPerGoroutine := 50

	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := fmt.Sprintf("key_%d_%d", goroutineID, i)
				entry := &CacheEntry{
					Value: []byte("value"),
					Size:  25,
				}
				_ = store.Set(ctx, key, entry, time.Hour)
			}
		}(g)
	}

	wg.Wait()

	stats, _ := store.Stats(ctx)
	// Should have evicted due to size constraints
	if stats.LastEviction.IsZero() && stats.KeyCount > int64(store.config.MaxKeys) {
		t.Errorf("Expected evictions to occur, LastEviction: %v, KeyCount: %d", stats.LastEviction, stats.KeyCount)
	}

	_ = evictions
}

// TestMemoryStoreGetAfterEviction tests Get works correctly after eviction
func TestMemoryStoreGetAfterEviction(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        100,
		MaxKeys:        3,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	// Fill beyond capacity
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("v"),
			Size:  50,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	// Get should work for remaining keys
	_, err := store.Get(ctx, "key_4")
	if err != nil {
		t.Fatalf("Failed to get after eviction: %v", err)
	}
}

// TestMemoryStoreDeleteNonExistent tests deleting non-existent key
func TestMemoryStoreDeleteNonExistent(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024,
		MaxKeys:        100,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	// Should not error when deleting non-existent key
	err := store.Delete(ctx, "non_existent")
	if err != nil {
		t.Fatalf("Delete of non-existent key should not error: %v", err)
	}
}

// TestMemoryStoreMultipleClear tests multiple clear operations
func TestMemoryStoreMultipleClear(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        1024,
		MaxKeys:        100,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		// Add data
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("key_%d_%d", i, j)
			entry := &CacheEntry{
				Value: []byte("data"),
				Size:  4,
			}
			store.Set(ctx, key, entry, time.Hour)
		}

		stats, _ := store.Stats(ctx)
		if stats.KeyCount == 0 {
			t.Fatalf("Keys should exist before clear iteration %d", i)
		}

		// Clear
		store.Clear(ctx)

		// Verify cleared
		stats, _ = store.Stats(ctx)
		if stats.KeyCount != 0 {
			t.Errorf("Expected 0 keys after clear iteration %d, got %d", i, stats.KeyCount)
		}
	}
}

// TestBackgroundCleanerStop tests stopping background cleaner
func TestBackgroundCleanerStop(t *testing.T) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:          1024,
		MaxKeys:          100,
		EvictionPolicy:   EvictionLRU,
		CleanupInterval:  100 * time.Millisecond,
		TTLCheckInterval: 100 * time.Millisecond,
	})

	// Add data
	entry := &CacheEntry{
		Value: []byte("data"),
		Size:  4,
	}
	store.Set(context.Background(), "key1", entry, time.Hour)

	// Close should stop background goroutines cleanly
	err := store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify cleaner is stopped
	if store.background.running {
		t.Error("Background cleaner should be stopped after Close")
	}
}

// BenchmarkMemoryStoreSetConcurrent benchmarks concurrent Set operations
func BenchmarkMemoryStoreSetConcurrent(b *testing.B) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        100 * 1024 * 1024,
		MaxKeys:        100000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i)
			entry := &CacheEntry{
				Value: []byte("benchmark"),
				Size:  9,
			}
			store.Set(ctx, key, entry, time.Hour)
			i++
		}
	})
}

// BenchmarkMemoryStoreGetConcurrent benchmarks concurrent Get operations
func BenchmarkMemoryStoreGetConcurrent(b *testing.B) {
	store := NewMemoryStore(MemoryStoreConfig{
		MaxSize:        100 * 1024 * 1024,
		MaxKeys:        100000,
		EvictionPolicy: EvictionLRU,
	})
	defer store.Close()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i)
		entry := &CacheEntry{
			Value: []byte("data"),
			Size:  4,
		}
		store.Set(ctx, key, entry, time.Hour)
	}

	b.ResetTimer()
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		i := atomic.AddInt64(&counter, 1)
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%10000)
			store.Get(ctx, key)
			i++
		}
	})
}
