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

// GeminiProvider implements the Provider interface for Google's Gemini API.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

// Gemini API types.
type geminiRequest struct {
	Contents         []geminiContent          `json:"contents"`
	GenerationConfig *geminiGenerationConfig  `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// NewGeminiProvider creates a provider for Google Gemini.
func NewGeminiProvider(apiKey, baseURL string, models []string) *GeminiProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	if len(models) == 0 {
		models = []string{"gemini-2.5-flash", "gemini-2.5-pro"}
	}
	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (p *GeminiProvider) Name() string            { return "gemini" }
func (p *GeminiProvider) SupportsStreaming() bool  { return true }

func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	gemReq := geminiRequest{}

	var systemText string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemText = m.Content
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		gemReq.Contents = append(gemReq.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	if systemText != "" {
		gemReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemText}},
		}
	}

	genConfig := &geminiGenerationConfig{}
	hasConfig := false
	if req.Temperature > 0 {
		t := req.Temperature
		genConfig.Temperature = &t
		hasConfig = true
	}
	if req.MaxTokens > 0 {
		m := req.MaxTokens
		genConfig.MaxOutputTokens = &m
		hasConfig = true
	}
	if req.TopP > 0 {
		tp := req.TopP
		genConfig.TopP = &tp
		hasConfig = true
	}
	if len(req.Stop) > 0 {
		genConfig.StopSequences = req.Stop
		hasConfig = true
	}
	if hasConfig {
		gemReq.GenerationConfig = genConfig
	}

	payload, err := json.Marshal(gemReq)
	if err != nil {
		return nil, &ProviderError{Provider: "gemini", Message: "marshal request", Err: err}
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Provider: "gemini", Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: "gemini", Message: "HTTP request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Provider: "gemini", StatusCode: resp.StatusCode, Message: "read response", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp geminiResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return nil, &ProviderError{
				Provider:   "gemini",
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("%s: %s", errResp.Error.Status, errResp.Error.Message),
			}
		}
		return nil, &ProviderError{
			Provider:   "gemini",
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return nil, &ProviderError{Provider: "gemini", Message: "unmarshal response", Err: err}
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, &ProviderError{Provider: "gemini", Message: "empty response from Gemini"}
	}

	return &ChatResponse{
		Content:      gemResp.Candidates[0].Content.Parts[0].Text,
		Model:        req.Model,
		Provider:     "gemini",
		FinishReason: gemResp.Candidates[0].FinishReason,
		Usage: Usage{
			InputTokens:  gemResp.UsageMetadata.PromptTokenCount,
			OutputTokens: gemResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  gemResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func (p *GeminiProvider) Models(_ context.Context) ([]Model, error) {
	out := make([]Model, len(p.models))
	for i, id := range p.models {
		out[i] = Model{
			ID:       id,
			Object:   "model",
			Created:  time.Now().Unix(),
			OwnedBy:  "google",
			Provider: "gemini",
		}
	}
	return out, nil
}
