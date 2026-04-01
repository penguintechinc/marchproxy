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

// WaddleAIProvider implements the Provider interface for the WaddleAI service.
// WaddleAI is a PenguinTech internal AI service that exposes an
// OpenAI-compatible chat completions endpoint.
type WaddleAIProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// NewWaddleAIProvider creates a provider for WaddleAI.
func NewWaddleAIProvider(apiKey, baseURL string, models []string) *WaddleAIProvider {
	if baseURL == "" {
		baseURL = "http://waddleai:8080"
	}
	if len(models) == 0 {
		models = []string{"waddleai-default"}
	}
	return &WaddleAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (p *WaddleAIProvider) Name() string            { return "waddleai" }
func (p *WaddleAIProvider) SupportsStreaming() bool  { return false }

func (p *WaddleAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// WaddleAI uses OpenAI-compatible format.
	msgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body := openAIRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   false,
		Stop:     req.Stop,
		User:     req.User,
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
		return nil, &ProviderError{Provider: "waddleai", Message: "marshal request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: "waddleai", Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "waddleai", Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: "waddleai", StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &ProviderError{
			Provider:   "waddleai",
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, &ProviderError{Provider: "waddleai", Message: "unmarshal response", Err: err}
	}

	if len(oaiResp.Choices) == 0 {
		return nil, &ProviderError{Provider: "waddleai", Message: "empty choices in response"}
	}

	return &ChatResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		Provider:     "waddleai",
		FinishReason: oaiResp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *WaddleAIProvider) Models(_ context.Context) ([]Model, error) {
	out := make([]Model, len(p.models))
	for i, id := range p.models {
		out[i] = Model{
			ID:       id,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "penguintech",
			Provider: "waddleai",
		}
	}
	return out, nil
}
