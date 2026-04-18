//go:build ci
// +build ci

package auth

import (
	"testing"

	"marchproxy-egress/internal/manager"
)

func TestAuthenticator_NewAuthenticator(t *testing.T) {
	tests := []struct {
		name     string
		services []manager.Service
		wantLen  int
	}{
		{
			name:     "empty services",
			services: []manager.Service{},
			wantLen:  0,
		},
		{
			name: "single service",
			services: []manager.Service{
				{ID: 1, Name: "test", AuthType: "base64", AuthToken: "token"},
			},
			wantLen: 1,
		},
		{
			name: "multiple services",
			services: []manager.Service{
				{ID: 1, Name: "svc1", AuthType: "base64"},
				{ID: 2, Name: "svc2", AuthType: "jwt"},
				{ID: 3, Name: "svc3", AuthType: "none"},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthenticator(tt.services)
			if auth == nil {
				t.Fatal("expected non-nil authenticator")
			}
			if len(auth.services) != tt.wantLen {
				t.Errorf("got %d services, want %d", len(auth.services), tt.wantLen)
			}
		})
	}
}

func TestAuthenticator_AuthenticateService(t *testing.T) {
	tests := []struct {
		name      string
		services  []manager.Service
		serviceID int
		token     string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "service not found",
			services:  []manager.Service{},
			serviceID: 1,
			token:     "any",
			wantErr:   true,
			errMsg:    "not found",
		},
		{
			name: "base64 auth success",
			services: []manager.Service{
				{ID: 1, Name: "svc1", AuthType: "base64", AuthToken: "mytoken"},
			},
			serviceID: 1,
			token:     "mytoken",
			wantErr:   false,
		},
		{
			name: "base64 auth wrong token",
			services: []manager.Service{
				{ID: 1, Name: "svc1", AuthType: "base64", AuthToken: "mytoken"},
			},
			serviceID: 1,
			token:     "wrongtoken",
			wantErr:   true,
		},
		{
			name: "base64 auth no token configured",
			services: []manager.Service{
				{ID: 1, Name: "svc1", AuthType: "base64", AuthToken: ""},
			},
			serviceID: 1,
			token:     "anything",
			wantErr:   true,
			errMsg:    "no Base64 token",
		},
		{
			name: "none auth always succeeds",
			services: []manager.Service{
				{ID: 1, Name: "svc1", AuthType: "none"},
			},
			serviceID: 1,
			token:     "",
			wantErr:   false,
		},
		{
			name: "unsupported auth type",
			services: []manager.Service{
				{ID: 1, Name: "svc1", AuthType: "unknown"},
			},
			serviceID: 1,
			token:     "any",
			wantErr:   true,
			errMsg:    "unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthenticator(tt.services)
			err := auth.AuthenticateService(tt.serviceID, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				errStr := err.Error()
				if !contains(errStr, tt.errMsg) {
					t.Errorf("error %q should contain %q", errStr, tt.errMsg)
				}
			}
		})
	}
}

func TestAuthenticator_UpdateServices(t *testing.T) {
	auth := NewAuthenticator([]manager.Service{
		{ID: 1, Name: "old"},
	})

	if len(auth.services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(auth.services))
	}

	newServices := []manager.Service{
		{ID: 2, Name: "new1"},
		{ID: 3, Name: "new2"},
	}
	auth.UpdateServices(newServices)

	if len(auth.services) != 2 {
		t.Errorf("expected 2 services after update, got %d", len(auth.services))
	}
	if _, exists := auth.services[1]; exists {
		t.Error("old service should be removed")
	}
	if _, exists := auth.services[2]; !exists {
		t.Error("new service 2 should exist")
	}
}

func TestAuthenticator_GetServiceAuthType(t *testing.T) {
	auth := NewAuthenticator([]manager.Service{
		{ID: 1, Name: "svc1", AuthType: "jwt"},
		{ID: 2, Name: "svc2", AuthType: "base64"},
	})

	tests := []struct {
		serviceID    int
		wantType     AuthType
		wantErr      bool
	}{
		{1, AuthTypeJWT, false},
		{2, AuthTypeBase64, false},
		{999, AuthTypeNone, true},
	}

	for _, tt := range tests {
		authType, err := auth.GetServiceAuthType(tt.serviceID)
		if (err != nil) != tt.wantErr {
			t.Errorf("service %d: got error %v, wantErr %v", tt.serviceID, err, tt.wantErr)
		}
		if !tt.wantErr && authType != tt.wantType {
			t.Errorf("service %d: got type %s, want %s", tt.serviceID, authType, tt.wantType)
		}
	}
}

func TestAuthenticator_GenerateJWTToken(t *testing.T) {
	auth := NewAuthenticator([]manager.Service{
		{ID: 1, Name: "svc1", JWTSecret: "secret123", JWTExpiry: 3600},
		{ID: 2, Name: "svc2", JWTSecret: ""},
	})

	tests := []struct {
		serviceID int
		wantErr   bool
		errMsg    string
	}{
		{1, false, ""},
		{2, true, "no JWT secret"},
		{999, true, "not found"},
	}

	for _, tt := range tests {
		token, err := auth.GenerateJWTToken(tt.serviceID)
		if (err != nil) != tt.wantErr {
			t.Errorf("service %d: got error %v, wantErr %v", tt.serviceID, err, tt.wantErr)
		}
		if !tt.wantErr && token == "" {
			t.Errorf("service %d: expected non-empty token", tt.serviceID)
		}
		if tt.wantErr && tt.errMsg != "" && err != nil {
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("service %d: error %q should contain %q", tt.serviceID, err.Error(), tt.errMsg)
			}
		}
	}
}

func TestAuthenticator_JWT_Integration(t *testing.T) {
	auth := NewAuthenticator([]manager.Service{
		{ID: 1, Name: "svc1", AuthType: "jwt", JWTSecret: "secret123", JWTExpiry: 3600},
	})

	// Generate a token
	token, err := auth.GenerateJWTToken(1)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Authenticate with the generated token
	err = auth.AuthenticateService(1, token)
	if err != nil {
		t.Errorf("authentication failed: %v", err)
	}
}

func TestAuthenticator_Base64_TokenLength(t *testing.T) {
	auth := NewAuthenticator([]manager.Service{
		{ID: 1, Name: "svc1", AuthType: "base64", AuthToken: "token"},
	})

	// Short token
	err := auth.AuthenticateService(1, "sh")
	if err == nil {
		t.Error("expected error for short token")
	}

	// Long token
	err = auth.AuthenticateService(1, "this_is_a_very_long_token_that_does_not_match")
	if err == nil {
		t.Error("expected error for long token")
	}
}

func TestAuthenticator_ConcurrentAccess(t *testing.T) {
	services := make([]manager.Service, 10)
	for i := 0; i < 10; i++ {
		services[i] = manager.Service{
			ID:        i + 1,
			Name:      "svc" + string(rune(i)),
			AuthType:  "none",
		}
	}
	auth := NewAuthenticator(services)

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			serviceID := (idx % 10) + 1
			_ = auth.AuthenticateService(serviceID, "")
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
