// Package router implements intelligent request routing across LLM providers
// with support for multiple strategies, provider health tracking, and failover.
package router

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
)

// providerStats tracks per-provider performance and health.
type providerStats struct {
	mu                  sync.Mutex
	totalRequests       int64
	successfulRequests  int64
	failedRequests      int64
	avgLatencyMs        float64
	lastSuccess         time.Time
	lastFailure         time.Time
	consecutiveFailures int
}

// Router selects a provider for each request based on the configured strategy.
type Router struct {
	registry        *providers.Registry
	defaultStrategy Strategy
	stats           sync.Map // map[string]*providerStats
	rrCounter       atomic.Uint64
}

// New creates a Router with the given registry and default strategy.
func New(registry *providers.Registry, strategy Strategy) *Router {
	return &Router{
		registry:        registry,
		defaultStrategy: strategy,
	}
}

// Route sends the request to the best available provider, with automatic
// failover to other providers on failure.
func (r *Router) Route(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return r.RouteWithStrategy(ctx, req, r.defaultStrategy)
}

// RouteWithStrategy sends the request using a specific strategy.
func (r *Router) RouteWithStrategy(ctx context.Context, req *providers.ChatRequest, strategy Strategy) (*providers.ChatResponse, error) {
	available := r.getAvailableProviders(req.Model)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available providers for model %q", req.Model)
	}

	primary := r.selectProvider(req.Model, available, strategy)
	return r.executeWithFallback(ctx, primary, available, req)
}

// RouteToProvider sends the request to a specific named provider.
func (r *Router) RouteToProvider(ctx context.Context, providerName string, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	p := r.registry.Get(providerName)
	if p == nil {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

	start := time.Now()
	resp, err := p.Chat(ctx, req)
	latency := time.Since(start)

	if err != nil {
		r.recordFailure(providerName)
		return nil, err
	}

	r.recordSuccess(providerName, latency)
	resp.Provider = providerName
	return resp, nil
}

// getAvailableProviders returns providers that can serve the given model
// and are not in a failure backoff state.
func (r *Router) getAvailableProviders(model string) []string {
	var available []string
	for _, p := range r.registry.ListAll() {
		name := p.Name()
		stats := r.getStats(name)
		stats.mu.Lock()

		// Skip providers with too many consecutive failures.
		if stats.consecutiveFailures >= 3 {
			// Allow retry after 5 minutes.
			if !stats.lastFailure.IsZero() && time.Since(stats.lastFailure) < 5*time.Minute {
				stats.mu.Unlock()
				continue
			}
		}
		stats.mu.Unlock()
		available = append(available, name)
	}

	if len(available) == 0 {
		// Fallback: return all providers if everything is in backoff.
		for _, p := range r.registry.ListAll() {
			available = append(available, p.Name())
		}
	}

	return available
}

// selectProvider picks a provider according to the given strategy.
func (r *Router) selectProvider(model string, available []string, strategy Strategy) string {
	if len(available) == 0 {
		return ""
	}
	if len(available) == 1 {
		return available[0]
	}

	switch strategy {
	case StrategyRoundRobin:
		idx := r.rrCounter.Add(1) - 1
		return available[idx%uint64(len(available))]

	case StrategyLatencyOptimized:
		return r.lowestLatency(available)

	case StrategyLoadBalanced:
		return r.leastLoaded(available)

	case StrategyFailover:
		return r.failoverOrder(available)

	case StrategyRandom:
		return available[rand.Intn(len(available))]

	case StrategyCostOptimized:
		// Cost-optimized: prefer local/free providers, then cheapest cloud.
		// Priority: ollama > llamacpp > waddleai > mistral > openai > anthropic > gemini
		priority := []string{"ollama", "llamacpp", "waddleai", "mistral", "openai", "anthropic", "gemini"}
		set := make(map[string]bool, len(available))
		for _, a := range available {
			set[a] = true
		}
		for _, p := range priority {
			if set[p] {
				return p
			}
		}
		return available[0]

	default:
		return available[0]
	}
}

func (r *Router) lowestLatency(available []string) string {
	best := available[0]
	bestLatency := float64(1<<62 - 1)

	for _, name := range available {
		stats := r.getStats(name)
		stats.mu.Lock()
		lat := stats.avgLatencyMs
		success := stats.successfulRequests
		stats.mu.Unlock()

		if success > 0 && lat < bestLatency {
			bestLatency = lat
			best = name
		}
	}
	return best
}

func (r *Router) leastLoaded(available []string) string {
	best := available[0]
	bestScore := int64(1<<62 - 1)

	for _, name := range available {
		stats := r.getStats(name)
		stats.mu.Lock()
		score := stats.totalRequests - stats.successfulRequests + int64(stats.consecutiveFailures*10)
		stats.mu.Unlock()

		if score < bestScore {
			bestScore = score
			best = name
		}
	}
	return best
}

func (r *Router) failoverOrder(available []string) string {
	priority := []string{"openai", "anthropic", "gemini", "mistral", "ollama", "llamacpp", "waddleai"}
	set := make(map[string]bool, len(available))
	for _, a := range available {
		set[a] = true
	}
	for _, p := range priority {
		if set[p] {
			return p
		}
	}
	return available[0]
}

// executeWithFallback tries the primary provider, then falls back to others.
func (r *Router) executeWithFallback(ctx context.Context, primary string, available []string, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	order := make([]string, 0, len(available))
	order = append(order, primary)
	for _, name := range available {
		if name != primary {
			order = append(order, name)
		}
	}

	var lastErr error
	for _, name := range order {
		p := r.registry.Get(name)
		if p == nil {
			continue
		}

		start := time.Now()
		resp, err := p.Chat(ctx, req)
		latency := time.Since(start)

		if err != nil {
			slog.Warn("provider failed, trying next", "provider", name, "model", req.Model, "error", err)
			r.recordFailure(name)
			lastErr = err
			continue
		}

		r.recordSuccess(name, latency)
		resp.Provider = name
		slog.Info("routed request successfully", "provider", name, "model", req.Model, "latency_ms", latency.Milliseconds())
		return resp, nil
	}

	return nil, fmt.Errorf("all providers failed for model %q: %w", req.Model, lastErr)
}

func (r *Router) getStats(name string) *providerStats {
	val, _ := r.stats.LoadOrStore(name, &providerStats{})
	return val.(*providerStats)
}

func (r *Router) recordSuccess(name string, latency time.Duration) {
	stats := r.getStats(name)
	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.totalRequests++
	stats.successfulRequests++
	stats.lastSuccess = time.Now()
	stats.consecutiveFailures = 0

	latMs := float64(latency.Milliseconds())
	if stats.avgLatencyMs == 0 {
		stats.avgLatencyMs = latMs
	} else {
		stats.avgLatencyMs = stats.avgLatencyMs*0.9 + latMs*0.1
	}
}

func (r *Router) recordFailure(name string) {
	stats := r.getStats(name)
	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.totalRequests++
	stats.failedRequests++
	stats.lastFailure = time.Now()
	stats.consecutiveFailures++
}

// Stats returns current routing statistics for all providers.
func (r *Router) Stats() map[string]map[string]any {
	result := make(map[string]map[string]any)
	r.stats.Range(func(key, value any) bool {
		name := key.(string)
		stats := value.(*providerStats)
		stats.mu.Lock()
		defer stats.mu.Unlock()

		result[name] = map[string]any{
			"total_requests":       stats.totalRequests,
			"successful_requests":  stats.successfulRequests,
			"failed_requests":      stats.failedRequests,
			"avg_latency_ms":       stats.avgLatencyMs,
			"consecutive_failures": stats.consecutiveFailures,
		}
		return true
	})
	return result
}
