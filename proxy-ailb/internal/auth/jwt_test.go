package auth_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/auth"
)

func TestClaimsHasScope(t *testing.T) {
	tests := []struct {
		name     string
		claims   auth.Claims
		scope    string
		expected bool
	}{
		{
			"single scope match",
			auth.Claims{Scope: "read"},
			"read",
			true,
		},
		{
			"multiple scopes match first",
			auth.Claims{Scope: "read write admin"},
			"read",
			true,
		},
		{
			"multiple scopes match middle",
			auth.Claims{Scope: "read write admin"},
			"write",
			true,
		},
		{
			"multiple scopes match last",
			auth.Claims{Scope: "read write admin"},
			"admin",
			true,
		},
		{
			"scope not present",
			auth.Claims{Scope: "read write"},
			"delete",
			false,
		},
		{
			"empty scope",
			auth.Claims{Scope: ""},
			"read",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.claims.HasScope(tt.scope)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClaimsIsExpired(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name     string
		expTime  int64
		expected bool
	}{
		{"expired", now - 3600, true},
		{"not expired", now + 3600, false},
		{"just expired", now - 1, true},
		{"just not expired", now + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &auth.Claims{Exp: tt.expTime}
			result := claims.IsExpired()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	secret := "test-secret-key"
	validator := auth.NewValidator(secret)
	if validator == nil {
		t.Fatal("expected validator to be created, got nil")
	}
}

func TestValidateRequestNoHeader(t *testing.T) {
	validator := auth.NewValidator("secret")
	req := httptest.NewRequest("GET", "/test", nil)

	_, err := validator.ValidateRequest(req)
	if err != auth.ErrNoAuthHeader {
		t.Errorf("expected ErrNoAuthHeader, got %v", err)
	}
}

func TestValidateRequestInvalidFormat(t *testing.T) {
	validator := auth.NewValidator("secret")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")

	_, err := validator.ValidateRequest(req)
	if err != auth.ErrInvalidFormat {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestValidateRequestInvalidBearerFormat(t *testing.T) {
	validator := auth.NewValidator("secret")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer")

	_, err := validator.ValidateRequest(req)
	if err != auth.ErrInvalidFormat {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestValidateInvalidTokenStructure(t *testing.T) {
	validator := auth.NewValidator("secret")
	_, err := validator.Validate("invalid.token")
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateInvalidSignature(t *testing.T) {
	validator := auth.NewValidator("secret")
	// Create a token with mismatched signature
	invalidToken := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.invalid_signature"
	_, err := validator.Validate(invalidToken)
	if err != auth.ErrInvalidSig {
		t.Errorf("expected ErrInvalidSig, got %v", err)
	}
}

func TestValidateValidToken(t *testing.T) {
	secret := "test-secret-key"
	validator := auth.NewValidator(secret)

	// Create a valid token manually
	claims := auth.Claims{
		Sub:    "user123",
		Iss:    "issuer",
		Aud:    []string{"audience"},
		Iat:    time.Now().Unix(),
		Exp:    time.Now().Add(1 * time.Hour).Unix(),
		Scope:  "read write",
		Tenant: "tenant-123",
		Teams:  []string{"team1"},
		Roles:  []string{"admin"},
	}

	token := createTestToken(t, secret, &claims)
	result, err := validator.Validate(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Sub != "user123" {
		t.Errorf("expected Sub user123, got %s", result.Sub)
	}
	if result.Scope != "read write" {
		t.Errorf("expected Scope 'read write', got %s", result.Scope)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	secret := "test-secret-key"
	validator := auth.NewValidator(secret)

	claims := auth.Claims{
		Sub: "user123",
		Exp: time.Now().Add(-1 * time.Hour).Unix(),
	}

	token := createTestToken(t, secret, &claims)
	_, err := validator.Validate(token)
	if err != auth.ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateRequestWithValidToken(t *testing.T) {
	secret := "test-secret-key"
	validator := auth.NewValidator(secret)

	claims := auth.Claims{
		Sub:    "user123",
		Exp:    time.Now().Add(1 * time.Hour).Unix(),
		Tenant: "tenant-123",
	}

	token := createTestToken(t, secret, &claims)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	result, err := validator.ValidateRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Sub != "user123" {
		t.Errorf("expected Sub user123, got %s", result.Sub)
	}
}

func TestValidatorWithEmptySecret(t *testing.T) {
	validator := auth.NewValidator("")
	_, err := validator.Validate("any.token.here")
	// Should fail because secret is empty
	if err == nil {
		t.Fatal("expected error for empty secret, got nil")
	}
}

func TestValidateMalformedClaims(t *testing.T) {
	validator := auth.NewValidator("secret")
	// Create token with invalid base64 in claims
	invalidToken := "eyJhbGciOiJIUzI1NiJ9.!!!invalid_base64!!!.sig"
	_, err := validator.Validate(invalidToken)
	if err == nil {
		t.Fatal("expected error for malformed claims, got nil")
	}
}

func TestValidateBearerCaseInsensitive(t *testing.T) {
	secret := "test-secret-key"
	validator := auth.NewValidator(secret)

	claims := auth.Claims{
		Sub: "user123",
		Exp: time.Now().Add(1 * time.Hour).Unix(),
	}

	token := createTestToken(t, secret, &claims)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "bearer "+token) // lowercase

	result, err := validator.ValidateRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Sub != "user123" {
		t.Errorf("expected Sub user123, got %s", result.Sub)
	}
}

func TestClaimsDefaultValues(t *testing.T) {
	claims := &auth.Claims{}
	if claims.HasScope("any") {
		t.Error("expected HasScope to return false for empty scope")
	}
	// Note: zero Exp (Exp=0) is in the past, so IsExpired() returns true
	// This is expected behavior - tokens must have valid expiration
	if !claims.IsExpired() {
		t.Error("expected IsExpired to return true for zero expiration")
	}
}

func createTestToken(t *testing.T, secret string, claims *auth.Claims) string {
	// Create header
	header := map[string]string{"alg": "HS256"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Create claims
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Compute signature (simplified HMAC-SHA256)
	sigInput := headerB64 + "." + claimsB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return sigInput + "." + sig
}
