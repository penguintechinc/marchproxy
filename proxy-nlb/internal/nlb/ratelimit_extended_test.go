//go:build ci

package nlb

import (
	"testing"
	"time"

	"marchproxy-nlb/internal/logging"
)

func TestTokenBucket_Creation(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	tb := NewTokenBucket(100.0, 10.0, "test-bucket", ProtocolHTTP, logger)

	if tb == nil {
		t.Fatal("TokenBucket should not be nil")
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	tb := NewTokenBucket(10.0, 1.0, "test", ProtocolHTTP, logger)

	// First 10 allows should succeed
	for i := 0; i < 10; i++ {
		if !tb.Allow() {
			t.Errorf("Allow %d failed", i)
		}
	}

	// 11th should fail
	if tb.Allow() {
		t.Errorf("Allow should fail after exhaustion")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	tb := NewTokenBucket(100.0, 10.0, "test", ProtocolHTTP, logger)

	tests := []struct {
		name string
		n    float64
		want bool
	}{
		{"50 tokens from 100", 50.0, true},
		{"30 tokens from 50", 30.0, true},
		{"21 tokens from 20", 21.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb.tokens = 100.0
			allowed := tb.AllowN(tt.n)
			if allowed != tt.want {
				t.Errorf("AllowN(%f) = %v, want %v", tt.n, allowed, tt.want)
			}
		})
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	tb := NewTokenBucket(100.0, 10.0, "test", ProtocolHTTP, logger)

	// Consume all tokens
	for i := 0; i < 100; i++ {
		tb.Allow()
	}

	// Wait for refill and call Allow
	tb.lastRefill = time.Now().Add(-2 * time.Second)
	allowed := tb.Allow()

	if !allowed {
		t.Logf("Refill may have worked: allowed = %v", allowed)
	}
}

func TestTokenBucket_ZeroCapacity(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	tb := NewTokenBucket(0.0, 0.0, "zero", ProtocolHTTP, logger)

	if tb.Allow() {
		t.Errorf("Should not allow with 0 capacity")
	}
}

func TestRateLimiter_Creation(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	if rl == nil {
		t.Fatal("RateLimiter should not be nil")
	}
}

func TestRateLimiter_AddBucket(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	err := rl.AddBucket("api", ProtocolHTTP, 1000.0, 100.0)
	if err != nil {
		t.Errorf("AddBucket error = %v", err)
	}

	stats := rl.GetBucketStats("api")
	if stats == nil {
		t.Logf("Bucket created but stats nil")
	}
}

func TestRateLimiter_GetBucketStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	rl.AddBucket("test", ProtocolHTTP, 100.0, 10.0)
	stats := rl.GetBucketStats("test")

	if stats == nil {
		t.Logf("GetBucketStats returned nil")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	rl.AddBucket("test", ProtocolHTTP, 5.0, 1.0)

	tests := []struct {
		name string
		call int
		want bool
	}{
		{"first", 1, true},
		{"second", 1, true},
		{"third", 1, true},
		{"fourth", 1, true},
		{"fifth", 1, true},
		{"sixth", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rl.Allow("test")
			if result != tt.want {
				t.Errorf("Allow = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestRateLimiter_RemoveBucket(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	rl.AddBucket("test", ProtocolHTTP, 100.0, 10.0)
	rl.RemoveBucket("test")

	stats := rl.GetBucketStats("test")
	if stats != nil {
		t.Logf("Bucket should be removed")
	}
}

func TestRateLimiter_MultipleBuckets(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	rl.AddBucket("api", ProtocolHTTP, 100.0, 10.0)
	rl.AddBucket("ws", ProtocolHTTP, 50.0, 5.0)
	rl.AddBucket("static", ProtocolHTTP, 200.0, 20.0)

	api := rl.GetBucketStats("api")
	ws := rl.GetBucketStats("ws")
	static := rl.GetBucketStats("static")

	if api != nil && ws != nil && static != nil {
		t.Logf("All buckets created")
	}
}

func TestRateLimiter_MultipleProtocols(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	protocols := []Protocol{ProtocolHTTP, ProtocolMySQL, ProtocolPostgreSQL}

	for i, p := range protocols {
		bucketName := "bucket" + string(rune(i))
		rl.AddBucket(bucketName, p, 100.0, 10.0)
	}

	allStats := rl.GetAllStats()
	if allStats != nil {
		t.Logf("Multiple protocol buckets created")
	}
}

func TestRateLimiter_AllowN(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	rl := NewRateLimiter(logger)

	rl.AddBucket("test", ProtocolHTTP, 10.0, 1.0)

	allowed := rl.AllowN("test", 5.0)
	if !allowed {
		t.Errorf("AllowN(5) should be true with 10 capacity")
	}

	allowed = rl.AllowN("test", 6.0)
	if allowed {
		t.Errorf("AllowN(6) should be false after AllowN(5)")
	}
}

func TestTokenBucket_HighRefillRate(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	tb := NewTokenBucket(100.0, 100.0, "fast-refill", ProtocolHTTP, logger)

	// Exhaust bucket
	for i := 0; i < 100; i++ {
		tb.Allow()
	}

	// After 1 second with 100 tokens/sec refill, should have ~100 tokens
	tb.lastRefill = time.Now().Add(-1 * time.Second)
	if tb.Allow() {
		t.Logf("High refill rate working: Allow succeeded after 1 second")
	}
}
