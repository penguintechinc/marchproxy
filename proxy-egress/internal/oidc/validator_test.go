package oidc

import (
	"context"
	"testing"
)

func TestValidatorNotConfigured(t *testing.T) {
	v := New()

	// Should return ErrNotConfigured when no provider is set
	err := v.Validate(context.Background(), "")
	if err != ErrNotConfigured {
		t.Errorf("expected ErrNotConfigured, got %v", err)
	}
}

func TestValidatorSetProvider(t *testing.T) {
	v := New()

	cfg := Config{
		IssuerURL: "https://auth.example.com",
		ClientID:  "test-client",
		Audience:  "test-audience",
	}

	v.SetProvider(cfg)

	// After setting provider, validation should fail on invalid token (not configured error)
	err := v.Validate(context.Background(), "invalid.token.format")
	if err == ErrNotConfigured {
		t.Error("expected validation to proceed after provider config, got ErrNotConfigured")
	}
}
