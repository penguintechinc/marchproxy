// Package xds provides the complete xDS control plane server
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// Server represents the xDS control plane server with complete state management
type Server struct {
	cache      *SnapshotCache
	callbacks  *Callbacks
	configAPI  *ConfigAPI
	mu         sync.RWMutex
	nodes      map[string]*NodeInfo
	shutdown   chan struct{}
}

// NodeInfo tracks information about connected Envoy nodes
type NodeInfo struct {
	NodeID          string
	LastSeen        int64
	Version         string
	ClusterID       int
	EnvoyVersion    string
	ConnectionCount int
}

// NewServer creates a new xDS control plane server
func NewServer(debug bool, nodeID string) *Server {
	cache := NewSnapshotCache(debug)

	callbacks := &Callbacks{
		Signal:   make(chan struct{}),
		Fetches:  0,
		Requests: 0,
		Debug:    debug,
	}

	configAPI := NewConfigAPI(cache, nodeID)

	return &Server{
		cache:     cache,
		callbacks: callbacks,
		configAPI: configAPI,
		nodes:     make(map[string]*NodeInfo),
		shutdown:  make(chan struct{}),
	}
}

// GetCache returns the snapshot cache
func (s *Server) GetCache() *SnapshotCache {
	return s.cache
}

// GetCallbacks returns the server callbacks
func (s *Server) GetCallbacks() *Callbacks {
	return s.callbacks
}

// GetConfigAPI returns the config API
func (s *Server) GetConfigAPI() *ConfigAPI {
	return s.configAPI
}

// RegisterNode registers a new Envoy node
func (s *Server) RegisterNode(nodeID string, clusterID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[nodeID]; !exists {
		s.nodes[nodeID] = &NodeInfo{
			NodeID:    nodeID,
			ClusterID: clusterID,
		}
		log.Printf("Registered new node: %s (cluster: %d)", nodeID, clusterID)
	}
}

// UnregisterNode removes a node from tracking
func (s *Server) UnregisterNode(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node, exists := s.nodes[nodeID]; exists {
		delete(s.nodes, nodeID)
		log.Printf("Unregistered node: %s", node.NodeID)
	}
}

// GetNodeInfo returns information about a specific node
func (s *Server) GetNodeInfo(nodeID string) (*NodeInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.nodes[nodeID]
	return info, exists
}

// GetAllNodes returns all registered nodes
func (s *Server) GetAllNodes() map[string]*NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	nodesCopy := make(map[string]*NodeInfo)
	for k, v := range s.nodes {
		nodesCopy[k] = v
	}

	return nodesCopy
}

// UpdateNodeSnapshot updates the snapshot for a specific node
func (s *Server) UpdateNodeSnapshot(nodeID string, snapshot cache.ResourceSnapshot) error {
	concreteSnapshot, ok := snapshot.(*cache.Snapshot)
	if !ok {
		return fmt.Errorf("invalid snapshot type")
	}

	if err := s.cache.SetSnapshot(context.Background(), nodeID, concreteSnapshot); err != nil {
		return fmt.Errorf("failed to set snapshot for node %s: %w", nodeID, err)
	}

	log.Printf("Updated snapshot for node: %s", nodeID)
	return nil
}

// ClearNodeSnapshot clears the snapshot for a specific node
func (s *Server) ClearNodeSnapshot(nodeID string) {
	s.cache.ClearSnapshot(nodeID)
	log.Printf("Cleared snapshot for node: %s", nodeID)
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	s.mu.RLock()
	nodeCount := len(s.nodes)
	s.mu.RUnlock()

	return map[string]interface{}{
		"node_count":    nodeCount,
		"cache_version": s.cache.GetVersion(),
		"requests":      s.callbacks.GetRequestCount(),
		"fetches":       s.callbacks.GetFetchCount(),
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	close(s.shutdown)
	log.Println("xDS server shutting down...")
}

// WaitForShutdown blocks until shutdown is initiated
func (s *Server) WaitForShutdown() {
	<-s.shutdown
}
