package nlb

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	blueGreenSplits = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nlb_bluegreen_traffic_split",
			Help: "Current traffic split percentage for blue/green deployments",
		},
		[]string{"protocol", "version", "color"},
	)

	blueGreenDeployments = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nlb_bluegreen_deployments_total",
			Help: "Total number of blue/green deployments",
		},
		[]string{"protocol", "status"},
	)

	blueGreenRollbacks = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nlb_bluegreen_rollbacks_total",
			Help: "Total number of blue/green rollbacks",
		},
	)
)

// DeploymentColor represents blue or green deployment
type DeploymentColor string

const (
	DeploymentBlue  DeploymentColor = "blue"
	DeploymentGreen DeploymentColor = "green"
)

// DeploymentState represents the state of a blue/green deployment
type DeploymentState struct {
	Protocol       Protocol
	BlueVersion    string
	GreenVersion   string
	ActiveColor    DeploymentColor
	BlueWeight     int // 0-100 percentage
	GreenWeight    int // 0-100 percentage
	StartTime      time.Time
	LastUpdate     time.Time
	Status         string // "stable", "transitioning", "canary", "rollback"
	TargetBlue     int    // Target weight for gradual rollout
	TargetGreen    int
	StepSize       int           // Weight increment per step
	StepDuration   time.Duration // Duration between steps
}

// BlueGreenController manages blue/green deployments
type BlueGreenController struct {
	deployments map[Protocol]*DeploymentState
	router      *Router
	mu          sync.RWMutex
	logger      *logrus.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewBlueGreenController creates a new blue/green controller
func NewBlueGreenController(router *Router, logger *logrus.Logger) *BlueGreenController {
	ctx, cancel := context.WithCancel(context.Background())

	return &BlueGreenController{
		deployments: make(map[Protocol]*DeploymentState),
		router:      router,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// InitializeDeployment initializes a blue/green deployment for a protocol
func (bgc *BlueGreenController) InitializeDeployment(protocol Protocol, version string, color DeploymentColor) error {
	if protocol == ProtocolUnknown {
		return errors.New("invalid protocol")
	}

	bgc.mu.Lock()
	defer bgc.mu.Unlock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		deployment = &DeploymentState{
			Protocol:    protocol,
			ActiveColor: DeploymentBlue,
			BlueWeight:  100,
			GreenWeight: 0,
			Status:      "stable",
			StartTime:   time.Now(),
			LastUpdate:  time.Now(),
		}
		bgc.deployments[protocol] = deployment
	}

	if color == DeploymentBlue {
		deployment.BlueVersion = version
	} else {
		deployment.GreenVersion = version
	}

	blueGreenSplits.WithLabelValues(protocol.String(), deployment.BlueVersion, "blue").Set(float64(deployment.BlueWeight))
	blueGreenSplits.WithLabelValues(protocol.String(), deployment.GreenVersion, "green").Set(float64(deployment.GreenWeight))

	bgc.logger.WithFields(logrus.Fields{
		"protocol": protocol.String(),
		"version":  version,
		"color":    color,
	}).Info("Deployment initialized")

	return nil
}

// StartCanaryDeployment starts a canary deployment with gradual traffic shift
func (bgc *BlueGreenController) StartCanaryDeployment(protocol Protocol, newVersion string, targetColor DeploymentColor, stepSize int, stepDuration time.Duration) error {
	bgc.mu.Lock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		bgc.mu.Unlock()
		return errors.New("deployment not initialized")
	}

	if deployment.Status == "transitioning" {
		bgc.mu.Unlock()
		return errors.New("deployment already in progress")
	}

	// Set new version
	if targetColor == DeploymentBlue {
		deployment.BlueVersion = newVersion
		deployment.TargetBlue = 100
		deployment.TargetGreen = 0
	} else {
		deployment.GreenVersion = newVersion
		deployment.TargetBlue = 0
		deployment.TargetGreen = 100
	}

	deployment.Status = "canary"
	deployment.StepSize = stepSize
	deployment.StepDuration = stepDuration
	deployment.LastUpdate = time.Now()

	bgc.mu.Unlock()

	blueGreenDeployments.WithLabelValues(protocol.String(), "started").Inc()

	bgc.logger.WithFields(logrus.Fields{
		"protocol":      protocol.String(),
		"new_version":   newVersion,
		"target_color":  targetColor,
		"step_size":     stepSize,
		"step_duration": stepDuration,
	}).Info("Canary deployment started")

	// Start gradual rollout
	bgc.wg.Add(1)
	go bgc.gradualRollout(protocol)

	return nil
}

// gradualRollout performs gradual traffic shift
func (bgc *BlueGreenController) gradualRollout(protocol Protocol) {
	defer bgc.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bgc.ctx.Done():
			return
		case <-ticker.C:
			done, err := bgc.stepRollout(protocol)
			if err != nil {
				bgc.logger.WithError(err).Error("Rollout step failed")
				bgc.Rollback(protocol)
				return
			}
			if done {
				bgc.logger.WithField("protocol", protocol.String()).Info("Gradual rollout completed")
				return
			}
		}
	}
}

// stepRollout performs one step of the rollout
func (bgc *BlueGreenController) stepRollout(protocol Protocol) (bool, error) {
	bgc.mu.Lock()
	defer bgc.mu.Unlock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		return true, errors.New("deployment not found")
	}

	if deployment.Status != "canary" {
		return true, nil
	}

	// Check if it's time for next step
	if time.Since(deployment.LastUpdate) < deployment.StepDuration {
		return false, nil
	}

	// Calculate new weights
	if deployment.TargetBlue > deployment.BlueWeight {
		deployment.BlueWeight = min(deployment.BlueWeight+deployment.StepSize, deployment.TargetBlue)
		deployment.GreenWeight = 100 - deployment.BlueWeight
	} else if deployment.TargetGreen > deployment.GreenWeight {
		deployment.GreenWeight = min(deployment.GreenWeight+deployment.StepSize, deployment.TargetGreen)
		deployment.BlueWeight = 100 - deployment.GreenWeight
	}

	deployment.LastUpdate = time.Now()

	// Update metrics
	blueGreenSplits.WithLabelValues(protocol.String(), deployment.BlueVersion, "blue").Set(float64(deployment.BlueWeight))
	blueGreenSplits.WithLabelValues(protocol.String(), deployment.GreenVersion, "green").Set(float64(deployment.GreenWeight))

	bgc.logger.WithFields(logrus.Fields{
		"protocol":     protocol.String(),
		"blue_weight":  deployment.BlueWeight,
		"green_weight": deployment.GreenWeight,
	}).Debug("Rollout step completed")

	// Check if complete
	if deployment.BlueWeight == deployment.TargetBlue && deployment.GreenWeight == deployment.TargetGreen {
		deployment.Status = "stable"
		if deployment.BlueWeight == 100 {
			deployment.ActiveColor = DeploymentBlue
		} else {
			deployment.ActiveColor = DeploymentGreen
		}
		blueGreenDeployments.WithLabelValues(protocol.String(), "completed").Inc()
		return true, nil
	}

	return false, nil
}

// InstantSwitch performs an instant blue/green switch
func (bgc *BlueGreenController) InstantSwitch(protocol Protocol, targetColor DeploymentColor) error {
	bgc.mu.Lock()
	defer bgc.mu.Unlock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		return errors.New("deployment not initialized")
	}

	if targetColor == DeploymentBlue {
		deployment.BlueWeight = 100
		deployment.GreenWeight = 0
		deployment.ActiveColor = DeploymentBlue
	} else {
		deployment.BlueWeight = 0
		deployment.GreenWeight = 100
		deployment.ActiveColor = DeploymentGreen
	}

	deployment.Status = "stable"
	deployment.LastUpdate = time.Now()

	blueGreenSplits.WithLabelValues(protocol.String(), deployment.BlueVersion, "blue").Set(float64(deployment.BlueWeight))
	blueGreenSplits.WithLabelValues(protocol.String(), deployment.GreenVersion, "green").Set(float64(deployment.GreenWeight))

	blueGreenDeployments.WithLabelValues(protocol.String(), "instant_switch").Inc()

	bgc.logger.WithFields(logrus.Fields{
		"protocol":     protocol.String(),
		"target_color": targetColor,
	}).Info("Instant switch completed")

	return nil
}

// Rollback rolls back to the previous deployment
func (bgc *BlueGreenController) Rollback(protocol Protocol) error {
	bgc.mu.Lock()
	defer bgc.mu.Unlock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		return errors.New("deployment not found")
	}

	// Switch back to previous active color
	if deployment.ActiveColor == DeploymentBlue {
		deployment.BlueWeight = 0
		deployment.GreenWeight = 100
		deployment.ActiveColor = DeploymentGreen
	} else {
		deployment.BlueWeight = 100
		deployment.GreenWeight = 0
		deployment.ActiveColor = DeploymentBlue
	}

	deployment.Status = "rollback"
	deployment.LastUpdate = time.Now()

	blueGreenSplits.WithLabelValues(protocol.String(), deployment.BlueVersion, "blue").Set(float64(deployment.BlueWeight))
	blueGreenSplits.WithLabelValues(protocol.String(), deployment.GreenVersion, "green").Set(float64(deployment.GreenWeight))

	blueGreenRollbacks.Inc()

	bgc.logger.WithFields(logrus.Fields{
		"protocol":     protocol.String(),
		"active_color": deployment.ActiveColor,
	}).Warn("Deployment rolled back")

	return nil
}

// GetDeploymentState returns the current deployment state
func (bgc *BlueGreenController) GetDeploymentState(protocol Protocol) (*DeploymentState, error) {
	bgc.mu.RLock()
	defer bgc.mu.RUnlock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		return nil, errors.New("deployment not found")
	}

	// Return a copy
	stateCopy := *deployment
	return &stateCopy, nil
}

// ShouldRouteToColor determines which color to route to based on weights
func (bgc *BlueGreenController) ShouldRouteToColor(protocol Protocol, randomValue int) (DeploymentColor, error) {
	bgc.mu.RLock()
	defer bgc.mu.RUnlock()

	deployment, exists := bgc.deployments[protocol]
	if !exists {
		return DeploymentBlue, errors.New("deployment not found")
	}

	// randomValue should be 0-99
	if randomValue < deployment.BlueWeight {
		return DeploymentBlue, nil
	}
	return DeploymentGreen, nil
}

// Stop stops the blue/green controller
func (bgc *BlueGreenController) Stop() {
	bgc.cancel()
	bgc.wg.Wait()
	bgc.logger.Info("Blue/Green controller stopped")
}

// GetStats returns blue/green deployment statistics
func (bgc *BlueGreenController) GetStats() map[string]interface{} {
	bgc.mu.RLock()
	defer bgc.mu.RUnlock()

	stats := make(map[string]interface{})
	deploymentStats := make(map[string]interface{})

	for protocol, deployment := range bgc.deployments {
		deploymentStats[protocol.String()] = map[string]interface{}{
			"blue_version":  deployment.BlueVersion,
			"green_version": deployment.GreenVersion,
			"active_color":  deployment.ActiveColor,
			"blue_weight":   deployment.BlueWeight,
			"green_weight":  deployment.GreenWeight,
			"status":        deployment.Status,
			"start_time":    deployment.StartTime,
			"last_update":   deployment.LastUpdate,
		}
	}

	stats["deployments"] = deploymentStats
	stats["total_deployments"] = len(bgc.deployments)

	return stats
}
