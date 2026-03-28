package multicloud_test

import (
	"testing"

	"marchproxy-l3l4/internal/multicloud"
)

// helpers

func healthyBackend(name string, latency int64, cost float64, conns int, weight int) *multicloud.Backend {
	return &multicloud.Backend{
		Name:        name,
		Healthy:     true,
		Latency:     latency,
		Cost:        cost,
		Connections: conns,
		Weight:      weight,
		Cloud:       "aws",
		Region:      "us-east-1",
	}
}

func unhealthyBackend(name string) *multicloud.Backend {
	return &multicloud.Backend{
		Name:    name,
		Healthy: false,
	}
}

var emptyRequest = &multicloud.Request{}

// ----- LatencyBasedAlgorithm -----

func TestLatencyBasedAlgorithm_Name(t *testing.T) {
	a := &multicloud.LatencyBasedAlgorithm{}
	if a.Name() != "latency" {
		t.Errorf("Name() = %q, want %q", a.Name(), "latency")
	}
}

func TestLatencyBasedAlgorithm_SelectsLowestLatency(t *testing.T) {
	a := &multicloud.LatencyBasedAlgorithm{}
	backends := []*multicloud.Backend{
		healthyBackend("slow", 500, 0.1, 0, 1),
		healthyBackend("fast", 10, 0.1, 0, 1),
		healthyBackend("medium", 200, 0.1, 0, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil {
		t.Fatal("Select() returned nil, want non-nil")
	}
	if selected.Name != "fast" {
		t.Errorf("Select() = %q, want %q", selected.Name, "fast")
	}
}

func TestLatencyBasedAlgorithm_SkipsUnhealthyBackends(t *testing.T) {
	a := &multicloud.LatencyBasedAlgorithm{}
	backends := []*multicloud.Backend{
		unhealthyBackend("bad"),
		healthyBackend("good", 100, 0.1, 0, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil {
		t.Fatal("Select() returned nil")
	}
	if selected.Name != "good" {
		t.Errorf("Select() = %q, want %q", selected.Name, "good")
	}
}

func TestLatencyBasedAlgorithm_AllUnhealthyReturnsNil(t *testing.T) {
	a := &multicloud.LatencyBasedAlgorithm{}
	backends := []*multicloud.Backend{
		unhealthyBackend("b1"),
		unhealthyBackend("b2"),
	}
	if got := a.Select(backends, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil when all backends unhealthy", got)
	}
}

func TestLatencyBasedAlgorithm_EmptyBackendsReturnsNil(t *testing.T) {
	a := &multicloud.LatencyBasedAlgorithm{}
	if got := a.Select([]*multicloud.Backend{}, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil for empty backend list", got)
	}
}

// ----- CostBasedAlgorithm -----

func TestCostBasedAlgorithm_Name(t *testing.T) {
	a := &multicloud.CostBasedAlgorithm{}
	if a.Name() != "cost" {
		t.Errorf("Name() = %q, want %q", a.Name(), "cost")
	}
}

func TestCostBasedAlgorithm_SelectsLowestCost(t *testing.T) {
	a := &multicloud.CostBasedAlgorithm{}
	backends := []*multicloud.Backend{
		healthyBackend("expensive", 100, 1.0, 0, 1),
		healthyBackend("cheap", 100, 0.01, 0, 1),
		healthyBackend("medium", 100, 0.5, 0, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil {
		t.Fatal("Select() returned nil")
	}
	if selected.Name != "cheap" {
		t.Errorf("Select() = %q, want %q", selected.Name, "cheap")
	}
}

func TestCostBasedAlgorithm_SkipsUnhealthy(t *testing.T) {
	a := &multicloud.CostBasedAlgorithm{}
	backends := []*multicloud.Backend{
		{Name: "very-cheap", Cost: 0.001, Healthy: false},
		healthyBackend("affordable", 100, 0.5, 0, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil || selected.Name != "affordable" {
		t.Errorf("Select() = %v, want \"affordable\"", selected)
	}
}

func TestCostBasedAlgorithm_AllUnhealthyReturnsNil(t *testing.T) {
	a := &multicloud.CostBasedAlgorithm{}
	backends := []*multicloud.Backend{unhealthyBackend("b1")}
	if got := a.Select(backends, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil", got)
	}
}

// ----- GeoProximityAlgorithm -----

func TestGeoProximityAlgorithm_Name(t *testing.T) {
	a := &multicloud.GeoProximityAlgorithm{}
	if a.Name() != "geo" {
		t.Errorf("Name() = %q, want %q", a.Name(), "geo")
	}
}

func TestGeoProximityAlgorithm_SelectsFirstHealthy(t *testing.T) {
	a := &multicloud.GeoProximityAlgorithm{}
	backends := []*multicloud.Backend{
		unhealthyBackend("b1"),
		healthyBackend("b2", 100, 0.1, 0, 1),
		healthyBackend("b3", 200, 0.1, 0, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil || selected.Name != "b2" {
		t.Errorf("Select() = %v, want \"b2\"", selected)
	}
}

func TestGeoProximityAlgorithm_AllUnhealthyReturnsNil(t *testing.T) {
	a := &multicloud.GeoProximityAlgorithm{}
	if got := a.Select([]*multicloud.Backend{unhealthyBackend("x")}, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil", got)
	}
}

// ----- RoundRobinAlgorithm -----

func TestRoundRobinAlgorithm_Name(t *testing.T) {
	a := &multicloud.RoundRobinAlgorithm{}
	if a.Name() != "roundrobin" {
		t.Errorf("Name() = %q, want %q", a.Name(), "roundrobin")
	}
}

func TestRoundRobinAlgorithm_EmptyBackendsReturnsNil(t *testing.T) {
	a := &multicloud.RoundRobinAlgorithm{}
	if got := a.Select([]*multicloud.Backend{}, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil for empty list", got)
	}
}

func TestRoundRobinAlgorithm_AllUnhealthyReturnsNil(t *testing.T) {
	a := &multicloud.RoundRobinAlgorithm{}
	backends := []*multicloud.Backend{unhealthyBackend("b1"), unhealthyBackend("b2")}
	if got := a.Select(backends, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil when all unhealthy", got)
	}
}

func TestRoundRobinAlgorithm_ReturnsDifferentBackendsInSequence(t *testing.T) {
	a := &multicloud.RoundRobinAlgorithm{}
	b1 := healthyBackend("b1", 100, 0.1, 0, 1)
	b2 := healthyBackend("b2", 100, 0.1, 0, 1)
	backends := []*multicloud.Backend{b1, b2}

	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		selected := a.Select(backends, emptyRequest)
		if selected == nil {
			t.Fatal("Select() returned nil with healthy backends")
		}
		seen[selected.Name] = true
	}
	if !seen["b1"] || !seen["b2"] {
		t.Errorf("round-robin did not distribute across both backends; seen: %v", seen)
	}
}

func TestRoundRobinAlgorithm_SingleHealthyBackend(t *testing.T) {
	a := &multicloud.RoundRobinAlgorithm{}
	b := healthyBackend("only", 100, 0.1, 0, 1)
	for i := 0; i < 5; i++ {
		selected := a.Select([]*multicloud.Backend{b}, emptyRequest)
		if selected == nil || selected.Name != "only" {
			t.Errorf("iteration %d: Select() = %v, want \"only\"", i, selected)
		}
	}
}

// ----- LeastConnectionAlgorithm -----

func TestLeastConnectionAlgorithm_Name(t *testing.T) {
	a := &multicloud.LeastConnectionAlgorithm{}
	if a.Name() != "leastconn" {
		t.Errorf("Name() = %q, want %q", a.Name(), "leastconn")
	}
}

func TestLeastConnectionAlgorithm_SelectsFewestConnections(t *testing.T) {
	a := &multicloud.LeastConnectionAlgorithm{}
	backends := []*multicloud.Backend{
		healthyBackend("busy", 100, 0.1, 100, 1),
		healthyBackend("idle", 100, 0.1, 1, 1),
		healthyBackend("moderate", 100, 0.1, 50, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil || selected.Name != "idle" {
		t.Errorf("Select() = %v, want \"idle\"", selected)
	}
}

func TestLeastConnectionAlgorithm_SkipsUnhealthy(t *testing.T) {
	a := &multicloud.LeastConnectionAlgorithm{}
	backends := []*multicloud.Backend{
		{Name: "zero-conns", Connections: 0, Healthy: false},
		healthyBackend("few-conns", 100, 0.1, 5, 1),
	}
	selected := a.Select(backends, emptyRequest)
	if selected == nil || selected.Name != "few-conns" {
		t.Errorf("Select() = %v, want \"few-conns\"", selected)
	}
}

func TestLeastConnectionAlgorithm_AllUnhealthyReturnsNil(t *testing.T) {
	a := &multicloud.LeastConnectionAlgorithm{}
	backends := []*multicloud.Backend{unhealthyBackend("b1")}
	if got := a.Select(backends, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil", got)
	}
}

// ----- WeightedRoundRobinAlgorithm -----

func TestWeightedRoundRobinAlgorithm_Name(t *testing.T) {
	a := &multicloud.WeightedRoundRobinAlgorithm{}
	if a.Name() != "weighted_roundrobin" {
		t.Errorf("Name() = %q, want %q", a.Name(), "weighted_roundrobin")
	}
}

func TestWeightedRoundRobinAlgorithm_EmptyReturnsNil(t *testing.T) {
	a := &multicloud.WeightedRoundRobinAlgorithm{}
	if got := a.Select([]*multicloud.Backend{}, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil", got)
	}
}

func TestWeightedRoundRobinAlgorithm_AllUnhealthyReturnsNil(t *testing.T) {
	a := &multicloud.WeightedRoundRobinAlgorithm{}
	backends := []*multicloud.Backend{
		{Name: "b1", Healthy: false, Weight: 5},
	}
	if got := a.Select(backends, emptyRequest); got != nil {
		t.Errorf("Select() = %v, want nil", got)
	}
}

func TestWeightedRoundRobinAlgorithm_HigherWeightSelectedMoreOften(t *testing.T) {
	a := &multicloud.WeightedRoundRobinAlgorithm{}
	low := healthyBackend("low", 100, 0.1, 0, 1)   // weight 1
	high := healthyBackend("high", 100, 0.1, 0, 9)  // weight 9
	backends := []*multicloud.Backend{low, high}

	counts := map[string]int{}
	for i := 0; i < 100; i++ {
		b := a.Select(backends, emptyRequest)
		if b != nil {
			counts[b.Name]++
		}
	}
	// "high" should appear ~9x more often than "low"
	if counts["high"] <= counts["low"] {
		t.Errorf("weighted RR favoured low-weight backend: high=%d low=%d", counts["high"], counts["low"])
	}
}

// ----- Backend struct -----

func TestBackend_FieldsAccessible(t *testing.T) {
	b := &multicloud.Backend{
		Name:        "test",
		URL:         "http://127.0.0.1:8080",
		Weight:      3,
		Priority:    1,
		Cloud:       "gcp",
		Region:      "eu-west1",
		Cost:        0.02,
		Healthy:     true,
		Latency:     42,
		Connections: 7,
	}
	if b.Name != "test" || b.Cloud != "gcp" || b.Latency != 42 {
		t.Errorf("unexpected Backend field values: %+v", b)
	}
}

// ----- Request struct -----

func TestRequest_FieldsAccessible(t *testing.T) {
	r := &multicloud.Request{
		SourceIP:   "10.0.0.1",
		DestIP:     "10.0.0.2",
		Protocol:   "tcp",
		SourceLat:  37.7749,
		SourceLong: -122.4194,
	}
	if r.SourceIP != "10.0.0.1" || r.Protocol != "tcp" {
		t.Errorf("unexpected Request field values: %+v", r)
	}
}
