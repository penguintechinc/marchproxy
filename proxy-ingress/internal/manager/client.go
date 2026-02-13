package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"marchproxy-ingress/internal/config"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string

	lastConfigHash string
	lastConfigTime time.Time

	clusterID   int
	clusterName string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.GetManagerTimeout(),
		},
		baseURL: cfg.Manager.URL,
		apiKey:  cfg.Manager.APIKey,
	}
}

type RegistrationRequest struct {
	Name         string   `json:"name"`
	Hostname     string   `json:"hostname"`
	Version      string   `json:"version"`
	ProxyType    string   `json:"proxy_type"`
	Capabilities []string `json:"capabilities"`
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
}

type RegistrationResponse struct {
	Success      bool   `json:"success"`
	ProxyID      int    `json:"proxy_id"`
	ClusterName  string `json:"cluster_name"`
	Message      string `json:"message"`
	Error        string `json:"error,omitempty"`
}

type VirtualHost struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	Hostname     string                 `json:"hostname"`
	SSLEnabled   bool                   `json:"ssl_enabled"`
	CertID       *int                   `json:"cert_id,omitempty"`
	Backend      string                 `json:"backend"`
	RoutingRules []RoutingRule          `json:"routing_rules"`
	Headers      map[string]string      `json:"headers"`
	Middleware   []string               `json:"middleware"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type RoutingRule struct {
	ID            int               `json:"id"`
	PathPattern   string            `json:"path_pattern"`
	PathType      string            `json:"path_type"`
	Backend       string            `json:"backend"`
	Priority      int               `json:"priority"`
	Methods       []string          `json:"methods"`
	Headers       map[string]string `json:"headers"`
	QueryParams   map[string]string `json:"query_params"`
	Rewrite       *RewriteRule      `json:"rewrite,omitempty"`
	RateLimiting  *RateLimitRule    `json:"rate_limiting,omitempty"`
	Authentication *AuthRule        `json:"authentication,omitempty"`
}

type RewriteRule struct {
	StripPrefix string            `json:"strip_prefix"`
	AddPrefix   string            `json:"add_prefix"`
	Replace     map[string]string `json:"replace"`
}

type RateLimitRule struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	WindowSize        time.Duration `json:"window_size"`
}

type AuthRule struct {
	Required     bool     `json:"required"`
	Methods      []string `json:"methods"`
	ClientCerts  []string `json:"client_certs"`
	AllowedCNs   []string `json:"allowed_cns"`
	AllowedOUs   []string `json:"allowed_ous"`
}

type Backend struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Endpoints       []BackendEndpoint      `json:"endpoints"`
	LoadBalancing   LoadBalancingConfig    `json:"load_balancing"`
	HealthCheck     HealthCheckConfig      `json:"health_check"`
	CircuitBreaker  CircuitBreakerConfig   `json:"circuit_breaker"`
	Timeout         time.Duration          `json:"timeout"`
	RetryPolicy     RetryPolicyConfig      `json:"retry_policy"`
	TLSConfig       BackendTLSConfig       `json:"tls_config"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type BackendEndpoint struct {
	ID     int    `json:"id"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"`
	Active bool   `json:"active"`
}

type LoadBalancingConfig struct {
	Algorithm        string        `json:"algorithm"`
	StickySession    bool          `json:"sticky_session"`
	SessionCookie    string        `json:"session_cookie"`
	HealthThreshold  int           `json:"health_threshold"`
	UnhealthyTimeout time.Duration `json:"unhealthy_timeout"`
}

type HealthCheckConfig struct {
	Enabled         bool          `json:"enabled"`
	Path            string        `json:"path"`
	Interval        time.Duration `json:"interval"`
	Timeout         time.Duration `json:"timeout"`
	HealthyStatus   []int         `json:"healthy_status"`
	UnhealthyLimit  int           `json:"unhealthy_limit"`
	HealthyLimit    int           `json:"healthy_limit"`
}

type CircuitBreakerConfig struct {
	Enabled           bool          `json:"enabled"`
	FailureThreshold  int           `json:"failure_threshold"`
	RecoveryTimeout   time.Duration `json:"recovery_timeout"`
	HalfOpenRequests  int           `json:"half_open_requests"`
}

type RetryPolicyConfig struct {
	Enabled     bool          `json:"enabled"`
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
	BackoffType string        `json:"backoff_type"`
}

type BackendTLSConfig struct {
	Enabled            bool     `json:"enabled"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify"`
	ServerName         string   `json:"server_name"`
	ClientCertID       *int     `json:"client_cert_id,omitempty"`
	CACerts            []string `json:"ca_certs"`
}

type Certificate struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Subject    string   `json:"subject"`
	Issuer     string   `json:"issuer"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	SANDomains []string `json:"san_domains"`
	KeyUsage   []string `json:"key_usage"`
	CertData   string   `json:"cert_data,omitempty"`
	KeyData    string   `json:"key_data,omitempty"`
	CAData     string   `json:"ca_data,omitempty"`
}

type LoggingConfig struct {
	SyslogEndpoint string            `json:"syslog_endpoint"`
	LogLevel       string            `json:"log_level"`
	LogAuth        bool              `json:"log_auth"`
	LogNetflow     bool              `json:"log_netflow"`
	LogDebug       bool              `json:"log_debug"`
	LogRequests    bool              `json:"log_requests"`
	LogFormat      string            `json:"log_format"`
	Fields         map[string]string `json:"fields"`
}

type SecurityPolicy struct {
	ID             int                    `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Rules          []SecurityRule         `json:"rules"`
	DefaultAction  string                 `json:"default_action"`
	Priority       int                    `json:"priority"`
	Enabled        bool                   `json:"enabled"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type SecurityRule struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Action      string                 `json:"action"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ClusterConfig struct {
	Cluster         ClusterInfo        `json:"cluster"`
	VirtualHosts    []VirtualHost      `json:"virtual_hosts"`
	Backends        []Backend          `json:"backends"`
	Certificates    []Certificate      `json:"certificates"`
	Logging         LoggingConfig      `json:"logging"`
	SecurityPolicies []SecurityPolicy  `json:"security_policies"`
	ConfigHash      string             `json:"config_hash"`
	Version         string             `json:"version"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type ClusterInfo struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ProxyCount  int                    `json:"proxy_count"`
	MaxProxies  int                    `json:"max_proxies"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type ConfigResponse struct {
	Success bool          `json:"success"`
	Data    ClusterConfig `json:"data"`
	Error   string        `json:"error,omitempty"`
	Hash    string        `json:"hash"`
}

type HealthReportRequest struct {
	ProxyID       int                    `json:"proxy_id"`
	Status        string                 `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	Uptime        time.Duration          `json:"uptime"`
	CPUUsage      float64                `json:"cpu_usage"`
	MemoryUsage   int64                  `json:"memory_usage"`
	Connections   int                    `json:"connections"`
	RequestCount  uint64                 `json:"request_count"`
	ErrorCount    uint64                 `json:"error_count"`
	VirtualHosts  map[string]interface{} `json:"virtual_hosts"`
	Backends      map[string]interface{} `json:"backends"`
	Certificates  map[string]interface{} `json:"certificates"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type HealthReportResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (c *Client) Register(ctx context.Context, proxyName, hostname, version string, capabilities []string) (*RegistrationResponse, error) {
	req := RegistrationRequest{
		Name:         proxyName,
		Hostname:     hostname,
		Version:      version,
		ProxyType:    "ingress",
		Capabilities: capabilities,
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}

	var resp RegistrationResponse
	err := c.makeRequest(ctx, "POST", "/api/v1/proxies/register", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	if resp.Success {
		c.clusterID = resp.ProxyID
		c.clusterName = resp.ClusterName
	}

	return &resp, nil
}

func (c *Client) GetConfig(ctx context.Context) (*ClusterConfig, error) {
	var resp ConfigResponse
	err := c.makeRequest(ctx, "GET", "/api/v1/config", nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("config request failed: %s", resp.Error)
	}

	if resp.Hash != c.lastConfigHash {
		c.lastConfigHash = resp.Hash
		c.lastConfigTime = time.Now()
	}

	return &resp.Data, nil
}

func (c *Client) ReportHealth(ctx context.Context, report HealthReportRequest) error {
	report.ProxyID = c.clusterID
	report.Timestamp = time.Now()

	var resp HealthReportResponse
	err := c.makeRequest(ctx, "POST", "/api/v1/health", report, &resp)
	if err != nil {
		return fmt.Errorf("health report failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("health report rejected: %s", resp.Error)
	}

	return nil
}

func (c *Client) GetCertificate(ctx context.Context, certID int) (*Certificate, error) {
	var cert Certificate
	err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/certificates/%d", certID), nil, &cert)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	return &cert, nil
}

func (c *Client) ListCertificates(ctx context.Context) ([]Certificate, error) {
	var certs []Certificate
	err := c.makeRequest(ctx, "GET", "/api/v1/certificates", nil, &certs)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	return certs, nil
}

func (c *Client) NotifyConfigUpdate(ctx context.Context, updateType, message string) error {
	notification := map[string]interface{}{
		"proxy_id":    c.clusterID,
		"update_type": updateType,
		"message":     message,
		"timestamp":   time.Now(),
	}

	var resp map[string]interface{}
	err := c.makeRequest(ctx, "POST", "/api/v1/notifications", notification, &resp)
	if err != nil {
		return fmt.Errorf("notification failed: %w", err)
	}

	return nil
}

func (c *Client) PollConfigChanges(ctx context.Context, interval time.Duration) <-chan *ClusterConfig {
	configChan := make(chan *ClusterConfig)

	go func() {
		defer close(configChan)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				config, err := c.GetConfig(ctx)
				if err != nil {
					continue
				}

				configChan <- config
			}
		}
	}()

	return configChan
}

func (c *Client) GetBackend(ctx context.Context, backendName string) (*Backend, error) {
	var backend Backend
	err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/backends/%s", backendName), nil, &backend)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}

	return &backend, nil
}

func (c *Client) ListBackends(ctx context.Context) ([]Backend, error) {
	var backends []Backend
	err := c.makeRequest(ctx, "GET", "/api/v1/backends", nil, &backends)
	if err != nil {
		return nil, fmt.Errorf("failed to list backends: %w", err)
	}

	return backends, nil
}

func (c *Client) GetVirtualHost(ctx context.Context, vhostName string) (*VirtualHost, error) {
	var vhost VirtualHost
	err := c.makeRequest(ctx, "GET", fmt.Sprintf("/api/v1/vhosts/%s", vhostName), nil, &vhost)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual host: %w", err)
	}

	return &vhost, nil
}

func (c *Client) ListVirtualHosts(ctx context.Context) ([]VirtualHost, error) {
	var vhosts []VirtualHost
	err := c.makeRequest(ctx, "GET", "/api/v1/vhosts", nil, &vhosts)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual hosts: %w", err)
	}

	return vhosts, nil
}

func (c *Client) Ping(ctx context.Context) error {
	var resp map[string]interface{}
	err := c.makeRequest(ctx, "GET", "/api/v1/ping", nil, &resp)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "MarchProxy-Ingress/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // Best-effort read of error response body
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) SetClusterInfo(clusterID int, clusterName string) {
	c.clusterID = clusterID
	c.clusterName = clusterName
}

func (c *Client) GetClusterID() int {
	return c.clusterID
}

func (c *Client) GetClusterName() string {
	return c.clusterName
}

func (c *Client) GetLastConfigTime() time.Time {
	return c.lastConfigTime
}

func (c *Client) GetLastConfigHash() string {
	return c.lastConfigHash
}