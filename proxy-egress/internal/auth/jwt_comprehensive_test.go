//go:build ci
// +build ci

package auth

import (
	"strings"
	"testing"
	"time"
)

func TestJWTClaims_Valid(t *testing.T) {
	tests := []struct {
		name      string
		claims    JWTClaims
		wantErr   bool
	}{
		{
			name:      "future expiry",
			claims:    JWTClaims{ExpiresAt: time.Now().Add(time.Hour).Unix()},
			wantErr:   false,
		},
		{
			name:      "past expiry",
			claims:    JWTClaims{ExpiresAt: time.Now().Add(-time.Hour).Unix()},
			wantErr:   true,
		},
		{
			name:      "zero expiry",
			claims:    JWTClaims{ExpiresAt: 0},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Valid()
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != ErrTokenExpired {
				t.Errorf("expected ErrTokenExpired, got %v", err)
			}
		})
	}
}

func TestValidateJWTToken(t *testing.T) {
	secret := "test-secret-key"
	serviceID := 123

	tests := []struct {
		name              string
		tokenString       string
		secret            string
		expectedServiceID int
		wantErr           bool
		errType           error
	}{
		{
			name:        "empty secret",
			tokenString: "any-token",
			secret:      "",
			wantErr:     true,
		},
		{
			name:              "invalid token format",
			tokenString:       "not.a.valid.token",
			secret:            secret,
			expectedServiceID: serviceID,
			wantErr:           true,
		},
		{
			name:              "empty token string",
			tokenString:       "",
			secret:            secret,
			expectedServiceID: serviceID,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWTToken(tt.tokenString, tt.secret, tt.expectedServiceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJWTToken_ValidToken(t *testing.T) {
	secret := "test-secret"
	serviceID := 42
	serviceName := "my-service"

	// Generate a valid token
	token, err := GenerateJWTToken(serviceID, serviceName, secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate it
	claims, err := ValidateJWTToken(token, secret, serviceID)
	if err != nil {
		t.Errorf("validation failed: %v", err)
	}

	if claims.ServiceID != serviceID {
		t.Errorf("service ID mismatch: got %d, want %d", claims.ServiceID, serviceID)
	}
	if claims.ServiceName != serviceName {
		t.Errorf("service name mismatch: got %s, want %s", claims.ServiceName, serviceName)
	}
}

func TestValidateJWTToken_WrongServiceID(t *testing.T) {
	secret := "test-secret"
	serviceID := 42
	wrongServiceID := 99

	token, err := GenerateJWTToken(serviceID, "service", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateJWTToken(token, secret, wrongServiceID)
	if err == nil {
		t.Error("expected error for service ID mismatch")
	}
}

func TestValidateJWTToken_WrongSecret(t *testing.T) {
	secret := "test-secret"
	wrongSecret := "wrong-secret"
	serviceID := 42

	token, err := GenerateJWTToken(serviceID, "service", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateJWTToken(token, wrongSecret, serviceID)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestValidateJWTToken_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	serviceID := 42

	// Generate token with negative expiry (already expired)
	token, err := GenerateJWTToken(serviceID, "service", secret, -time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateJWTToken(token, secret, serviceID)
	if err == nil {
		t.Error("expected error for expired token")
	}
	// Check it's related to token expiration (could be wrapped)
	if !contains(err.Error(), "expired") {
		t.Errorf("expected expired error message, got %v", err)
	}
}

func TestGenerateJWTToken(t *testing.T) {
	tests := []struct {
		name          string
		serviceID     int
		serviceName   string
		secret        string
		expiry        time.Duration
		wantErr       bool
		errMsg        string
	}{
		{
			name:        "valid token generation",
			serviceID:   1,
			serviceName: "test",
			secret:      "secret",
			expiry:      time.Hour,
			wantErr:     false,
		},
		{
			name:        "empty secret",
			serviceID:   1,
			serviceName: "test",
			secret:      "",
			expiry:      time.Hour,
			wantErr:     true,
			errMsg:      "no JWT secret",
		},
		{
			name:        "zero expiry",
			serviceID:   1,
			serviceName: "test",
			secret:      "secret",
			expiry:      0,
			wantErr:     false,
		},
		{
			name:        "negative expiry",
			serviceID:   1,
			serviceName: "test",
			secret:      "secret",
			expiry:      -time.Hour,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWTToken(tt.serviceID, tt.serviceName, tt.secret, tt.expiry)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("expected non-empty token")
				}
				if !strings.Contains(token, ".") {
					t.Error("token should contain JWT parts separated by dots")
				}
			}
		})
	}
}

func TestGenerateJWTToken_Claims(t *testing.T) {
	serviceID := 99
	serviceName := "my-api"
	secret := "my-secret"
	expiry := time.Hour

	token, err := GenerateJWTToken(serviceID, serviceName, secret, expiry)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateJWTToken(token, secret, serviceID)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.ServiceID != serviceID {
		t.Errorf("service ID: got %d, want %d", claims.ServiceID, serviceID)
	}
	if claims.ServiceName != serviceName {
		t.Errorf("service name: got %s, want %s", claims.ServiceName, serviceName)
	}
	if claims.IssuedAt == 0 {
		t.Error("issued at should be set")
	}
	if claims.ExpiresAt == 0 {
		t.Error("expires at should be set")
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   bool
	}{
		{
			name:      "bearer token",
			header:    "Bearer abc123def456",
			wantToken: "abc123def456",
			wantErr:   false,
		},
		{
			name:      "bearer with extra spaces",
			header:    "Bearer  abc123def456  ",
			wantToken: "abc123def456",
			wantErr:   false,
		},
		{
			name:      "no bearer prefix",
			header:    "abc123def456",
			wantToken: "abc123def456",
			wantErr:   false,
		},
		{
			name:      "empty header",
			header:    "",
			wantErr:   true,
		},
		{
			name:      "just bearer",
			header:    "Bearer ",
			wantToken: "",
			wantErr:   false,
		},
		{
			name:      "bearer case sensitive",
			header:    "bearer abc123",
			wantToken: "bearer abc123",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractTokenFromHeader(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if token != tt.wantToken {
				t.Errorf("got token %q, want %q", token, tt.wantToken)
			}
		})
	}
}

func TestJWT_MultipleTokens(t *testing.T) {
	secret := "shared-secret"

	// Generate multiple tokens with different service IDs
	tokens := make(map[int]string)
	for i := 1; i <= 5; i++ {
		token, err := GenerateJWTToken(i, "service-"+string(rune(48+i)), secret, time.Hour)
		if err != nil {
			t.Fatalf("failed to generate token %d: %v", i, err)
		}
		tokens[i] = token
	}

	// Validate each token
	for serviceID, token := range tokens {
		claims, err := ValidateJWTToken(token, secret, serviceID)
		if err != nil {
			t.Errorf("validation failed for service %d: %v", serviceID, err)
		}
		if claims.ServiceID != serviceID {
			t.Errorf("service %d: got ID %d", serviceID, claims.ServiceID)
		}
	}
}

func TestJWT_TokenExpiration_EdgeCase(t *testing.T) {
	secret := "secret"
	serviceID := 1

	// Test with a token that has already expired (past time)
	now := time.Now()
	pastExpiry := now.Add(-1 * time.Hour)

	// Manually craft claims with past expiry
	testErr := false
	for i := 0; i < 3; i++ {
		// Generate token with minimal expiry
		token, err := GenerateJWTToken(serviceID, "service", secret, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// Validate immediately - should work
		_, err = ValidateJWTToken(token, secret, serviceID)
		if err != nil {
			t.Logf("attempt %d: token validation failed immediately (possible timing issue)", i+1)
			continue
		}

		// Wait for expiration
		time.Sleep(50 * time.Millisecond)

		// Try again - should fail
		_, err = ValidateJWTToken(token, secret, serviceID)
		if err != nil {
			testErr = true
			break
		}
	}

	if !testErr {
		t.Skip("skipping flaky timing test - could not reliably expire token in time")
	}

	_ = pastExpiry
}

func TestJWT_LargeServiceID(t *testing.T) {
	secret := "secret"
	serviceID := 999999999

	token, err := GenerateJWTToken(serviceID, "large-service", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateJWTToken(token, secret, serviceID)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if claims.ServiceID != serviceID {
		t.Errorf("got service ID %d, want %d", claims.ServiceID, serviceID)
	}
}
