// Package providers defines the Provider interface and common types
// for LLM provider integrations.
package providers

import (
	"context"
	"time"
)

// Provider is the interface that all LLM provider connectors must implement.
type Provider interface {
	// Name returns the provider identifier (e.g. "openai", "anthropic").
	Name() string

	// Chat sends a chat completion request and returns the response.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// Models returns the list of models available from this provider.
	Models(ctx context.Context) ([]Model, error)

	// SupportsStreaming indicates whether the provider supports streaming responses.
	SupportsStreaming() bool
}

// Message represents a single message in a chat conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request in a provider-agnostic format.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
	User        string    `json:"user,omitempty"`
}

// Usage holds token usage information for a chat completion.
type Usage struct {
	InputTokens  int `json:"prompt_tokens"`
	OutputTokens int `json:"completion_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ChatResponse represents the result of a chat completion request.
type ChatResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	Usage        Usage  `json:"usage"`
	FinishReason string `json:"finish_reason"`
}

// Model represents an LLM model exposed through the provider.
type Model struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Created  int64  `json:"created"`
	OwnedBy  string `json:"owned_by"`
	Provider string `json:"provider,omitempty"`
}

// ProviderError wraps provider-specific errors with metadata.
type ProviderError struct {
	Provider   string
	StatusCode int
	Message    string
	Err        error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Provider + ": " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// DefaultTimeout is the default HTTP timeout for provider API calls.
const DefaultTimeout = 5 * time.Minute
