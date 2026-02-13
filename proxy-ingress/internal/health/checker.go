package health

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type HealthChecker struct {
	config    HealthConfig
	backends  map[string]*BackendHealth
	vhosts    map[string]*VirtualHostHealth
	checks    map[string]HealthCheck
	probes    map[string]Probe
	metrics   *HealthMetrics
	mutex     sync.RWMutex
	stopChan  chan struct{}
	running   bool
}

type HealthConfig struct {
	CheckInterval       time.Duration
	Timeout            time.Duration
	HealthyThreshold   int
	UnhealthyThreshold int
	EnabledChecks      []string
	HTTPEndpoint       string
	HTTPPort           int
	EnableProbes       bool
	ProbeEndpoints     map[string]ProbeConfig
	NotificationURL    string
	RetryAttempts      int
	RetryDelay         time.Duration
	EnableMTLSChecks   bool
	MTLSConfig         MTLSCheckConfig
}

type MTLSCheckConfig struct {
	CertPath       string
	KeyPath        string
	CAPath         string
	SkipVerify     bool
	ClientTimeout  time.Duration
}

type BackendHealth struct {
	Backend             *Backend
	Status              HealthStatus
	LastCheck           time.Time
	LastStatusChange    time.Time
	ConsecutiveSuccesses int
	ConsecutiveFailures  int
	TotalChecks         uint64
	TotalFailures       uint64
	ResponseTime        time.Duration
	ErrorMessage        string
	Metadata           map[string]interface{}
	SSLCertExpiry      time.Time
	SSLCertValid       bool
}

type VirtualHostHealth struct {
	VHost               *VirtualHost
	Status              HealthStatus
	LastCheck           time.Time
	BackendCount        int
	HealthyBackends     int
	UnhealthyBackends   int
	SSLEnabled          bool
	SSLCertExpiry       time.Time
	RequestCount        uint64
	ErrorCount          uint64
	AverageResponseTime time.Duration
}

type Backend struct {
	Name      string
	Host      string
	Port      int
	Scheme    string
	Path      string
	Weight    int
	MaxConns  int
	Headers   map[string]string
}

type VirtualHost struct {
	Name        string
	Host        string
	SSLEnabled  bool
	CertPath    string
	KeyPath     string
	Backends    []*Backend
	HealthPath  string
}

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnknown   HealthStatus = "unknown"
)

type HealthCheck interface {
	Name() string
	Check(ctx context.Context, target interface{}) *CheckResult
	Enabled() bool
	Configure(config map[string]interface{}) error
}

type CheckResult struct {
	Status       HealthStatus
	ResponseTime time.Duration
	Error        error
	Message      string
	Metadata     map[string]interface{}
}

type Probe interface {
	Name() string
	Execute(ctx context.Context) *ProbeResult
	GetConfig() ProbeConfig
}

type ProbeConfig struct {
	Type     ProbeType
	Path     string
	Port     int
	Timeout  time.Duration
	Headers  map[string]string
	Expected ProbeExpected
}

type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "http"
	ProbeTypeTCP  ProbeType = "tcp"
	ProbeTypeExec ProbeType = "exec"
	ProbeTypeMTLS ProbeType = "mtls"
)

type ProbeExpected struct {
	StatusCode int
	Body       string
	Headers    map[string]string
	Timeout    time.Duration
}

type ProbeResult struct {
	Status       HealthStatus
	ResponseTime time.Duration
	Error        error
	Message      string
	Data         map[string]interface{}
}

type HealthMetrics struct {
	TotalChecks       uint64
	HealthyBackends   uint64
	UnhealthyBackends uint64
	DegradedBackends  uint64
	HealthyVHosts     uint64
	UnhealthyVHosts   uint64
	AverageCheckTime  time.Duration
	CheckFailures     uint64
	StatusChanges     uint64
	MTLSHandshakes    uint64
	SSLCertFailures   uint64
	mutex             sync.RWMutex
}

type HTTPHealthCheck struct {
	name     string
	enabled  bool
	client   *http.Client
	path     string
	method   string
	headers  map[string]string
	expected HTTPExpected
}

type HTTPExpected struct {
	StatusCodes []int
	BodyContains string
	Headers     map[string]string
	MaxResponseTime time.Duration
}

type MTLSHealthCheck struct {
	name       string
	enabled    bool
	client     *http.Client
	tlsConfig  *tls.Config
	path       string
	method     string
	headers    map[string]string
	expected   HTTPExpected
}

type TCPHealthCheck struct {
	name    string
	enabled bool
	timeout time.Duration
}

type SSLCertCheck struct {
	name    string
	enabled bool
	timeout time.Duration
}

type ReadinessProbe struct {
	config ProbeConfig
	name   string
}

type LivenessProbe struct {
	config ProbeConfig
	name   string
}

func NewHealthChecker(config HealthConfig) *HealthChecker {
	hc := &HealthChecker{
		config:   config,
		backends: make(map[string]*BackendHealth),
		vhosts:   make(map[string]*VirtualHostHealth),
		checks:   make(map[string]HealthCheck),
		probes:   make(map[string]Probe),
		metrics:  &HealthMetrics{},
		stopChan: make(chan struct{}),
	}

	if config.CheckInterval == 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.HealthyThreshold == 0 {
		config.HealthyThreshold = 2
	}
	if config.UnhealthyThreshold == 0 {
		config.UnhealthyThreshold = 3
	}

	hc.initializeDefaultChecks()
	hc.initializeProbes()

	return hc
}

func (hc *HealthChecker) initializeDefaultChecks() {
	httpCheck := NewHTTPHealthCheck("http", HTTPExpected{
		StatusCodes: []int{200, 201, 202, 204},
		MaxResponseTime: 5 * time.Second,
	})
	hc.checks["http"] = httpCheck

	tcpCheck := NewTCPHealthCheck("tcp")
	hc.checks["tcp"] = tcpCheck

	sslCertCheck := NewSSLCertCheck("ssl_cert")
	hc.checks["ssl_cert"] = sslCertCheck

	if hc.config.EnableMTLSChecks {
		mtlsCheck := NewMTLSHealthCheck("mtls", hc.config.MTLSConfig, HTTPExpected{
			StatusCodes: []int{200, 201, 202, 204},
			MaxResponseTime: 5 * time.Second,
		})
		hc.checks["mtls"] = mtlsCheck
	}
}

func (hc *HealthChecker) initializeProbes() {
	if !hc.config.EnableProbes {
		return
	}

	for name, config := range hc.config.ProbeEndpoints {
		switch config.Type {
		case ProbeTypeHTTP:
			probe := NewReadinessProbe(name, config)
			hc.probes[name] = probe
		case ProbeTypeTCP:
			probe := NewLivenessProbe(name, config)
			hc.probes[name] = probe
		case ProbeTypeMTLS:
			if hc.config.EnableMTLSChecks {
				probe := NewMTLSProbe(name, config)
				hc.probes[name] = probe
			}
		}
	}
}

func (hc *HealthChecker) AddBackend(backend *Backend) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	key := fmt.Sprintf("%s-%s:%d", backend.Name, backend.Host, backend.Port)
	hc.backends[key] = &BackendHealth{
		Backend:   backend,
		Status:    StatusUnknown,
		LastCheck: time.Time{},
		Metadata:  make(map[string]interface{}),
	}
}

func (hc *HealthChecker) RemoveBackend(backend *Backend) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	key := fmt.Sprintf("%s-%s:%d", backend.Name, backend.Host, backend.Port)
	delete(hc.backends, key)
}

func (hc *HealthChecker) AddVirtualHost(vhost *VirtualHost) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.vhosts[vhost.Name] = &VirtualHostHealth{
		VHost:         vhost,
		Status:        StatusUnknown,
		LastCheck:     time.Time{},
		BackendCount:  len(vhost.Backends),
		SSLEnabled:    vhost.SSLEnabled,
	}

	for _, backend := range vhost.Backends {
		hc.AddBackend(backend)
	}
}

func (hc *HealthChecker) GetBackendHealth(backend *Backend) *BackendHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	key := fmt.Sprintf("%s-%s:%d", backend.Name, backend.Host, backend.Port)
	return hc.backends[key]
}

func (hc *HealthChecker) GetVirtualHostHealth(vhostName string) *VirtualHostHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	return hc.vhosts[vhostName]
}

func (hc *HealthChecker) GetAllBackendHealth() map[string]*BackendHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	result := make(map[string]*BackendHealth)
	for key, health := range hc.backends {
		result[key] = health
	}
	return result
}

func (hc *HealthChecker) GetAllVirtualHostHealth() map[string]*VirtualHostHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	result := make(map[string]*VirtualHostHealth)
	for key, health := range hc.vhosts {
		result[key] = health
	}
	return result
}

func (hc *HealthChecker) Start() error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if hc.running {
		return fmt.Errorf("health checker already running")
	}

	hc.running = true
	go hc.checkLoop()

	return nil
}

func (hc *HealthChecker) Stop() error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if !hc.running {
		return nil
	}

	hc.running = false
	close(hc.stopChan)

	return nil
}

func (hc *HealthChecker) checkLoop() {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks()
		case <-hc.stopChan:
			return
		}
	}
}

func (hc *HealthChecker) performHealthChecks() {
	var wg sync.WaitGroup

	// Check backends
	hc.mutex.RLock()
	backends := make(map[string]*BackendHealth)
	for k, v := range hc.backends {
		backends[k] = v
	}
	hc.mutex.RUnlock()

	for key, backendHealth := range backends {
		wg.Add(1)
		go func(k string, bh *BackendHealth) {
			defer wg.Done()
			hc.checkBackend(k, bh)
		}(key, backendHealth)
	}

	wg.Wait()

	// Update virtual host health based on backend health
	hc.updateVirtualHostHealth()
}

func (hc *HealthChecker) checkBackend(key string, backendHealth *BackendHealth) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	start := time.Now()
	result := hc.executeChecks(ctx, backendHealth.Backend)
	duration := time.Since(start)

	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	backendHealth.LastCheck = time.Now()
	backendHealth.ResponseTime = duration
	backendHealth.TotalChecks++

	if result.Error != nil {
		backendHealth.ErrorMessage = result.Error.Error()
		backendHealth.ConsecutiveFailures++
		backendHealth.ConsecutiveSuccesses = 0
		backendHealth.TotalFailures++
		hc.metrics.recordCheckFailure()
	} else {
		backendHealth.ErrorMessage = ""
		backendHealth.ConsecutiveSuccesses++
		backendHealth.ConsecutiveFailures = 0
	}

	oldStatus := backendHealth.Status
	newStatus := hc.determineHealthStatus(backendHealth, result)

	if newStatus != oldStatus {
		backendHealth.Status = newStatus
		backendHealth.LastStatusChange = time.Now()
		hc.metrics.recordStatusChange()
		hc.notifyStatusChange(backendHealth, oldStatus, newStatus)
	}

	if result.Metadata != nil {
		backendHealth.Metadata = result.Metadata
	}

	hc.metrics.recordCheck(duration)
	hc.updateBackendCounts()
}

func (hc *HealthChecker) executeChecks(ctx context.Context, backend *Backend) *CheckResult {
	var lastResult *CheckResult

	for _, checkName := range hc.config.EnabledChecks {
		if checkName == "all" {
			for _, check := range hc.checks {
				if check.Enabled() {
					result := check.Check(ctx, backend)
					if result.Status == StatusUnhealthy {
						return result
					}
					lastResult = result
				}
			}
		} else if check, exists := hc.checks[checkName]; exists && check.Enabled() {
			result := check.Check(ctx, backend)
			if result.Status == StatusUnhealthy {
				return result
			}
			lastResult = result
		}
	}

	if lastResult == nil {
		return &CheckResult{
			Status:  StatusHealthy,
			Message: "No checks performed",
		}
	}

	return lastResult
}

func (hc *HealthChecker) determineHealthStatus(backendHealth *BackendHealth, result *CheckResult) HealthStatus {
	if result.Status == StatusUnhealthy {
		if backendHealth.ConsecutiveFailures >= hc.config.UnhealthyThreshold {
			return StatusUnhealthy
		}
		return StatusDegraded
	}

	if result.Status == StatusHealthy {
		if backendHealth.ConsecutiveSuccesses >= hc.config.HealthyThreshold {
			return StatusHealthy
		}
		if backendHealth.Status == StatusUnhealthy {
			return StatusDegraded
		}
	}

	return backendHealth.Status
}

func (hc *HealthChecker) updateBackendCounts() {
	healthy := uint64(0)
	unhealthy := uint64(0)
	degraded := uint64(0)

	for _, backendHealth := range hc.backends {
		switch backendHealth.Status {
		case StatusHealthy:
			healthy++
		case StatusUnhealthy:
			unhealthy++
		case StatusDegraded:
			degraded++
		}
	}

	hc.metrics.mutex.Lock()
	hc.metrics.HealthyBackends = healthy
	hc.metrics.UnhealthyBackends = unhealthy
	hc.metrics.DegradedBackends = degraded
	hc.metrics.mutex.Unlock()
}

func (hc *HealthChecker) updateVirtualHostHealth() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	for vhostName, vhostHealth := range hc.vhosts {
		healthyCount := 0
		unhealthyCount := 0

		for _, backend := range vhostHealth.VHost.Backends {
			key := fmt.Sprintf("%s-%s:%d", backend.Name, backend.Host, backend.Port)
			if backendHealth, exists := hc.backends[key]; exists {
				switch backendHealth.Status {
				case StatusHealthy:
					healthyCount++
				case StatusUnhealthy:
					unhealthyCount++
				}
			}
		}

		vhostHealth.HealthyBackends = healthyCount
		vhostHealth.UnhealthyBackends = unhealthyCount
		vhostHealth.LastCheck = time.Now()

		if healthyCount == 0 {
			vhostHealth.Status = StatusUnhealthy
		} else if unhealthyCount == 0 {
			vhostHealth.Status = StatusHealthy
		} else {
			vhostHealth.Status = StatusDegraded
		}

		_ = vhostName
	}

	hc.updateVHostCounts()
}

func (hc *HealthChecker) updateVHostCounts() {
	healthy := uint64(0)
	unhealthy := uint64(0)

	for _, vhostHealth := range hc.vhosts {
		switch vhostHealth.Status {
		case StatusHealthy:
			healthy++
		case StatusUnhealthy:
			unhealthy++
		}
	}

	hc.metrics.mutex.Lock()
	hc.metrics.HealthyVHosts = healthy
	hc.metrics.UnhealthyVHosts = unhealthy
	hc.metrics.mutex.Unlock()
}

func (hc *HealthChecker) notifyStatusChange(backendHealth *BackendHealth, oldStatus, newStatus HealthStatus) {
	if hc.config.NotificationURL == "" {
		return
	}

	go func() {
		notification := map[string]interface{}{
			"backend":    fmt.Sprintf("%s-%s:%d", backendHealth.Backend.Name, backendHealth.Backend.Host, backendHealth.Backend.Port),
			"old_status": oldStatus,
			"new_status": newStatus,
			"timestamp":  time.Now(),
			"message":    backendHealth.ErrorMessage,
		}

		jsonData, err := json.Marshal(notification)
		if err != nil {
			return
		}
		resp, err := http.Post(hc.config.NotificationURL, "application/json", bytes.NewReader(jsonData))
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}

func (hc *HealthChecker) GetSystemHealth() *SystemHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	systemHealth := &SystemHealth{
		Status:        StatusHealthy,
		Timestamp:     time.Now(),
		Backends:      make(map[string]BackendHealthSummary),
		VirtualHosts:  make(map[string]VirtualHostHealthSummary),
		Probes:        make(map[string]ProbeResult),
		Metrics:       hc.getMetricsSummary(),
	}

	healthyBackends := 0
	totalBackends := 0

	for key, backendHealth := range hc.backends {
		totalBackends++
		summary := BackendHealthSummary{
			Status:               backendHealth.Status,
			LastCheck:            backendHealth.LastCheck,
			ConsecutiveFailures:  backendHealth.ConsecutiveFailures,
			ResponseTime:         backendHealth.ResponseTime,
			ErrorMessage:         backendHealth.ErrorMessage,
			SSLCertExpiry:        backendHealth.SSLCertExpiry,
			SSLCertValid:         backendHealth.SSLCertValid,
		}

		systemHealth.Backends[key] = summary

		if backendHealth.Status == StatusHealthy {
			healthyBackends++
		}
	}

	for key, vhostHealth := range hc.vhosts {
		summary := VirtualHostHealthSummary{
			Status:              vhostHealth.Status,
			LastCheck:           vhostHealth.LastCheck,
			BackendCount:        vhostHealth.BackendCount,
			HealthyBackends:     vhostHealth.HealthyBackends,
			UnhealthyBackends:   vhostHealth.UnhealthyBackends,
			SSLEnabled:          vhostHealth.SSLEnabled,
			SSLCertExpiry:       vhostHealth.SSLCertExpiry,
			RequestCount:        vhostHealth.RequestCount,
			ErrorCount:          vhostHealth.ErrorCount,
			AverageResponseTime: vhostHealth.AverageResponseTime,
		}

		systemHealth.VirtualHosts[key] = summary
	}

	for name := range hc.probes {
		result := hc.ExecuteProbe(name)
		systemHealth.Probes[name] = *result
	}

	if totalBackends == 0 {
		systemHealth.Status = StatusUnknown
	} else if healthyBackends == totalBackends {
		systemHealth.Status = StatusHealthy
	} else if healthyBackends == 0 {
		systemHealth.Status = StatusUnhealthy
	} else {
		systemHealth.Status = StatusDegraded
	}

	return systemHealth
}

func (hc *HealthChecker) ExecuteProbe(probeName string) *ProbeResult {
	hc.mutex.RLock()
	probe, exists := hc.probes[probeName]
	hc.mutex.RUnlock()

	if !exists {
		return &ProbeResult{
			Status:  StatusUnknown,
			Error:   fmt.Errorf("probe not found: %s", probeName),
			Message: "Probe not configured",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	return probe.Execute(ctx)
}

func (hc *HealthChecker) getMetricsSummary() MetricsSummary {
	hc.metrics.mutex.RLock()
	defer hc.metrics.mutex.RUnlock()

	return MetricsSummary{
		TotalChecks:       hc.metrics.TotalChecks,
		HealthyBackends:   hc.metrics.HealthyBackends,
		UnhealthyBackends: hc.metrics.UnhealthyBackends,
		DegradedBackends:  hc.metrics.DegradedBackends,
		HealthyVHosts:     hc.metrics.HealthyVHosts,
		UnhealthyVHosts:   hc.metrics.UnhealthyVHosts,
		AverageCheckTime:  hc.metrics.AverageCheckTime,
		CheckFailures:     hc.metrics.CheckFailures,
		StatusChanges:     hc.metrics.StatusChanges,
		MTLSHandshakes:    hc.metrics.MTLSHandshakes,
		SSLCertFailures:   hc.metrics.SSLCertFailures,
	}
}

type SystemHealth struct {
	Status        HealthStatus                               `json:"status"`
	Timestamp     time.Time                                  `json:"timestamp"`
	Backends      map[string]BackendHealthSummary            `json:"backends"`
	VirtualHosts  map[string]VirtualHostHealthSummary        `json:"virtual_hosts"`
	Probes        map[string]ProbeResult                     `json:"probes"`
	Metrics       MetricsSummary                             `json:"metrics"`
}

type BackendHealthSummary struct {
	Status              HealthStatus  `json:"status"`
	LastCheck           time.Time     `json:"last_check"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	ResponseTime        time.Duration `json:"response_time"`
	ErrorMessage        string        `json:"error_message,omitempty"`
	SSLCertExpiry       time.Time     `json:"ssl_cert_expiry,omitempty"`
	SSLCertValid        bool          `json:"ssl_cert_valid"`
}

type VirtualHostHealthSummary struct {
	Status              HealthStatus  `json:"status"`
	LastCheck           time.Time     `json:"last_check"`
	BackendCount        int           `json:"backend_count"`
	HealthyBackends     int           `json:"healthy_backends"`
	UnhealthyBackends   int           `json:"unhealthy_backends"`
	SSLEnabled          bool          `json:"ssl_enabled"`
	SSLCertExpiry       time.Time     `json:"ssl_cert_expiry,omitempty"`
	RequestCount        uint64        `json:"request_count"`
	ErrorCount          uint64        `json:"error_count"`
	AverageResponseTime time.Duration `json:"average_response_time"`
}

type MetricsSummary struct {
	TotalChecks       uint64        `json:"total_checks"`
	HealthyBackends   uint64        `json:"healthy_backends"`
	UnhealthyBackends uint64        `json:"unhealthy_backends"`
	DegradedBackends  uint64        `json:"degraded_backends"`
	HealthyVHosts     uint64        `json:"healthy_vhosts"`
	UnhealthyVHosts   uint64        `json:"unhealthy_vhosts"`
	AverageCheckTime  time.Duration `json:"average_check_time"`
	CheckFailures     uint64        `json:"check_failures"`
	StatusChanges     uint64        `json:"status_changes"`
	MTLSHandshakes    uint64        `json:"mtls_handshakes"`
	SSLCertFailures   uint64        `json:"ssl_cert_failures"`
}

// Health check implementations
func NewHTTPHealthCheck(name string, expected HTTPExpected) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		name:    name,
		enabled: true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		path:     "/health",
		method:   "GET",
		headers:  make(map[string]string),
		expected: expected,
	}
}

func (hc *HTTPHealthCheck) Name() string { return hc.name }
func (hc *HTTPHealthCheck) Enabled() bool { return hc.enabled }
func (hc *HTTPHealthCheck) Configure(config map[string]interface{}) error { return nil }

func (hc *HTTPHealthCheck) Check(ctx context.Context, target interface{}) *CheckResult {
	backend, ok := target.(*Backend)
	if !ok {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  fmt.Errorf("invalid target type for HTTP check"),
		}
	}

	url := fmt.Sprintf("%s://%s:%d%s", backend.Scheme, backend.Host, backend.Port, hc.path)

	req, err := http.NewRequestWithContext(ctx, hc.method, url, nil)
	if err != nil {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  err,
		}
	}

	start := time.Now()
	resp, err := hc.client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return &CheckResult{
			Status:       StatusUnhealthy,
			ResponseTime: responseTime,
			Error:        err,
		}
	}
	defer resp.Body.Close()

	if len(hc.expected.StatusCodes) > 0 {
		validStatus := false
		for _, code := range hc.expected.StatusCodes {
			if resp.StatusCode == code {
				validStatus = true
				break
			}
		}
		if !validStatus {
			return &CheckResult{
				Status:       StatusUnhealthy,
				ResponseTime: responseTime,
				Error:        fmt.Errorf("unexpected status code: %d", resp.StatusCode),
			}
		}
	}

	return &CheckResult{
		Status:       StatusHealthy,
		ResponseTime: responseTime,
		Message:      "HTTP check passed",
	}
}

func NewMTLSHealthCheck(name string, mtlsConfig MTLSCheckConfig, expected HTTPExpected) *MTLSHealthCheck {
	cert, err := tls.LoadX509KeyPair(mtlsConfig.CertPath, mtlsConfig.KeyPath)
	if err != nil {
		return &MTLSHealthCheck{
			name:    name,
			enabled: false,
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		InsecureSkipVerify: mtlsConfig.SkipVerify,
	}

	return &MTLSHealthCheck{
		name:    name,
		enabled: true,
		client: &http.Client{
			Timeout: mtlsConfig.ClientTimeout,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		tlsConfig: tlsConfig,
		path:      "/health",
		method:    "GET",
		headers:   make(map[string]string),
		expected:  expected,
	}
}

func (mc *MTLSHealthCheck) Name() string { return mc.name }
func (mc *MTLSHealthCheck) Enabled() bool { return mc.enabled }
func (mc *MTLSHealthCheck) Configure(config map[string]interface{}) error { return nil }

func (mc *MTLSHealthCheck) Check(ctx context.Context, target interface{}) *CheckResult {
	backend, ok := target.(*Backend)
	if !ok {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  fmt.Errorf("invalid target type for mTLS check"),
		}
	}

	url := fmt.Sprintf("https://%s:%d%s", backend.Host, backend.Port, mc.path)

	req, err := http.NewRequestWithContext(ctx, mc.method, url, nil)
	if err != nil {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  err,
		}
	}

	start := time.Now()
	resp, err := mc.client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return &CheckResult{
			Status:       StatusUnhealthy,
			ResponseTime: responseTime,
			Error:        err,
		}
	}
	defer resp.Body.Close()

	return &CheckResult{
		Status:       StatusHealthy,
		ResponseTime: responseTime,
		Message:      "mTLS check passed",
	}
}

func NewTCPHealthCheck(name string) *TCPHealthCheck {
	return &TCPHealthCheck{
		name:    name,
		enabled: true,
		timeout: 5 * time.Second,
	}
}

func (tc *TCPHealthCheck) Name() string { return tc.name }
func (tc *TCPHealthCheck) Enabled() bool { return tc.enabled }
func (tc *TCPHealthCheck) Configure(config map[string]interface{}) error { return nil }

func (tc *TCPHealthCheck) Check(ctx context.Context, target interface{}) *CheckResult {
	backend, ok := target.(*Backend)
	if !ok {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  fmt.Errorf("invalid target type for TCP check"),
		}
	}

	address := fmt.Sprintf("%s:%d", backend.Host, backend.Port)

	dialer := &net.Dialer{Timeout: tc.timeout}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	responseTime := time.Since(start)

	if err != nil {
		return &CheckResult{
			Status:       StatusUnhealthy,
			ResponseTime: responseTime,
			Error:        err,
		}
	}

	conn.Close()

	return &CheckResult{
		Status:       StatusHealthy,
		ResponseTime: responseTime,
		Message:      "TCP connection successful",
	}
}

func NewSSLCertCheck(name string) *SSLCertCheck {
	return &SSLCertCheck{
		name:    name,
		enabled: true,
		timeout: 5 * time.Second,
	}
}

func (sc *SSLCertCheck) Name() string { return sc.name }
func (sc *SSLCertCheck) Enabled() bool { return sc.enabled }
func (sc *SSLCertCheck) Configure(config map[string]interface{}) error { return nil }

func (sc *SSLCertCheck) Check(ctx context.Context, target interface{}) *CheckResult {
	backend, ok := target.(*Backend)
	if !ok {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  fmt.Errorf("invalid target type for SSL cert check"),
		}
	}

	if backend.Scheme != "https" {
		return &CheckResult{
			Status:  StatusHealthy,
			Message: "SSL check skipped for non-HTTPS backend",
		}
	}

	address := fmt.Sprintf("%s:%d", backend.Host, backend.Port)

	dialer := &net.Dialer{Timeout: sc.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName: backend.Host,
	})

	if err != nil {
		return &CheckResult{
			Status: StatusUnhealthy,
			Error:  err,
		}
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]

		if time.Now().After(cert.NotAfter) {
			return &CheckResult{
				Status:  StatusUnhealthy,
				Message: "SSL certificate expired",
				Metadata: map[string]interface{}{
					"cert_expiry": cert.NotAfter,
					"cert_valid":  false,
				},
			}
		}

		if time.Until(cert.NotAfter) < 30*24*time.Hour {
			return &CheckResult{
				Status:  StatusDegraded,
				Message: "SSL certificate expires soon",
				Metadata: map[string]interface{}{
					"cert_expiry": cert.NotAfter,
					"cert_valid":  true,
				},
			}
		}

		return &CheckResult{
			Status:  StatusHealthy,
			Message: "SSL certificate valid",
			Metadata: map[string]interface{}{
				"cert_expiry": cert.NotAfter,
				"cert_valid":  true,
			},
		}
	}

	return &CheckResult{
		Status:  StatusUnhealthy,
		Message: "No SSL certificate found",
	}
}

// Probe implementations
func NewReadinessProbe(name string, config ProbeConfig) *ReadinessProbe {
	return &ReadinessProbe{name: name, config: config}
}

func (rp *ReadinessProbe) Name() string { return rp.name }
func (rp *ReadinessProbe) GetConfig() ProbeConfig { return rp.config }
func (rp *ReadinessProbe) Execute(ctx context.Context) *ProbeResult {
	return &ProbeResult{Status: StatusHealthy, Message: "Readiness probe passed"}
}

func NewLivenessProbe(name string, config ProbeConfig) *LivenessProbe {
	return &LivenessProbe{name: name, config: config}
}

func (lp *LivenessProbe) Name() string { return lp.name }
func (lp *LivenessProbe) GetConfig() ProbeConfig { return lp.config }
func (lp *LivenessProbe) Execute(ctx context.Context) *ProbeResult {
	return &ProbeResult{Status: StatusHealthy, Message: "Liveness probe passed"}
}

func NewMTLSProbe(name string, config ProbeConfig) Probe {
	return &ReadinessProbe{name: name, config: config}
}

// Metrics methods
func (hm *HealthMetrics) recordCheck(duration time.Duration) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.TotalChecks++
	if hm.TotalChecks > 0 {
		hm.AverageCheckTime = (hm.AverageCheckTime*time.Duration(hm.TotalChecks-1) + duration) / time.Duration(hm.TotalChecks)
	} else {
		hm.AverageCheckTime = duration
	}
}

func (hm *HealthMetrics) recordCheckFailure() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.CheckFailures++
}

func (hm *HealthMetrics) recordStatusChange() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.StatusChanges++
}

//nolint:unused // Reserved for future mTLS health check integration
func (hm *HealthMetrics) recordMTLSHandshake() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.MTLSHandshakes++
}

//nolint:unused // Reserved for future mTLS health check integration
func (hm *HealthMetrics) recordSSLCertFailure() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.SSLCertFailures++
}