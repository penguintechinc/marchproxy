package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/billing"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/metrics"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/router"
)

// OpenAI-compatible request/response types for the /v1/chat/completions endpoint.

type chatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []messageRequest `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Stop        json.RawMessage `json:"stop,omitempty"`
	User        string          `json:"user,omitempty"`
}

type messageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   chatCompletionUsage    `json:"usage"`
}

type chatCompletionChoice struct {
	Index        int            `json:"index"`
	Message      messageRequest `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ChatHandler handles POST /v1/chat/completions (OpenAI-compatible).
type ChatHandler struct {
	registry  *providers.Registry
	router    *router.Router
	reporter  *billing.Reporter
	metrics   *metrics.Metrics
	waddleAI  *providers.WaddleAIGRPCClient
	routingAI bool
}

// NewChatHandler creates a new chat completions handler.
func NewChatHandler(reg *providers.Registry, rtr *router.Router, reporter *billing.Reporter, m *metrics.Metrics) *ChatHandler {
	return &ChatHandler{
		registry: reg,
		router:   rtr,
		reporter: reporter,
		metrics:  m,
	}
}

// SetWaddleAI enables WaddleAI gRPC integration for routing and security.
func (h *ChatHandler) SetWaddleAI(client *providers.WaddleAIGRPCClient, routingAI bool) {
	h.waddleAI = client
	h.routingAI = routingAI
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request")
		return
	}

	var req chatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), "invalid_request")
		return
	}

	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required", "invalid_request")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages is required", "invalid_request")
		return
	}

	// Extract user identity from JWT claims.
	claims := ClaimsFromContext(r.Context())
	userID := ""
	if claims != nil {
		userID = claims.Sub
	}

	// Concatenate messages for prompt analysis.
	lastMessage := req.Messages[len(req.Messages)-1].Content

	// WaddleAI security evaluation (fail-open).
	if h.waddleAI != nil && h.waddleAI.Enabled() {
		secResp, _ := h.waddleAI.EvaluateSecurity(r.Context(), lastMessage, "chat", userID, nil)
		if secResp != nil && secResp.GetBlocked() {
			slog.Warn("request blocked by WaddleAI security",
				"user_id", userID,
				"threat_type", secResp.GetThreatType(),
				"risk_score", secResp.GetRiskScore(),
			)
			writeError(w, http.StatusForbidden,
				"request blocked by security policy: "+secResp.GetExplanation(), "security_block")
			return
		}
	}

	// Convert to internal format.
	msgs := make([]providers.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = providers.Message{Role: m.Role, Content: m.Content}
	}

	chatReq := &providers.ChatRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   req.Stream,
		User:     req.User,
	}
	if req.Temperature != nil {
		chatReq.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		chatReq.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		chatReq.TopP = *req.TopP
	}

	// Parse stop sequences.
	if req.Stop != nil {
		var stopStr string
		var stopArr []string
		if json.Unmarshal(req.Stop, &stopStr) == nil {
			chatReq.Stop = []string{stopStr}
		} else if json.Unmarshal(req.Stop, &stopArr) == nil {
			chatReq.Stop = stopArr
		}
	}

	// WaddleAI routing evaluation (fail-open): use recommended model if available.
	if h.routingAI && h.waddleAI != nil && h.waddleAI.Enabled() {
		routeResp, _ := h.waddleAI.EvaluateRoute(r.Context(), lastMessage, "chat", "", userID, "", nil)
		if routeResp != nil && routeResp.GetRecommendedModel() != "" {
			slog.Debug("waddleai routing recommendation",
				"original_model", chatReq.Model,
				"recommended_model", routeResp.GetRecommendedModel(),
				"complexity", routeResp.GetComplexity(),
				"confidence", routeResp.GetConfidence(),
			)
			chatReq.Model = routeResp.GetRecommendedModel()
		}
	}

	start := time.Now()

	// Check X-Model-Selector header for direct provider routing.
	var chatResp *providers.ChatResponse
	var err error

	if selector := r.Header.Get("X-Model-Selector"); selector != "" {
		providerName, modelName, ok := router.ParseModelSelector(selector)
		if ok {
			chatReq.Model = modelName
			chatResp, err = h.router.RouteToProvider(r.Context(), providerName, chatReq)
		} else {
			chatResp, err = h.router.Route(r.Context(), chatReq)
		}
	} else {
		chatResp, err = h.router.Route(r.Context(), chatReq)
	}

	duration := time.Since(start)

	if err != nil {
		slog.Error("chat completion failed", "model", req.Model, "error", err)
		h.metrics.RecordRequest("unknown", req.Model, "error", duration.Seconds(), 0, 0)
		writeError(w, http.StatusBadGateway, "all providers failed: "+err.Error(), "provider_error")
		return
	}

	// Record metrics.
	h.metrics.RecordRequest(chatResp.Provider, chatResp.Model, "success",
		duration.Seconds(), chatResp.Usage.InputTokens, chatResp.Usage.OutputTokens)

	// Fire-and-forget usage reporting to WaddleAI.
	if h.reporter != nil {
		h.reporter.ReportAsync(billing.UsageReport{
			UserID:       userID,
			Model:        chatResp.Model,
			Provider:     chatResp.Provider,
			InputTokens:  chatResp.Usage.InputTokens,
			OutputTokens: chatResp.Usage.OutputTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
			LatencyMs:    int(duration.Milliseconds()),
			RequestID:    fmt.Sprintf("ailb-%d", time.Now().UnixNano()),
		})
	}

	// Fire-and-forget StoreTurn to WaddleAI memory.
	if h.waddleAI != nil && h.waddleAI.Enabled() {
		go h.waddleAI.StoreTurn(
			context.Background(),
			"", // sessionID (not available in stateless API)
			userID,
			lastMessage,
			chatResp.Content,
			chatResp.Model,
			chatResp.Provider,
			map[string]string{
				"latency_ms": fmt.Sprintf("%d", duration.Milliseconds()),
			},
		)
	}

	// Build OpenAI-compatible response.
	resp := chatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   chatResp.Model,
		Choices: []chatCompletionChoice{
			{
				Index: 0,
				Message: messageRequest{
					Role:    "assistant",
					Content: chatResp.Content,
				},
				FinishReason: chatResp.FinishReason,
			},
		},
		Usage: chatCompletionUsage{
			PromptTokens:     chatResp.Usage.InputTokens,
			CompletionTokens: chatResp.Usage.OutputTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, message, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{
		Error: errorDetail{
			Message: message,
			Type:    errType,
			Code:    http.StatusText(status),
		},
	})
}
