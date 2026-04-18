package oidc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareNotConfigured(t *testing.T) {
	validator := New()
	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should pass through when not configured and no token provided
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMiddlewareMissingAuthHeader(t *testing.T) {
	validator := New()
	validator.SetProvider(Config{
		IssuerURL: "https://auth.example.com",
		ClientID:  "test-client",
		Audience:  "test-api",
	})

	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 401 when provider configured but no token provided
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestMiddlewareMissingBearer(t *testing.T) {
	validator := New()
	validator.SetProvider(Config{
		IssuerURL: "https://auth.example.com",
		ClientID:  "test-client",
		Audience:  "test-api",
	})

	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNzd2Q=")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 401 when Authorization header is not Bearer
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestMiddlewareInvalidToken(t *testing.T) {
	validator := New()
	validator.SetProvider(Config{
		IssuerURL: "https://auth.example.com",
		ClientID:  "test-client",
		Audience:  "test-api",
	})

	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 401 for invalid token
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestMiddlewarePassesContextWithoutProvider(t *testing.T) {
	validator := New()

	called := false
	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected next handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMiddlewareBearerPrefixExtraction(t *testing.T) {
	validator := New()
	validator.SetProvider(Config{
		IssuerURL: "https://auth.example.com",
		ClientID:  "test-client",
		Audience:  "test-api",
	})

	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
		code   int
	}{
		{
			name:   "bearer with space",
			header: "Bearer token123",
			code:   http.StatusUnauthorized,
		},
		{
			name:   "bearer lowercase",
			header: "bearer token123",
			code:   http.StatusUnauthorized,
		},
		{
			name:   "no bearer prefix",
			header: "token123",
			code:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.code {
				t.Errorf("expected status %d, got %d", tt.code, w.Code)
			}
		})
	}
}
