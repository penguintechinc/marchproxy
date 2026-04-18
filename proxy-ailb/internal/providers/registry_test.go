//go:build ci
// +build ci

package providers

import (
	"context"
	"testing"
)

// MockProvider is a test implementation of Provider
type MockProvider struct {
	name   string
	models []Model
	err    error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Models(ctx context.Context) ([]Model, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models, nil
}

func (m *MockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return nil, nil
}

func (m *MockProvider) SupportsStreaming() bool {
	return false
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("registry should not be nil")
	}
	if registry.providers == nil {
		t.Fatal("providers map should be initialized")
	}
	if registry.modelIndex == nil {
		t.Fatal("modelIndex should be initialized")
	}
	if len(registry.providers) != 0 {
		t.Errorf("registry should start empty, has %d providers", len(registry.providers))
	}
}

func TestRegisterProvider(t *testing.T) {
	registry := NewRegistry()

	provider := &MockProvider{
		name: "test-provider",
		models: []Model{
			{ID: "model-1"},
			{ID: "model-2"},
		},
	}

	registry.Register(provider)

	if registry.Get("test-provider") == nil {
		t.Fatal("provider should be registered")
	}
	if registry.Get("TEST-PROVIDER") == nil {
		t.Fatal("provider lookup should be case-insensitive")
	}
}

func TestRegisterProviderIndexesModels(t *testing.T) {
	registry := NewRegistry()

	provider := &MockProvider{
		name: "openai",
		models: []Model{
			{ID: "gpt-4"},
			{ID: "gpt-3.5-turbo"},
		},
	}

	registry.Register(provider)

	// Check model index
	if _, ok := registry.modelIndex["gpt-4"]; !ok {
		t.Fatal("gpt-4 should be indexed")
	}
	if _, ok := registry.modelIndex["gpt-3.5-turbo"]; !ok {
		t.Fatal("gpt-3.5-turbo should be indexed")
	}
}

func TestGet(t *testing.T) {
	registry := NewRegistry()

	provider := &MockProvider{
		name: "openai",
		models: []Model{
			{ID: "gpt-4"},
		},
	}

	registry.Register(provider)

	// Exact case
	if registry.Get("openai") == nil {
		t.Fatal("Get should find provider with exact case")
	}

	// Different case
	if registry.Get("OpenAI") == nil {
		t.Fatal("Get should find provider with different case")
	}

	// Not found
	if registry.Get("anthropic") != nil {
		t.Fatal("Get should return nil for non-existent provider")
	}
}

func TestGetByModel(t *testing.T) {
	registry := NewRegistry()

	// Register OpenAI
	openaiProvider := &MockProvider{
		name: "openai",
		models: []Model{
			{ID: "gpt-4"},
		},
	}
	registry.Register(openaiProvider)

	// Register Anthropic
	anthropicProvider := &MockProvider{
		name: "anthropic",
		models: []Model{
			{ID: "claude-3-opus"},
		},
	}
	registry.Register(anthropicProvider)

	// Test exact match from index
	provider, err := registry.GetByModel("gpt-4")
	if err != nil {
		t.Fatalf("GetByModel should find gpt-4, got error: %v", err)
	}
	if provider.Name() != "openai" {
		t.Errorf("expected openai provider, got %s", provider.Name())
	}

	// Test heuristic matching by prefix
	provider, err = registry.GetByModel("gpt-4-turbo")
	if err != nil {
		t.Fatalf("GetByModel should find gpt-4-turbo by prefix, got error: %v", err)
	}
	if provider.Name() != "openai" {
		t.Errorf("expected openai provider for gpt prefix, got %s", provider.Name())
	}

	// Test Claude prefix
	provider, err = registry.GetByModel("claude-sonnet")
	if err != nil {
		t.Fatalf("GetByModel should find claude model, got error: %v", err)
	}
	if provider.Name() != "anthropic" {
		t.Errorf("expected anthropic provider for claude prefix, got %s", provider.Name())
	}

	// Test case insensitivity
	provider, err = registry.GetByModel("GPT-4")
	if err != nil {
		t.Fatalf("GetByModel should be case-insensitive, got error: %v", err)
	}
	if provider == nil {
		t.Fatal("GetByModel should find GPT-4")
	}
}

func TestGetByModelNotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.GetByModel("unknown-model")
	if err == nil {
		t.Fatal("GetByModel should return error for unknown model")
	}
}

func TestListAll(t *testing.T) {
	registry := NewRegistry()

	providers := []string{"openai", "anthropic", "google"}
	for _, name := range providers {
		registry.Register(&MockProvider{
			name:   name,
			models: []Model{{ID: "test"}},
		})
	}

	all := registry.ListAll()
	if len(all) != 3 {
		t.Errorf("expected 3 providers, got %d", len(all))
	}
}

func TestNames(t *testing.T) {
	registry := NewRegistry()

	providerNames := []string{"openai", "anthropic", "google"}
	for _, name := range providerNames {
		registry.Register(&MockProvider{
			name:   name,
			models: []Model{{ID: "test"}},
		})
	}

	names := registry.Names()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Check all names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	for _, expected := range providerNames {
		if !nameMap[expected] {
			t.Errorf("expected name '%s' not found", expected)
		}
	}
}

func TestRefreshModelIndex(t *testing.T) {
	registry := NewRegistry()

	// Register provider with initial models
	provider := &MockProvider{
		name: "openai",
		models: []Model{
			{ID: "gpt-4"},
		},
	}
	registry.Register(provider)

	if _, ok := registry.modelIndex["gpt-4"]; !ok {
		t.Fatal("gpt-4 should be in index after registration")
	}

	// Update models in provider (simulate dynamic update)
	provider.models = []Model{
		{ID: "gpt-4-turbo"},
		{ID: "gpt-4o"},
	}

	// Refresh index
	registry.RefreshModelIndex(context.Background())

	// Old model should still be in index (not cleared unless provider is re-registered)
	// New models should be added
	if _, ok := registry.modelIndex["gpt-4-turbo"]; !ok {
		t.Fatal("gpt-4-turbo should be in index after refresh")
	}
}

func TestMultipleProviders(t *testing.T) {
	registry := NewRegistry()

	// Register multiple providers
	openai := &MockProvider{
		name:   "openai",
		models: []Model{{ID: "gpt-4"}},
	}
	anthropic := &MockProvider{
		name:   "anthropic",
		models: []Model{{ID: "claude-opus"}},
	}
	google := &MockProvider{
		name:   "google",
		models: []Model{{ID: "gemini-pro"}},
	}

	registry.Register(openai)
	registry.Register(anthropic)
	registry.Register(google)

	// Verify all are registered
	if registry.Get("openai") == nil {
		t.Fatal("openai should be registered")
	}
	if registry.Get("anthropic") == nil {
		t.Fatal("anthropic should be registered")
	}
	if registry.Get("google") == nil {
		t.Fatal("google should be registered")
	}

	// Verify all are in ListAll
	all := registry.ListAll()
	if len(all) != 3 {
		t.Errorf("expected 3 providers, got %d", len(all))
	}
}

func TestGetByModelPrefixes(t *testing.T) {
	registry := NewRegistry()

	ollama := &MockProvider{
		name:   "ollama",
		models: []Model{{ID: "llama2"}},
	}
	registry.Register(ollama)

	// Test Mistral prefixes
	mistral := &MockProvider{
		name:   "mistral",
		models: []Model{{ID: "mistral-large"}},
	}
	registry.Register(mistral)

	tests := []struct {
		model           string
		expectedProvider string
	}{
		{"mistral-large", "mistral"},
		{"mixtral-8x7b", "mistral"},
		{"codestral-latest", "mistral"},
		{"pixtral-12b", "mistral"},
	}

	for _, tt := range tests {
		provider, err := registry.GetByModel(tt.model)
		if err != nil {
			t.Fatalf("GetByModel(%s) failed: %v", tt.model, err)
		}
		if provider.Name() != tt.expectedProvider {
			t.Errorf("GetByModel(%s) expected %s, got %s",
				tt.model, tt.expectedProvider, provider.Name())
		}
	}
}

func TestEmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	all := registry.ListAll()
	if len(all) != 0 {
		t.Errorf("empty registry should return 0 providers, got %d", len(all))
	}

	names := registry.Names()
	if len(names) != 0 {
		t.Errorf("empty registry should return 0 names, got %d", len(names))
	}
}

func TestConcurrentRegistration(t *testing.T) {
	registry := NewRegistry()
	done := make(chan bool)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			provider := &MockProvider{
				name:   "provider" + string(rune('1'+idx)),
				models: []Model{{ID: "model"}},
			}
			registry.Register(provider)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	if len(registry.Names()) != 5 {
		t.Errorf("expected 5 providers after concurrent registration, got %d", len(registry.Names()))
	}
}
