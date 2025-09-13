package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/penguintech/marchproxy/internal/manager"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeBase64 AuthType = "base64"
	AuthTypeJWT    AuthType = "jwt"
	AuthTypeNone   AuthType = "none"
)

// JWTClaims represents JWT token claims
type JWTClaims struct {
	ServiceID int    `json:"service_id"`
	ServiceName string `json:"service_name"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// Authenticator handles authentication for proxy connections
type Authenticator struct {
	services map[int]*manager.Service
}

// NewAuthenticator creates a new authenticator with service configuration
func NewAuthenticator(services []manager.Service) *Authenticator {
	serviceMap := make(map[int]*manager.Service)
	for i := range services {
		serviceMap[services[i].ID] = &services[i]
	}
	
	return &Authenticator{
		services: serviceMap,
	}
}

// AuthenticateService authenticates a service using the provided credentials
func (a *Authenticator) AuthenticateService(serviceID int, token string) error {
	service, exists := a.services[serviceID]
	if !exists {
		return fmt.Errorf("service %d not found", serviceID)
	}
	
	switch AuthType(service.AuthType) {
	case AuthTypeBase64:
		return a.validateBase64Token(service, token)
	case AuthTypeJWT:
		return a.validateJWTToken(service, token)
	case AuthTypeNone:
		return nil // No authentication required
	default:
		return fmt.Errorf("unsupported auth type: %s", service.AuthType)
	}
}

// validateBase64Token validates a Base64 encoded token
func (a *Authenticator) validateBase64Token(service *manager.Service, token string) error {
	if service.AuthToken == "" {
		return fmt.Errorf("no Base64 token configured for service %s", service.Name)
	}
	
	// Simple constant-time comparison
	expectedToken := service.AuthToken
	if len(token) != len(expectedToken) {
		return fmt.Errorf("invalid token length for service %s", service.Name)
	}
	
	// Use HMAC for constant-time comparison
	h1 := hmac.New(sha256.New, []byte("comparison"))
	h1.Write([]byte(token))
	mac1 := h1.Sum(nil)
	
	h2 := hmac.New(sha256.New, []byte("comparison"))
	h2.Write([]byte(expectedToken))
	mac2 := h2.Sum(nil)
	
	if !hmac.Equal(mac1, mac2) {
		return fmt.Errorf("invalid Base64 token for service %s", service.Name)
	}
	
	return nil
}

// validateJWTToken validates a JWT token
func (a *Authenticator) validateJWTToken(service *manager.Service, token string) error {
	if service.JWTSecret == "" {
		return fmt.Errorf("no JWT secret configured for service %s", service.Name)
	}
	
	// Split JWT token into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format for service %s", service.Name)
	}
	
	header := parts[0]
	payload := parts[1]
	signature := parts[2]
	
	// Verify signature
	expectedSig, err := a.generateJWTSignature(header+"."+payload, service.JWTSecret)
	if err != nil {
		return fmt.Errorf("failed to generate expected signature: %w", err)
	}
	
	// Constant-time comparison
	expectedSigBytes, err := base64.RawURLEncoding.DecodeString(expectedSig)
	if err != nil {
		return fmt.Errorf("failed to decode expected signature: %w", err)
	}
	
	actualSigBytes, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode provided signature: %w", err)
	}
	
	if !hmac.Equal(expectedSigBytes, actualSigBytes) {
		return fmt.Errorf("invalid JWT signature for service %s", service.Name)
	}
	
	// Decode and validate payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("failed to decode JWT payload: %w", err)
	}
	
	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return fmt.Errorf("failed to parse JWT claims: %w", err)
	}
	
	// Validate claims
	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return fmt.Errorf("JWT token expired for service %s", service.Name)
	}
	
	if claims.ServiceID != service.ID {
		return fmt.Errorf("JWT service ID mismatch for service %s", service.Name)
	}
	
	return nil
}

// generateJWTSignature generates HMAC-SHA256 signature for JWT
func (a *Authenticator) generateJWTSignature(data, secret string) (string, error) {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	signature := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature), nil
}

// UpdateServices updates the authenticator with new service configuration
func (a *Authenticator) UpdateServices(services []manager.Service) {
	serviceMap := make(map[int]*manager.Service)
	for i := range services {
		serviceMap[services[i].ID] = &services[i]
	}
	
	a.services = serviceMap
}

// GetServiceAuthType returns the authentication type for a service
func (a *Authenticator) GetServiceAuthType(serviceID int) (AuthType, error) {
	service, exists := a.services[serviceID]
	if !exists {
		return AuthTypeNone, fmt.Errorf("service %d not found", serviceID)
	}
	
	return AuthType(service.AuthType), nil
}

// GenerateJWTToken generates a JWT token for testing purposes
func (a *Authenticator) GenerateJWTToken(serviceID int) (string, error) {
	service, exists := a.services[serviceID]
	if !exists {
		return "", fmt.Errorf("service %d not found", serviceID)
	}
	
	if service.JWTSecret == "" {
		return "", fmt.Errorf("no JWT secret configured for service %s", service.Name)
	}
	
	// Create header
	header := map[string]interface{}{
		"typ": "JWT",
		"alg": "HS256",
	}
	
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}
	
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	
	// Create claims
	now := time.Now().Unix()
	expiry := now + int64(service.JWTExpiry)
	if service.JWTExpiry == 0 {
		expiry = now + 3600 // Default 1 hour
	}
	
	claims := JWTClaims{
		ServiceID:   service.ID,
		ServiceName: service.Name,
		IssuedAt:    now,
		ExpiresAt:   expiry,
	}
	
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsBytes)
	
	// Generate signature
	signature, err := a.generateJWTSignature(headerEncoded+"."+claimsEncoded, service.JWTSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate signature: %w", err)
	}
	
	return headerEncoded + "." + claimsEncoded + "." + signature, nil
}