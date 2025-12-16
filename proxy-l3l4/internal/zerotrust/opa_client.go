package zerotrust

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OPAClient handles communication with OPA server
type OPAClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// OPARequest represents a request to OPA server
type OPARequest struct {
	Input interface{} `json:"input"`
}

// OPAResponse represents a response from OPA server
type OPAResponse struct {
	Result interface{} `json:"result"`
}

// NewOPAClient creates a new OPA HTTP client
func NewOPAClient(baseURL string, timeout time.Duration) (*OPAClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("OPA server URL cannot be empty")
	}

	return &OPAClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		timeout: timeout,
	}, nil
}

// EvaluatePolicy sends a policy evaluation request to OPA server
func (c *OPAClient) EvaluatePolicy(ctx context.Context, policyPath string, input interface{}) ([]byte, error) {
	// Construct the URL for the policy
	url := fmt.Sprintf("%s/v1/data/%s", c.baseURL, policyPath)

	// Create request body
	reqBody := OPARequest{Input: input}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var opaResp OPAResponse
	if err := json.Unmarshal(respBody, &opaResp); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	// Marshal the result back to JSON
	resultBytes, err := json.Marshal(opaResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultBytes, nil
}

// UploadPolicy uploads a Rego policy to OPA server
func (c *OPAClient) UploadPolicy(ctx context.Context, policyName string, policyContent string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, policyName)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader([]byte(policyContent)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload policy, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeletePolicy deletes a policy from OPA server
func (c *OPAClient) DeletePolicy(ctx context.Context, policyName string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, policyName)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete policy, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListPolicies retrieves list of policies from OPA server
func (c *OPAClient) ListPolicies(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/v1/policies", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list policies, status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	policies := make([]string, len(result.Result))
	for i, p := range result.Result {
		policies[i] = p.ID
	}

	return policies, nil
}

// HealthCheck checks if OPA server is reachable and healthy
func (c *OPAClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OPA health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OPA server unhealthy, status: %d", resp.StatusCode)
	}

	return nil
}

// Close closes the HTTP client
func (c *OPAClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
