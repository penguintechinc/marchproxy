package qos_test

import (
	"testing"
	"time"

	"marchproxy-l3l4/internal/qos"
)

func TestNewTokenBucket_InitialTokensEqualCapacity(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 5000)
	available := tb.Available()
	if available != 5000 {
		t.Errorf("Available() = %d, want 5000 (capacity)", available)
	}
}

func TestNewTokenBucket_RateGetter(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 5000)
	if tb.Rate() != 1000 {
		t.Errorf("Rate() = %d, want 1000", tb.Rate())
	}
}

func TestTryConsume_SucceedsWhenTokensAvailable(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 5000)
	if !tb.TryConsume(100) {
		t.Error("TryConsume(100) = false, want true when bucket is full")
	}
}

func TestTryConsume_ReducesAvailableTokens(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 5000)
	tb.TryConsume(200)
	if tb.Available() != 4800 {
		t.Errorf("Available() = %d after consuming 200, want 4800", tb.Available())
	}
}

func TestTryConsume_ReturnsFalseWhenInsufficient(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 100)
	// Drain the bucket completely.
	tb.TryConsume(100)
	if tb.TryConsume(1) {
		t.Error("TryConsume(1) = true on empty bucket, want false")
	}
}

func TestTryConsume_FullConsumptionSucceeds(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 500)
	if !tb.TryConsume(500) {
		t.Error("TryConsume(500) = false on full bucket with capacity 500, want true")
	}
	if tb.Available() != 0 {
		t.Errorf("Available() = %d after full consumption, want 0", tb.Available())
	}
}

func TestTryConsume_ExactCapacityRequestSucceeds(t *testing.T) {
	tb := qos.NewTokenBucket(10000, 1000)
	if !tb.TryConsume(1000) {
		t.Error("TryConsume(1000) on bucket with capacity 1000 should succeed")
	}
}

func TestTryConsume_MoreThanCapacityFails(t *testing.T) {
	tb := qos.NewTokenBucket(10000, 1000)
	// Even with full bucket, consuming more than capacity must fail.
	if tb.TryConsume(1001) {
		t.Error("TryConsume(1001) on bucket with capacity 1000 should fail")
	}
}

func TestTokensRefillOverTime(t *testing.T) {
	// Rate: 10,000 tokens/sec; capacity: 10,000; start full then drain.
	rate := int64(10_000)
	capacity := int64(10_000)
	tb := qos.NewTokenBucket(rate, capacity)

	// Drain completely.
	tb.TryConsume(capacity)
	if tb.Available() != 0 {
		t.Fatalf("bucket should be empty after draining, got %d", tb.Available())
	}

	// Wait long enough for at least 100 tokens to be refilled (100/10000 = 10ms, wait 50ms).
	time.Sleep(50 * time.Millisecond)

	after := tb.Available()
	if after == 0 {
		t.Error("tokens should have refilled after 50ms but Available() == 0")
	}
}

func TestTokensDoNotExceedCapacity(t *testing.T) {
	// Rate very high; bucket starts full. After waiting, should not exceed capacity.
	capacity := int64(1000)
	tb := qos.NewTokenBucket(1_000_000, capacity)
	time.Sleep(10 * time.Millisecond)
	available := tb.Available()
	if available > capacity {
		t.Errorf("Available() = %d, exceeds capacity %d", available, capacity)
	}
}

func TestSetRate_UpdatesRate(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 5000)
	tb.SetRate(9999)
	if tb.Rate() != 9999 {
		t.Errorf("Rate() = %d after SetRate(9999), want 9999", tb.Rate())
	}
}

func TestSetRate_DoesNotChangeExistingTokens(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 5000)
	// Consume some tokens, then change rate — token count should remain ~4900.
	tb.TryConsume(100)
	beforeRate := tb.Available()
	tb.SetRate(500)
	// Available should be the same (give or take a tiny refill during SetRate call).
	afterRate := tb.Available()
	if afterRate < beforeRate {
		t.Errorf("SetRate() reduced tokens from %d to %d, should not decrease", beforeRate, afterRate)
	}
}

func TestConsume_BlocksUntilTokensAvailable(t *testing.T) {
	// Rate: 100,000/sec; capacity: 100. Drain it, then Consume(1) should
	// complete in a bounded time once tokens refill.
	tb := qos.NewTokenBucket(100_000, 100)
	tb.TryConsume(100)

	done := make(chan struct{})
	go func() {
		tb.Consume(1)
		close(done)
	}()

	select {
	case <-done:
		// Passed — Consume() returned after tokens refilled.
	case <-time.After(500 * time.Millisecond):
		t.Error("Consume(1) did not return within 500ms")
	}
}

func TestNewTokenBucket_ZeroCapacity(t *testing.T) {
	tb := qos.NewTokenBucket(1000, 0)
	if tb.Available() != 0 {
		t.Errorf("Available() = %d for zero-capacity bucket, want 0", tb.Available())
	}
	if tb.TryConsume(1) {
		t.Error("TryConsume(1) should fail on zero-capacity bucket")
	}
}

func TestTokenBucket_ConcurrentTryConsume(t *testing.T) {
	// Verify no race conditions under concurrent access.
	tb := qos.NewTokenBucket(1_000_000, 1_000_000)
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tb.TryConsume(1)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
