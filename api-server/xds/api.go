// Package xds provides HTTP API for xDS configuration updates
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// ConfigAPI provides HTTP endpoints for configuration updates
type ConfigAPI struct {
	cache           cache.SnapshotCache
	nodeID          string
	mu              sync.RWMutex
	version         int
	snapshotHistory map[int]*cache.Snapshot
	maxHistory      int
}

// NewConfigAPI creates a new configuration API
func NewConfigAPI(cache cache.SnapshotCache, nodeID string) *ConfigAPI {
	return &ConfigAPI{
		cache:           cache,
		nodeID:          nodeID,
		version:         1,
		snapshotHistory: make(map[int]*cache.Snapshot),
		maxHistory:      10, // Keep last 10 snapshots for rollback
	}
}

// UpdateConfigHandler handles configuration update requests from the API server
func (api *ConfigAPI) UpdateConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse configuration
	config, err := ParseConfig(body)
	if err != nil {
		log.Printf("Failed to parse configuration: %v", err)
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Generate new version
	api.mu.Lock()
	api.version++
	version := fmt.Sprintf("%d", api.version)
	config.Version = version
	api.mu.Unlock()

	// Generate snapshot
	snapshot, err := GenerateSnapshot(*config)
	if err != nil {
		log.Printf("Failed to generate snapshot: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	// Update cache
	if err := api.cache.SetSnapshot(context.Background(), api.nodeID, snapshot); err != nil {
		log.Printf("Failed to set snapshot: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusInternalServerError)
		return
	}

	// Store snapshot in history for rollback capability
	api.storeSnapshotInHistory(api.version, snapshot)

	log.Printf("Configuration updated to version %s", version)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"version": version,
		"message": "Configuration updated successfully",
	})
}

// storeSnapshotInHistory stores a snapshot for rollback capability
func (api *ConfigAPI) storeSnapshotInHistory(version int, snapshot *cache.Snapshot) {
	api.snapshotHistory[version] = snapshot

	// Remove oldest snapshots if we exceed maxHistory
	if len(api.snapshotHistory) > api.maxHistory {
		// Find oldest version
		oldestVersion := version
		for v := range api.snapshotHistory {
			if v < oldestVersion {
				oldestVersion = v
			}
		}
		delete(api.snapshotHistory, oldestVersion)
	}
}

// GetConfigHandler returns the current configuration version
func (api *ConfigAPI) GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	api.mu.RLock()
	version := api.version
	api.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version": version,
		"node_id": api.nodeID,
	})
}

// HealthHandler returns the health status of the xDS server
func (api *ConfigAPI) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "marchproxy-xds-server",
	})
}

// GetSnapshotHandler returns information about a specific snapshot version
func (api *ConfigAPI) GetSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse version from URL path (e.g., /v1/snapshot/5)
	var requestedVersion int
	if _, err := fmt.Sscanf(r.URL.Path, "/v1/snapshot/%d", &requestedVersion); err != nil {
		http.Error(w, "Invalid version in path", http.StatusBadRequest)
		return
	}

	api.mu.RLock()
	snapshot, exists := api.snapshotHistory[requestedVersion]
	currentVersion := api.version
	api.mu.RUnlock()

	if !exists {
		http.Error(w, "Snapshot version not found", http.StatusNotFound)
		return
	}

	// Return snapshot metadata
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":         requestedVersion,
		"current_version": currentVersion,
		"snapshot_version": snapshot.GetVersion(cache.ResponseType("type.googleapis.com/envoy.config.listener.v3.Listener")),
		"available":       true,
	})
}

// RollbackHandler rolls back to a previous snapshot version
func (api *ConfigAPI) RollbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse version from URL path (e.g., /v1/rollback/5)
	var targetVersion int
	if _, err := fmt.Sscanf(r.URL.Path, "/v1/rollback/%d", &targetVersion); err != nil {
		http.Error(w, "Invalid version in path", http.StatusBadRequest)
		return
	}

	api.mu.Lock()
	snapshot, exists := api.snapshotHistory[targetVersion]
	if !exists {
		api.mu.Unlock()
		http.Error(w, "Target version not found in history", http.StatusNotFound)
		return
	}

	// Create new version for the rollback
	api.version++
	newVersion := api.version
	api.mu.Unlock()

	// Apply the snapshot from history
	if err := api.cache.SetSnapshot(context.Background(), api.nodeID, snapshot); err != nil {
		log.Printf("Failed to rollback to version %d: %v", targetVersion, err)
		http.Error(w, fmt.Sprintf("Failed to rollback: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Rolled back to version %d (new version: %d)", targetVersion, newVersion)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"rolled_back_to":  targetVersion,
		"new_version":     newVersion,
		"message":         fmt.Sprintf("Successfully rolled back to version %d", targetVersion),
	})
}

// StartHTTPAPI starts the HTTP API server for configuration updates
func StartHTTPAPI(api *ConfigAPI, port uint) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/config", api.UpdateConfigHandler)
	mux.HandleFunc("/v1/version", api.GetConfigHandler)
	mux.HandleFunc("/v1/snapshot/", api.GetSnapshotHandler)
	mux.HandleFunc("/v1/rollback/", api.RollbackHandler)
	mux.HandleFunc("/healthz", api.HealthHandler)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Failed to start HTTP API: %v", err)
		}
	}()
}
