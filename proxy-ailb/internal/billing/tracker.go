// Package billing provides cost tracking and usage reporting for the AILB service.
package billing

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ModelPricing holds per-1K token pricing for a model.
type ModelPricing struct {
	Provider        string
	InputPer1K      float64
	OutputPer1K     float64
}

// UsageRecord tracks a single request's usage.
type UsageRecord struct {
	KeyID        string
	Model        string
	Provider     string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64
	Timestamp    time.Time
	RequestID    string
}

// defaultPricing contains default model pricing per 1K tokens (USD).
var defaultPricing = map[string]ModelPricing{
	"gpt-4":                     {Provider: "openai", InputPer1K: 0.03, OutputPer1K: 0.06},
	"gpt-4-turbo":               {Provider: "openai", InputPer1K: 0.01, OutputPer1K: 0.03},
	"gpt-4o":                    {Provider: "openai", InputPer1K: 0.005, OutputPer1K: 0.015},
	"gpt-4o-mini":               {Provider: "openai", InputPer1K: 0.00015, OutputPer1K: 0.0006},
	"gpt-3.5-turbo":             {Provider: "openai", InputPer1K: 0.0005, OutputPer1K: 0.0015},
	"claude-sonnet-4-20250514":  {Provider: "anthropic", InputPer1K: 0.003, OutputPer1K: 0.015},
	"claude-3-5-haiku-20241022": {Provider: "anthropic", InputPer1K: 0.001, OutputPer1K: 0.005},
	"claude-3-opus-20240229":    {Provider: "anthropic", InputPer1K: 0.015, OutputPer1K: 0.075},
	"gemini-2.5-flash":          {Provider: "google", InputPer1K: 0.00015, OutputPer1K: 0.0006},
	"gemini-2.5-pro":            {Provider: "google", InputPer1K: 0.00125, OutputPer1K: 0.01},
	"mistral-large-latest":      {Provider: "mistral", InputPer1K: 0.002, OutputPer1K: 0.006},
	"mistral-small-latest":      {Provider: "mistral", InputPer1K: 0.001, OutputPer1K: 0.003},
}

// Tracker tracks token usage and calculates costs.
type Tracker struct {
	mu       sync.Mutex
	records  []UsageRecord
	pricing  map[string]ModelPricing
}

// NewTracker creates a cost tracker with default pricing.
func NewTracker() *Tracker {
	pricing := make(map[string]ModelPricing, len(defaultPricing))
	for k, v := range defaultPricing {
		pricing[k] = v
	}
	return &Tracker{
		pricing: pricing,
	}
}

// CalculateCost computes the USD cost for a request.
func (t *Tracker) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	pricing, ok := t.pricing[strings.ToLower(model)]
	if !ok {
		// Default fallback pricing.
		pricing = ModelPricing{InputPer1K: 0.001, OutputPer1K: 0.002}
	}

	return (float64(inputTokens)/1000.0)*pricing.InputPer1K +
		(float64(outputTokens)/1000.0)*pricing.OutputPer1K
}

// RecordUsage records a usage event and returns the calculated cost.
func (t *Tracker) RecordUsage(keyID, model, provider string, inputTokens, outputTokens int, requestID string) float64 {
	cost := t.CalculateCost(model, inputTokens, outputTokens)

	record := UsageRecord{
		KeyID:        keyID,
		Model:        strings.ToLower(model),
		Provider:     strings.ToLower(provider),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Cost:         cost,
		Timestamp:    time.Now(),
		RequestID:    requestID,
	}

	t.mu.Lock()
	t.records = append(t.records, record)
	t.mu.Unlock()

	slog.Debug("recorded usage",
		"key_id", keyID,
		"model", model,
		"tokens", inputTokens+outputTokens,
		"cost", cost,
	)

	return cost
}

// UpdatePricing sets or updates pricing for a model.
func (t *Tracker) UpdatePricing(model, provider string, inputPer1K, outputPer1K float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pricing[strings.ToLower(model)] = ModelPricing{
		Provider:   provider,
		InputPer1K: inputPer1K,
		OutputPer1K: outputPer1K,
	}
}
