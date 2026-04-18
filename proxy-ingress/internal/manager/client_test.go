//go:build ci

package manager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-ingress/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{}
	cfg.Manager.URL = "http://localhost:8080"
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL 'http://localhost:8080', got '%s'", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got '%s'", client.apiKey)
	}
}

func TestSetClusterInfo(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	client.SetClusterInfo(123, "test-cluster")

	if client.GetClusterID() != 123 {
		t.Errorf("expected cluster ID 123, got %d", client.GetClusterID())
	}
	if client.GetClusterName() != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got '%s'", client.GetClusterName())
	}
}

func TestGetClusterID(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)
	client.clusterID = 42

	if client.GetClusterID() != 42 {
		t.Errorf("expected cluster ID 42, got %d", client.GetClusterID())
	}
}

func TestGetClusterName(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)
	client.clusterName = "my-cluster"

	if client.GetClusterName() != "my-cluster" {
		t.Errorf("expected cluster name 'my-cluster', got '%s'", client.GetClusterName())
	}
}

func TestGetLastConfigTime(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	expected := time.Now()
	client.lastConfigTime = expected

	if client.GetLastConfigTime() != expected {
		t.Error("expected time to match")
	}
}

func TestGetLastConfigHash(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	client.lastConfigHash = "abc123"

	if client.GetLastConfigHash() != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", client.GetLastConfigHash())
	}
}

func TestRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/proxies/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("expected 'Bearer test-key', got '%s'", auth)
		}

		response := RegistrationResponse{
			Success:     true,
			ProxyID:     999,
			ClusterName: "registered-cluster",
			Message:     "Success",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	resp, err := client.Register(context.Background(), "test-proxy", "hostname", "1.0.0", []string{"cap1"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success response")
	}
	if resp.ProxyID != 999 {
		t.Errorf("expected proxy ID 999, got %d", resp.ProxyID)
	}
	if client.GetClusterID() != 999 {
		t.Errorf("expected cluster ID set to 999, got %d", client.GetClusterID())
	}
}

func TestGetConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := ConfigResponse{
			Success: true,
			Hash:    "hash123",
			Data: ClusterConfig{
				ConfigHash: "hash123",
				Version:    "1.0.0",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	result, err := client.GetConfig(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil config")
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", result.Version)
	}
	if client.GetLastConfigHash() != "hash123" {
		t.Errorf("expected last config hash 'hash123', got '%s'", client.GetLastConfigHash())
	}
}

func TestGetConfigError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ConfigResponse{
			Success: false,
			Error:   "Config not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	_, err := client.GetConfig(context.Background())

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReportHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		response := HealthReportResponse{
			Success: true,
			Message: "Health report received",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	client.clusterID = 42

	report := HealthReportRequest{
		Status: "healthy",
		CPUUsage: 25.5,
		MemoryUsage: 512000,
	}

	err := client.ReportHealth(context.Background(), report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportHealthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := HealthReportResponse{
			Success: false,
			Error:   "Invalid proxy ID",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)

	err := client.ReportHealth(context.Background(), HealthReportRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCertificate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/certificates/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := Certificate{
			ID:   42,
			Name: "test-cert",
			Type: "ssl",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	cert, err := client.GetCertificate(context.Background(), 42)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert.ID != 42 {
		t.Errorf("expected cert ID 42, got %d", cert.ID)
	}
	if cert.Name != "test-cert" {
		t.Errorf("expected cert name 'test-cert', got '%s'", cert.Name)
	}
}

func TestListCertificates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/certificates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := []Certificate{
			{ID: 1, Name: "cert1", Type: "ssl"},
			{ID: 2, Name: "cert2", Type: "tls"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	certs, err := client.ListCertificates(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(certs) != 2 {
		t.Errorf("expected 2 certificates, got %d", len(certs))
	}
}

func TestNotifyConfigUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notifications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"success": true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	err := client.NotifyConfigUpdate(context.Background(), "update", "Config updated")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/backends/backend1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := Backend{
			ID:   1,
			Name: "backend1",
			Type: "http",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	backend, err := client.GetBackend(context.Background(), "backend1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend.Name != "backend1" {
		t.Errorf("expected name 'backend1', got '%s'", backend.Name)
	}
}

func TestListBackends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/backends" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := []Backend{
			{ID: 1, Name: "backend1", Type: "http"},
			{ID: 2, Name: "backend2", Type: "grpc"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	backends, err := client.ListBackends(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(backends))
	}
}

func TestGetVirtualHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vhosts/vhost1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := VirtualHost{
			ID:       1,
			Name:     "vhost1",
			Hostname: "example.com",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	vhost, err := client.GetVirtualHost(context.Background(), "vhost1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vhost.Hostname != "example.com" {
		t.Errorf("expected hostname 'example.com', got '%s'", vhost.Hostname)
	}
}

func TestListVirtualHosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vhosts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := []VirtualHost{
			{ID: 1, Name: "vhost1", Hostname: "example1.com"},
			{ID: 2, Name: "vhost2", Hostname: "example2.com"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	vhosts, err := client.ListVirtualHosts(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vhosts) != 2 {
		t.Errorf("expected 2 vhosts, got %d", len(vhosts))
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ping" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"status": "pong",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	err := client.Ping(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMakeRequestBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	err := client.Ping(context.Background())

	if err == nil {
		t.Fatal("expected error for bad status code")
	}
}

func TestMakeRequestBadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)
	_, err := client.GetConfig(context.Background())

	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestPollConfigChanges(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := ConfigResponse{
			Success: true,
			Hash:    "hash1",
			Data: ClusterConfig{
				ConfigHash: "hash1",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.Manager.URL = server.URL
	cfg.Manager.APIKey = "test-key"

	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	configChan := client.PollConfigChanges(ctx, 50*time.Millisecond)

	// Consume at least one config
	select {
	case config := <-configChan:
		if config == nil {
			t.Fatal("expected non-nil config")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for config")
	}
}
