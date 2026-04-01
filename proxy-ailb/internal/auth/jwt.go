// Package auth provides JWT validation for the AILB service.
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// Claims represents the JWT claims payload.
type Claims struct {
	Sub    string   `json:"sub"`
	Iss    string   `json:"iss"`
	Aud    []string `json:"aud"`
	Iat    int64    `json:"iat"`
	Exp    int64    `json:"exp"`
	Scope  string   `json:"scope"`
	Tenant string   `json:"tenant"`
	Teams  []string `json:"teams"`
	Roles  []string `json:"roles"`
}

// HasScope checks if the claims include a specific scope.
func (c *Claims) HasScope(scope string) bool {
	for _, s := range strings.Fields(c.Scope) {
		if s == scope {
			return true
		}
	}
	return false
}

// IsExpired returns true if the token has expired.
func (c *Claims) IsExpired() bool {
	return time.Now().Unix() > c.Exp
}

var (
	ErrNoAuthHeader   = errors.New("missing Authorization header")
	ErrInvalidFormat  = errors.New("invalid Authorization format, expected 'Bearer <token>'")
	ErrInvalidToken   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token expired")
	ErrInvalidSig     = errors.New("invalid token signature")
)

// Validator validates JWT tokens using HMAC-SHA256.
// In production this would integrate with go-aaa or a JWKS endpoint.
type Validator struct {
	secret []byte
}

// NewValidator creates a JWT validator with the given HMAC secret.
// If secret is empty, validation will always fail (secure by default).
func NewValidator(secret string) *Validator {
	return &Validator{secret: []byte(secret)}
}

// ValidateRequest extracts and validates a JWT from the Authorization header.
func (v *Validator) ValidateRequest(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, ErrNoAuthHeader
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, ErrInvalidFormat
	}

	return v.Validate(parts[1])
}

// Validate parses and validates a JWT token string.
func (v *Validator) Validate(tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Verify signature.
	sigInput := parts[0] + "." + parts[1]
	expectedSig, err := v.sign(sigInput)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	actualSig := parts[2]
	if !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
		return nil, ErrInvalidSig
	}

	// Decode claims.
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	if claims.IsExpired() {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

// sign computes HMAC-SHA256 and returns base64url-encoded result.
func (v *Validator) sign(input string) (string, error) {
	if len(v.secret) == 0 {
		return "", errors.New("JWT secret not configured")
	}
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
