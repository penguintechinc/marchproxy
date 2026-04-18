//go:build ci

package envoy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/envoy"
	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/logging"
)

func newTestXDSClient(t *testing.T, serverAddr string) *envoy.XDSClient {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return envoy.NewXDSClient(serverAddr, logger)
}

func TestNewXDSClient(t *testing.T) {
	client := newTestXDSClient(t, "localhost:18000")
	if client == nil {
		t.Fatal("expected non-nil xDS client")
	}
}

func TestGetRoutes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/routes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "route-1",
					"prefix": "/api",
					"cluster_name": "backend",
					"hosts": ["example.com"],
					"timeout": 30,
					"enabled": true
				}
			]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	routes, err := client.GetRoutes()

	if err != nil {
		t.Fatalf("GetRoutes() unexpected error: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}

	if routes[0].Name != "route-1" {
		t.Errorf("expected route name 'route-1', got %q", routes[0].Name)
	}

	if routes[0].Prefix != "/api" {
		t.Errorf("expected prefix '/api', got %q", routes[0].Prefix)
	}

	if routes[0].ClusterName != "backend" {
		t.Errorf("expected cluster 'backend', got %q", routes[0].ClusterName)
	}

	if !routes[0].Enabled {
		t.Error("expected route to be enabled")
	}
}

func TestGetRoutes_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	_, err := client.GetRoutes()

	if err == nil {
		t.Error("expected error for server error, got nil")
	}
}

func TestGetClusters_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clusters" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "backend",
					"protocol": "http",
					"endpoints": [
						{"host": "10.0.0.1", "port": 8080, "weight": 50},
						{"host": "10.0.0.2", "port": 8080, "weight": 50}
					]
				}
			]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	clusters, err := client.GetClusters()

	if err != nil {
		t.Fatalf("GetClusters() unexpected error: %v", err)
	}

	if len(clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	if clusters[0].Name != "backend" {
		t.Errorf("expected cluster name 'backend', got %q", clusters[0].Name)
	}

	if len(clusters[0].Endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(clusters[0].Endpoints))
	}

	if clusters[0].Endpoints[0].Host != "10.0.0.1" {
		t.Errorf("expected endpoint host '10.0.0.1', got %q", clusters[0].Endpoints[0].Host)
	}
}

func TestGetClusters_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clusters" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	clusters, err := client.GetClusters()

	if err != nil {
		t.Fatalf("GetClusters() unexpected error: %v", err)
	}

	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestUpdateRouteRateLimit(t *testing.T) {
	client := newTestXDSClient(t, "localhost:18000")

	rateLimit := &envoy.RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
		Enabled:           true,
	}

	err := client.UpdateRouteRateLimit("route-1", rateLimit)
	if err != nil {
		t.Fatalf("UpdateRouteRateLimit() unexpected error: %v", err)
	}
}

func TestUpdateTrafficWeights(t *testing.T) {
	client := newTestXDSClient(t, "localhost:18000")

	weights := map[string]int{
		"cluster-1": 80,
		"cluster-2": 20,
	}

	err := client.UpdateTrafficWeights("route-1", weights)
	if err != nil {
		t.Fatalf("UpdateTrafficWeights() unexpected error: %v", err)
	}
}

func TestHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	err := client.HealthCheck()

	if err != nil {
		t.Fatalf("HealthCheck() unexpected error: %v", err)
	}
}

func TestHealthCheck_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	err := client.HealthCheck()

	if err == nil {
		t.Error("expected error for unhealthy server, got nil")
	}
}

func TestHealthCheck_Unreachable(t *testing.T) {
	client := newTestXDSClient(t, "localhost:1")
	err := client.HealthCheck()

	if err == nil {
		t.Error("expected error for unreachable server, got nil")
	}
}

func TestRouteConfig_WithRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/routes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "limited-route",
					"prefix": "/api/v1",
					"cluster_name": "backend",
					"hosts": ["api.example.com"],
					"timeout": 60,
					"enabled": true,
					"rate_limit": {
						"requests_per_second": 500,
						"burst_size": 50,
						"enabled": true
					}
				}
			]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	routes, err := client.GetRoutes()

	if err != nil {
		t.Fatalf("GetRoutes() unexpected error: %v", err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	if routes[0].RateLimit == nil {
		t.Fatal("expected non-nil rate limit")
	}

	if routes[0].RateLimit.RequestsPerSecond != 500 {
		t.Errorf("expected RPS 500, got %d", routes[0].RateLimit.RequestsPerSecond)
	}

	if routes[0].RateLimit.BurstSize != 50 {
		t.Errorf("expected burst size 50, got %d", routes[0].RateLimit.BurstSize)
	}
}

func TestMultipleEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clusters" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"name": "multi-endpoint",
					"protocol": "http",
					"endpoints": [
						{"host": "10.0.0.1", "port": 8080, "weight": 40},
						{"host": "10.0.0.2", "port": 8080, "weight": 30},
						{"host": "10.0.0.3", "port": 8080, "weight": 30}
					]
				}
			]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	clusters, err := client.GetClusters()

	if err != nil {
		t.Fatalf("GetClusters() unexpected error: %v", err)
	}

	if len(clusters[0].Endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(clusters[0].Endpoints))
	}

	// Verify weights
	totalWeight := 0
	for _, ep := range clusters[0].Endpoints {
		totalWeight += ep.Weight
	}

	if totalWeight != 100 {
		t.Errorf("expected total weight 100, got %d", totalWeight)
	}
}

func TestNewXDSClientNilLogger(t *testing.T) {
	client := envoy.NewXDSClient("localhost:18000", nil)
	if client == nil {
		t.Fatal("expected non-nil xDS client")
	}
}

func TestGetRoutes_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/routes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	_, err := client.GetRoutes()

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGetClusters_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clusters" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`not-an-array`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestXDSClient(t, server.Listener.Addr().String())
	_, err := client.GetClusters()

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGetRoutes_Unreachable(t *testing.T) {
	client := newTestXDSClient(t, "localhost:1")
	_, err := client.GetRoutes()

	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestGetClusters_Unreachable(t *testing.T) {
	client := newTestXDSClient(t, "localhost:1")
	_, err := client.GetClusters()

	if err == nil {
		t.Error("expected error for unreachable server")
	}
}
