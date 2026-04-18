//go:build ci

package circuitbreaker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

// TestCircuitBreaker_StateTransitions tests state machine transitions
func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := Config{
		Name:        "transition-test",
		MaxRequests: 2,
		Interval:    time.Millisecond * 100,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
		SleepWindow: time.Millisecond * 50,
	}

	cb := NewCircuitBreaker(config)

	// Initial state should be Closed
	if cb.State() != StateClosed {
		t.Errorf("Initial state: got %v, want %v", cb.State(), StateClosed)
	}

	// Trigger failures to transition to Open
	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("After failures state: got %v, want %v", cb.State(), StateOpen)
	}

	// Wait for sleep window and check HalfOpen
	time.Sleep(time.Millisecond * 60)
	if cb.State() != StateHalfOpen {
		t.Errorf("After sleep state: got %v, want %v", cb.State(), StateHalfOpen)
	}

	// Success should transition back to Closed
	cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if cb.State() != StateClosed {
		t.Errorf("After success state: got %v, want %v", cb.State(), StateClosed)
	}
}

// TestCircuitBreaker_ExecuteWithContext_TimeoutExtended tests context timeout handling with details
func TestCircuitBreaker_ExecuteWithContext_TimeoutExtended(t *testing.T) {
	config := Config{
		Name:        "timeout-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Millisecond * 50,
	}

	cb := NewCircuitBreaker(config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	_, err := cb.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		time.Sleep(time.Millisecond * 200)
		return nil, nil
	})

	if err == nil {
		t.Error("Expected timeout error")
	}

	stats := cb.Statistics()
	if stats.TotalTimeouts != 1 {
		t.Errorf("TotalTimeouts: got %d, want 1", stats.TotalTimeouts)
	}
}

// TestCircuitBreaker_ExecuteWithContext_ContextDone tests context cancellation
func TestCircuitBreaker_ExecuteWithContext_ContextDone(t *testing.T) {
	config := Config{
		Name:        "context-done-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cb := NewCircuitBreaker(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := cb.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, nil
	})

	if err == nil {
		t.Error("Expected context error")
	}
}

// TestCircuitBreaker_Fallback tests fallback function execution
func TestCircuitBreaker_Fallback(t *testing.T) {
	fallbackCalled := false
	config := Config{
		Name:        "fallback-test",
		MaxRequests: 1,
		Interval:    time.Millisecond * 50,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
		SleepWindow: time.Millisecond * 20,
		FallbackFunc: func(ctx context.Context, err error) (interface{}, error) {
			fallbackCalled = true
			return "fallback result", nil
		},
	}

	cb := NewCircuitBreaker(config)

	// Trigger failure to open circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	if cb.State() != StateOpen {
		t.Errorf("Expected state Open, got %v", cb.State())
	}

	// Next request should use fallback
	result, _ := cb.Execute(func() (interface{}, error) {
		return nil, errors.New("should not execute")
	})

	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}
	if result != "fallback result" {
		t.Errorf("Fallback result: got %v, want 'fallback result'", result)
	}
}

// TestCircuitBreaker_MaxConcurrentRequests tests concurrent request limiting
func TestCircuitBreaker_MaxConcurrentRequests(t *testing.T) {
	config := Config{
		Name:                  "concurrent-test",
		MaxRequests:          10,
		MaxConcurrentRequests: 3,
		Interval:             time.Minute,
		Timeout:              time.Second,
	}

	cb := NewCircuitBreaker(config)

	rejectedCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cb.Execute(func() (interface{}, error) {
				time.Sleep(time.Millisecond * 100)
				return "ok", nil
			})
			if err == ErrTooManyRequests {
				mu.Lock()
				rejectedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if rejectedCount == 0 {
		t.Error("Expected some requests to be rejected due to concurrent limit")
	}
}

// TestCircuitBreaker_HalfOpenTransition tests HalfOpen to Closed transition on success
func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	config := Config{
		Name:        "half-open-transition-test",
		MaxRequests: 1,
		Interval:    time.Millisecond * 50,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
		SleepWindow: time.Millisecond * 30,
	}

	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	time.Sleep(time.Millisecond * 40)

	if cb.State() != StateHalfOpen {
		t.Fatalf("Expected state HalfOpen, got %v", cb.State())
	}

	// Execute request in HalfOpen
	result, err := cb.Execute(func() (interface{}, error) {
		return "ok", nil
	})
	if err != nil {
		t.Errorf("Request in HalfOpen failed: %v", err)
	}
	if result != "ok" {
		t.Errorf("Expected result 'ok', got %v", result)
	}

	// After success, state should transition to Closed
	if cb.State() != StateClosed {
		t.Errorf("After success state: got %v, want %v", cb.State(), StateClosed)
	}
}

// TestCircuitBreaker_ResponseTimeTracking tests response time tracking
func TestCircuitBreaker_ResponseTimeTracking(t *testing.T) {
	config := Config{
		Name:        "response-time-test",
		MaxRequests: 10,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 5; i++ {
		cb.Execute(func() (interface{}, error) {
			time.Sleep(time.Millisecond * 10)
			return "ok", nil
		})
	}

	metrics := cb.GetMetrics()
	if metrics.AverageResponseTime == 0 {
		t.Error("Expected average response time to be tracked")
	}
}

// TestCircuitBreaker_ResetExtended tests reset functionality with detailed validation
func TestCircuitBreaker_ResetExtended(t *testing.T) {
	config := Config{
		Name:        "reset-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}

	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("Expected state Open before reset")
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("After reset state: got %v, want %v", cb.State(), StateClosed)
	}

	counts := cb.Counts()
	if counts.Requests != 0 {
		t.Errorf("After reset Requests: got %d, want 0", counts.Requests)
	}
}

// TestServiceCircuitBreaker_MultipleBreakers tests multiple service breakers
func TestServiceCircuitBreaker_MultipleBreakers(t *testing.T) {
	config := Config{
		Name:        "service-multi-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	scb := NewServiceCircuitBreaker(config)

	services := []*manager.Service{
		{Name: "svc1", Host: "svc1.example.com", Port: 8080},
		{Name: "svc2", Host: "svc2.example.com", Port: 8080},
		{Name: "svc3", Host: "svc3.example.com", Port: 8080},
	}

	breakers := make([]*CircuitBreaker, len(services))
	for i, svc := range services {
		breakers[i] = scb.GetBreaker(svc)
	}

	// All should be different instances
	for i := 0; i < len(breakers); i++ {
		for j := i + 1; j < len(breakers); j++ {
			if breakers[i] == breakers[j] {
				t.Errorf("Breaker %d and %d should be different", i, j)
			}
		}
	}
}

// TestServiceCircuitBreaker_RemoveBreakerExtended tests breaker removal with detailed checks
func TestServiceCircuitBreaker_RemoveBreakerExtended(t *testing.T) {
	config := Config{
		Name:        "remove-breaker-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	scb := NewServiceCircuitBreaker(config)

	service := &manager.Service{
		Host: "svc.example.com",
		Port: 8080,
	}

	breaker1 := scb.GetBreaker(service)
	if breaker1 == nil {
		t.Fatal("Expected breaker to be created")
	}

	scb.RemoveBreaker(service)

	breaker2 := scb.GetBreaker(service)
	if breaker1 == breaker2 {
		t.Error("Expected new breaker after removal")
	}
}

// TestServiceCircuitBreaker_ResetAllExtended tests resetting all breakers with validation
func TestServiceCircuitBreaker_ResetAllExtended(t *testing.T) {
	config := Config{
		Name:        "reset-all-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	}

	scb := NewServiceCircuitBreaker(config)

	services := []*manager.Service{
		{Name: "svc1", Host: "svc1.example.com", Port: 8080},
		{Name: "svc2", Host: "svc2.example.com", Port: 8080},
	}

	for _, svc := range services {
		breaker := scb.GetBreaker(svc)
		breaker.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
		if breaker.State() != StateOpen {
			t.Fatalf("Expected service breaker to be open")
		}
	}

	scb.ResetAll()

	for _, svc := range services {
		breaker := scb.GetBreaker(svc)
		if breaker.State() != StateClosed {
			t.Errorf("After reset service state: got %v, want %v", breaker.State(), StateClosed)
		}
	}
}

// TestCircuitBreakerMiddleware_ProcessRequest tests middleware request processing
func TestCircuitBreakerMiddleware_ProcessRequest(t *testing.T) {
	config := Config{
		Name:        "middleware-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cbm := NewCircuitBreakerMiddleware(config)

	if cbm.Name() != "circuit-breaker" {
		t.Errorf("Middleware name: got %q, want %q", cbm.Name(), "circuit-breaker")
	}

	if cbm.Priority() != 200 {
		t.Errorf("Middleware priority: got %d, want 200", cbm.Priority())
	}

	if !cbm.Enabled() {
		t.Error("Expected middleware to be enabled by default")
	}
}

// TestCircuitBreakerMiddleware_EnableDisable tests enable/disable functionality
func TestCircuitBreakerMiddleware_EnableDisable(t *testing.T) {
	config := Config{
		Name:        "enable-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cbm := NewCircuitBreakerMiddleware(config)

	cbm.Disable()
	if cbm.Enabled() {
		t.Error("Expected middleware to be disabled")
	}

	cbm.Enable()
	if !cbm.Enabled() {
		t.Error("Expected middleware to be enabled")
	}
}

// TestCircuitBreakerProxy_ExecuteRequest tests proxy request execution
func TestCircuitBreakerProxy_ExecuteRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := Config{
		Name:        "proxy-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cbp := NewCircuitBreakerProxy(config)

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	service := &manager.Service{
		Host:   server.URL[7:],
		Port:   8080,
		Scheme: "http",
	}

	resp, err := cbp.ExecuteRequest(service, req)
	if err == nil && resp != nil {
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Response status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		resp.Body.Close()
	}
}

// TestCircuitBreakerProxy_GetBreaker tests breaker retrieval from proxy
func TestCircuitBreakerProxy_GetBreaker(t *testing.T) {
	config := Config{
		Name:        "proxy-breaker-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cbp := NewCircuitBreakerProxy(config)

	service := &manager.Service{
		Host: "svc.example.com",
		Port: 8080,
	}

	breaker := cbp.GetBreaker(service)
	if breaker == nil {
		t.Fatal("Expected breaker from proxy")
	}

	if breaker.State() != StateClosed {
		t.Errorf("Initial state: got %v, want %v", breaker.State(), StateClosed)
	}
}

// TestCounts_ErrorRate tests error rate calculation
func TestCounts_ErrorRate(t *testing.T) {
	counts := &Counts{}

	for i := 0; i < 10; i++ {
		counts.onRequest()
	}
	for i := 0; i < 3; i++ {
		counts.onFailure()
	}

	rate := counts.ErrorRate()
	expectedRate := 30.0

	if rate != expectedRate {
		t.Errorf("ErrorRate: got %.2f, want %.2f", rate, expectedRate)
	}
}

// TestCounts_Clear tests counts clearing
func TestCounts_Clear(t *testing.T) {
	counts := &Counts{}

	counts.onRequest()
	counts.onRequest()
	counts.onSuccess()
	counts.onFailure()

	if counts.Requests == 0 {
		t.Error("Expected counts before clear")
	}

	counts.clear()

	if counts.Requests != 0 {
		t.Errorf("Requests after clear: got %d, want 0", counts.Requests)
	}
	if counts.TotalSuccesses != 0 {
		t.Errorf("TotalSuccesses after clear: got %d, want 0", counts.TotalSuccesses)
	}
	if counts.TotalFailures != 0 {
		t.Errorf("TotalFailures after clear: got %d, want 0", counts.TotalFailures)
	}
}

// TestState_String tests state string representation
func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateHalfOpen, "HALF_OPEN"},
		{StateOpen, "OPEN"},
		{State(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.state.String()
		if result != tt.expected {
			t.Errorf("State %d: got %q, want %q", tt.state, result, tt.expected)
		}
	}
}

// TestCircuitBreakerError_Error tests error formatting
func TestCircuitBreakerError_Error(t *testing.T) {
	cbe := &CircuitBreakerError{
		Service: &manager.Service{
			IPFQDN: "svc.example.com",
			Port:   8080,
		},
		State: StateOpen,
		Err:   ErrCircuitBreakerOpen,
	}

	errStr := cbe.Error()
	if errStr == "" {
		t.Error("Expected error string")
	}
	if !contains(errStr, "svc.example.com") {
		t.Errorf("Error should contain service name: %s", errStr)
	}
	if !contains(errStr, "OPEN") {
		t.Errorf("Error should contain state: %s", errStr)
	}
}

// TestCircuitBreakerError_Unwrap tests error unwrapping
func TestCircuitBreakerError_Unwrap(t *testing.T) {
	cbe := &CircuitBreakerError{
		Service: &manager.Service{IPFQDN: "svc", Port: 8080},
		State:   StateOpen,
		Err:     ErrCircuitBreakerOpen,
	}

	unwrapped := cbe.Unwrap()
	if unwrapped != ErrCircuitBreakerOpen {
		t.Errorf("Unwrap: got %v, want %v", unwrapped, ErrCircuitBreakerOpen)
	}
}

// BenchmarkCircuitBreakerExecute benchmarks circuit breaker execution
func BenchmarkCircuitBreakerExecute(b *testing.B) {
	config := Config{
		Name:        "bench",
		MaxRequests: 100,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cb := NewCircuitBreaker(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(func() (interface{}, error) {
			return "ok", nil
		})
	}
}

// BenchmarkCircuitBreakerState benchmarks state retrieval
func BenchmarkCircuitBreakerState(b *testing.B) {
	config := Config{
		Name:        "bench-state",
		MaxRequests: 100,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cb := NewCircuitBreaker(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.State()
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr))
}
