//go:build ci

package providers

import (
	"context"
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	p := NewOpenAIProvider("openai", "test-key", "https://api.openai.com", nil)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "openai" {
		t.Errorf("expected 'openai', got %q", p.Name())
	}
	if !p.SupportsStreaming() {
		t.Error("openai should support streaming")
	}
}

func TestOpenAIProviderWithModels(t *testing.T) {
	models := []string{"gpt-test"}
	p := NewOpenAIProvider("openai", "key", "https://api.openai.com", models)
	if len(p.models) != 1 || p.models[0] != "gpt-test" {
		t.Error("models not set correctly")
	}
}

func TestOpenAIProviderDefaultModels(t *testing.T) {
	p := NewOpenAIProvider("openai", "key", "https://api.openai.com", nil)
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestOpenAIChat(t *testing.T) {
	p := NewOpenAIProvider("openai", "invalid-key", "https://api.openai.com", []string{"gpt-test"})

	req := &ChatRequest{
		Model: "gpt-test",
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

func TestOpenAIModels(t *testing.T) {
	p := NewOpenAIProvider("openai", "key", "https://api.openai.com", []string{"gpt-1", "gpt-2"})
	models, err := p.Models(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}
