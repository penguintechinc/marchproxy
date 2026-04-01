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

// MistralProvider implements the Provider interface for the Mistral AI API.
// Mistral uses an OpenAI-compatible chat completions format.
type MistralProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// NewMistralProvider creates a provider for the Mistral AI API.
func NewMistralProvider(apiKey, baseURL string, models []string) *MistralProvider {
	if baseURL == "" {
		baseURL = "https://api.mistral.ai/v1"
	}
	if len(models) == 0 {
		models = []string{"mistral-large-latest", "mistral-small-latest"}
	}
	return &MistralProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (p *MistralProvider) Name() string            { return "mistral" }
func (p *MistralProvider) SupportsStreaming() bool  { return true }

func (p *MistralProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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
	if req.TopP > 0 {
		tp := req.TopP
		body.TopP = &tp
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &ProviderError{Provider: "mistral", Message: "marshal request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: "mistral", Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "mistral", Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: "mistral", StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openAIResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return nil, &ProviderError{
				Provider:   "mistral",
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message),
			}
		}
		return nil, &ProviderError{
			Provider:   "mistral",
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, &ProviderError{Provider: "mistral", Message: "unmarshal response", Err: err}
	}

	if len(oaiResp.Choices) == 0 {
		return nil, &ProviderError{Provider: "mistral", Message: "empty choices in response"}
	}

	return &ChatResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		Provider:     "mistral",
		FinishReason: oaiResp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *MistralProvider) Models(_ context.Context) ([]Model, error) {
	out := make([]Model, len(p.models))
	for i, id := range p.models {
		out[i] = Model{
			ID:       id,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "mistralai",
			Provider: "mistral",
		}
	}
	return out, nil
}
