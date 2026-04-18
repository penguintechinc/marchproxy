package levers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClusterRegistry(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}
	if registry == nil {
		t.Fatal("registry should not be nil")
	}
	if registry.clusters == nil {
		t.Fatal("clusters map should be initialized")
	}
	if registry.logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestHandlePushSuccess(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	clusters := []ClusterDef{
		{
			Name:      "service-a",
			Endpoints: []string{"10.0.0.1:8080", "10.0.0.2:8080"},
			LBPolicy:  "ROUND_ROBIN",
		},
		{
			Name:      "service-b",
			Endpoints: []string{"10.0.0.3:8080"},
			LBPolicy:  "LEAST_REQUEST",
		},
	}

	body, err := json.Marshal(clusters)
	if err != nil {
		t.Fatalf("Failed to marshal clusters: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader(body))
	w := httptest.NewRecorder()

	registry.HandlePush(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify clusters were stored
	stored, ok := registry.Get("service-a")
	if !ok {
		t.Fatal("service-a should be stored")
	}
	if stored.Name != "service-a" {
		t.Errorf("expected name 'service-a', got '%s'", stored.Name)
	}
	if len(stored.Endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(stored.Endpoints))
	}
	if stored.LBPolicy != "ROUND_ROBIN" {
		t.Errorf("expected LBPolicy 'ROUND_ROBIN', got '%s'", stored.LBPolicy)
	}
}

func TestHandlePushInvalidMethod(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/levers/clusters", nil)
	w := httptest.NewRecorder()

	registry.HandlePush(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandlePushInvalidBody(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	registry.HandlePush(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandlePushMultipleTimes(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	// First push
	clusters1 := []ClusterDef{
		{
			Name:      "service-a",
			Endpoints: []string{"10.0.0.1:8080"},
			LBPolicy:  "ROUND_ROBIN",
		},
	}
	body1, _ := json.Marshal(clusters1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	registry.HandlePush(w1, req1)

	// Second push (override)
	clusters2 := []ClusterDef{
		{
			Name:      "service-a",
			Endpoints: []string{"10.0.0.1:8080", "10.0.0.2:8080", "10.0.0.3:8080"},
			LBPolicy:  "LEAST_REQUEST",
		},
	}
	body2, _ := json.Marshal(clusters2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	registry.HandlePush(w2, req2)

	// Verify last update is stored
	stored, ok := registry.Get("service-a")
	if !ok {
		t.Fatal("service-a should be stored")
	}
	if len(stored.Endpoints) != 3 {
		t.Errorf("expected 3 endpoints after update, got %d", len(stored.Endpoints))
	}
	if stored.LBPolicy != "LEAST_REQUEST" {
		t.Errorf("expected LBPolicy 'LEAST_REQUEST', got '%s'", stored.LBPolicy)
	}
}

func TestGet(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	clusters := []ClusterDef{
		{
			Name:      "test-cluster",
			Endpoints: []string{"10.0.0.1:8080"},
			LBPolicy:  "ROUND_ROBIN",
		},
	}
	body, _ := json.Marshal(clusters)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader(body))
	w := httptest.NewRecorder()
	registry.HandlePush(w, req)

	// Get existing cluster
	cluster, ok := registry.Get("test-cluster")
	if !ok {
		t.Fatal("cluster should be found")
	}
	if cluster.Name != "test-cluster" {
		t.Errorf("expected name 'test-cluster', got '%s'", cluster.Name)
	}

	// Get non-existent cluster
	_, ok = registry.Get("non-existent")
	if ok {
		t.Fatal("non-existent cluster should not be found")
	}
}

func TestList(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	// Empty list
	list := registry.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d clusters", len(list))
	}

	// Push clusters
	clusters := []ClusterDef{
		{
			Name:      "service-a",
			Endpoints: []string{"10.0.0.1:8080"},
			LBPolicy:  "ROUND_ROBIN",
		},
		{
			Name:      "service-b",
			Endpoints: []string{"10.0.0.2:8080"},
			LBPolicy:  "LEAST_REQUEST",
		},
		{
			Name:      "service-c",
			Endpoints: []string{"10.0.0.3:8080"},
			LBPolicy:  "WEIGHTED",
		},
	}
	body, _ := json.Marshal(clusters)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader(body))
	w := httptest.NewRecorder()
	registry.HandlePush(w, req)

	// List should return all clusters
	list = registry.List()
	if len(list) != 3 {
		t.Errorf("expected 3 clusters, got %d", len(list))
	}

	// Verify all clusters are in list
	names := make(map[string]bool)
	for _, c := range list {
		names[c.Name] = true
	}
	if !names["service-a"] || !names["service-b"] || !names["service-c"] {
		t.Fatal("not all clusters found in list")
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry, err := NewClusterRegistry()
	if err != nil {
		t.Fatalf("NewClusterRegistry failed: %v", err)
	}

	// Push initial cluster
	clusters := []ClusterDef{
		{
			Name:      "test",
			Endpoints: []string{"10.0.0.1:8080"},
			LBPolicy:  "ROUND_ROBIN",
		},
	}
	body, _ := json.Marshal(clusters)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/levers/clusters", bytes.NewReader(body))
	w := httptest.NewRecorder()
	registry.HandlePush(w, req)

	// Concurrent reads should not panic
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			registry.Get("test")
			registry.List()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
