//go:build ci

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

func TestNewOllamaHandler(t *testing.T) {
	reg := providers.NewRegistry()
	rtr := router.New(reg, router.StrategyRoundRobin)

	h := NewOllamaHandler(reg, rtr, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.registry != reg {
		t.Error("registry not set correctly")
	}
	if h.router != rtr {
		t.Error("router not set correctly")
	}
}

func TestOllamaChatHandlerMethodNotAllowed(t *testing.T) {
	h := &OllamaHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/chat", nil)

	handler := h.ChatHandler()
	handler(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestOllamaChatHandlerInvalidJSON(t *testing.T) {
	h := &OllamaHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader([]byte("bad json")))

	handler := h.ChatHandler()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateHandlerMethodNotAllowed(t *testing.T) {
	h := &OllamaHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/generate", nil)

	handler := h.GenerateHandler()
	handler(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestGenerateHandlerInvalidJSON(t *testing.T) {
	h := &OllamaHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewReader([]byte("bad json")))

	handler := h.GenerateHandler()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestTagsHandlerMethodNotAllowed(t *testing.T) {
	h := &OllamaHandler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/tags", nil)

	handler := h.TagsHandler()
	handler(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

