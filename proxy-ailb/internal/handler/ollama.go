package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/metrics"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

// Ollama-compatible request/response types.

type ollamaChatRequest struct {
	Model    string                `json:"model"`
	Messages []ollamaChatMessage   `json:"messages"`
	Stream   bool                  `json:"stream"`
	Options  *ollamaChatOptions    `json:"options,omitempty"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatOptions struct {
	Temperature float64  `json:"temperature,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
	TopP        float64  `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

type ollamaChatResponse struct {
	Model           string            `json:"model"`
	CreatedAt       string            `json:"created_at"`
	Message         ollamaChatMessage `json:"message"`
	Done            bool              `json:"done"`
	DoneReason      string            `json:"done_reason"`
	TotalDuration   int64             `json:"total_duration"`
	PromptEvalCount int               `json:"prompt_eval_count"`
	EvalCount       int               `json:"eval_count"`
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Model           string `json:"model"`
	CreatedAt       string `json:"created_at"`
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	DoneReason      string `json:"done_reason"`
	TotalDuration   int64  `json:"total_duration"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
}

type ollamaTagModel struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

type ollamaTagsResponse struct {
	Models []ollamaTagModel `json:"models"`
}

// OllamaHandler handles Ollama-compatible API endpoints.
type OllamaHandler struct {
	registry *providers.Registry
	router   *router.Router
	metrics  *metrics.Metrics
}

// NewOllamaHandler creates handlers for Ollama-compatible endpoints.
func NewOllamaHandler(reg *providers.Registry, rtr *router.Router, m *metrics.Metrics) *OllamaHandler {
	return &OllamaHandler{
		registry: reg,
		router:   rtr,
		metrics:  m,
	}
}

// ChatHandler handles POST /api/chat.
func (h *OllamaHandler) ChatHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		// Normalize to internal format.
		msgs := make([]providers.Message, len(req.Messages))
		for i, m := range req.Messages {
			msgs[i] = providers.Message{Role: m.Role, Content: m.Content}
		}

		chatReq := &providers.ChatRequest{
			Model:    req.Model,
			Messages: msgs,
		}
		if req.Options != nil {
			chatReq.Temperature = req.Options.Temperature
			chatReq.MaxTokens = req.Options.NumPredict
			chatReq.TopP = req.Options.TopP
			chatReq.Stop = req.Options.Stop
		}

		start := time.Now()
		chatResp, err := h.router.Route(r.Context(), chatReq)
		duration := time.Since(start)

		if err != nil {
			slog.Error("ollama chat failed", "model", req.Model, "error", err)
			h.metrics.RecordRequest("unknown", req.Model, "error", duration.Seconds(), 0, 0)
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
			return
		}

		h.metrics.RecordRequest(chatResp.Provider, chatResp.Model, "success",
			duration.Seconds(), chatResp.Usage.InputTokens, chatResp.Usage.OutputTokens)

		resp := ollamaChatResponse{
			Model:     chatResp.Model,
			CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Message: ollamaChatMessage{
				Role:    "assistant",
				Content: chatResp.Content,
			},
			Done:            true,
			DoneReason:      chatResp.FinishReason,
			TotalDuration:   duration.Nanoseconds(),
			PromptEvalCount: chatResp.Usage.InputTokens,
			EvalCount:       chatResp.Usage.OutputTokens,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// GenerateHandler handles POST /api/generate.
func (h *OllamaHandler) GenerateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ollamaGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		chatReq := &providers.ChatRequest{
			Model: req.Model,
			Messages: []providers.Message{
				{Role: "user", Content: req.Prompt},
			},
		}

		start := time.Now()
		chatResp, err := h.router.Route(r.Context(), chatReq)
		duration := time.Since(start)

		if err != nil {
			slog.Error("ollama generate failed", "model", req.Model, "error", err)
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
			return
		}

		h.metrics.RecordRequest(chatResp.Provider, chatResp.Model, "success",
			duration.Seconds(), chatResp.Usage.InputTokens, chatResp.Usage.OutputTokens)

		resp := ollamaGenerateResponse{
			Model:           chatResp.Model,
			CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
			Response:        chatResp.Content,
			Done:            true,
			DoneReason:      chatResp.FinishReason,
			TotalDuration:   duration.Nanoseconds(),
			PromptEvalCount: chatResp.Usage.InputTokens,
			EvalCount:       chatResp.Usage.OutputTokens,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// TagsHandler handles GET /api/tags (list models in Ollama format).
func (h *OllamaHandler) TagsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var allModels []ollamaTagModel
		for _, p := range h.registry.ListAll() {
			models, err := p.Models(r.Context())
			if err != nil {
				slog.Warn("failed to list models from provider", "provider", p.Name(), "error", err)
				continue
			}
			for _, m := range models {
				allModels = append(allModels, ollamaTagModel{
					Name:       m.ID,
					ModifiedAt: time.Now().UTC().Format(time.RFC3339),
					Size:       0,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaTagsResponse{Models: allModels})
	}
}
