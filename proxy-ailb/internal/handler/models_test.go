//go:build ci

package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/handler"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
)

// mockModelProvider implements providers.Provider with Models support
type mockModelProvider struct {
	name   string
	models []providers.Model
	err    error
}

func (m *mockModelProvider) Name() string {
	return m.name
}

func (m *mockModelProvider) Chat(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return &providers.ChatResponse{
		Content:  "test response",
		Provider: m.name,
	}, nil
}

func (m *mockModelProvider) Models(ctx context.Context) ([]providers.Model, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models, nil
}

func (m *mockModelProvider) SupportsStreaming() bool {
	return false
}

func TestNewModelsHandler(t *testing.T) {
	registry := providers.NewRegistry()
	handler := handler.NewModelsHandler(registry)

	if handler == nil {
		t.Error("expected models handler to be created")
	}
}

func TestModelsHandlerGetMethod(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockModelProvider{
		name: "test",
		models: []providers.Model{
			{ID: "model-1", Object: "model", Created: 1000, OwnedBy: "test-owner", Provider: "test"},
		},
	})

	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestModelsHandlerPostMethodNotAllowed(t *testing.T) {
	registry := providers.NewRegistry()
	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodPost, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestModelsHandlerEmptyRegistry(t *testing.T) {
	registry := providers.NewRegistry()
	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Object string        `json:"object"`
		Data   []interface{} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Object != "list" {
		t.Errorf("expected object 'list', got %v", resp.Object)
	}

	if len(resp.Data) != 0 {
		t.Errorf("expected empty data for empty registry, got %d models", len(resp.Data))
	}
}

func TestModelsHandlerSingleProvider(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockModelProvider{
		name: "openai",
		models: []providers.Model{
			{ID: "gpt-4", Object: "model", Created: 1000, OwnedBy: "openai", Provider: "openai"},
			{ID: "gpt-3.5-turbo", Object: "model", Created: 900, OwnedBy: "openai", Provider: "openai"},
		},
	})

	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 2 {
		t.Errorf("expected 2 models, got %d", len(resp.Data))
	}
}

func TestModelsHandlerMultipleProviders(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockModelProvider{
		name: "openai",
		models: []providers.Model{
			{ID: "gpt-4", Object: "model", Created: 1000, OwnedBy: "openai", Provider: "openai"},
		},
	})
	registry.Register(&mockModelProvider{
		name: "anthropic",
		models: []providers.Model{
			{ID: "claude-3", Object: "model", Created: 2000, OwnedBy: "anthropic", Provider: "anthropic"},
		},
	})

	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 2 {
		t.Errorf("expected 2 models from 2 providers, got %d", len(resp.Data))
	}
}

func TestModelsHandlerProviderError(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockModelProvider{
		name: "working",
		models: []providers.Model{
			{ID: "model-1", Object: "model", Created: 1000, OwnedBy: "working", Provider: "working"},
		},
	})
	registry.Register(&mockModelProvider{
		name: "failing",
		err:  context.DeadlineExceeded,
	})

	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 200 with the models from the working provider
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 1 {
		t.Errorf("expected 1 model from working provider, got %d", len(resp.Data))
	}
}

func TestModelsHandlerModelObject(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Register(&mockModelProvider{
		name: "test",
		models: []providers.Model{
			{ID: "test-model", Object: "model", Created: 1234567890, OwnedBy: "test-org", Provider: "test"},
		},
	})

	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) == 0 {
		t.Fatal("expected at least one model")
	}

	model := resp.Data[0]
	if model["id"] != "test-model" {
		t.Errorf("expected id 'test-model', got %v", model["id"])
	}

	if model["object"] != "model" {
		t.Errorf("expected object 'model', got %v", model["object"])
	}

	if model["owned_by"] != "test-org" {
		t.Errorf("expected owned_by 'test-org', got %v", model["owned_by"])
	}

	if model["provider"] != "test" {
		t.Errorf("expected provider 'test', got %v", model["provider"])
	}

	created := int64(model["created"].(float64))
	if created != 1234567890 {
		t.Errorf("expected created 1234567890, got %d", created)
	}
}

func TestModelsHandlerPutMethodNotAllowed(t *testing.T) {
	registry := providers.NewRegistry()
	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodPut, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestModelsHandlerDeleteMethodNotAllowed(t *testing.T) {
	registry := providers.NewRegistry()
	handler := handler.NewModelsHandler(registry)

	req := httptest.NewRequest(http.MethodDelete, "/v1/models", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
