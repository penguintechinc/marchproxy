package multicloud

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CostAnalyzer analyzes and optimizes cloud egress costs
type CostAnalyzer struct {
	mu sync.RWMutex

	backends []*Backend
	logger   *logrus.Logger

	// Cost tracking
	backendCosts map[string]*CostStats

	stopChan chan struct{}
	stopped  bool
}

// CostStats tracks cost statistics for a backend
type CostStats struct {
	TotalBytes   uint64
	TotalCost    float64
	CostPerGB    float64
	LastUpdated  time.Time
}

// NewCostAnalyzer creates a new cost analyzer
func NewCostAnalyzer(backends []*Backend, logger *logrus.Logger) *CostAnalyzer {
	ca := &CostAnalyzer{
		backends:     backends,
		logger:       logger,
		backendCosts: make(map[string]*CostStats),
		stopChan:     make(chan struct{}),
	}

	// Initialize cost stats
	for _, backend := range backends {
		ca.backendCosts[backend.Name] = &CostStats{
			CostPerGB:   backend.Cost,
			LastUpdated: time.Now(),
		}
	}

	return ca
}

// Start starts cost analysis
func (ca *CostAnalyzer) Start() {
	go ca.analysisLoop()
	ca.logger.Info("Cost analyzer started")
}

// Stop stops cost analysis
func (ca *CostAnalyzer) Stop() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if !ca.stopped {
		close(ca.stopChan)
		ca.stopped = true
		ca.logger.Info("Cost analyzer stopped")
	}
}

// analysisLoop runs periodic cost analysis
func (ca *CostAnalyzer) analysisLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ca.analyze()
		case <-ca.stopChan:
			return
		}
	}
}

// analyze performs cost analysis
func (ca *CostAnalyzer) analyze() {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	totalCost := 0.0
	for name, stats := range ca.backendCosts {
		totalCost += stats.TotalCost
		ca.logger.WithFields(logrus.Fields{
			"backend":    name,
			"total_gb":   float64(stats.TotalBytes) / 1e9,
			"total_cost": stats.TotalCost,
			"cost_per_gb": stats.CostPerGB,
		}).Debug("Backend cost analysis")
	}

	ca.logger.WithField("total_cost", totalCost).Info("Total egress cost")
}

// RecordTraffic records traffic for cost calculation
func (ca *CostAnalyzer) RecordTraffic(backendName string, bytes uint64) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	stats, ok := ca.backendCosts[backendName]
	if !ok {
		return
	}

	stats.TotalBytes += bytes
	stats.TotalCost = float64(stats.TotalBytes) / 1e9 * stats.CostPerGB
	stats.LastUpdated = time.Now()
}

// GetOptimalBackend returns the most cost-effective backend
func (ca *CostAnalyzer) GetOptimalBackend(backends []*Backend) *Backend {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var optimal *Backend
	lowestCost := float64(-1)

	for _, backend := range backends {
		if !backend.Healthy {
			continue
		}

		if lowestCost < 0 || backend.Cost < lowestCost {
			lowestCost = backend.Cost
			optimal = backend
		}
	}

	return optimal
}

// GetStats returns cost statistics
func (ca *CostAnalyzer) GetStats() map[string]interface{} {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	stats := make(map[string]interface{})
	totalCost := 0.0

	for name, costStats := range ca.backendCosts {
		stats[name] = map[string]interface{}{
			"total_gb":   float64(costStats.TotalBytes) / 1e9,
			"total_cost": costStats.TotalCost,
			"cost_per_gb": costStats.CostPerGB,
		}
		totalCost += costStats.TotalCost
	}

	stats["total_cost"] = totalCost
	return stats
}
