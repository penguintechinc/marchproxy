//go:build ci

package providers

import (
	"context"
	"testing"
)

func TestNewMistralProvider(t *testing.T) {
	p := NewMistralProvider("test-key", "https://api.mistral.ai", nil)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "mistral" {
		t.Errorf("expected 'mistral', got %q", p.Name())
	}
	if !p.SupportsStreaming() {
		t.Error("mistral should support streaming")
	}
}

func TestMistralProviderWithModels(t *testing.T) {
	models := []string{"mistral-test"}
	p := NewMistralProvider("key", "https://api.mistral.ai", models)
	if len(p.models) != 1 || p.models[0] != "mistral-test" {
		t.Error("models not set correctly")
	}
}

func TestMistralProviderDefaultModels(t *testing.T) {
	p := NewMistralProvider("key", "https://api.mistral.ai", nil)
	if len(p.models) == 0 {
		t.Error("expected default models")
	}
}

func TestMistralChat(t *testing.T) {
	p := NewMistralProvider("invalid-key", "https://api.mistral.ai", []string{"mistral-test"})

	req := &ChatRequest{
		Model: "mistral-test",
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

func TestMistralModels(t *testing.T) {
	p := NewMistralProvider("key", "https://api.mistral.ai", []string{"mistral-1", "mistral-2"})
	models, err := p.Models(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}
