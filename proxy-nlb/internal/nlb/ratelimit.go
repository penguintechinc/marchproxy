package nlb

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	rateLimitAllowed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_ratelimit_allowed_total",
			Help: "Total number of requests allowed by rate limiter",
		},
		[]string{"protocol", "bucket"},
	)

	rateLimitDenied = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_ratelimit_denied_total",
			Help: "Total number of requests denied by rate limiter",
		},
		[]string{"protocol", "bucket"},
	)

	rateLimitTokens = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nlb_ratelimit_tokens_available",
			Help: "Number of tokens available in rate limit bucket",
		},
		[]string{"protocol", "bucket"},
	)
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity      float64       // Maximum tokens in bucket
	tokens        float64       // Current tokens available
	refillRate    float64       // Tokens added per second
	lastRefill    time.Time     // Last refill timestamp
	mu            sync.Mutex
	name          string        // Bucket identifier
	protocol      Protocol
	logger        *logrus.Logger
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(capacity float64, refillRate float64, name string, protocol Protocol, logger *logrus.Logger) *TokenBucket {
	tb := &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
		name:       name,
		protocol:   protocol,
		logger:     logger,
	}

	rateLimitTokens.WithLabelValues(protocol.String(), name).Set(capacity)

	return tb
}

// Allow checks if a request should be allowed (consumes 1 token)
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if a request consuming N tokens should be allowed
func (tb *TokenBucket) AllowN(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= n {
		tb.tokens -= n
		rateLimitTokens.WithLabelValues(tb.protocol.String(), tb.name).Set(tb.tokens)
		rateLimitAllowed.WithLabelValues(tb.protocol.String(), tb.name).Inc()
		return true
	}

	rateLimitDenied.WithLabelValues(tb.protocol.String(), tb.name).Inc()
	return false
}

// refill adds tokens based on elapsed time (must be called with lock held)
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	if elapsed <= 0 {
		return
	}

	tokensToAdd := elapsed * tb.refillRate
	tb.tokens = min64(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now
}

// GetAvailableTokens returns current available tokens
func (tb *TokenBucket) GetAvailableTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// GetCapacity returns bucket capacity
func (tb *TokenBucket) GetCapacity() float64 {
	return tb.capacity
}

// GetRefillRate returns refill rate
func (tb *TokenBucket) GetRefillRate() float64 {
	return tb.refillRate
}

// RateLimiter manages multiple token buckets for different protocols and services
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	logger  *logrus.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger *logrus.Logger) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		logger:  logger,
	}
}

// AddBucket adds a new token bucket for a specific protocol/service
func (rl *RateLimiter) AddBucket(name string, protocol Protocol, capacity float64, refillRate float64) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, exists := rl.buckets[name]; exists {
		rl.logger.WithField("bucket", name).Warn("Bucket already exists, updating parameters")
	}

	rl.buckets[name] = NewTokenBucket(capacity, refillRate, name, protocol, rl.logger)

	rl.logger.WithFields(logrus.Fields{
		"bucket":      name,
		"protocol":    protocol.String(),
		"capacity":    capacity,
		"refill_rate": refillRate,
	}).Info("Rate limit bucket created")

	return nil
}

// RemoveBucket removes a token bucket
func (rl *RateLimiter) RemoveBucket(name string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.buckets, name)
	rl.logger.WithField("bucket", name).Info("Rate limit bucket removed")
}

// Allow checks if a request should be allowed for a specific bucket
func (rl *RateLimiter) Allow(bucketName string) bool {
	return rl.AllowN(bucketName, 1)
}

// AllowN checks if a request consuming N tokens should be allowed
func (rl *RateLimiter) AllowN(bucketName string, n float64) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[bucketName]
	rl.mu.RUnlock()

	if !exists {
		// No bucket configured - allow by default
		return true
	}

	return bucket.AllowN(n)
}

// AllowWithContext checks rate limit with context support
func (rl *RateLimiter) AllowWithContext(ctx context.Context, bucketName string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return rl.Allow(bucketName)
	}
}

// GetBucketStats returns statistics for a specific bucket
func (rl *RateLimiter) GetBucketStats(name string) map[string]interface{} {
	rl.mu.RLock()
	bucket, exists := rl.buckets[name]
	rl.mu.RUnlock()

	if !exists {
		return nil
	}

	return map[string]interface{}{
		"name":              bucket.name,
		"protocol":          bucket.protocol.String(),
		"capacity":          bucket.GetCapacity(),
		"refill_rate":       bucket.GetRefillRate(),
		"available_tokens":  bucket.GetAvailableTokens(),
		"utilization":       (bucket.GetCapacity() - bucket.GetAvailableTokens()) / bucket.GetCapacity(),
	}
}

// GetAllStats returns statistics for all buckets
func (rl *RateLimiter) GetAllStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := make(map[string]interface{})
	bucketStats := make(map[string]interface{})

	for name, bucket := range rl.buckets {
		bucketStats[name] = map[string]interface{}{
			"protocol":         bucket.protocol.String(),
			"capacity":         bucket.GetCapacity(),
			"refill_rate":      bucket.GetRefillRate(),
			"available_tokens": bucket.GetAvailableTokens(),
			"utilization":      (bucket.GetCapacity() - bucket.GetAvailableTokens()) / bucket.GetCapacity(),
		}
	}

	stats["buckets"] = bucketStats
	stats["total_buckets"] = len(rl.buckets)

	return stats
}

// Helper function for float64 min
func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
