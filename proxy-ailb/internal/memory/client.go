// Package memory provides a client for the WaddleAI memory service (mem0)
// to store and retrieve conversation context.
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Client communicates with the WaddleAI mem0 HTTP API for conversation memory.
type Client struct {
	endpoint   string
	httpClient *http.Client
	enabled    bool
}

// MemoryEntry represents a single memory result.
type MemoryEntry struct {
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
	Score    float64           `json:"score"`
}

// ContextResult holds retrieved context for a session.
type ContextResult struct {
	SessionID       string        `json:"session_id"`
	RelevantMemories []MemoryEntry `json:"relevant_memories"`
	MemoryCount     int           `json:"memory_count"`
}

// NewClient creates a memory client. If endpoint is empty, the client is disabled.
func NewClient(endpoint string, enabled bool) *Client {
	if endpoint == "" {
		endpoint = "http://waddleai-proxy:8080/mem0"
	}
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		enabled: enabled,
	}
}

// GetContext retrieves relevant conversation context for a session.
func (c *Client) GetContext(ctx context.Context, sessionID string, query string, limit int) (*ContextResult, error) {
	if !c.enabled {
		return &ContextResult{SessionID: sessionID}, nil
	}

	if limit <= 0 {
		limit = 5
	}

	reqBody := map[string]any{
		"query":     query,
		"user_id":   sessionID,
		"agent_id":  sessionID,
		"limit":     limit,
		"threshold": 0.7,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal search request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/memories/search", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Warn("memory search failed, continuing without context", "error", err)
		return &ContextResult{SessionID: sessionID}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("memory search returned non-200", "status", resp.StatusCode)
		return &ContextResult{SessionID: sessionID}, nil
	}

	var result struct {
		Results []struct {
			Memory   string            `json:"memory"`
			Metadata map[string]string `json:"metadata"`
			Score    float64           `json:"score"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	memories := make([]MemoryEntry, 0, len(result.Results))
	for _, r := range result.Results {
		memories = append(memories, MemoryEntry{
			Content:  r.Memory,
			Metadata: r.Metadata,
			Score:    r.Score,
		})
	}

	return &ContextResult{
		SessionID:       sessionID,
		RelevantMemories: memories,
		MemoryCount:     len(memories),
	}, nil
}

// StoreTurn saves a conversation turn to memory.
func (c *Client) StoreTurn(ctx context.Context, sessionID, userMessage, assistantResponse string, metadata map[string]string) error {
	if !c.enabled {
		return nil
	}

	reqBody := map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
			{"role": "assistant", "content": assistantResponse},
		},
		"user_id":  sessionID,
		"agent_id": sessionID,
		"metadata": metadata,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal store request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/memories", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create store request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Warn("memory store failed", "error", err)
		return nil // Fail-open: don't break chat if memory is unavailable.
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		slog.Warn("memory store returned non-success", "status", resp.StatusCode)
	}

	return nil
}
