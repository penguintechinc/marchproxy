//go:build ci

package providers

import (
	"context"
	"testing"
)

func TestNewAnthropicProvider(t *testing.T) {
	p := NewAnthropicProvider("test-key", nil)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "anthropic" {
		t.Errorf("expected 'anthropic', got %q", p.Name())
	}
	if !p.SupportsStreaming() {
		t.Error("anthropic should support streaming")
	}
}

func TestAnthropicProviderWithModels(t *testing.T) {
	models := []string{"claude-test"}
	p := NewAnthropicProvider("key", models)
	if len(p.models) != 1 || p.models[0] != "claude-test" {
		t.Error("models not set correctly")
	}
}

func TestAnthropicProviderDefaultModels(t *testing.T) {
	p := NewAnthropicProvider("key", nil)
	if len(p.models) == 0 {
		t.Error("expected default models")
	}
}

func TestAnthropicChat(t *testing.T) {
	p := NewAnthropicProvider("invalid-key", []string{"claude-test"})

	req := &ChatRequest{
		Model: "claude-test",
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

func TestAnthropicModels(t *testing.T) {
	p := NewAnthropicProvider("key", []string{"claude-1", "claude-2"})
	models, err := p.Models(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}
