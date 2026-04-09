// Package levers implements the MarchProxy levers API — a simple push-receiver
// for compiled rule sets from an external controller (e.g., hub-policy).
// MarchProxy enforces received rules locally; it has no policy evaluation logic.
package levers

import (
	"encoding/json"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
	"marchproxy-egress/internal/oidc"
)

// RuleSet is the compiled, flat rule set pushed by the controller.
// All entries are simple instructions — no evaluation logic in the proxy.
type RuleSet struct {
	BlockCIDRs    []string       `json:"block_cidrs"`
	AllowCIDRs    []string       `json:"allow_cidrs"`
	BlockDomains  []string       `json:"block_domains"`
	AllowDomains  []string       `json:"allow_domains"`
	RouteClusters []ClusterDef   `json:"route_clusters"`
	RateLimits    map[string]int `json:"rate_limits"` // src_ip → pps
	OIDCProvider  *OIDCConfig    `json:"oidc_provider,omitempty"`
}

type ClusterDef struct {
	Name      string   `json:"name"`
	Endpoints []string `json:"endpoints"`
	LBPolicy  string   `json:"lb_policy"` // ROUND_ROBIN, LEAST_REQUEST, WEIGHTED
}

type OIDCConfig struct {
	IssuerURL string `json:"issuer_url"`
	ClientID  string `json:"client_id"`
	Audience  string `json:"audience"`
}

// Server receives rule pushes from the controller and applies them via the Enforcer.
type Server struct {
	enforcer  *Enforcer
	validator *oidc.Validator
	mu        sync.RWMutex
	current   RuleSet
}

func NewServer(enforcer *Enforcer, validator *oidc.Validator) *Server {
	return &Server{enforcer: enforcer, validator: validator}
}

// RegisterRoutes registers the levers API endpoints on the given mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/levers/rules", s.handleRules)
	mux.HandleFunc("/api/v1/levers/status", s.handleStatus)
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rules RuleSet
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		log.WithError(err).Error("levers: failed to decode rule set")
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := s.enforcer.Apply(rules); err != nil {
		log.WithError(err).Error("levers: failed to apply rule set")
		http.Error(w, "apply failed", http.StatusInternalServerError)
		return
	}

	// Configure OIDC provider if present in rules
	if rules.OIDCProvider != nil && s.validator != nil {
		s.validator.SetProvider(oidc.Config{
			IssuerURL: rules.OIDCProvider.IssuerURL,
			ClientID:  rules.OIDCProvider.ClientID,
			Audience:  rules.OIDCProvider.Audience,
		})
		log.WithField("issuer", rules.OIDCProvider.IssuerURL).Info("levers: configured OIDC provider")
	}

	s.mu.Lock()
	s.current = rules
	s.mu.Unlock()

	log.WithFields(log.Fields{
		"block_cidrs":    len(rules.BlockCIDRs),
		"allow_cidrs":    len(rules.AllowCIDRs),
		"block_domains":  len(rules.BlockDomains),
		"allow_domains":  len(rules.AllowDomains),
		"route_clusters": len(rules.RouteClusters),
		"rate_limits":    len(rules.RateLimits),
		"oidc_provider":  rules.OIDCProvider != nil,
	}).Info("levers: applied rule set")

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"block_cidrs":    len(s.current.BlockCIDRs),
		"allow_cidrs":    len(s.current.AllowCIDRs),
		"block_domains":  len(s.current.BlockDomains),
		"route_clusters": len(s.current.RouteClusters),
	})
}
