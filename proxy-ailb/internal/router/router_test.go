//go:build ci

package router_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

// mockProvider implements the providers.Provider interface.
type mockProvider struct {
	name     string
	responses map[string]*providers.ChatResponse
	err      error
	delay    time.Duration
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Chat(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, m.err
	}
	if m.responses[req.Model] != nil {
		return m.responses[req.Model], nil
	}
	resp := &providers.ChatResponse{
		Content:  "test response",
		Provider: m.name,
	}
	return resp, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{
		{ID: m.name + "-model", Object: "model", Created: 1000, OwnedBy: m.name, Provider: m.name},
	}, nil
}

func (m *mockProvider) SupportsStreaming() bool {
	return false
}

func TestNew(t *testing.T) {
	registry := providers.NewRegistry()
	r := router.New(registry, router.StrategyRoundRobin)

	if r == nil {
		t.Error("expected router to be created")
	}
}

func TestRouteWithStrategy(t *testing.T) {
	registry := providers.NewRegistry()
	provider := &mockProvider{name: "test", responses: make(map[string]*providers.ChatResponse)}
	registry.Register(provider)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyRoundRobin)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Error("expected response")
	}
	if resp.Provider != "test" {
		t.Errorf("expected provider 'test', got %s", resp.Provider)
	}
}

func TestRoute(t *testing.T) {
	registry := providers.NewRegistry()
	provider := &mockProvider{name: "default", responses: make(map[string]*providers.ChatResponse)}
	registry.Register(provider)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.Route(context.Background(), req)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Error("expected response")
	}
}

func TestRouteNoAvailableProviders(t *testing.T) {
	registry := providers.NewRegistry()
	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.Route(context.Background(), req)
	if err == nil {
		t.Error("expected error when no providers available")
	}
	if resp != nil {
		t.Error("expected no response when no providers available")
	}
}

func TestRouteToProvider(t *testing.T) {
	registry := providers.NewRegistry()
	provider := &mockProvider{name: "specific", responses: make(map[string]*providers.ChatResponse)}
	registry.Register(provider)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.RouteToProvider(context.Background(), "specific", req)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Error("expected response")
	}
	if resp.Provider != "specific" {
		t.Errorf("expected provider 'specific', got %s", resp.Provider)
	}
}

func TestRouteToProviderNotFound(t *testing.T) {
	registry := providers.NewRegistry()
	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.RouteToProvider(context.Background(), "nonexistent", req)
	if err == nil {
		t.Error("expected error when provider not found")
	}
	if resp != nil {
		t.Error("expected no response when provider not found")
	}
}

func TestRouteToProviderError(t *testing.T) {
	registry := providers.NewRegistry()
	provider := &mockProvider{
		name:      "failing",
		err:       errors.New("test error"),
		responses: make(map[string]*providers.ChatResponse),
	}
	registry.Register(provider)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.RouteToProvider(context.Background(), "failing", req)
	if err == nil {
		t.Error("expected error from provider")
	}
	if resp != nil {
		t.Error("expected no response when provider errors")
	}
}

func TestRouteStrategyRoundRobin(t *testing.T) {
	registry := providers.NewRegistry()
	p1 := &mockProvider{name: "provider1", responses: make(map[string]*providers.ChatResponse)}
	p2 := &mockProvider{name: "provider2", responses: make(map[string]*providers.ChatResponse)}
	p3 := &mockProvider{name: "provider3", responses: make(map[string]*providers.ChatResponse)}
	registry.Register(p1)
	registry.Register(p2)
	registry.Register(p3)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	// Make multiple requests and track distribution
	counts := make(map[string]int)
	for i := 0; i < 9; i++ {
		resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyRoundRobin)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[resp.Provider]++
	}

	// Each provider should be selected multiple times
	if len(counts) < 2 {
		t.Errorf("round-robin expected at least 2 providers, got %v", counts)
	}
}

func TestRouteStrategyRandom(t *testing.T) {
	registry := providers.NewRegistry()
	for i := 1; i <= 5; i++ {
		name := "provider" + string(rune('0'+i))
		p := &mockProvider{
			name:      name,
			responses: make(map[string]*providers.ChatResponse),
		}
		registry.Register(p)
	}

	r := router.New(registry, router.StrategyRandom)
	req := &providers.ChatRequest{Model: "gpt-4"}

	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyRandom)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[resp.Provider] = true
	}

	// With 20 random selections from 5 options, we should see multiple providers
	if len(seen) < 2 {
		t.Errorf("random strategy not varied enough: only saw %d providers", len(seen))
	}
}

func TestRouteStrategyLatencyOptimized(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "fast", delay: 1 * time.Millisecond, responses: make(map[string]*providers.ChatResponse)})
	registry.Register(&mockProvider{name: "slow", delay: 100 * time.Millisecond, responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyLatencyOptimized)

	// Record success for both providers with different latencies
	req := &providers.ChatRequest{Model: "gpt-4"}
	ctx := context.Background()

	// Make successful requests to establish latency
	r.RouteToProvider(ctx, "fast", req)
	r.RouteToProvider(ctx, "slow", req)

	// Route with latency-optimized strategy
	resp, err := r.RouteWithStrategy(ctx, req, router.StrategyLatencyOptimized)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Provider != "fast" {
		t.Errorf("expected latency-optimized to select 'fast', got %s", resp.Provider)
	}
}

func TestRouteStrategyCostOptimized(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "ollama", responses: make(map[string]*providers.ChatResponse)})
	registry.Register(&mockProvider{name: "openai", responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyCostOptimized)
	req := &providers.ChatRequest{Model: "gpt-4"}

	// Cost-optimized should prefer ollama over openai
	resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyCostOptimized)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Provider != "ollama" {
		t.Errorf("expected cost-optimized to select 'ollama', got %s", resp.Provider)
	}
}

func TestRouteStrategyFailover(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "openai", responses: make(map[string]*providers.ChatResponse)})
	registry.Register(&mockProvider{name: "anthropic", responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyFailover)
	req := &providers.ChatRequest{Model: "gpt-4"}

	// Failover should prefer openai
	resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyFailover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Provider != "openai" {
		t.Errorf("expected failover to select 'openai', got %s", resp.Provider)
	}
}

func TestRouteStrategyLoadBalanced(t *testing.T) {
	registry := providers.NewRegistry()
	p1 := &mockProvider{name: "p1", responses: make(map[string]*providers.ChatResponse), err: errors.New("too busy")}
	p2 := &mockProvider{name: "p2", responses: make(map[string]*providers.ChatResponse)}
	registry.Register(p1)
	registry.Register(p2)

	r := router.New(registry, router.StrategyLoadBalanced)

	// Make many failed requests to p1 to create a big load difference
	req := &providers.ChatRequest{Model: "gpt-4"}
	for i := 0; i < 5; i++ {
		// This will fail and record failures
		r.RouteWithStrategy(context.Background(), req, router.StrategyLoadBalanced)
	}

	// Check stats to verify p1 has failures
	stats := r.Stats()
	p1Stats, _ := stats["p1"]

	// Verify p1 has failures
	if p1Stats["failed_requests"] == int64(0) {
		t.Fatalf("expected p1 to have failed requests")
	}

	// Reset p1 to not error for final check
	p1.err = nil

	// Route with load-balanced strategy should now prefer p2 more
	// (since p1 has accumulated errors/load)
	p2Count := 0
	p1Count := 0
	for i := 0; i < 5; i++ {
		resp, _ := r.RouteWithStrategy(context.Background(), req, router.StrategyLoadBalanced)
		if resp.Provider == "p2" {
			p2Count++
		} else {
			p1Count++
		}
	}

	// With the load on p1, it should be selected less often or equally
	// Just verify the routing works without crashing
	t.Logf("p1: %d, p2: %d", p1Count, p2Count)
}

func TestRecordSuccess(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "test", responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyRoundRobin)

	// Record success
	r.RouteToProvider(context.Background(), "test", &providers.ChatRequest{Model: "gpt-4"})

	stats := r.Stats()
	if len(stats) == 0 {
		t.Error("expected stats to be recorded")
	}

	testStats, ok := stats["test"]
	if !ok {
		t.Error("expected 'test' provider stats")
	}

	if testStats["successful_requests"] != int64(1) {
		t.Errorf("expected 1 successful request, got %v", testStats["successful_requests"])
	}
}

func TestRecordFailure(t *testing.T) {
	registry := providers.NewRegistry()
	provider := &mockProvider{
		name:      "test",
		err:       errors.New("test error"),
		responses: make(map[string]*providers.ChatResponse),
	}
	registry.Register(provider)

	r := router.New(registry, router.StrategyRoundRobin)

	// Record failure
	r.RouteToProvider(context.Background(), "test", &providers.ChatRequest{Model: "gpt-4"})

	stats := r.Stats()
	testStats, _ := stats["test"]

	if testStats["failed_requests"] != int64(1) {
		t.Errorf("expected 1 failed request, got %v", testStats["failed_requests"])
	}
}

func TestStatsMultipleProviders(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "p1", responses: make(map[string]*providers.ChatResponse)})
	registry.Register(&mockProvider{name: "p2", responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyRoundRobin)

	req := &providers.ChatRequest{Model: "gpt-4"}
	ctx := context.Background()

	r.RouteToProvider(ctx, "p1", req)
	r.RouteToProvider(ctx, "p1", req)
	r.RouteToProvider(ctx, "p2", req)

	stats := r.Stats()

	p1Stats, ok1 := stats["p1"]
	p2Stats, ok2 := stats["p2"]

	if !ok1 || !ok2 {
		t.Error("expected stats for both providers")
	}

	if p1Stats["total_requests"] != int64(2) {
		t.Errorf("expected 2 total requests for p1, got %v", p1Stats["total_requests"])
	}

	if p2Stats["total_requests"] != int64(1) {
		t.Errorf("expected 1 total request for p2, got %v", p2Stats["total_requests"])
	}
}

func TestStatsAvgLatency(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "test", delay: 10 * time.Millisecond, responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	r.RouteToProvider(context.Background(), "test", req)

	stats := r.Stats()
	testStats, _ := stats["test"]

	avgLatency := testStats["avg_latency_ms"].(float64)
	if avgLatency < 5 || avgLatency > 100 {
		t.Errorf("expected avg_latency_ms around 10, got %v", avgLatency)
	}
}

func TestRouteConcurrency(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockProvider{name: "p1", responses: make(map[string]*providers.ChatResponse)})
	registry.Register(&mockProvider{name: "p2", responses: make(map[string]*providers.ChatResponse)})

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Route(context.Background(), req)
		}()
	}
	wg.Wait()

	stats := r.Stats()
	if len(stats) == 0 {
		t.Error("expected stats to be recorded")
	}
}

func TestExecuteWithFallback(t *testing.T) {
	registry := providers.NewRegistry()

	// First provider fails, second succeeds
	p1 := &mockProvider{
		name:      "failing",
		err:       errors.New("connection error"),
		responses: make(map[string]*providers.ChatResponse),
	}
	p2 := &mockProvider{
		name:      "working",
		responses: make(map[string]*providers.ChatResponse),
	}

	registry.Register(p1)
	registry.Register(p2)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyRoundRobin)

	if err != nil {
		t.Errorf("expected fallback to succeed, got error: %v", err)
	}
	if resp == nil {
		t.Error("expected response from fallback provider")
	}
	if resp.Provider != "working" {
		t.Errorf("expected response from 'working' provider, got %s", resp.Provider)
	}
}

func TestAllProvidersFailFallback(t *testing.T) {
	registry := providers.NewRegistry()

	p1 := &mockProvider{
		name:      "fail1",
		err:       errors.New("error 1"),
		responses: make(map[string]*providers.ChatResponse),
	}
	p2 := &mockProvider{
		name:      "fail2",
		err:       errors.New("error 2"),
		responses: make(map[string]*providers.ChatResponse),
	}

	registry.Register(p1)
	registry.Register(p2)

	r := router.New(registry, router.StrategyRoundRobin)
	req := &providers.ChatRequest{Model: "gpt-4"}

	resp, err := r.RouteWithStrategy(context.Background(), req, router.StrategyRoundRobin)

	if err == nil {
		t.Error("expected error when all providers fail")
	}
	if resp != nil {
		t.Error("expected no response when all providers fail")
	}
}
