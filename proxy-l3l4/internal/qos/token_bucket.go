package qos

import (
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	mu sync.Mutex

	rate       int64     // Tokens per second
	capacity   int64     // Maximum tokens
	tokens     int64     // Current tokens
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rate, capacity int64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
	}
}

// TryConsume attempts to consume tokens
func (tb *TokenBucket) TryConsume(tokens int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}

	return false
}

// Consume blocks until tokens are available
func (tb *TokenBucket) Consume(tokens int64) {
	for {
		if tb.TryConsume(tokens) {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int64(float64(tb.rate) * elapsed.Seconds())

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
}

// SetRate updates the token generation rate
func (tb *TokenBucket) SetRate(rate int64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	tb.rate = rate
}

// Available returns the number of available tokens
func (tb *TokenBucket) Available() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// Rate returns the current rate
func (tb *TokenBucket) Rate() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.rate
}
