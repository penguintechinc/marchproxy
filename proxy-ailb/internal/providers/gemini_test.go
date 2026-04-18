//go:build ci

package providers

import (
	"context"
	"testing"
)

func TestNewGeminiProvider(t *testing.T) {
	p := NewGeminiProvider("test-key", "https://generativelanguage.googleapis.com", nil)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "gemini" {
		t.Errorf("expected 'gemini', got %q", p.Name())
	}
	if !p.SupportsStreaming() {
		t.Error("gemini should support streaming")
	}
}

func TestGeminiProviderWithModels(t *testing.T) {
	models := []string{"gemini-test"}
	p := NewGeminiProvider("key", "https://api.gemini.com", models)
	if len(p.models) != 1 || p.models[0] != "gemini-test" {
		t.Error("models not set correctly")
	}
}

func TestGeminiProviderDefaultModels(t *testing.T) {
	p := NewGeminiProvider("key", "https://api.gemini.com", nil)
	if len(p.models) == 0 {
		t.Error("expected default models")
	}
}

func TestGeminiChat(t *testing.T) {
	p := NewGeminiProvider("invalid-key", "https://api.gemini.com", []string{"gemini-test"})

	req := &ChatRequest{
		Model: "gemini-test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	}

	resp, err := p.Chat(context.Background(), req)

	if err == nil {
		t.Error("expected error for invalid API key")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}
}

func TestGeminiModels(t *testing.T) {
	p := NewGeminiProvider("key", "https://api.gemini.com", []string{"gemini-1", "gemini-2"})
	models, err := p.Models(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}
