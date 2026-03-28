package nlb

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func testLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// TestNewTokenBucket verifies construction of a TokenBucket.
func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(100, 10, "test-bucket", ProtocolHTTP, testLogger())
	if tb == nil {
		t.Fatal("expected non-nil TokenBucket")
	}
	if tb.GetCapacity() != 100 {
		t.Errorf("expected capacity 100, got %f", tb.GetCapacity())
	}
	if tb.GetRefillRate() != 10 {
		t.Errorf("expected refill rate 10, got %f", tb.GetRefillRate())
	}
}

// TestTokenBucketAllowInitially verifies Allow() returns true on a fresh bucket.
func TestTokenBucketAllowInitially(t *testing.T) {
	tb := NewTokenBucket(100, 10, "allow-test", ProtocolHTTP, testLogger())

	if !tb.Allow() {
		t.Error("expected Allow() to return true on fresh bucket")
	}
}

// TestTokenBucketGetAvailableTokens checks initial available tokens equals capacity.
func TestTokenBucketGetAvailableTokens(t *testing.T) {
	const capacity = 50.0
	tb := NewTokenBucket(capacity, 5, "tokens-test", ProtocolMySQL, testLogger())

	available := tb.GetAvailableTokens()
	if available > capacity || available < capacity-1 {
		t.Errorf("expected available tokens ~%f, got %f", capacity, available)
	}
}

// TestTokenBucketExhaustion verifies that Allow() returns false after exhausting tokens.
func TestTokenBucketExhaustion(t *testing.T) {
	// 5 tokens, refill rate 0 (so no refill happens)
	tb := NewTokenBucket(5, 0, "exhaust-test", ProtocolHTTP, testLogger())

	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("expected Allow() true on iteration %d", i)
		}
	}

	// Bucket should now be empty.
	if tb.Allow() {
		t.Error("expected Allow() false after exhausting all tokens")
	}
}

// TestTokenBucketAllowN verifies AllowN correctly consumes multiple tokens.
func TestTokenBucketAllowN(t *testing.T) {
	tb := NewTokenBucket(10, 0, "allown-test", ProtocolRedis, testLogger())

	if !tb.AllowN(5) {
		t.Error("expected AllowN(5) true with 10 tokens available")
	}
	if !tb.AllowN(5) {
		t.Error("expected AllowN(5) true with 5 tokens remaining")
	}
	if tb.AllowN(1) {
		t.Error("expected AllowN(1) false with 0 tokens remaining")
	}
}

// TestNewRateLimiter verifies RateLimiter construction.
func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	if rl == nil {
		t.Fatal("expected non-nil RateLimiter")
	}
}

// TestRateLimiterAddBucket verifies AddBucket creates a named bucket.
func TestRateLimiterAddBucket(t *testing.T) {
	rl := NewRateLimiter(testLogger())

	if err := rl.AddBucket("http-bucket", ProtocolHTTP, 1000, 100); err != nil {
		t.Fatalf("AddBucket() unexpected error: %v", err)
	}

	stats := rl.GetBucketStats("http-bucket")
	if stats == nil {
		t.Fatal("expected non-nil stats for 'http-bucket'")
	}
	if stats["capacity"] != 1000.0 {
		t.Errorf("expected capacity 1000, got %v", stats["capacity"])
	}
}

// TestRateLimiterAllowOnKnownBucket verifies Allow() on a known bucket.
func TestRateLimiterAllowOnKnownBucket(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	_ = rl.AddBucket("test-bucket", ProtocolHTTP, 100, 10)

	if !rl.Allow("test-bucket") {
		t.Error("expected Allow() true on fresh bucket")
	}
}

// TestRateLimiterAllowOnUnknownBucket verifies unknown bucket defaults to allow.
func TestRateLimiterAllowOnUnknownBucket(t *testing.T) {
	rl := NewRateLimiter(testLogger())

	if !rl.Allow("nonexistent-bucket") {
		t.Error("expected Allow() true for unknown bucket (default allow)")
	}
}

// TestRateLimiterRemoveBucket verifies RemoveBucket removes the bucket.
func TestRateLimiterRemoveBucket(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	_ = rl.AddBucket("temp-bucket", ProtocolHTTP, 100, 10)

	rl.RemoveBucket("temp-bucket")

	stats := rl.GetBucketStats("temp-bucket")
	if stats != nil {
		t.Error("expected nil stats after removing bucket")
	}
}

// TestRateLimiterGetAllStats verifies GetAllStats returns info for all buckets.
func TestRateLimiterGetAllStats(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	_ = rl.AddBucket("bucket-a", ProtocolHTTP, 100, 10)
	_ = rl.AddBucket("bucket-b", ProtocolMySQL, 200, 20)

	stats := rl.GetAllStats()
	if stats["total_buckets"] != 2 {
		t.Errorf("expected total_buckets 2, got %v", stats["total_buckets"])
	}
}

// TestRateLimiterAllowWithContextCancelled verifies cancelled context is rejected.
func TestRateLimiterAllowWithContextCancelled(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	_ = rl.AddBucket("ctx-bucket", ProtocolHTTP, 100, 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	if rl.AllowWithContext(ctx, "ctx-bucket") {
		t.Error("expected AllowWithContext to return false for cancelled context")
	}
}

// TestRateLimiterAllowWithContextActive verifies active context is allowed.
func TestRateLimiterAllowWithContextActive(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	_ = rl.AddBucket("ctx-bucket2", ProtocolHTTP, 100, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !rl.AllowWithContext(ctx, "ctx-bucket2") {
		t.Error("expected AllowWithContext to return true for active context")
	}
}

// TestRateLimiterBucketStatUtilization verifies utilization is 0 on fresh bucket.
func TestRateLimiterBucketStatUtilization(t *testing.T) {
	rl := NewRateLimiter(testLogger())
	_ = rl.AddBucket("util-bucket", ProtocolHTTP, 100, 0)

	stats := rl.GetBucketStats("util-bucket")
	utilization, ok := stats["utilization"].(float64)
	if !ok {
		t.Fatal("expected utilization to be float64")
	}
	if utilization < 0 || utilization > 0.01 {
		t.Errorf("expected utilization near 0 on fresh bucket, got %f", utilization)
	}
}
