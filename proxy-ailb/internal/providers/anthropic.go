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

// AnthropicProvider implements the Provider interface for Anthropic's Messages API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// Anthropic API types.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewAnthropicProvider creates a provider for the Anthropic API.
func NewAnthropicProvider(apiKey string, models []string) *AnthropicProvider {
	if len(models) == 0 {
		models = []string{
			"claude-sonnet-4-20250514",
			"claude-3-5-haiku-20241022",
		}
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com",
		models:  models,
		client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (p *AnthropicProvider) Name() string            { return "anthropic" }
func (p *AnthropicProvider) SupportsStreaming() bool  { return true }

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	var systemMsg string
	var msgs []anthropicMessage

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemMsg = m.Content
			continue
		}
		msgs = append(msgs, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	body := anthropicRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		System:    systemMsg,
		Messages:  msgs,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &ProviderError{Provider: "anthropic", Message: "marshal request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: "anthropic", Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "anthropic", Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: "anthropic", StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return nil, &ProviderError{
				Provider:   "anthropic",
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message),
			}
		}
		return nil, &ProviderError{
			Provider:   "anthropic",
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var antResp anthropicResponse
	if err := json.Unmarshal(respBody, &antResp); err != nil {
		return nil, &ProviderError{Provider: "anthropic", Message: "unmarshal response", Err: err}
	}

	var content string
	for _, block := range antResp.Content {
		if block.Type == "text" {
			content = block.Text
			break
		}
	}

	return &ChatResponse{
		Content:      content,
		Model:        antResp.Model,
		Provider:     "anthropic",
		FinishReason: antResp.StopReason,
		Usage: Usage{
			InputTokens:  antResp.Usage.InputTokens,
			OutputTokens: antResp.Usage.OutputTokens,
			TotalTokens:  antResp.Usage.InputTokens + antResp.Usage.OutputTokens,
		},
	}, nil
}

func (p *AnthropicProvider) Models(_ context.Context) ([]Model, error) {
	out := make([]Model, len(p.models))
	for i, id := range p.models {
		out[i] = Model{
			ID:       id,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "anthropic",
			Provider: "anthropic",
		}
	}
	return out, nil
}
