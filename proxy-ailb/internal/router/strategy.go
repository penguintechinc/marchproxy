package router

// Strategy defines how requests are distributed across providers.
type Strategy string

const (
	StrategyRoundRobin       Strategy = "round_robin"
	StrategyCostOptimized    Strategy = "cost_optimized"
	StrategyLatencyOptimized Strategy = "latency_optimized"
	StrategyLoadBalanced     Strategy = "load_balanced"
	StrategyFailover         Strategy = "failover"
	StrategyRandom           Strategy = "random"
)

// ParseStrategy converts a string to a Strategy, defaulting to round_robin.
func ParseStrategy(s string) Strategy {
	switch Strategy(s) {
	case StrategyRoundRobin, StrategyCostOptimized, StrategyLatencyOptimized,
		StrategyLoadBalanced, StrategyFailover, StrategyRandom:
		return Strategy(s)
	default:
		return StrategyRoundRobin
	}
}
