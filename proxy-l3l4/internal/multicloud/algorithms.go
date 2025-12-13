package multicloud

import (
	"math"
	"sync/atomic"
)

// RoutingAlgorithm defines the interface for routing algorithms
type RoutingAlgorithm interface {
	Select(backends []*Backend, request *Request) *Backend
	Name() string
}

// LatencyBasedAlgorithm selects backend with lowest latency
type LatencyBasedAlgorithm struct{}

func (a *LatencyBasedAlgorithm) Select(backends []*Backend, request *Request) *Backend {
	var selected *Backend
	minLatency := int64(math.MaxInt64)

	for _, backend := range backends {
		if backend.Healthy && backend.Latency < minLatency {
			minLatency = backend.Latency
			selected = backend
		}
	}

	return selected
}

func (a *LatencyBasedAlgorithm) Name() string {
	return "latency"
}

// CostBasedAlgorithm selects backend with lowest cost
type CostBasedAlgorithm struct{}

func (a *CostBasedAlgorithm) Select(backends []*Backend, request *Request) *Backend {
	var selected *Backend
	minCost := math.MaxFloat64

	for _, backend := range backends {
		if backend.Healthy && backend.Cost < minCost {
			minCost = backend.Cost
			selected = backend
		}
	}

	return selected
}

func (a *CostBasedAlgorithm) Name() string {
	return "cost"
}

// GeoProximityAlgorithm selects backend based on geographic proximity
type GeoProximityAlgorithm struct{}

func (a *GeoProximityAlgorithm) Select(backends []*Backend, request *Request) *Backend {
	// Simplified: select first healthy backend
	// In production, this would use actual geo-location data
	for _, backend := range backends {
		if backend.Healthy {
			return backend
		}
	}
	return nil
}

func (a *GeoProximityAlgorithm) Name() string {
	return "geo"
}

// RoundRobinAlgorithm distributes traffic evenly
type RoundRobinAlgorithm struct {
	counter uint64
}

func (a *RoundRobinAlgorithm) Select(backends []*Backend, request *Request) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Filter healthy backends
	healthy := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		if backend.Healthy {
			healthy = append(healthy, backend)
		}
	}

	if len(healthy) == 0 {
		return nil
	}

	// Round-robin selection
	idx := atomic.AddUint64(&a.counter, 1) % uint64(len(healthy))
	return healthy[idx]
}

func (a *RoundRobinAlgorithm) Name() string {
	return "roundrobin"
}

// LeastConnectionAlgorithm selects backend with fewest connections
type LeastConnectionAlgorithm struct{}

func (a *LeastConnectionAlgorithm) Select(backends []*Backend, request *Request) *Backend {
	var selected *Backend
	minConns := int(^uint(0) >> 1) // Max int

	for _, backend := range backends {
		if backend.Healthy && backend.Connections < minConns {
			minConns = backend.Connections
			selected = backend
		}
	}

	return selected
}

func (a *LeastConnectionAlgorithm) Name() string {
	return "leastconn"
}

// WeightedRoundRobinAlgorithm distributes traffic based on weights
type WeightedRoundRobinAlgorithm struct {
	counter uint64
}

func (a *WeightedRoundRobinAlgorithm) Select(backends []*Backend, request *Request) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Build weighted list
	var weighted []*Backend
	for _, backend := range backends {
		if backend.Healthy {
			for i := 0; i < backend.Weight; i++ {
				weighted = append(weighted, backend)
			}
		}
	}

	if len(weighted) == 0 {
		return nil
	}

	idx := atomic.AddUint64(&a.counter, 1) % uint64(len(weighted))
	return weighted[idx]
}

func (a *WeightedRoundRobinAlgorithm) Name() string {
	return "weighted_roundrobin"
}
