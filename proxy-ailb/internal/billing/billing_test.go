//go:build ci

package billing_test

import (
	"math"
	"testing"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/billing"
)

func TestNewTracker(t *testing.T) {
	tracker := billing.NewTracker()

	if tracker == nil {
		t.Error("expected tracker to be created")
	}
}

func TestCalculateCostKnownModel(t *testing.T) {
	tracker := billing.NewTracker()

	// GPT-4: $0.03 per 1K input, $0.06 per 1K output
	cost := tracker.CalculateCost("gpt-4", 1000, 1000)

	// Expected: (1000/1000)*0.03 + (1000/1000)*0.06 = 0.09
	expected := 0.09
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected cost %.4f, got %.4f", expected, cost)
	}
}

func TestCalculateCostCaseInsensitive(t *testing.T) {
	tracker := billing.NewTracker()

	cost1 := tracker.CalculateCost("gpt-4", 1000, 1000)
	cost2 := tracker.CalculateCost("GPT-4", 1000, 1000)
	cost3 := tracker.CalculateCost("Gpt-4", 1000, 1000)

	if cost1 != cost2 || cost2 != cost3 {
		t.Errorf("expected case-insensitive pricing: %f, %f, %f", cost1, cost2, cost3)
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	tracker := billing.NewTracker()

	// Unknown models should use fallback pricing: $0.001 input, $0.002 output
	cost := tracker.CalculateCost("unknown-model-xyz", 1000, 1000)

	// Expected: (1000/1000)*0.001 + (1000/1000)*0.002 = 0.003
	expected := 0.003
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected fallback cost %.4f, got %.4f", expected, cost)
	}
}

func TestCalculateCostZeroTokens(t *testing.T) {
	tracker := billing.NewTracker()

	cost := tracker.CalculateCost("gpt-4", 0, 0)

	if cost != 0 {
		t.Errorf("expected zero cost for zero tokens, got %f", cost)
	}
}

func TestCalculateCostVariousModels(t *testing.T) {
	tracker := billing.NewTracker()

	tests := []struct {
		model       string
		inputTokens int
		outputTokens int
		// We just check cost is positive and reasonable
	}{
		{"gpt-4-turbo", 1000, 1000},
		{"gpt-4o", 1000, 1000},
		{"gpt-3.5-turbo", 1000, 1000},
		{"claude-sonnet-4-20250514", 1000, 1000},
		{"claude-3-5-haiku-20241022", 1000, 1000},
		{"gemini-2.5-flash", 1000, 1000},
		{"mistral-large-latest", 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			cost := tracker.CalculateCost(tt.model, tt.inputTokens, tt.outputTokens)
			if cost <= 0 {
				t.Errorf("expected positive cost for %s, got %f", tt.model, cost)
			}
		})
	}
}

func TestRecordUsage(t *testing.T) {
	tracker := billing.NewTracker()

	cost := tracker.RecordUsage("key-123", "gpt-4", "openai", 1000, 1000, "req-456")

	if cost <= 0 {
		t.Errorf("expected positive cost, got %f", cost)
	}
}

func TestRecordUsageMultipleRecords(t *testing.T) {
	tracker := billing.NewTracker()

	tracker.RecordUsage("key-1", "gpt-4", "openai", 1000, 1000, "req-1")
	tracker.RecordUsage("key-2", "gpt-3.5-turbo", "openai", 500, 500, "req-2")
	tracker.RecordUsage("key-1", "gpt-4", "openai", 2000, 2000, "req-3")

	// We can't directly check internal records, but recording should not error
}

func TestUpdatePricing(t *testing.T) {
	tracker := billing.NewTracker()

	// Update pricing for a custom model
	tracker.UpdatePricing("custom-model", "custom-provider", 0.01, 0.02)

	// Calculate cost with updated pricing
	cost := tracker.CalculateCost("custom-model", 1000, 1000)

	// Expected: (1000/1000)*0.01 + (1000/1000)*0.02 = 0.03
	expected := 0.03
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected updated cost %.4f, got %.4f", expected, cost)
	}
}

func TestUpdatePricingOverride(t *testing.T) {
	tracker := billing.NewTracker()

	// Get original gpt-4 pricing
	originalCost := tracker.CalculateCost("gpt-4", 1000, 1000)

	// Override with custom pricing
	tracker.UpdatePricing("gpt-4", "custom", 0.001, 0.001)

	// Calculate cost with updated pricing
	newCost := tracker.CalculateCost("gpt-4", 1000, 1000)

	// New cost should be different
	if originalCost == newCost {
		t.Errorf("expected pricing update to change cost, got same value %f", originalCost)
	}

	// Expected new cost: (1000/1000)*0.001 + (1000/1000)*0.001 = 0.002
	expected := 0.002
	if math.Abs(newCost-expected) > 0.001 {
		t.Errorf("expected new cost %.4f, got %.4f", expected, newCost)
	}
}

func TestUpdatePricingCaseInsensitive(t *testing.T) {
	tracker := billing.NewTracker()

	// Update with different case
	tracker.UpdatePricing("GPT-4", "provider", 0.05, 0.10)

	// Query with different case
	cost := tracker.CalculateCost("gpt-4", 1000, 1000)

	expected := 0.15
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected case-insensitive update to work, got %f", cost)
	}
}

func TestCalculateCostFractionalTokens(t *testing.T) {
	tracker := billing.NewTracker()

	// Test with small token counts
	cost := tracker.CalculateCost("gpt-4", 500, 250)

	// Expected: (500/1000)*0.03 + (250/1000)*0.06 = 0.015 + 0.015 = 0.03
	expected := 0.03
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected cost %.4f, got %.4f", expected, cost)
	}
}

func TestCalculateCostLargeTokens(t *testing.T) {
	tracker := billing.NewTracker()

	// Test with large token counts
	cost := tracker.CalculateCost("gpt-4", 100000, 100000)

	// Expected: (100000/1000)*0.03 + (100000/1000)*0.06 = 3 + 6 = 9
	expected := 9.0
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected cost %.2f, got %.2f", expected, cost)
	}
}

func TestRecordUsageCostCalculation(t *testing.T) {
	tracker := billing.NewTracker()

	// Record usage and verify cost matches calculation
	cost := tracker.RecordUsage("key-1", "gpt-4", "openai", 1000, 1000, "req-1")
	expectedCost := tracker.CalculateCost("gpt-4", 1000, 1000)

	if math.Abs(cost-expectedCost) > 0.001 {
		t.Errorf("expected recorded cost %.4f to match calculated %.4f", cost, expectedCost)
	}
}

func TestMultipleTrackerInstances(t *testing.T) {
	tracker1 := billing.NewTracker()
	tracker2 := billing.NewTracker()

	// Update pricing in tracker1
	tracker1.UpdatePricing("test-model", "provider", 1.0, 1.0)

	// tracker2 should use default pricing
	cost1 := tracker1.CalculateCost("test-model", 1000, 1000)
	cost2 := tracker2.CalculateCost("test-model", 1000, 1000)

	if cost1 == cost2 {
		t.Error("expected different costs between tracker instances after update")
	}
}

func TestClaudePricingModels(t *testing.T) {
	tracker := billing.NewTracker()

	tests := []struct {
		model    string
		minCost  float64
		maxCost  float64
	}{
		{"claude-sonnet-4-20250514", 0, 100},
		{"claude-3-5-haiku-20241022", 0, 100},
		{"claude-3-opus-20240229", 0, 100},
	}

	for _, tt := range tests {
		cost := tracker.CalculateCost(tt.model, 10000, 10000)
		if cost < tt.minCost || cost > tt.maxCost {
			t.Errorf("expected cost for %s to be between %.2f-%.2f, got %.2f", tt.model, tt.minCost, tt.maxCost, cost)
		}
	}
}

func TestGeminiPricingModels(t *testing.T) {
	tracker := billing.NewTracker()

	tests := []struct {
		model string
	}{
		{"gemini-2.5-flash"},
		{"gemini-2.5-pro"},
	}

	for _, tt := range tests {
		cost := tracker.CalculateCost(tt.model, 1000, 1000)
		if cost <= 0 {
			t.Errorf("expected positive cost for %s, got %f", tt.model, cost)
		}
	}
}

func TestMistralPricingModels(t *testing.T) {
	tracker := billing.NewTracker()

	tests := []struct {
		model string
	}{
		{"mistral-large-latest"},
		{"mistral-small-latest"},
	}

	for _, tt := range tests {
		cost := tracker.CalculateCost(tt.model, 1000, 1000)
		if cost <= 0 {
			t.Errorf("expected positive cost for %s, got %f", tt.model, cost)
		}
	}
}

func TestCostCalculationWithZeroInput(t *testing.T) {
	tracker := billing.NewTracker()

	cost := tracker.CalculateCost("gpt-4", 0, 1000)

	expected := 0.06 // Only output tokens count
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected cost %.4f, got %.4f", expected, cost)
	}
}

func TestCostCalculationWithZeroOutput(t *testing.T) {
	tracker := billing.NewTracker()

	cost := tracker.CalculateCost("gpt-4", 1000, 0)

	expected := 0.03 // Only input tokens count
	if math.Abs(cost-expected) > 0.001 {
		t.Errorf("expected cost %.4f, got %.4f", expected, cost)
	}
}

// Reporter tests
func TestNewReporter(t *testing.T) {
	reporter := billing.NewReporter("localhost:50051")

	if reporter == nil {
		t.Error("expected reporter to be created")
	}
}

func TestReporterWithDifferentAddresses(t *testing.T) {
	tests := []string{
		"localhost:50051",
		"127.0.0.1:9000",
		"waddleai:50051",
		"",
	}

	for _, addr := range tests {
		reporter := billing.NewReporter(addr)
		if reporter == nil {
			t.Errorf("expected reporter creation to succeed with address %q", addr)
		}
	}
}

func TestReporterClose(t *testing.T) {
	reporter := billing.NewReporter("localhost:50051")
	// Close without Connect should not panic
	reporter.Close()
}

func TestReporterReportAsyncNotConnected(t *testing.T) {
	reporter := billing.NewReporter("localhost:50051")

	// ReportAsync should not panic when not connected
	report := billing.UsageReport{
		UserID:       "user-1",
		APIKeyID:     "key-1",
		Model:        "gpt-4",
		Provider:     "openai",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
		LatencyMs:    100,
		RequestID:    "req-1",
		Metadata:     map[string]string{"test": "value"},
	}

	// This should return immediately without error
	reporter.ReportAsync(report)

	// Allow goroutine to finish (it shouldn't do anything)
	time.Sleep(100 * time.Millisecond)
}

func TestReporterReportAsyncWithNilClient(t *testing.T) {
	reporter := billing.NewReporter("localhost:50051")
	reporter.ReportAsync(billing.UsageReport{
		UserID:       "user-1",
		APIKeyID:     "key-1",
		Model:        "gpt-4",
		Provider:     "openai",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
		LatencyMs:    100,
		RequestID:    "req-1",
		Metadata:     map[string]string{},
	})

	time.Sleep(100 * time.Millisecond)
}

func TestUsageReportStructure(t *testing.T) {
	report := billing.UsageReport{
		UserID:       "user-123",
		APIKeyID:     "key-abc",
		Model:        "gpt-4",
		Provider:     "openai",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
		LatencyMs:    250,
		RequestID:    "req-xyz",
		Metadata: map[string]string{
			"session": "sess-1",
			"version": "v1",
		},
	}

	if report.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", report.UserID)
	}
	if report.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got %q", report.Model)
	}
	if report.TotalTokens != 1500 {
		t.Errorf("expected TotalTokens 1500, got %d", report.TotalTokens)
	}
	if len(report.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(report.Metadata))
	}
}
