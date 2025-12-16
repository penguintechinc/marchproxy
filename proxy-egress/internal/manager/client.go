package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"marchproxy-egress/internal/config"
)

// Client handles communication with the MarchProxy manager API
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	
	// Configuration state
	lastConfigHash string
	lastConfigTime time.Time
	
	// Cluster information
	clusterID   int
	clusterName string
}

// NewClient creates a new manager API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ConnectionTimeout) * time.Second,
		},
		baseURL: cfg.ManagerURL,
		apiKey:  cfg.ClusterAPIKey,
	}
}

// Registration types
type RegistrationRequest struct {
	Name         string   `json:"name"`
	Hostname     string   `json:"hostname"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

type RegistrationResponse struct {
	Success      bool   `json:"success"`
	ProxyID      int    `json:"proxy_id"`
	ClusterName  string `json:"cluster_name"`
	Message      string `json:"message"`
	Error        string `json:"error,omitempty"`
}

// Configuration types
type Service struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	IPFQDN     string `json:"ip_fqdn"`
	Collection string `json:"collection"`
	AuthType   string `json:"auth_type"`
	AuthToken  string `json:"auth_token,omitempty"`
	JWTSecret  string `json:"jwt_secret,omitempty"`
	JWTExpiry  int    `json:"jwt_expiry,omitempty"`
}

type Mapping struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	SourceServices  []int    `json:"source_services"`
	DestServices    []int    `json:"dest_services"`
	Protocols       []string `json:"protocols"`
	Ports           string   `json:"ports"`
	AuthRequired    bool     `json:"auth_required"`
	AuthType        string   `json:"auth_type"`
	Priority        int      `json:"priority"`
	Timeout         int      `json:"timeout"`
}

type Certificate struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Subject    string   `json:"subject"`
	NotAfter   string   `json:"not_after"`
	SANDomains []string `json:"san_domains"`
}

type LoggingConfig struct {
	SyslogEndpoint string `json:"syslog_endpoint"`
	LogAuth        bool   `json:"log_auth"`
	LogNetflow     bool   `json:"log_netflow"`
	LogDebug       bool   `json:"log_debug"`
}

type ClusterConfig struct {
	Cluster      ClusterInfo     `json:"cluster"`
	Logging      LoggingConfig   `json:"logging"`
	Services     []Service       `json:"services"`
	Mappings     []Mapping       `json:"mappings"`
	Certificates []Certificate   `json:"certificates"`
	Version      string          `json:"version"`
	GeneratedAt  string          `json:"generated_at"`
}

type ClusterInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// License types
type LicenseStatus struct {
	Edition        string   `json:"edition"`
	Valid          bool     `json:"valid"`
	ProxyLimit     int      `json:"proxy_limit"`
	Features       []string `json:"features"`
	ExpiresAt      string   `json:"expires_at,omitempty"`
	ClusterID      int      `json:"cluster_id"`
	ClusterName    string   `json:"cluster_name"`
	CurrentProxies int      `json:"current_proxies"`
	MaxProxies     int      `json:"max_proxies"`
	CanRegister    bool     `json:"can_register"`
	Error          string   `json:"error,omitempty"`
}

// Heartbeat types
type HeartbeatRequest struct {
	Name             string  `json:"name"`
	CPUUsage         float64 `json:"cpu_usage"`
	MemoryUsage      float64 `json:"memory_usage"`
	Connections      int     `json:"connections"`
	BytesTransferred int64   `json:"bytes_transferred"`
}

type HeartbeatResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// Register registers this proxy with the manager
func (c *Client) Register(cfg *config.Config) error {
	fmt.Printf("Registering proxy with manager...\n")
	
	req := RegistrationRequest{
		Name:         cfg.ProxyName,
		Hostname:     cfg.Hostname,
		Version:      getVersion(),
		Capabilities: cfg.GetCapabilities(),
	}
	
	var resp RegistrationResponse
	if err := c.makeRequest("POST", "/api/proxy/register", req, &resp); err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Error)
	}
	
	// Store cluster information
	c.clusterID = resp.ProxyID
	c.clusterName = resp.ClusterName
	
	fmt.Printf("Proxy registered successfully - ID: %d, Cluster: %s\n", resp.ProxyID, resp.ClusterName)
	
	return nil
}

// GetConfig retrieves the current configuration from the manager
func (c *Client) GetConfig() (*ClusterConfig, error) {
	if c.clusterID == 0 {
		return nil, fmt.Errorf("proxy not registered, call Register() first")
	}
	
	endpoint := fmt.Sprintf("/api/config/%d", c.clusterID)
	
	var config ClusterConfig
	if err := c.makeRequest("GET", endpoint, nil, &config); err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	
	// Update local state
	c.lastConfigHash = config.Version
	c.lastConfigTime = time.Now()
	
	fmt.Printf("Retrieved config - Services: %d, Mappings: %d, Version: %s\n", 
		len(config.Services), len(config.Mappings), config.Version)
	
	return &config, nil
}

// GetLicenseStatus retrieves the current license status
func (c *Client) GetLicenseStatus() (*LicenseStatus, error) {
	var status LicenseStatus
	if err := c.makeRequest("GET", "/api/license-status", nil, &status); err != nil {
		return nil, fmt.Errorf("failed to get license status: %w", err)
	}
	
	return &status, nil
}

// SendHeartbeat sends a heartbeat with current proxy status
func (c *Client) SendHeartbeat(cfg *config.Config, stats SystemStats) error {
	req := HeartbeatRequest{
		Name:             cfg.ProxyName,
		CPUUsage:         stats.CPUUsage,
		MemoryUsage:      stats.MemoryUsage,
		Connections:      stats.ActiveConnections,
		BytesTransferred: stats.BytesTransferred,
	}
	
	var resp HeartbeatResponse
	if err := c.makeRequest("POST", "/api/proxy/heartbeat", req, &resp); err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("heartbeat failed: %s", resp.Error)
	}
	
	return nil
}

// StartConfigRefresh starts a goroutine that periodically refreshes configuration
func (c *Client) StartConfigRefresh(ctx context.Context, cfg *config.Config, onConfigUpdate func(*ClusterConfig)) {
	interval := time.Duration(cfg.ConfigUpdateInterval) * time.Second
	jitter := time.Duration(30) * time.Second // Add randomization
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	fmt.Printf("Starting config refresh loop - interval: %v, jitter: %v\n", interval, jitter)
	
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Configuration refresh loop stopped\n")
			return
			
		case <-ticker.C:
			// Add random jitter to prevent thundering herd
			jitterDuration := time.Duration(rand.Int63n(int64(jitter)))
			time.Sleep(jitterDuration)
			
			config, err := c.GetConfig()
			if err != nil {
				fmt.Printf("Failed to refresh configuration: %v\n", err)
				continue
			}
			
			// Check if configuration changed
			if config.Version != c.lastConfigHash {
				fmt.Printf("Configuration updated - old: %s, new: %s\n", c.lastConfigHash, config.Version)
				onConfigUpdate(config)
			}
		}
	}
}

// StartHeartbeat starts a goroutine that periodically sends heartbeat
func (c *Client) StartHeartbeat(ctx context.Context, cfg *config.Config, getStats func() SystemStats) {
	interval := time.Duration(cfg.HeartbeatInterval) * time.Second
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	fmt.Printf("Starting heartbeat loop - interval: %v\n", interval)
	
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Heartbeat loop stopped\n")
			return
			
		case <-ticker.C:
			stats := getStats()
			if err := c.SendHeartbeat(cfg, stats); err != nil {
				fmt.Printf("Failed to send heartbeat: %v\n", err)
			}
		}
	}
}

// makeRequest makes an HTTP request to the manager API
func (c *Client) makeRequest(method, endpoint string, reqBody interface{}, respBody interface{}) error {
	url := c.baseURL + endpoint
	
	var bodyReader io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", "MarchProxy-Proxy/"+getVersion())
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Check status code
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	// Parse response
	if respBody != nil {
		if err := json.Unmarshal(bodyBytes, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	
	return nil
}

// SystemStats represents current system statistics
type SystemStats struct {
	CPUUsage          float64
	MemoryUsage       float64
	ActiveConnections int
	BytesTransferred  int64
}

// GetSystemStats returns current system statistics
func GetSystemStats() SystemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return SystemStats{
		CPUUsage:    0.0, // TODO: Implement CPU usage calculation
		MemoryUsage: float64(m.Sys) / 1024 / 1024, // MB
		// ActiveConnections and BytesTransferred would be populated by the proxy server
	}
}

// getVersion returns the current version of the proxy
func getVersion() string {
	return "v0.1.1" // Would be set at build time
}