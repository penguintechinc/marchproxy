//go:build ci

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/auth"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/billing"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

func TestNewChatHandler(t *testing.T) {
	reg := providers.NewRegistry()
	rtr := router.New(reg, router.StrategyRoundRobin)
	reporter := billing.NewReporter("127.0.0.1:5000")

	h := NewChatHandler(reg, rtr, reporter, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.registry != reg {
		t.Error("registry not set correctly")
	}
}

func TestChatHandlerSetWaddleAI(t *testing.T) {
	h := &ChatHandler{}
	client := &providers.WaddleAIGRPCClient{}
	h.SetWaddleAI(client, true)

	if h.waddleAI != client {
		t.Error("waddleAI not set correctly")
	}
	if !h.routingAI {
		t.Error("routingAI not set correctly")
	}
}

func TestChatHandlerMethodNotAllowed(t *testing.T) {
	h := &ChatHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)

	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestChatHandlerInvalidJSON(t *testing.T) {
	h := &ChatHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte("invalid json")))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
	var errResp errorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error.Code == "" {
		t.Error("error code should be set")
	}
}

func TestChatHandlerMissingModel(t *testing.T) {
	h := &ChatHandler{}
	w := httptest.NewRecorder()

	req := chatCompletionRequest{
		Messages: []messageRequest{{Role: "user", Content: "hello"}},
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestChatHandlerMissingMessages(t *testing.T) {
	h := &ChatHandler{}
	w := httptest.NewRecorder()

	req := chatCompletionRequest{
		Model: "test-model",
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestClaimsFromContext(t *testing.T) {
	claims := &auth.Claims{Sub: "test-user"}
	ctx := context.WithValue(context.Background(), contextKey("claims"), claims)

	result := ClaimsFromContext(ctx)
	if result != claims {
		t.Error("claims not extracted correctly")
	}

	result = ClaimsFromContext(context.Background())
	if result != nil {
		t.Error("expected nil for context without claims")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "test message", "test_type")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp errorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error.Message != "test message" {
		t.Errorf("expected 'test message', got %q", errResp.Error.Message)
	}
	if errResp.Error.Type != "test_type" {
		t.Errorf("expected 'test_type', got %q", errResp.Error.Type)
	}
}
