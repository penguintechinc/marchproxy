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

// OpenAIProvider implements the Provider interface for OpenAI and any
// OpenAI-compatible API (Together, Fireworks, Groq, Cerebras, vLLM,
// LocalAI, DeepSeek, etc.) via a configurable base URL.
type OpenAIProvider struct {
	name    string
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// OpenAI API request/response types.
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stream      bool            `json:"stream"`
	Stop        []string        `json:"stop,omitempty"`
	User        string          `json:"user,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *openAIErrorPayload `json:"error,omitempty"`
}

type openAIErrorPayload struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type openAIModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// NewOpenAIProvider creates a provider for OpenAI-compatible APIs.
//
// Parameters:
//   - name: identifier for this provider instance (e.g. "openai", "together", "groq")
//   - apiKey: API key (may be empty for local endpoints like vLLM)
//   - baseURL: base URL including /v1 path (e.g. "https://api.openai.com/v1")
//   - models: list of supported model IDs; if empty, discovered via /models
func NewOpenAIProvider(name, apiKey, baseURL string, models []string) *OpenAIProvider {
	return &OpenAIProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (p *OpenAIProvider) Name() string            { return p.name }
func (p *OpenAIProvider) SupportsStreaming() bool  { return true }

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	msgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body := openAIRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   false, // Streaming handled separately when needed.
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
	if req.TopP > 0 {
		tp := req.TopP
		body.TopP = &tp
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &ProviderError{Provider: p.name, Message: "marshal request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: p.name, Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: p.name, Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: p.name, StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openAIResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return nil, &ProviderError{
				Provider:   p.name,
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message),
			}
		}
		return nil, &ProviderError{
			Provider:   p.name,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, &ProviderError{Provider: p.name, Message: "unmarshal response", Err: err}
	}

	if len(oaiResp.Choices) == 0 {
		return nil, &ProviderError{Provider: p.name, Message: "empty choices in response"}
	}

	return &ChatResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		Provider:     p.name,
		FinishReason: oaiResp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *OpenAIProvider) Models(ctx context.Context) ([]Model, error) {
	// If models are explicitly configured, return those without calling the API.
	if len(p.models) > 0 {
		out := make([]Model, len(p.models))
		for i, id := range p.models {
			out[i] = Model{
				ID:       id,
				Object:   "model",
				Created:  time.Now().Unix(),
				OwnedBy:  p.name,
				Provider: p.name,
			}
		}
		return out, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return nil, &ProviderError{Provider: p.name, Message: "create models request", Err: err}
	}
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return p.fallbackModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.fallbackModels(), nil
	}

	var modelsResp openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return p.fallbackModels(), nil
	}

	out := make([]Model, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		out = append(out, Model{
			ID:       m.ID,
			Object:   m.Object,
			Created:  m.Created,
			OwnedBy:  m.OwnedBy,
			Provider: p.name,
		})
	}
	return out, nil
}

func (p *OpenAIProvider) fallbackModels() []Model {
	defaults := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"}
	if len(p.models) > 0 {
		defaults = p.models
	}
	out := make([]Model, len(defaults))
	for i, id := range defaults {
		out[i] = Model{ID: id, Object: "model", OwnedBy: p.name, Provider: p.name}
	}
	return out
}
