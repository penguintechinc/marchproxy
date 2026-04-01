package providers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// Registry manages registered LLM providers and provides lookup by name or model.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	// modelIndex maps model ID -> provider name for fast lookup.
	modelIndex map[string]string
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers:  make(map[string]Provider),
		modelIndex: make(map[string]string),
	}
}

// Register adds a provider to the registry and indexes its models.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(p.Name())
	r.providers[name] = p

	// Attempt to index models; non-blocking on error.
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	models, err := p.Models(ctx)
	if err != nil {
		slog.Warn("failed to index models for provider", "provider", name, "error", err)
		return
	}
	for _, m := range models {
		r.modelIndex[strings.ToLower(m.ID)] = name
	}

	slog.Info("registered provider", "provider", name, "models", len(models))
}

// Get returns a provider by name (case-insensitive). Returns nil if not found.
func (r *Registry) Get(name string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[strings.ToLower(name)]
}

// GetByModel returns the provider that serves a given model ID.
// It first checks the model index, then falls back to heuristic matching.
func (r *Registry) GetByModel(modelID string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lower := strings.ToLower(modelID)

	// Exact match from index.
	if provName, ok := r.modelIndex[lower]; ok {
		if p, exists := r.providers[provName]; exists {
			return p, nil
		}
	}

	// Heuristic: match model name prefixes to known providers.
	providerHints := map[string][]string{
		"openai":    {"gpt-", "o1-", "o3-", "o4-", "chatgpt-", "davinci", "text-"},
		"anthropic": {"claude-"},
		"gemini":    {"gemini-"},
		"mistral":   {"mistral-", "mixtral-", "codestral-", "pixtral-"},
	}

	for provName, prefixes := range providerHints {
		for _, prefix := range prefixes {
			if strings.HasPrefix(lower, prefix) {
				if p, exists := r.providers[provName]; exists {
					return p, nil
				}
			}
		}
	}

	// Fall back to Ollama for unrecognised models (local inference).
	if p, exists := r.providers["ollama"]; exists {
		return p, nil
	}

	return nil, fmt.Errorf("no provider found for model %q", modelID)
}

// ListAll returns every registered provider.
func (r *Registry) ListAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

// Names returns the names of all registered providers.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.providers))
	for name := range r.providers {
		out = append(out, name)
	}
	return out
}

// RefreshModelIndex re-indexes models from all providers.
func (r *Registry) RefreshModelIndex(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.modelIndex = make(map[string]string)
	for name, p := range r.providers {
		models, err := p.Models(ctx)
		if err != nil {
			slog.Warn("failed to refresh model index", "provider", name, "error", err)
			continue
		}
		for _, m := range models {
			r.modelIndex[strings.ToLower(m.ID)] = name
		}
	}
}
