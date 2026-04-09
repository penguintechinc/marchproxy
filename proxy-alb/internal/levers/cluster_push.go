// Package levers implements the proxy-alb levers API for receiving cluster
// configuration pushes from the hub-policy controller.
package levers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/PenguinTech/MarchProxy/proxy-alb/internal/logging"
)

// ClusterDef defines an upstream cluster with load balancing policy.
type ClusterDef struct {
	Name      string   `json:"name"`
	Endpoints []string `json:"endpoints"`
	LBPolicy  string   `json:"lb_policy"` // ROUND_ROBIN, LEAST_REQUEST, WEIGHTED
}

// ClusterRegistry holds the current cluster definitions pushed by the controller.
type ClusterRegistry struct {
	mu       sync.RWMutex
	clusters map[string]ClusterDef
	logger   *logging.LogrusAdapter
}

func NewClusterRegistry() (*ClusterRegistry, error) {
	logger, err := logging.NewLogrusAdapter("levers")
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	return &ClusterRegistry{
		clusters: make(map[string]ClusterDef),
		logger:   logger,
	}, nil
}

// HandlePush is an HTTP handler for POST /api/v1/levers/clusters
func (cr *ClusterRegistry) HandlePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var clusters []ClusterDef
	if err := json.NewDecoder(r.Body).Decode(&clusters); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	cr.mu.Lock()
	for _, c := range clusters {
		cr.clusters[c.Name] = c
	}
	cr.mu.Unlock()
	cr.logger.WithFields(map[string]interface{}{
		"count": len(clusters),
	}).Info("alb levers: cluster definitions updated")
	// TODO: trigger Envoy xDS cluster config reload
	w.WriteHeader(http.StatusOK)
}

// Get returns a cluster definition by name. Returns false if not found.
func (cr *ClusterRegistry) Get(name string) (ClusterDef, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	c, ok := cr.clusters[name]
	return c, ok
}

// List returns all current cluster definitions.
func (cr *ClusterRegistry) List() []ClusterDef {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	out := make([]ClusterDef, 0, len(cr.clusters))
	for _, c := range cr.clusters {
		out = append(out, c)
	}
	return out
}
