//go:build ci

package providers

import (
	"context"
	"testing"
)

func TestNewLlamaCppProvider(t *testing.T) {
	p := NewLlamaCppProvider("http://localhost:8000", nil)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "llamacpp" {
		t.Errorf("expected 'llamacpp', got %q", p.Name())
	}
}

func TestLlamaCppProviderWithModels(t *testing.T) {
	models := []string{"llama-test"}
	p := NewLlamaCppProvider("http://localhost:8000", models)
	if len(p.models) != 1 || p.models[0] != "llama-test" {
		t.Error("models not set correctly")
	}
}

func TestLlamaCppProviderDefaultModels(t *testing.T) {
	p := NewLlamaCppProvider("http://localhost:8000", nil)
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestLlamaCppChat(t *testing.T) {
	p := NewLlamaCppProvider("http://invalid:99999", []string{"llama-test"})

	req := &ChatRequest{
		Model: "llama-test",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	}

	resp, err := p.Chat(context.Background(), req)

	if err == nil {
		t.Error("expected error for unreachable server")
	}
	if resp != nil {
		t.Error("expected nil response on error")
	}
}

func TestLlamaCppModels(t *testing.T) {
	p := NewLlamaCppProvider("http://localhost:8000", []string{"llama-1", "llama-2"})
	models, err := p.Models(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
}
