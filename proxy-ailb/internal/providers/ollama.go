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

// OllamaProvider implements the Provider interface for Ollama local inference.
type OllamaProvider struct {
	baseURL string
	models  []string
	client  *http.Client
}

// Ollama API types.
type ollamaChatRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  *ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

type ollamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
	DoneReason string       `json:"done_reason"`
	TotalDuration   int64   `json:"total_duration"`
	PromptEvalCount int     `json:"prompt_eval_count"`
	EvalCount       int     `json:"eval_count"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		ModifiedAt string `json:"modified_at"`
		Size       int64  `json:"size"`
		Digest     string `json:"digest"`
	} `json:"models"`
}

// NewOllamaProvider creates a provider for Ollama.
func NewOllamaProvider(baseURL string, models []string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		models:  models,
		client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (p *OllamaProvider) Name() string            { return "ollama" }
func (p *OllamaProvider) SupportsStreaming() bool  { return true }

func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	msgs := make([]ollamaMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}

	body := ollamaChatRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   false,
	}

	opts := &ollamaOptions{}
	hasOpts := false
	if req.Temperature > 0 {
		opts.Temperature = req.Temperature
		hasOpts = true
	}
	if req.MaxTokens > 0 {
		opts.NumPredict = req.MaxTokens
		hasOpts = true
	}
	if req.TopP > 0 {
		opts.TopP = req.TopP
		hasOpts = true
	}
	if len(req.Stop) > 0 {
		opts.Stop = req.Stop
		hasOpts = true
	}
	if hasOpts {
		body.Options = opts
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &ProviderError{Provider: "ollama", Message: "marshal request", Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: "ollama", Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "ollama", Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: "ollama", StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &ProviderError{
			Provider:   "ollama",
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, &ProviderError{Provider: "ollama", Message: "unmarshal response", Err: err}
	}

	return &ChatResponse{
		Content:      ollamaResp.Message.Content,
		Model:        ollamaResp.Model,
		Provider:     "ollama",
		FinishReason: ollamaResp.DoneReason,
		Usage: Usage{
			InputTokens:  ollamaResp.PromptEvalCount,
			OutputTokens: ollamaResp.EvalCount,
			TotalTokens:  ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

func (p *OllamaProvider) Models(ctx context.Context) ([]Model, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return p.fallbackModels(), nil
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return p.fallbackModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.fallbackModels(), nil
	}

	var tagsResp ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return p.fallbackModels(), nil
	}

	out := make([]Model, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		out = append(out, Model{
			ID:       m.Name,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "ollama",
			Provider: "ollama",
		})
	}
	return out, nil
}

func (p *OllamaProvider) fallbackModels() []Model {
	if len(p.models) == 0 {
		return nil
	}
	out := make([]Model, len(p.models))
	for i, id := range p.models {
		out[i] = Model{ID: id, Object: "model", OwnedBy: "ollama", Provider: "ollama"}
	}
	return out
}
