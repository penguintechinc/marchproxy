package health_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/health"
)

func TestHealthHandler(t *testing.T) {
	handler := health.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var response map[string]string
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	status, exists := response["status"]
	if !exists {
		t.Fatal("expected 'status' field in response")
	}
	if status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", status)
	}
}

func TestHealthHandlerMethod(t *testing.T) {
	handler := health.Handler()
	tests := []struct {
		name   string
		method string
		expect bool
	}{
		{"GET", "GET", true},
		{"POST", "POST", true},
		{"PUT", "PUT", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		})
	}
}

func TestHealthHandlerResponse(t *testing.T) {
	handler := health.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty response body")
	}

	if !contains(body, "status") {
		t.Errorf("expected 'status' in response, got %s", body)
	}
	if !contains(body, "healthy") {
		t.Errorf("expected 'healthy' in response, got %s", body)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
