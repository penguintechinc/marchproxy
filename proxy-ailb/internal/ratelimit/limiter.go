// Package ratelimit provides token and request rate limiting using
// a sliding window algorithm.
package ratelimit

import (
	"log/slog"
	"sync"
	"time"
)

// Config holds rate limit configuration for a key.
type Config struct {
	TPMLimit      int           // Tokens per minute limit (0 = unlimited).
	RPMLimit      int           // Requests per minute limit (0 = unlimited).
	WindowSeconds int           // Sliding window duration in seconds.
	Enabled       bool
}

// DefaultConfig returns a sensible default rate limit configuration.
func DefaultConfig() Config {
	return Config{
		TPMLimit:      10000,
		RPMLimit:      60,
		WindowSeconds: 60,
		Enabled:       true,
	}
}

// Status reports current rate limit state.
type Status struct {
	CurrentTPM   int
	CurrentRPM   int
	IsLimited    bool
	LimitReason  string
	RemainingTPM int
	RemainingRPM int
}

type requestRecord struct {
	timestamp time.Time
	tokens    int
}

type windowData struct {
	requests    []requestRecord
	totalTokens int
	totalReqs   int
}

// Limiter implements sliding window rate limiting per API key.
type Limiter struct {
	mu       sync.Mutex
	windows  map[string]*windowData
	configs  map[string]Config
	defaults Config
}

// NewLimiter creates a rate limiter with default configuration.
func NewLimiter(defaults Config) *Limiter {
	return &Limiter{
		windows:  make(map[string]*windowData),
		configs:  make(map[string]Config),
		defaults: defaults,
	}
}

// SetConfig sets a per-key rate limit configuration.
func (l *Limiter) SetConfig(keyID string, cfg Config) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.configs[keyID] = cfg
}

// GetConfig returns the rate limit configuration for a key.
func (l *Limiter) GetConfig(keyID string) Config {
	l.mu.Lock()
	defer l.mu.Unlock()
	if cfg, ok := l.configs[keyID]; ok {
		return cfg
	}
	return l.defaults
}

// CheckLimit verifies whether a request with the given token count is allowed.
func (l *Limiter) CheckLimit(keyID string, tokens int) (bool, Status) {
	cfg := l.GetConfig(keyID)
	if !cfg.Enabled {
		return true, Status{}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	w := l.getOrCreateWindow(keyID)
	l.cleanWindow(w, cfg.WindowSeconds)

	currentTPM := w.totalTokens
	currentRPM := w.totalReqs

	tpmExceeded := cfg.TPMLimit > 0 && (currentTPM+tokens) > cfg.TPMLimit
	rpmExceeded := cfg.RPMLimit > 0 && (currentRPM+1) > cfg.RPMLimit

	allowed := !tpmExceeded && !rpmExceeded

	reason := ""
	if tpmExceeded {
		reason = "tpm_exceeded"
	} else if rpmExceeded {
		reason = "rpm_exceeded"
	}

	remainTPM := 0
	if cfg.TPMLimit > 0 {
		remainTPM = cfg.TPMLimit - currentTPM
		if remainTPM < 0 {
			remainTPM = 0
		}
	}
	remainRPM := 0
	if cfg.RPMLimit > 0 {
		remainRPM = cfg.RPMLimit - currentRPM
		if remainRPM < 0 {
			remainRPM = 0
		}
	}

	if !allowed {
		slog.Warn("rate limit exceeded",
			"key_id", keyID[:min(8, len(keyID))],
			"reason", reason,
			"tpm", currentTPM, "tpm_limit", cfg.TPMLimit,
			"rpm", currentRPM, "rpm_limit", cfg.RPMLimit,
		)
	}

	return allowed, Status{
		CurrentTPM:   currentTPM,
		CurrentRPM:   currentRPM,
		IsLimited:    !allowed,
		LimitReason:  reason,
		RemainingTPM: remainTPM,
		RemainingRPM: remainRPM,
	}
}

// RecordRequest records a successful request.
func (l *Limiter) RecordRequest(keyID string, tokens int) {
	cfg := l.GetConfig(keyID)
	if !cfg.Enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	w := l.getOrCreateWindow(keyID)
	w.requests = append(w.requests, requestRecord{
		timestamp: time.Now(),
		tokens:    tokens,
	})
	w.totalTokens += tokens
	w.totalReqs++
}

// Reset clears rate limit tracking for a key.
func (l *Limiter) Reset(keyID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, keyID)
}

func (l *Limiter) getOrCreateWindow(keyID string) *windowData {
	w, ok := l.windows[keyID]
	if !ok {
		w = &windowData{}
		l.windows[keyID] = w
	}
	return w
}

func (l *Limiter) cleanWindow(w *windowData, windowSeconds int) {
	cutoff := time.Now().Add(-time.Duration(windowSeconds) * time.Second)
	i := 0
	for i < len(w.requests) && w.requests[i].timestamp.Before(cutoff) {
		w.totalTokens -= w.requests[i].tokens
		w.totalReqs--
		i++
	}
	if i > 0 {
		w.requests = w.requests[i:]
	}
}

// CleanupExpired removes windows that have been idle for the given duration.
func (l *Limiter) CleanupExpired(maxIdle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-maxIdle)
	for keyID, w := range l.windows {
		if len(w.requests) == 0 || w.requests[len(w.requests)-1].timestamp.Before(cutoff) {
			delete(l.windows, keyID)
		}
	}
}
