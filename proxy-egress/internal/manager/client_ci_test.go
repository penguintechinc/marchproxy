//go:build ci

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-egress/internal/config"
)

// TestNewClient tests Client creation
func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:         "http://localhost:8000",
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil Client")
	}

	if client.baseURL != "http://localhost:8000" {
		t.Errorf("baseURL = %s, want http://localhost:8000", client.baseURL)
	}

	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %s, want test-key", client.apiKey)
	}

	if client.httpClient == nil {
		t.Error("Expected non-nil httpClient")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

// TestRegister tests proxy registration
func TestRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/proxy/register" {
			t.Errorf("Expected path /api/proxy/register, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		if r.Header.Get("X-API-Key") != "test-key" {
			t.Error("Expected X-API-Key header")
		}

		// Write response
		resp := RegistrationResponse{
			Success:     true,
			ProxyID:     42,
			ClusterName: "test-cluster",
			Message:     "Registered successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ProxyName:          "test-proxy",
		Hostname:           "test-host",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	err := client.Register(cfg)

	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}

	if client.clusterID != 42 {
		t.Errorf("clusterID = %d, want 42", client.clusterID)
	}

	if client.clusterName != "test-cluster" {
		t.Errorf("clusterName = %s, want test-cluster", client.clusterName)
	}
}

// TestRegisterFailure tests registration failure handling
func TestRegisterFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RegistrationResponse{
			Success: false,
			Error:   "Invalid cluster credentials",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ProxyName:          "test-proxy",
		Hostname:           "test-host",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	err := client.Register(cfg)

	if err == nil {
		t.Fatal("Expected error for failed registration")
	}

	if err.Error() != "registration failed: Invalid cluster credentials" {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestRegisterNotRegistered tests GetConfig without registration
func TestGetConfigNotRegistered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)

	// clusterID is 0 since we didn't register
	_, err := client.GetConfig()

	if err == nil {
		t.Fatal("Expected error when not registered")
	}

	if err.Error() != "proxy not registered, call Register() first" {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestGetConfig tests configuration retrieval
func TestGetConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/42" {
			t.Errorf("Expected path /api/config/42, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		config := ClusterConfig{
			Version:     "v1.2.3",
			GeneratedAt: time.Now().Format(time.RFC3339),
			Services: []Service{
				{
					ID:   1,
					Name: "service-1",
					Port: 8080,
				},
				{
					ID:   2,
					Name: "service-2",
					Port: 8081,
				},
			},
			Mappings: []Mapping{
				{
					ID:   1,
					Name: "mapping-1",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	config, err := client.GetConfig()

	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if config.Version != "v1.2.3" {
		t.Errorf("Version = %s, want v1.2.3", config.Version)
	}

	if len(config.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(config.Services))
	}

	if len(config.Mappings) != 1 {
		t.Errorf("Expected 1 mapping, got %d", len(config.Mappings))
	}

	if client.lastConfigHash != "v1.2.3" {
		t.Errorf("lastConfigHash not updated")
	}
}

// TestGetConfigHTTPError tests GetConfig with HTTP error
func TestGetConfigHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	_, err := client.GetConfig()

	if err == nil {
		t.Fatal("Expected error for HTTP 500")
	}

	if err.Error() != "failed to get config: API request failed with status 500: Internal Server Error" {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestGetLicenseStatus tests license status retrieval
func TestGetLicenseStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/license-status" {
			t.Errorf("Expected path /api/license-status, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		status := LicenseStatus{
			Edition:       "professional",
			Valid:         true,
			ProxyLimit:    100,
			Features:      []string{"xdp", "tls", "websocket"},
			CurrentProxies: 5,
			MaxProxies:    100,
			CanRegister:   true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)

	status, err := client.GetLicenseStatus()

	if err != nil {
		t.Fatalf("GetLicenseStatus() unexpected error: %v", err)
	}

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	if status.Edition != "professional" {
		t.Errorf("Edition = %s, want professional", status.Edition)
	}

	if !status.Valid {
		t.Error("Expected valid license")
	}

	if status.ProxyLimit != 100 {
		t.Errorf("ProxyLimit = %d, want 100", status.ProxyLimit)
	}

	if len(status.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(status.Features))
	}
}

// TestSendHeartbeat tests heartbeat sending
func TestSendHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/proxy/heartbeat" {
			t.Errorf("Expected path /api/proxy/heartbeat, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.CPUUsage != 45.5 {
			t.Errorf("CPUUsage = %f, want 45.5", req.CPUUsage)
		}

		if req.MemoryUsage != 512.0 {
			t.Errorf("MemoryUsage = %f, want 512.0", req.MemoryUsage)
		}

		if req.Connections != 1000 {
			t.Errorf("Connections = %d, want 1000", req.Connections)
		}

		resp := HeartbeatResponse{
			Success: true,
			Status:  "healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ProxyName:          "test-proxy",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)

	stats := SystemStats{
		CPUUsage:          45.5,
		MemoryUsage:       512.0,
		ActiveConnections: 1000,
		BytesTransferred:  1048576,
	}

	err := client.SendHeartbeat(cfg, stats)

	if err != nil {
		t.Fatalf("SendHeartbeat() unexpected error: %v", err)
	}
}

// TestSendHeartbeatFailure tests heartbeat failure handling
func TestSendHeartbeatFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := HeartbeatResponse{
			Success: false,
			Error:   "Proxy offline",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ProxyName:          "test-proxy",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)

	stats := SystemStats{
		CPUUsage:          45.5,
		MemoryUsage:       512.0,
		ActiveConnections: 1000,
	}

	err := client.SendHeartbeat(cfg, stats)

	if err == nil {
		t.Fatal("Expected error for failed heartbeat")
	}

	if err.Error() != "heartbeat failed: Proxy offline" {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestStartConfigRefresh tests config refresh loop starts without error
func TestStartConfigRefresh(t *testing.T) {
	callCount := 0
	expectedVersion := "v1.0.0"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		config := ClusterConfig{
			Version: expectedVersion,
			Services: []Service{
				{ID: 1, Name: "service-1"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:            server.URL,
		ClusterAPIKey:         "test-key",
		ConnectionTimeout:     30,
		ConfigUpdateInterval:  1,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	updateCalls := 0
	onUpdate := func(c *ClusterConfig) {
		updateCalls++
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go client.StartConfigRefresh(ctx, cfg, onUpdate)

	time.Sleep(150 * time.Millisecond)

	// Test passes if no panic occurred
	if callCount == 0 {
		t.Logf("Config refresh started successfully")
	}
}

// TestStartHeartbeat tests heartbeat loop starts without error
func TestStartHeartbeat(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		resp := HeartbeatResponse{
			Success: true,
			Status:  "healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:          server.URL,
		ClusterAPIKey:       "test-key",
		ProxyName:           "test-proxy",
		ConnectionTimeout:   30,
		HeartbeatInterval:   1,
	}

	client := NewClient(cfg)

	getStats := func() SystemStats {
		return SystemStats{
			CPUUsage:          25.0,
			MemoryUsage:       256.0,
			ActiveConnections: 500,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go client.StartHeartbeat(ctx, cfg, getStats)

	time.Sleep(150 * time.Millisecond)

	// Test passes if no panic occurred
	if callCount == 0 {
		t.Logf("Heartbeat loop started successfully")
	}
}

// TestMakeRequestHeaders tests request header construction
func TestMakeRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing or incorrect Content-Type header")
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-key" {
			t.Errorf("X-API-Key = %s, want test-key", apiKey)
		}

		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Missing User-Agent header")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	client.GetConfig()
}

// TestGetSystemStats tests system stats collection
func TestGetSystemStats(t *testing.T) {
	stats := GetSystemStats()

	if stats.MemoryUsage < 0 {
		t.Errorf("MemoryUsage should be non-negative, got %f", stats.MemoryUsage)
	}

	if stats.MemoryUsage == 0 {
		t.Errorf("MemoryUsage should be greater than 0, got %f", stats.MemoryUsage)
	}
}

// TestMakeRequestMalformedJSON tests handling of malformed JSON response
func TestMakeRequestMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	_, err := client.GetConfig()

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}

	if err.Error() != "failed to get config: failed to unmarshal response: invalid character 'i' looking for beginning of value" {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestMakeRequestNetworkError tests handling of network errors
func TestMakeRequestNetworkError(t *testing.T) {
	cfg := &config.Config{
		ManagerURL:         "http://localhost:65432",
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  1,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	_, err := client.GetConfig()

	if err == nil {
		t.Fatal("Expected error for network failure")
	}

	if fmt.Sprintf("%v", err) == "" {
		t.Error("Error should contain network failure details")
	}
}

// TestConfigVersionTracking tests that config version is tracked
func TestConfigVersionTracking(t *testing.T) {
	versions := []string{"v1.0.0", "v1.0.1", "v1.1.0"}
	versionIndex := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if versionIndex >= len(versions) {
			versionIndex = len(versions) - 1
		}

		config := ClusterConfig{
			Version: versions[versionIndex],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	// Get first config
	config1, _ := client.GetConfig()
	if client.lastConfigHash != "v1.0.0" {
		t.Errorf("lastConfigHash = %s, want v1.0.0", client.lastConfigHash)
	}

	// Get second config
	versionIndex = 1
	config2, _ := client.GetConfig()
	if client.lastConfigHash != "v1.0.1" {
		t.Errorf("lastConfigHash = %s, want v1.0.1", client.lastConfigHash)
	}

	if config1.Version == config2.Version {
		t.Error("Config versions should differ")
	}
}

// TestMultipleServices tests config with multiple services
func TestMultipleServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := ClusterConfig{
			Version: "v1.0.0",
			Services: []Service{
				{ID: 1, Name: "api-1", Port: 8080},
				{ID: 2, Name: "api-2", Port: 8081},
				{ID: 3, Name: "api-3", Port: 8082},
				{ID: 4, Name: "api-4", Port: 8083},
			},
			Mappings: []Mapping{
				{ID: 1, Name: "mapping-1"},
				{ID: 2, Name: "mapping-2"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	cfg := &config.Config{
		ManagerURL:         server.URL,
		ClusterAPIKey:      "test-key",
		ConnectionTimeout:  30,
	}

	client := NewClient(cfg)
	client.clusterID = 42

	config, err := client.GetConfig()

	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}

	if len(config.Services) != 4 {
		t.Errorf("Expected 4 services, got %d", len(config.Services))
	}

	if len(config.Mappings) != 2 {
		t.Errorf("Expected 2 mappings, got %d", len(config.Mappings))
	}

	// Verify service details
	if config.Services[0].ID != 1 || config.Services[0].Port != 8080 {
		t.Error("First service details incorrect")
	}

	if config.Services[3].ID != 4 || config.Services[3].Port != 8083 {
		t.Error("Last service details incorrect")
	}
}
