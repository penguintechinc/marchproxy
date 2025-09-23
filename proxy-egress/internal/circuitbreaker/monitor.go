package circuitbreaker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/MarchProxy/proxy/internal/manager"
)

type Monitor struct {
	serviceBreaker *ServiceCircuitBreaker
	metrics        *MetricsCollector
	alerts         *AlertManager
	dashboard      *Dashboard
	config         MonitorConfig
	running        bool
	mutex          sync.RWMutex
	stopChan       chan struct{}
}

type MonitorConfig struct {
	CollectionInterval   time.Duration
	RetentionPeriod     time.Duration
	AlertThresholds     AlertThresholds
	DashboardEnabled    bool
	DashboardPort       int
	LogEnabled          bool
	MetricsEnabled      bool
}

type AlertThresholds struct {
	ErrorRateThreshold     float64
	FailureCountThreshold  uint64
	StateChangeThreshold   uint64
	ResponseTimeThreshold  time.Duration
}

type MetricsCollector struct {
	metrics map[string][]*TimeSeriesData
	mutex   sync.RWMutex
	config  MonitorConfig
}

type TimeSeriesData struct {
	Timestamp time.Time     `json:"timestamp"`
	Metrics   BreakerMetrics `json:"metrics"`
}

type AlertManager struct {
	alerts    []Alert
	mutex     sync.RWMutex
	callbacks []AlertCallback
	config    MonitorConfig
}

type Alert struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	ServiceName string    `json:"service_name"`
	Type        AlertType `json:"type"`
	Severity    Severity  `json:"severity"`
	Message     string    `json:"message"`
	Data        map[string]interface{} `json:"data"`
	Resolved    bool      `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type AlertType string

const (
	AlertTypeStateChange     AlertType = "state_change"
	AlertTypeHighErrorRate   AlertType = "high_error_rate"
	AlertTypeHighFailures    AlertType = "high_failures"
	AlertTypeSlowResponses   AlertType = "slow_responses"
	AlertTypeServiceDown     AlertType = "service_down"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type AlertCallback func(alert Alert)

type Dashboard struct {
	server   *http.Server
	monitor  *Monitor
	enabled  bool
}

func NewMonitor(serviceBreaker *ServiceCircuitBreaker, config MonitorConfig) *Monitor {
	return &Monitor{
		serviceBreaker: serviceBreaker,
		metrics:        NewMetricsCollector(config),
		alerts:         NewAlertManager(config),
		dashboard:      NewDashboard(config),
		config:        config,
		stopChan:      make(chan struct{}),
	}
}

func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		CollectionInterval: 30 * time.Second,
		RetentionPeriod:   24 * time.Hour,
		AlertThresholds: AlertThresholds{
			ErrorRateThreshold:    50.0,
			FailureCountThreshold: 100,
			StateChangeThreshold:  5,
			ResponseTimeThreshold: 5 * time.Second,
		},
		DashboardEnabled: true,
		DashboardPort:   8090,
		LogEnabled:      true,
		MetricsEnabled:  true,
	}
}

func NewMetricsCollector(config MonitorConfig) *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string][]*TimeSeriesData),
		config:  config,
	}
}

func (mc *MetricsCollector) Collect(breakers map[string]*CircuitBreaker) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-mc.config.RetentionPeriod)
	
	for name, breaker := range breakers {
		metrics := breaker.GetMetrics()
		data := &TimeSeriesData{
			Timestamp: now,
			Metrics:   metrics,
		}
		
		mc.metrics[name] = append(mc.metrics[name], data)
		
		var filtered []*TimeSeriesData
		for _, entry := range mc.metrics[name] {
			if entry.Timestamp.After(cutoff) {
				filtered = append(filtered, entry)
			}
		}
		mc.metrics[name] = filtered
	}
}

func (mc *MetricsCollector) GetMetrics(serviceName string, duration time.Duration) []*TimeSeriesData {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	data, exists := mc.metrics[serviceName]
	if !exists {
		return nil
	}
	
	cutoff := time.Now().Add(-duration)
	var result []*TimeSeriesData
	
	for _, entry := range data {
		if entry.Timestamp.After(cutoff) {
			result = append(result, entry)
		}
	}
	
	return result
}

func (mc *MetricsCollector) GetAllMetrics() map[string][]*TimeSeriesData {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	result := make(map[string][]*TimeSeriesData)
	for name, data := range mc.metrics {
		result[name] = make([]*TimeSeriesData, len(data))
		copy(result[name], data)
	}
	
	return result
}

func (mc *MetricsCollector) GetLatestMetrics() map[string]*TimeSeriesData {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	result := make(map[string]*TimeSeriesData)
	for name, data := range mc.metrics {
		if len(data) > 0 {
			result[name] = data[len(data)-1]
		}
	}
	
	return result
}

func NewAlertManager(config MonitorConfig) *AlertManager {
	return &AlertManager{
		alerts: make([]Alert, 0),
		config: config,
	}
}

func (am *AlertManager) CheckAlerts(breakers map[string]*CircuitBreaker) {
	for name, breaker := range breakers {
		metrics := breaker.GetMetrics()
		am.checkBreakerAlerts(name, metrics)
	}
}

func (am *AlertManager) checkBreakerAlerts(serviceName string, metrics BreakerMetrics) {
	if metrics.ErrorRate >= am.config.AlertThresholds.ErrorRateThreshold {
		am.CreateAlert(AlertTypeHighErrorRate, SeverityHigh, serviceName, 
			fmt.Sprintf("High error rate: %.2f%%", metrics.ErrorRate),
			map[string]interface{}{
				"error_rate": metrics.ErrorRate,
				"threshold": am.config.AlertThresholds.ErrorRateThreshold,
			})
	}
	
	if metrics.TotalFailures >= am.config.AlertThresholds.FailureCountThreshold {
		am.CreateAlert(AlertTypeHighFailures, SeverityMedium, serviceName,
			fmt.Sprintf("High failure count: %d", metrics.TotalFailures),
			map[string]interface{}{
				"failure_count": metrics.TotalFailures,
				"threshold": am.config.AlertThresholds.FailureCountThreshold,
			})
	}
	
	if metrics.AverageResponseTime >= am.config.AlertThresholds.ResponseTimeThreshold {
		am.CreateAlert(AlertTypeSlowResponses, SeverityMedium, serviceName,
			fmt.Sprintf("Slow responses: %v", metrics.AverageResponseTime),
			map[string]interface{}{
				"response_time": metrics.AverageResponseTime,
				"threshold": am.config.AlertThresholds.ResponseTimeThreshold,
			})
	}
	
	if metrics.State == "OPEN" {
		am.CreateAlert(AlertTypeServiceDown, SeverityCritical, serviceName,
			"Service circuit breaker is OPEN - service unavailable",
			map[string]interface{}{
				"state": metrics.State,
				"last_state_change": metrics.LastStateChange,
			})
	}
	
	if metrics.StateChanges >= am.config.AlertThresholds.StateChangeThreshold {
		am.CreateAlert(AlertTypeStateChange, SeverityMedium, serviceName,
			fmt.Sprintf("Frequent state changes: %d", metrics.StateChanges),
			map[string]interface{}{
				"state_changes": metrics.StateChanges,
				"threshold": am.config.AlertThresholds.StateChangeThreshold,
			})
	}
}

func (am *AlertManager) CreateAlert(alertType AlertType, severity Severity, serviceName, message string, data map[string]interface{}) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	alert := Alert{
		ID:          fmt.Sprintf("%s-%s-%d", alertType, serviceName, time.Now().Unix()),
		Timestamp:   time.Now(),
		ServiceName: serviceName,
		Type:        alertType,
		Severity:    severity,
		Message:     message,
		Data:        data,
		Resolved:    false,
	}
	
	am.alerts = append(am.alerts, alert)
	
	for _, callback := range am.callbacks {
		go callback(alert)
	}
}

func (am *AlertManager) ResolveAlert(alertID string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	for i := range am.alerts {
		if am.alerts[i].ID == alertID && !am.alerts[i].Resolved {
			now := time.Now()
			am.alerts[i].Resolved = true
			am.alerts[i].ResolvedAt = &now
			break
		}
	}
}

func (am *AlertManager) GetAlerts(resolved bool) []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	var result []Alert
	for _, alert := range am.alerts {
		if alert.Resolved == resolved {
			result = append(result, alert)
		}
	}
	
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	
	return result
}

func (am *AlertManager) AddCallback(callback AlertCallback) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.callbacks = append(am.callbacks, callback)
}

func NewDashboard(config MonitorConfig) *Dashboard {
	return &Dashboard{
		enabled: config.DashboardEnabled,
	}
}

func (d *Dashboard) Start(monitor *Monitor) error {
	if !d.enabled {
		return nil
	}
	
	d.monitor = monitor
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleDashboard)
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	mux.HandleFunc("/api/alerts", d.handleAlerts)
	mux.HandleFunc("/api/breakers", d.handleBreakers)
	mux.HandleFunc("/api/breakers/reset", d.handleReset)
	
	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", monitor.config.DashboardPort),
		Handler: mux,
	}
	
	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard server error: %v", err)
		}
	}()
	
	return nil
}

func (d *Dashboard) Stop() error {
	if d.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return d.server.Shutdown(ctx)
	}
	return nil
}

func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Circuit Breaker Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .breaker { border: 1px solid #ccc; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .state-CLOSED { border-left: 5px solid green; }
        .state-OPEN { border-left: 5px solid red; }
        .state-HALF_OPEN { border-left: 5px solid orange; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
        .metric { background: #f5f5f5; padding: 10px; border-radius: 3px; }
        .alert { padding: 10px; margin: 5px 0; border-radius: 3px; }
        .alert-high { background: #ffebee; border-left: 5px solid #f44336; }
        .alert-critical { background: #ffcdd2; border-left: 5px solid #d32f2f; }
        .alert-medium { background: #fff3e0; border-left: 5px solid #ff9800; }
        .alert-low { background: #e8f5e8; border-left: 5px solid #4caf50; }
    </style>
    <script>
        async function loadData() {
            try {
                const [breakersResp, alertsResp] = await Promise.all([
                    fetch('/api/breakers'),
                    fetch('/api/alerts')
                ]);
                const breakers = await breakersResp.json();
                const alerts = await alertsResp.json();
                
                displayBreakers(breakers);
                displayAlerts(alerts);
            } catch (error) {
                console.error('Error loading data:', error);
            }
        }
        
        function displayBreakers(breakers) {
            const container = document.getElementById('breakers');
            container.innerHTML = '';
            
            for (const [name, metrics] of Object.entries(breakers)) {
                const div = document.createElement('div');
                div.className = 'breaker state-' + metrics.state;
                div.innerHTML = '<h3>' + name + ' (' + metrics.state + ')</h3>' +
                    '<div class="metrics">' +
                    '<div class="metric">Requests: ' + metrics.total_requests + '</div>' +
                    '<div class="metric">Success Rate: ' + (100 - metrics.error_rate).toFixed(2) + '%</div>' +
                    '<div class="metric">Failures: ' + metrics.total_failures + '</div>' +
                    '<div class="metric">Avg Response: ' + (metrics.average_response_time / 1000000).toFixed(2) + 'ms</div>' +
                    '<div class="metric">Current Requests: ' + metrics.current_requests + '</div>' +
                    '<div class="metric">State Changes: ' + metrics.state_changes + '</div>' +
                    '</div>';
                container.appendChild(div);
            }
        }
        
        function displayAlerts(alerts) {
            const container = document.getElementById('alerts');
            container.innerHTML = '';
            
            alerts.forEach(alert => {
                const div = document.createElement('div');
                div.className = 'alert alert-' + alert.severity;
                div.innerHTML = '<strong>' + alert.service_name + '</strong> - ' + alert.message +
                    '<br><small>' + new Date(alert.timestamp).toLocaleString() + '</small>';
                container.appendChild(div);
            });
        }
        
        async function resetBreakers() {
            try {
                await fetch('/api/breakers/reset', { method: 'POST' });
                loadData();
            } catch (error) {
                console.error('Error resetting breakers:', error);
            }
        }
        
        setInterval(loadData, 5000);
        window.onload = loadData;
    </script>
</head>
<body>
    <h1>Circuit Breaker Dashboard</h1>
    <button onclick="resetBreakers()">Reset All Breakers</button>
    
    <h2>Circuit Breakers</h2>
    <div id="breakers"></div>
    
    <h2>Recent Alerts</h2>
    <div id="alerts"></div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := d.monitor.metrics.GetAllMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (d *Dashboard) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := d.monitor.alerts.GetAlerts(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (d *Dashboard) handleBreakers(w http.ResponseWriter, r *http.Request) {
	breakers := d.monitor.serviceBreaker.GetAllBreakers()
	metrics := make(map[string]BreakerMetrics)
	
	for name, breaker := range breakers {
		metrics[name] = breaker.GetMetrics()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (d *Dashboard) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	d.monitor.serviceBreaker.ResetAll()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (m *Monitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.running {
		return fmt.Errorf("monitor is already running")
	}
	
	m.running = true
	
	if err := m.dashboard.Start(m); err != nil {
		return fmt.Errorf("failed to start dashboard: %w", err)
	}
	
	m.alerts.AddCallback(func(alert Alert) {
		if m.config.LogEnabled {
			log.Printf("ALERT [%s] %s: %s", alert.Severity, alert.ServiceName, alert.Message)
		}
	})
	
	go m.monitorLoop()
	
	return nil
}

func (m *Monitor) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.running {
		return nil
	}
	
	m.running = false
	close(m.stopChan)
	
	return m.dashboard.Stop()
}

func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(m.config.CollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			breakers := m.serviceBreaker.GetAllBreakers()
			
			if m.config.MetricsEnabled {
				m.metrics.Collect(breakers)
			}
			
			m.alerts.CheckAlerts(breakers)
			
		case <-m.stopChan:
			return
		}
	}
}

func (m *Monitor) GetMetrics(serviceName string, duration time.Duration) []*TimeSeriesData {
	return m.metrics.GetMetrics(serviceName, duration)
}

func (m *Monitor) GetAlerts(resolved bool) []Alert {
	return m.alerts.GetAlerts(resolved)
}

func (m *Monitor) AddAlertCallback(callback AlertCallback) {
	m.alerts.AddCallback(callback)
}