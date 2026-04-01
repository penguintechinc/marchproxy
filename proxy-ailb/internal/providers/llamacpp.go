package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LlamaCppProvider implements the Provider interface for llama.cpp's HTTP server.
// llama.cpp exposes an OpenAI-compatible /v1/chat/completions endpoint.
type LlamaCppProvider struct {
	baseURL string
	models  []string
	client  *http.Client
}

// NewLlamaCppProvider creates a provider for llama.cpp server.
func NewLlamaCppProvider(baseURL string, models []string) *LlamaCppProvider {
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	return &LlamaCppProvider{
		baseURL: baseURL,
		models:  models,
		client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (p *LlamaCppProvider) Name() string            { return "llamacpp" }
func (p *LlamaCppProvider) SupportsStreaming() bool  { return true }

func (p *LlamaCppProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// llama.cpp server uses OpenAI-compatible format.
	msgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body := openAIRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   false,
		Stop:     req.Stop,
	}
	if req.Temperature > 0 {
		t := req.Temperature
		body.Temperature = &t
	}
	if req.MaxTokens > 0 {
		m := req.MaxTokens
		body.MaxTokens = &m
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &ProviderError{Provider: "llamacpp", Message: "marshal request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: "llamacpp", Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "llamacpp", Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: "llamacpp", StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &ProviderError{
			Provider:   "llamacpp",
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, &ProviderError{Provider: "llamacpp", Message: "unmarshal response", Err: err}
	}

	if len(oaiResp.Choices) == 0 {
		return nil, &ProviderError{Provider: "llamacpp", Message: "empty choices in response"}
	}

	return &ChatResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		Provider:     "llamacpp",
		FinishReason: oaiResp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *LlamaCppProvider) Models(_ context.Context) ([]Model, error) {
	modelIDs := p.models
	if len(modelIDs) == 0 {
		modelIDs = []string{"default"}
	}
	out := make([]Model, len(modelIDs))
	for i, id := range modelIDs {
		out[i] = Model{
			ID:       id,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "llamacpp",
			Provider: "llamacpp",
		}
	}
	return out, nil
}
