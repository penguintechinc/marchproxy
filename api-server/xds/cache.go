package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

// SnapshotCache wraps the go-control-plane snapshot cache
type SnapshotCache struct {
	cache.SnapshotCache
	mu      sync.RWMutex
	version int64
	debug   bool
}

// NewSnapshotCache creates a new snapshot cache
func NewSnapshotCache(debug bool) *SnapshotCache {
	return &SnapshotCache{
		SnapshotCache: cache.NewSnapshotCache(false, cache.IDHash{}, nil),
		version:       0,
		debug:         debug,
	}
}

// SetSnapshot sets a new snapshot for the given node
func (c *SnapshotCache) SetSnapshot(ctx context.Context, nodeID string, snapshot *cache.Snapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.version++

	if c.debug {
		fmt.Printf("Setting snapshot v%d for node %s\n", c.version, nodeID)
		fmt.Printf("  Listeners: %d\n", len(snapshot.GetResources(resource.ListenerType)))
		fmt.Printf("  Routes: %d\n", len(snapshot.GetResources(resource.RouteType)))
		fmt.Printf("  Clusters: %d\n", len(snapshot.GetResources(resource.ClusterType)))
		fmt.Printf("  Endpoints: %d\n", len(snapshot.GetResources(resource.EndpointType)))
	}

	// Validate snapshot consistency
	if err := snapshot.Consistent(); err != nil {
		return fmt.Errorf("snapshot inconsistency: %w", err)
	}

	// Set the snapshot in the cache
	return c.SnapshotCache.SetSnapshot(ctx, nodeID, snapshot)
}

// GetVersion returns the current snapshot version
func (c *SnapshotCache) GetVersion() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.version
}

// ClearSnapshot clears the snapshot for a given node
func (c *SnapshotCache) ClearSnapshot(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.debug {
		fmt.Printf("Clearing snapshot for node %s\n", nodeID)
	}

	c.SnapshotCache.ClearSnapshot(nodeID)
}

// GetSnapshot returns the current snapshot for a node
func (c *SnapshotCache) GetSnapshot(nodeID string) (*cache.Snapshot, error) {
	snap, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}

	// Type assertion to get concrete snapshot
	if snapshot, ok := snap.(*cache.Snapshot); ok {
		return snapshot, nil
	}

	return nil, fmt.Errorf("invalid snapshot type")
}

// GetResourceNames returns the names of all resources of a given type
func (c *SnapshotCache) GetResourceNames(nodeID string, typeURL string) ([]string, error) {
	snapshot, err := c.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}

	resources := snapshot.GetResources(typeURL)
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}

	return names, nil
}

// GetStats returns cache statistics
func (c *SnapshotCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"version": c.version,
		// Additional stats can be added here
	}
}

// CreateEmptySnapshot creates an empty snapshot with consistent version
func CreateEmptySnapshot(version string) (*cache.Snapshot, error) {
	return cache.NewSnapshot(
		version,
		map[resource.Type][]types.Resource{
			resource.EndpointType: {},
			resource.ClusterType:  {},
			resource.RouteType:    {},
			resource.ListenerType: {},
		},
	)
}
