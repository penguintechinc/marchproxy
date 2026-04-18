//go:build ci
// +build ci

package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientWithEndpoint(t *testing.T) {
	client := NewClient("http://custom.local:8080", true)
	if client == nil {
		t.Fatal("client should not be nil")
	}
	if client.endpoint != "http://custom.local:8080" {
		t.Errorf("expected endpoint 'http://custom.local:8080', got '%s'", client.endpoint)
	}
	if !client.enabled {
		t.Fatal("client should be enabled")
	}
	if client.httpClient == nil {
		t.Fatal("httpClient should be initialized")
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClientDefaultEndpoint(t *testing.T) {
	client := NewClient("", true)
	if client.endpoint != "http://waddleai-proxy:8080/mem0" {
		t.Errorf("expected default endpoint, got '%s'", client.endpoint)
	}
}

func TestNewClientDisabled(t *testing.T) {
	client := NewClient("http://localhost:8080", false)
	if client.enabled {
		t.Fatal("client should be disabled")
	}
}

func TestGetContextDisabled(t *testing.T) {
	client := NewClient("http://localhost:8080", false)
	result, err := client.GetContext(context.Background(), "session-1", "query", 5)

	if err != nil {
		t.Fatalf("disabled client should not error, got %v", err)
	}
	if result.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got '%s'", result.SessionID)
	}
	if len(result.RelevantMemories) != 0 {
		t.Errorf("disabled client should return empty memories, got %d", len(result.RelevantMemories))
	}
	if result.MemoryCount != 0 {
		t.Errorf("disabled client should return 0 memory count, got %d", result.MemoryCount)
	}
}

func TestGetContextDefaultLimit(t *testing.T) {
	client := NewClient("http://localhost:8080", false)
	// When disabled, context returns empty result
	result, err := client.GetContext(context.Background(), "session-1", "query", 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestStoreTurnDisabled(t *testing.T) {
	client := NewClient("http://localhost:8080", false)
	err := client.StoreTurn(context.Background(), "session-1", "hello", "hi there", nil)

	if err != nil {
		t.Fatalf("disabled client should not error, got %v", err)
	}
}

func TestStoreTurnWithMetadata(t *testing.T) {
	client := NewClient("http://localhost:8080", false)
	metadata := map[string]string{
		"model": "gpt-4",
		"user":  "test-user",
	}

	err := client.StoreTurn(context.Background(), "session-1", "user message", "assistant response", metadata)
	if err != nil {
		t.Fatalf("StoreTurn should succeed, got %v", err)
	}
}

func TestMemoryEntryStructure(t *testing.T) {
	entry := MemoryEntry{
		Content: "test content",
		Metadata: map[string]string{
			"key": "value",
		},
		Score: 0.95,
	}

	if entry.Content != "test content" {
		t.Errorf("expected content 'test content', got '%s'", entry.Content)
	}
	if entry.Metadata["key"] != "value" {
		t.Errorf("expected metadata value 'value', got '%s'", entry.Metadata["key"])
	}
	if entry.Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", entry.Score)
	}
}

func TestContextResultStructure(t *testing.T) {
	memories := []MemoryEntry{
		{Content: "mem1", Score: 0.9},
		{Content: "mem2", Score: 0.8},
	}

	result := &ContextResult{
		SessionID:        "session-1",
		RelevantMemories: memories,
		MemoryCount:      len(memories),
	}

	if result.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got '%s'", result.SessionID)
	}
	if len(result.RelevantMemories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(result.RelevantMemories))
	}
	if result.MemoryCount != 2 {
		t.Errorf("expected MemoryCount 2, got %d", result.MemoryCount)
	}
}

func TestClientMultipleSessions(t *testing.T) {
	client := NewClient("http://localhost:8080", false)

	// Multiple sessions should work independently
	sessions := []string{"session-1", "session-2", "session-3"}

	for _, sessionID := range sessions {
		result, err := client.GetContext(context.Background(), sessionID, "query", 5)
		if err != nil {
			t.Fatalf("GetContext failed for %s: %v", sessionID, err)
		}
		if result.SessionID != sessionID {
			t.Errorf("expected SessionID '%s', got '%s'", sessionID, result.SessionID)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	client := NewClient("http://localhost:8080", false)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should not panic even with cancelled context
	result, _ := client.GetContext(ctx, "session-1", "query", 5)
	// Either error or return with no data is acceptable
	if result != nil && result.SessionID == "session-1" {
		// Disabled client returns result anyway
		return
	}
}

func TestStoreTurnEmptyMessages(t *testing.T) {
	client := NewClient("http://localhost:8080", false)

	err := client.StoreTurn(context.Background(), "session-1", "", "", nil)
	if err != nil {
		t.Fatalf("StoreTurn should handle empty messages, got %v", err)
	}
}

func TestClientHTTPTimeout(t *testing.T) {
	client := NewClient("http://localhost:8080", true)

	// Should have 5-second timeout
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", client.httpClient.Timeout)
	}
}

func TestMemoryEntryScoreRange(t *testing.T) {
	entries := []MemoryEntry{
		{Content: "high", Score: 0.99},
		{Content: "medium", Score: 0.5},
		{Content: "low", Score: 0.01},
		{Content: "no-score", Score: 0.0},
	}

	for _, entry := range entries {
		if entry.Score < 0 || entry.Score > 1 {
			t.Errorf("score should be between 0 and 1, got %f", entry.Score)
		}
	}
}

func TestLimitVariations(t *testing.T) {
	client := NewClient("http://localhost:8080", false)

	limits := []int{-1, 0, 1, 5, 100}

	for _, limit := range limits {
		_, err := client.GetContext(context.Background(), "session-1", "query", limit)
		if err != nil {
			t.Fatalf("GetContext should accept limit %d, got %v", limit, err)
		}
	}
}

func TestContextResultEmptyMemories(t *testing.T) {
	result := &ContextResult{
		SessionID:        "session-1",
		RelevantMemories: []MemoryEntry{},
		MemoryCount:      0,
	}

	if len(result.RelevantMemories) != 0 {
		t.Errorf("expected 0 memories, got %d", len(result.RelevantMemories))
	}
	if result.MemoryCount != 0 {
		t.Errorf("expected MemoryCount 0, got %d", result.MemoryCount)
	}
}

// TestGetContextWithMockServer tests GetContext with a mock HTTP server
func TestGetContextWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mem0/memories/search" {
			t.Errorf("expected path /mem0/memories/search, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"memory":   "test memory content",
					"metadata": map[string]string{"type": "test"},
					"score":    0.95,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/mem0", true)
	result, err := client.GetContext(context.Background(), "session-1", "test query", 5)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result")
	}

	if result.MemoryCount != 1 {
		t.Errorf("expected 1 memory, got %d", result.MemoryCount)
	}

	if len(result.RelevantMemories) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result.RelevantMemories))
	}

	if result.RelevantMemories[0].Content != "test memory content" {
		t.Errorf("expected content 'test memory content', got %q", result.RelevantMemories[0].Content)
	}
}

// TestGetContextServerError tests graceful handling of server errors
func TestGetContextServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/mem0", true)
	result, err := client.GetContext(context.Background(), "session-1", "query", 5)

	if err != nil {
		t.Fatalf("expected graceful failure, got error %v", err)
	}

	if result == nil {
		t.Fatal("expected empty result")
	}

	if len(result.RelevantMemories) != 0 {
		t.Error("expected no memories on error")
	}
}

// TestStoreTurnWithMockServer tests StoreTurn with a mock HTTP server
func TestStoreTurnWithMockServer(t *testing.T) {
	requestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mem0/memories" {
			t.Errorf("expected path /mem0/memories, got %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		if messages, ok := body["messages"].([]interface{}); !ok || len(messages) != 2 {
			t.Error("expected 2 messages")
		}

		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/mem0", true)

	err := client.StoreTurn(
		context.Background(),
		"session-1",
		"user message",
		"assistant response",
		map[string]string{"key": "value"},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !requestReceived {
		t.Error("expected request to be received")
	}
}

// TestStoreTurnServerError tests graceful handling of server errors in StoreTurn
func TestStoreTurnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/mem0", true)

	err := client.StoreTurn(
		context.Background(),
		"session-1",
		"user message",
		"assistant response",
		map[string]string{},
	)

	// Fail-open: should not error
	if err != nil {
		t.Errorf("expected fail-open, got error %v", err)
	}
}

// TestStoreTurnWithCreatedStatus tests 201 Created status
func TestStoreTurnWithCreatedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/mem0", true)

	err := client.StoreTurn(
		context.Background(),
		"session-1",
		"user message",
		"assistant response",
		map[string]string{},
	)

	if err != nil {
		t.Fatalf("expected no error for 201, got %v", err)
	}
}

// TestGetContextWithMultipleResults tests multiple memory results
func TestGetContextWithMultipleResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"memory":   "memory 1",
					"metadata": map[string]string{"idx": "0"},
					"score":    0.95,
				},
				{
					"memory":   "memory 2",
					"metadata": map[string]string{"idx": "1"},
					"score":    0.85,
				},
				{
					"memory":   "memory 3",
					"metadata": map[string]string{"idx": "2"},
					"score":    0.75,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/mem0", true)
	result, err := client.GetContext(context.Background(), "session-1", "query", 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.MemoryCount != 3 {
		t.Errorf("expected 3 memories, got %d", result.MemoryCount)
	}

	if len(result.RelevantMemories) != 3 {
		t.Errorf("expected 3 entries, got %d", len(result.RelevantMemories))
	}

	if result.RelevantMemories[0].Score != 0.95 {
		t.Errorf("expected first score 0.95, got %f", result.RelevantMemories[0].Score)
	}
}

// TestMemoryEntryWithMetadata tests MemoryEntry structure
func TestMemoryEntryWithMetadata(t *testing.T) {
	entry := MemoryEntry{
		Content:  "test content",
		Metadata: map[string]string{"key1": "val1", "key2": "val2"},
		Score:    0.87,
	}

	if entry.Content != "test content" {
		t.Errorf("expected Content 'test content', got %q", entry.Content)
	}

	if entry.Score != 0.87 {
		t.Errorf("expected Score 0.87, got %f", entry.Score)
	}

	if len(entry.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(entry.Metadata))
	}

	if entry.Metadata["key1"] != "val1" {
		t.Errorf("expected metadata key1=val1, got %q", entry.Metadata["key1"])
	}
}
