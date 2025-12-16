package multicloud

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthMonitor monitors backend health
type HealthMonitor struct {
	mu sync.RWMutex

	backends []*Backend
	interval time.Duration
	timeout  time.Duration
	logger   *logrus.Logger

	stopChan chan struct{}
	stopped  bool
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(backends []*Backend, logger *logrus.Logger) *HealthMonitor {
	return &HealthMonitor{
		backends: backends,
		interval: 30 * time.Second,
		timeout:  5 * time.Second,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start starts health monitoring
func (hm *HealthMonitor) Start() error {
	hm.mu.Lock()
	if hm.stopped {
		hm.mu.Unlock()
		return fmt.Errorf("health monitor already stopped")
	}
	hm.mu.Unlock()

	// Initial health check
	hm.checkAll()

	// Start periodic checks
	go hm.monitorLoop()

	hm.logger.WithField("interval", hm.interval).Info("Health monitor started")
	return nil
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.stopped {
		close(hm.stopChan)
		hm.stopped = true
		hm.logger.Info("Health monitor stopped")
	}
}

// monitorLoop runs the health check loop
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.checkAll()
		case <-hm.stopChan:
			return
		}
	}
}

// checkAll checks health of all backends
func (hm *HealthMonitor) checkAll() {
	var wg sync.WaitGroup

	for _, backend := range hm.backends {
		wg.Add(1)
		go func(b *Backend) {
			defer wg.Done()
			hm.checkBackend(b)
		}(backend)
	}

	wg.Wait()
}

// checkBackend checks health of a single backend
func (hm *HealthMonitor) checkBackend(backend *Backend) {
	start := time.Now()

	// Try TCP health check first
	healthy, latency := hm.tcpHealthCheck(backend)

	if !healthy {
		// Try HTTP health check
		healthy, latency = hm.httpHealthCheck(backend)
	}

	backend.Healthy = healthy
	backend.Latency = latency

	hm.logger.WithFields(logrus.Fields{
		"backend": backend.Name,
		"healthy": healthy,
		"latency": latency,
		"took":    time.Since(start),
	}).Debug("Health check completed")
}

// tcpHealthCheck performs a TCP health check
func (hm *HealthMonitor) tcpHealthCheck(backend *Backend) (bool, int64) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
	defer cancel()

	// Parse URL to get host and port
	host := backend.URL
	port := "80"

	// Simple TCP connection attempt
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return false, 0
	}
	defer conn.Close()

	latency := time.Since(start).Microseconds()
	return true, latency
}

// httpHealthCheck performs an HTTP health check
func (hm *HealthMonitor) httpHealthCheck(backend *Backend) (bool, int64) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
	defer cancel()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: hm.timeout,
	}

	// Health check URL
	healthURL := fmt.Sprintf("http://%s/healthz", backend.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false, 0
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()

	latency := time.Since(start).Microseconds()
	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300

	return healthy, latency
}

// SetInterval sets the health check interval
func (hm *HealthMonitor) SetInterval(interval time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.interval = interval
}

// SetTimeout sets the health check timeout
func (hm *HealthMonitor) SetTimeout(timeout time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.timeout = timeout
}
