package router_test

import (
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected router.Strategy
	}{
		{"round_robin", router.StrategyRoundRobin},
		{"cost_optimized", router.StrategyCostOptimized},
		{"latency_optimized", router.StrategyLatencyOptimized},
		{"load_balanced", router.StrategyLoadBalanced},
		{"failover", router.StrategyFailover},
		{"random", router.StrategyRandom},
		{"unknown", router.StrategyRoundRobin},
		{"", router.StrategyRoundRobin},
		{"ROUND_ROBIN", router.StrategyRoundRobin},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := router.ParseStrategy(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStrategyConstants(t *testing.T) {
	if router.StrategyRoundRobin != "round_robin" {
		t.Errorf("expected StrategyRoundRobin to be 'round_robin', got %s", router.StrategyRoundRobin)
	}
	if router.StrategyCostOptimized != "cost_optimized" {
		t.Errorf("expected StrategyCostOptimized to be 'cost_optimized', got %s", router.StrategyCostOptimized)
	}
	if router.StrategyLatencyOptimized != "latency_optimized" {
		t.Errorf("expected StrategyLatencyOptimized to be 'latency_optimized', got %s", router.StrategyLatencyOptimized)
	}
	if router.StrategyLoadBalanced != "load_balanced" {
		t.Errorf("expected StrategyLoadBalanced to be 'load_balanced', got %s", router.StrategyLoadBalanced)
	}
	if router.StrategyFailover != "failover" {
		t.Errorf("expected StrategyFailover to be 'failover', got %s", router.StrategyFailover)
	}
	if router.StrategyRandom != "random" {
		t.Errorf("expected StrategyRandom to be 'random', got %s", router.StrategyRandom)
	}
}

func TestParseStrategyDefaultsUnknown(t *testing.T) {
	result := router.ParseStrategy("nonexistent_strategy")
	if result != router.StrategyRoundRobin {
		t.Errorf("expected unknown strategy to default to round_robin, got %s", result)
	}
}

func TestParseStrategyCaseSensitive(t *testing.T) {
	// ParseStrategy should be case-sensitive
	result := router.ParseStrategy("Round_Robin")
	if result != router.StrategyRoundRobin {
		t.Errorf("expected case-sensitive match to fail, got %s", result)
	}
}

func TestAllStrategies(t *testing.T) {
	strategies := []router.Strategy{
		router.StrategyRoundRobin,
		router.StrategyCostOptimized,
		router.StrategyLatencyOptimized,
		router.StrategyLoadBalanced,
		router.StrategyFailover,
		router.StrategyRandom,
	}

	for _, s := range strategies {
		t.Run(string(s), func(t *testing.T) {
			parsed := router.ParseStrategy(string(s))
			if parsed != s {
				t.Errorf("expected %v to parse to itself, got %v", s, parsed)
			}
		})
	}
}
