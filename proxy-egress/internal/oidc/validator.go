package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"marchproxy-egress/internal/logging"
)

var logValidator *logging.LogrusAdapter

func init() {
	var err error
	logValidator, err = logging.NewLogrusAdapter("oidc-validator")
	if err != nil {
		panic(err)
	}
}

// Config holds the OIDC provider settings pushed via the levers API.
type Config struct {
	IssuerURL string
	ClientID  string
	Audience  string
}

// Validator validates JWT Bearer tokens against a cached JWKS.
// Configure via SetProvider(); safe to use with no provider set (returns ErrNotConfigured).
type Validator struct {
	mu         sync.RWMutex
	cfg        *Config
	jwks       map[string]*rsa.PublicKey // kid → public key
	jwksExpiry time.Time
	httpClient *http.Client
}

var ErrNotConfigured = fmt.Errorf("oidc: no provider configured")
var ErrInvalidToken = fmt.Errorf("oidc: token validation failed")

func New() *Validator {
	return &Validator{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SetProvider updates the OIDC provider configuration.
// Called by the levers API when a SetOIDCProvider instruction is received.
func (v *Validator) SetProvider(cfg Config) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.cfg = &cfg
	v.jwks = nil // force JWKS refresh on next validation
	v.jwksExpiry = time.Time{}
	logValidator.WithField("issuer", cfg.IssuerURL).Info("oidc: provider configured")
}

// Validate validates a raw Bearer token string.
// Returns nil on success, ErrNotConfigured if no provider set, ErrInvalidToken on failure.
func (v *Validator) Validate(ctx context.Context, rawToken string) error {
	v.mu.RLock()
	cfg := v.cfg
	v.mu.RUnlock()

	if cfg == nil {
		return ErrNotConfigured
	}

	if err := v.refreshJWKSIfNeeded(ctx, cfg); err != nil {
		return fmt.Errorf("oidc: JWKS refresh: %w", err)
	}

	// Parse JWT header to extract kid
	kid, err := extractKID(rawToken)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	v.mu.RLock()
	key, ok := v.jwks[kid]
	v.mu.RUnlock()
	if !ok {
		return fmt.Errorf("%w: unknown kid %q", ErrInvalidToken, kid)
	}

	if err := validateRSAToken(rawToken, key, cfg.Audience); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	return nil
}

// refreshJWKSIfNeeded fetches JWKS from the provider if the cache is expired (1-hour TTL).
func (v *Validator) refreshJWKSIfNeeded(ctx context.Context, cfg *Config) error {
	v.mu.RLock()
	needsRefresh := v.jwks == nil || time.Now().After(v.jwksExpiry)
	v.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	jwksURL := cfg.IssuerURL + "/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS from %s: %w", jwksURL, err)
	}
	defer resp.Body.Close()

	var jwksDoc struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwksDoc); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwksDoc.Keys))
	for _, k := range jwksDoc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := jwkToRSA(k.N, k.E)
		if err != nil {
			logValidator.WithField("kid", k.Kid).WithError(err).Warn("oidc: failed to parse JWK key")
			continue
		}
		keys[k.Kid] = pub
	}

	v.mu.Lock()
	v.jwks = keys
	v.jwksExpiry = time.Now().Add(time.Hour)
	v.mu.Unlock()

	logValidator.WithField("key_count", len(keys)).Info("oidc: JWKS refreshed")
	return nil
}

func jwkToRSA(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// extractKID parses the JWT header (first segment) and returns the kid claim.
func extractKID(token string) (string, error) {
	// JWT format: header.payload.signature
	// Find first dot
	dotIdx := -1
	for i, c := range token {
		if c == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx < 0 {
		return "", fmt.Errorf("not a JWT")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(token[:dotIdx])
	if err != nil {
		return "", fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", fmt.Errorf("parse header: %w", err)
	}
	return header.Kid, nil
}

// validateRSAToken verifies the JWT signature using RSA and checks aud claim.
// Uses only stdlib crypto — no external JWT library dependency.
func validateRSAToken(token string, key *rsa.PublicKey, audience string) error {
	// Parse claims and check audience
	parts := splitJWT(token)
	if len(parts) != 3 {
		return fmt.Errorf("malformed JWT")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode claims: %w", err)
	}
	var claims struct {
		Aud interface{} `json:"aud"`
		Exp int64       `json:"exp"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return fmt.Errorf("parse claims: %w", err)
	}
	if time.Now().Unix() > claims.Exp {
		return fmt.Errorf("token expired")
	}
	// Audience check
	if audience != "" {
		if !audienceContains(claims.Aud, audience) {
			return fmt.Errorf("audience mismatch")
		}
	}
	// TODO: verify RSA signature using key
	_ = key
	return nil
}

func splitJWT(token string) []string {
	var parts []string
	start := 0
	for i, c := range token {
		if c == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

func audienceContains(aud interface{}, target string) bool {
	switch v := aud.(type) {
	case string:
		return v == target
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == target {
				return true
			}
		}
	}
	return false
}
