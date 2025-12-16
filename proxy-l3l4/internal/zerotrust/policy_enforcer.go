// Package zerotrust provides zero-trust security features including OPA policy enforcement,
// mTLS verification, and immutable audit logging for Enterprise tier.
package zerotrust

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/rego"
	"github.com/sirupsen/logrus"
)

// PolicyEnforcer handles OPA policy evaluation for zero-trust security
type PolicyEnforcer struct {
	mu            sync.RWMutex
	client        *OPAClient
	localPolicies map[string]*rego.PreparedEvalQuery
	logger        *logrus.Logger
	auditLogger   *AuditLogger
	enabled       bool
	licenseValid  bool
}

// PolicyResult contains the result of a policy evaluation
type PolicyResult struct {
	Allowed     bool                   `json:"allowed"`
	Deny        bool                   `json:"deny,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
	RateLimit   *RateLimitInfo         `json:"rate_limit,omitempty"`
}

// RateLimitInfo contains rate limiting information from policy evaluation
type RateLimitInfo struct {
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Window    string `json:"window"`
}

// PolicyInput represents input data for policy evaluation
type PolicyInput struct {
	Service     string                 `json:"service"`
	User        string                 `json:"user,omitempty"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	SourceIP    string                 `json:"source_ip"`
	Timestamp   time.Time              `json:"timestamp"`
	Certificate *CertificateInfo       `json:"certificate,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CertificateInfo contains client certificate information
type CertificateInfo struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serial_number"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	DNSNames     []string  `json:"dns_names,omitempty"`
}

// NewPolicyEnforcer creates a new policy enforcer instance
func NewPolicyEnforcer(opaServerURL string, logger *logrus.Logger, auditLogger *AuditLogger) (*PolicyEnforcer, error) {
	client, err := NewOPAClient(opaServerURL, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA client: %w", err)
	}

	enforcer := &PolicyEnforcer{
		client:        client,
		localPolicies: make(map[string]*rego.PreparedEvalQuery),
		logger:        logger,
		auditLogger:   auditLogger,
		enabled:       true,
		licenseValid:  false,
	}

	return enforcer, nil
}

// SetLicenseStatus updates the license validation status for Enterprise feature gating
func (pe *PolicyEnforcer) SetLicenseStatus(valid bool) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.licenseValid = valid
	pe.logger.WithField("license_valid", valid).Info("Updated policy enforcer license status")
}

// LoadPolicy loads a policy into the local cache for faster evaluation
func (pe *PolicyEnforcer) LoadPolicy(name string, policyContent string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// Compile the Rego policy
	query, err := rego.New(
		rego.Query("data.marchproxy.allow"),
		rego.Module(name, policyContent),
	).PrepareForEval(context.Background())

	if err != nil {
		return fmt.Errorf("failed to compile policy %s: %w", name, err)
	}

	pe.localPolicies[name] = &query
	pe.logger.WithField("policy", name).Info("Loaded policy into local cache")
	return nil
}

// EvaluatePolicy evaluates a request against OPA policies
func (pe *PolicyEnforcer) EvaluatePolicy(ctx context.Context, policyName string, input *PolicyInput) (*PolicyResult, error) {
	// Check if zero-trust features are enabled and licensed
	pe.mu.RLock()
	enabled := pe.enabled
	licensed := pe.licenseValid
	pe.mu.RUnlock()

	if !enabled {
		pe.logger.Debug("Policy enforcement disabled, allowing request")
		return &PolicyResult{Allowed: true}, nil
	}

	if !licensed {
		return nil, fmt.Errorf("zero-trust features require Enterprise license")
	}

	startTime := time.Now()

	// Try local policy first (faster)
	if result, err := pe.evaluateLocalPolicy(ctx, policyName, input); err == nil {
		pe.logPolicyEvaluation(policyName, input, result, time.Since(startTime), "local")
		return result, nil
	}

	// Fallback to OPA server
	result, err := pe.evaluateRemotePolicy(ctx, policyName, input)
	if err != nil {
		pe.logger.WithError(err).WithField("policy", policyName).Error("Policy evaluation failed")
		return nil, err
	}

	pe.logPolicyEvaluation(policyName, input, result, time.Since(startTime), "remote")
	return result, nil
}

// evaluateLocalPolicy evaluates using locally cached Rego query
func (pe *PolicyEnforcer) evaluateLocalPolicy(ctx context.Context, policyName string, input *PolicyInput) (*PolicyResult, error) {
	pe.mu.RLock()
	query, exists := pe.localPolicies[policyName]
	pe.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("policy %s not found in local cache", policyName)
	}

	// Evaluate the policy
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, fmt.Errorf("policy evaluation error: %w", err)
	}

	if len(results) == 0 {
		return &PolicyResult{
			Allowed: false,
			Deny:    true,
			Reason:  "no policy results returned",
		}, nil
	}

	// Parse the result
	result := &PolicyResult{}

	// Check if allowed
	if len(results) > 0 && len(results[0].Expressions) > 0 {
		if allowed, ok := results[0].Expressions[0].Value.(bool); ok {
			result.Allowed = allowed
			result.Deny = !allowed
		}
	}

	// Extract additional metadata if present
	if len(results[0].Bindings) > 0 {
		if reason, ok := results[0].Bindings["reason"].(string); ok {
			result.Reason = reason
		}
	}

	return result, nil
}

// evaluateRemotePolicy evaluates using OPA server
func (pe *PolicyEnforcer) evaluateRemotePolicy(ctx context.Context, policyName string, input *PolicyInput) (*PolicyResult, error) {
	// Call OPA server
	response, err := pe.client.EvaluatePolicy(ctx, policyName, input)
	if err != nil {
		return nil, err
	}

	var result PolicyResult
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse policy result: %w", err)
	}

	return &result, nil
}

// logPolicyEvaluation logs policy evaluation for audit trail
func (pe *PolicyEnforcer) logPolicyEvaluation(policyName string, input *PolicyInput, result *PolicyResult, duration time.Duration, source string) {
	if pe.auditLogger != nil {
		event := &AuditEvent{
			Timestamp:   time.Now(),
			EventType:   "policy_evaluation",
			Service:     input.Service,
			User:        input.User,
			Action:      input.Action,
			Resource:    input.Resource,
			SourceIP:    input.SourceIP,
			Allowed:     result.Allowed,
			Reason:      result.Reason,
			PolicyName:  policyName,
			Duration:    duration,
			Metadata: map[string]interface{}{
				"source":      source,
				"annotations": result.Annotations,
			},
		}

		if err := pe.auditLogger.LogEvent(event); err != nil {
			pe.logger.WithError(err).Error("Failed to log audit event")
		}
	}

	// Also log to standard logger
	pe.logger.WithFields(logrus.Fields{
		"policy":   policyName,
		"service":  input.Service,
		"action":   input.Action,
		"allowed":  result.Allowed,
		"duration": duration,
		"source":   source,
	}).Info("Policy evaluation completed")
}

// Enable enables policy enforcement
func (pe *PolicyEnforcer) Enable() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.enabled = true
	pe.logger.Info("Policy enforcement enabled")
}

// Disable disables policy enforcement
func (pe *PolicyEnforcer) Disable() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.enabled = false
	pe.logger.Warn("Policy enforcement disabled")
}

// IsEnabled returns whether policy enforcement is enabled
func (pe *PolicyEnforcer) IsEnabled() bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.enabled && pe.licenseValid
}

// Close cleans up resources
func (pe *PolicyEnforcer) Close() error {
	if pe.client != nil {
		return pe.client.Close()
	}
	return nil
}
