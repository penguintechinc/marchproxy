//go:build ci
// +build ci

package billing

import (
	"testing"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()
	if tracker == nil {
		t.Fatal("tracker should not be nil")
	}
	if tracker.pricing == nil {
		t.Fatal("pricing map should be initialized")
	}
	if len(tracker.pricing) == 0 {
		t.Fatal("default pricing should be loaded")
	}
}

func TestCalculateCostKnownModel(t *testing.T) {
	tracker := NewTracker()

	// GPT-4: input $0.03/1K, output $0.06/1K
	// 100 input + 50 output = 0.1*0.03 + 0.05*0.06 = 0.003 + 0.003 = 0.006
	cost := tracker.CalculateCost("gpt-4", 100, 50)
	expected := 0.006
	if cost != expected {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateCostCaseInsensitive(t *testing.T) {
	tracker := NewTracker()

	cost1 := tracker.CalculateCost("gpt-4", 100, 50)
	cost2 := tracker.CalculateCost("GPT-4", 100, 50)
	cost3 := tracker.CalculateCost("GpT-4", 100, 50)

	if cost1 != cost2 || cost2 != cost3 {
		t.Errorf("costs should be equal regardless of case")
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	tracker := NewTracker()

	// Unknown model should use fallback pricing
	cost := tracker.CalculateCost("unknown-model", 1000, 500)
	// Fallback: input $0.001/1K, output $0.002/1K
	// 1*0.001 + 0.5*0.002 = 0.001 + 0.001 = 0.002
	expected := 0.002
	if cost != expected {
		t.Errorf("expected fallback cost %f, got %f", expected, cost)
	}
}

func TestCalculateCostZeroTokens(t *testing.T) {
	tracker := NewTracker()

	cost := tracker.CalculateCost("gpt-4", 0, 0)
	if cost != 0.0 {
		t.Errorf("expected cost 0 for zero tokens, got %f", cost)
	}
}

func TestCalculateCostVariousModels(t *testing.T) {
	tracker := NewTracker()

	tests := []struct {
		model        string
		input        int
		output       int
		expectedMin  float64
		expectedMax  float64
	}{
		{"gpt-4o-mini", 100, 100, 0.00007, 0.0001},
		{"claude-sonnet-4-20250514", 1000, 1000, 0.018, 0.020},
		{"gemini-2.5-flash", 500, 500, 0.0003, 0.0004},
	}

	for _, tt := range tests {
		cost := tracker.CalculateCost(tt.model, tt.input, tt.output)
		if cost < tt.expectedMin || cost > tt.expectedMax {
			t.Errorf("%s: expected cost between %f and %f, got %f",
				tt.model, tt.expectedMin, tt.expectedMax, cost)
		}
	}
}

func TestRecordUsage(t *testing.T) {
	tracker := NewTracker()

	cost := tracker.RecordUsage("key-123", "gpt-4", "openai", 100, 50, "req-1")
	if cost == 0 {
		t.Fatal("cost should be calculated")
	}

	// Verify record was added
	if len(tracker.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(tracker.records))
	}

	record := tracker.records[0]
	if record.KeyID != "key-123" {
		t.Errorf("expected KeyID 'key-123', got '%s'", record.KeyID)
	}
	if record.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got '%s'", record.Model)
	}
	if record.InputTokens != 100 {
		t.Errorf("expected InputTokens 100, got %d", record.InputTokens)
	}
	if record.OutputTokens != 50 {
		t.Errorf("expected OutputTokens 50, got %d", record.OutputTokens)
	}
	if record.TotalTokens != 150 {
		t.Errorf("expected TotalTokens 150, got %d", record.TotalTokens)
	}
	if record.Cost == 0 {
		t.Fatal("cost should be set")
	}
	if record.RequestID != "req-1" {
		t.Errorf("expected RequestID 'req-1', got '%s'", record.RequestID)
	}
}

func TestRecordUsageMultiple(t *testing.T) {
	tracker := NewTracker()

	// Record multiple usages
	for i := 0; i < 5; i++ {
		tracker.RecordUsage("key-1", "gpt-4", "openai", 100+i*10, 50, "req-"+string(rune('1'+i)))
	}

	if len(tracker.records) != 5 {
		t.Fatalf("expected 5 records, got %d", len(tracker.records))
	}
}

func TestUpdatePricing(t *testing.T) {
	tracker := NewTracker()

	// Update pricing for existing model
	tracker.UpdatePricing("gpt-4", "openai", 0.05, 0.10)

	// Cost with new pricing
	cost := tracker.CalculateCost("gpt-4", 1000, 1000)
	// 1*0.05 + 1*0.10 = 0.15
	expected := 0.15
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("expected cost around %f with updated pricing, got %f", expected, cost)
	}

	// Add pricing for new model
	tracker.UpdatePricing("custom-model", "custom", 0.01, 0.02)
	cost = tracker.CalculateCost("custom-model", 1000, 1000)
	expected = 0.03
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("expected cost around %f for custom model, got %f", expected, cost)
	}
}

func TestUpdatePricingCaseInsensitive(t *testing.T) {
	tracker := NewTracker()

	tracker.UpdatePricing("GPT-4", "openai", 0.05, 0.10)
	cost := tracker.CalculateCost("gpt-4", 1000, 1000)
	expected := 0.15
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("expected cost around %f, got %f", expected, cost)
	}
}

func TestRecordUsageCaseNormalization(t *testing.T) {
	tracker := NewTracker()

	tracker.RecordUsage("KEY-123", "GPT-4", "OPENAI", 100, 50, "req-1")
	record := tracker.records[0]

	if record.Model != "gpt-4" {
		t.Errorf("model should be lowercase, got '%s'", record.Model)
	}
	if record.Provider != "openai" {
		t.Errorf("provider should be lowercase, got '%s'", record.Provider)
	}
}

func TestConcurrentRecordUsage(t *testing.T) {
	tracker := NewTracker()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			tracker.RecordUsage("key-"+string(rune('1'+idx)), "gpt-4", "openai", 100, 50, "req-"+string(rune('1'+idx)))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if len(tracker.records) != 10 {
		t.Fatalf("expected 10 records from concurrent calls, got %d", len(tracker.records))
	}
}

func TestModelPricingStructure(t *testing.T) {
	pricing := ModelPricing{
		Provider:    "openai",
		InputPer1K:  0.03,
		OutputPer1K: 0.06,
	}

	if pricing.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", pricing.Provider)
	}
	if pricing.InputPer1K != 0.03 {
		t.Errorf("expected InputPer1K 0.03, got %f", pricing.InputPer1K)
	}
	if pricing.OutputPer1K != 0.06 {
		t.Errorf("expected OutputPer1K 0.06, got %f", pricing.OutputPer1K)
	}
}

func TestUsageRecordStructure(t *testing.T) {
	record := UsageRecord{
		KeyID:        "key-1",
		Model:        "gpt-4",
		Provider:     "openai",
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		Cost:         0.006,
		RequestID:    "req-1",
	}

	if record.KeyID != "key-1" {
		t.Errorf("expected KeyID 'key-1', got '%s'", record.KeyID)
	}
	if record.TotalTokens != record.InputTokens+record.OutputTokens {
		t.Fatal("TotalTokens should equal sum of input and output tokens")
	}
	if record.Cost == 0 {
		t.Fatal("Cost should be set")
	}
}
