//go:build ci
// +build ci

package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "CLOSED"},
		{StateHalfOpen, "HALF_OPEN"},
		{StateOpen, "OPEN"},
		{State(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNewCircuitBreaker_Defaults(t *testing.T) {
	cb := NewCircuitBreaker(Config{Name: "test"})

	if cb.state != StateClosed {
		t.Errorf("initial state: got %v, want CLOSED", cb.state)
	}
	if cb.maxRequests != 1 {
		t.Errorf("maxRequests: got %d, want 1", cb.maxRequests)
	}
	if cb.interval != 60*time.Second {
		t.Errorf("interval: got %v, want 60s", cb.interval)
	}
	if cb.timeout != 60*time.Second {
		t.Errorf("timeout: got %v, want 60s", cb.timeout)
	}
	if cb.maxConcurrentRequests != 100 {
		t.Errorf("maxConcurrentRequests: got %d, want 100", cb.maxConcurrentRequests)
	}
	if cb.readyToTrip == nil {
		t.Error("readyToTrip should not be nil")
	}
	if cb.isSuccessful == nil {
		t.Error("isSuccessful should not be nil")
	}
}

func TestNewCircuitBreaker_CustomConfig(t *testing.T) {
	config := Config{
		Name:                   "custom",
		MaxRequests:            5,
		Interval:               30 * time.Second,
		Timeout:                15 * time.Second,
		MaxConcurrentRequests:  50,
		RequestVolumeThreshold: 100,
		SleepWindow:            3 * time.Second,
		ErrorPercentThreshold:  75.0,
	}

	cb := NewCircuitBreaker(config)

	if cb.maxRequests != 5 {
		t.Errorf("maxRequests: got %d, want 5", cb.maxRequests)
	}
	if cb.interval != 30*time.Second {
		t.Errorf("interval: got %v, want 30s", cb.interval)
	}
	if cb.timeout != 15*time.Second {
		t.Errorf("timeout: got %v, want 15s", cb.timeout)
	}
}

func TestCircuitBreaker_Name(t *testing.T) {
	cb := NewCircuitBreaker(Config{Name: "my-breaker"})
	if cb.Name() != "my-breaker" {
		t.Errorf("Name() = %q, want my-breaker", cb.Name())
	}
}

func TestCircuitBreaker_State(t *testing.T) {
	cb := NewCircuitBreaker(Config{Name: "test"})
	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := NewCircuitBreaker(Config{Name: "test"})
	counts := cb.Counts()

	if counts.Requests != 0 {
		t.Errorf("Requests: got %d, want 0", counts.Requests)
	}
	if counts.TotalSuccesses != 0 {
		t.Errorf("TotalSuccesses: got %d, want 0", counts.TotalSuccesses)
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:        "test",
		MaxRequests: 1,
	})

	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("Execute() result = %v, want success", result)
	}

	counts := cb.Counts()
	if counts.Requests != 1 {
		t.Errorf("Requests: got %d, want 1", counts.Requests)
	}
	if counts.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses: got %d, want 1", counts.TotalSuccesses)
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:        "test",
		MaxRequests: 1,
	})

	testErr := errors.New("test error")
	result, err := cb.Execute(func() (interface{}, error) {
		return nil, testErr
	})

	if err != testErr {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
	if result != nil {
		t.Errorf("Execute() result = %v, want nil", result)
	}

	counts := cb.Counts()
	if counts.TotalFailures != 1 {
		t.Errorf("TotalFailures: got %d, want 1", counts.TotalFailures)
	}
}

func TestCircuitBreaker_Execute_OpenCircuit(t *testing.T) {
	stateChanges := []State{}
	var mu sync.Mutex

	config := Config{
		Name:        "test",
		MaxRequests: 1,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures > 2
		},
		OnStateChange: func(name string, from, to State) {
			mu.Lock()
			stateChanges = append(stateChanges, to)
			mu.Unlock()
		},
	}

	cb := NewCircuitBreaker(config)

	// Trigger failures to open circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("fail")
		})
	}

	// Check circuit is open
	if cb.State() != StateOpen {
		t.Errorf("State: got %v, want OPEN", cb.State())
	}

	// Try to execute - should get error immediately
	_, err := cb.Execute(func() (interface{}, error) {
		return "should not execute", nil
	})

	if err != ErrCircuitBreakerOpen {
		t.Errorf("Execute() error = %v, want ErrCircuitBreakerOpen", err)
	}

	// Verify state change was recorded
	if len(stateChanges) == 0 || stateChanges[len(stateChanges)-1] != StateOpen {
		t.Error("state change to OPEN not recorded")
	}
}

func TestCircuitBreaker_ExecuteWithContext_Timeout(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:    "test",
		Timeout: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := cb.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	if err == nil {
		t.Error("ExecuteWithContext should timeout")
	}
}

func TestCircuitBreaker_ExecuteWithContext_Success(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:    "test",
		Timeout: 1 * time.Second,
	})

	ctx := context.Background()
	result, err := cb.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		return "done", nil
	})

	if err != nil {
		t.Errorf("ExecuteWithContext error = %v, want nil", err)
	}
	if result != "done" {
		t.Errorf("result = %v, want done", result)
	}
}

func TestCircuitBreaker_Statistics(t *testing.T) {
	cb := NewCircuitBreaker(Config{Name: "test"})

	// Execute some requests
	cb.Execute(func() (interface{}, error) { return "ok", nil })
	cb.Execute(func() (interface{}, error) { return nil, errors.New("err") })
	cb.Execute(func() (interface{}, error) { return "ok", nil })

	stats := cb.Statistics()

	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests: got %d, want 3", stats.TotalRequests)
	}
	if stats.TotalSuccesses != 2 {
		t.Errorf("TotalSuccesses: got %d, want 2", stats.TotalSuccesses)
	}
	if stats.TotalFailures != 1 {
		t.Errorf("TotalFailures: got %d, want 1", stats.TotalFailures)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:        "test",
		ReadyToTrip: func(counts Counts) bool { return counts.ConsecutiveFailures > 1 },
	})

	// Trigger failures
	cb.Execute(func() (interface{}, error) { return nil, errors.New("fail") })
	cb.Execute(func() (interface{}, error) { return nil, errors.New("fail") })

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN state before reset")
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("State after reset: got %v, want CLOSED", cb.State())
	}

	counts := cb.Counts()
	if counts.Requests != 0 || counts.TotalFailures != 0 {
		t.Errorf("counts not reset: %+v", counts)
	}
}

func TestServiceCircuitBreaker_GetBreaker(t *testing.T) {
	scb := NewServiceCircuitBreaker(Config{Name: "svc-breaker"})

	svc1 := &manager.Service{ID: 1, Name: "svc1", Host: "host1", Port: 8080}
	svc2 := &manager.Service{ID: 2, Name: "svc2", IPFQDN: "svc2.example.com"}

	breaker1 := scb.GetBreaker(svc1)
	breaker2 := scb.GetBreaker(svc2)

	if breaker1 == nil {
		t.Error("expected non-nil breaker1")
	}
	if breaker2 == nil {
		t.Error("expected non-nil breaker2")
	}

	// Same service should return same breaker
	breaker1Again := scb.GetBreaker(svc1)
	if breaker1 != breaker1Again {
		t.Error("should return same breaker for same service")
	}
}

func TestServiceCircuitBreaker_ExecuteRequest(t *testing.T) {
	scb := NewServiceCircuitBreaker(Config{Name: "test"})
	svc := &manager.Service{ID: 1, Name: "test-svc", Host: "host", Port: 8080}

	result, err := scb.ExecuteRequest(svc, func() (interface{}, error) {
		return "executed", nil
	})

	if err != nil {
		t.Errorf("ExecuteRequest error = %v, want nil", err)
	}
	if result != "executed" {
		t.Errorf("result = %v, want executed", result)
	}
}

func TestServiceCircuitBreaker_RemoveBreaker(t *testing.T) {
	scb := NewServiceCircuitBreaker(Config{Name: "test"})
	svc := &manager.Service{ID: 1, Name: "test", Host: "host", Port: 8080}

	// Get breaker
	_ = scb.GetBreaker(svc)

	breakers := scb.GetAllBreakers()
	if len(breakers) != 1 {
		t.Fatalf("expected 1 breaker, got %d", len(breakers))
	}

	// Remove breaker
	scb.RemoveBreaker(svc)

	breakers = scb.GetAllBreakers()
	if len(breakers) != 0 {
		t.Errorf("expected 0 breakers after removal, got %d", len(breakers))
	}
}

func TestServiceCircuitBreaker_ResetAll(t *testing.T) {
	scb := NewServiceCircuitBreaker(Config{
		Name:        "test",
		ReadyToTrip: func(c Counts) bool { return c.ConsecutiveFailures > 1 },
	})

	svc1 := &manager.Service{ID: 1, Name: "svc1", Host: "h1", Port: 1}
	svc2 := &manager.Service{ID: 2, Name: "svc2", Host: "h2", Port: 2}

	// Get breakers
	b1 := scb.GetBreaker(svc1)
	b2 := scb.GetBreaker(svc2)

	// Trigger failures on both
	b1.Execute(func() (interface{}, error) { return nil, errors.New("fail") })
	b1.Execute(func() (interface{}, error) { return nil, errors.New("fail") })
	b2.Execute(func() (interface{}, error) { return nil, errors.New("fail") })
	b2.Execute(func() (interface{}, error) { return nil, errors.New("fail") })

	// ResetAll
	scb.ResetAll()

	// Verify all are closed
	if b1.State() != StateClosed || b2.State() != StateClosed {
		t.Error("breakers not reset to closed")
	}
}

func TestCircuitBreaker_Counts_Operations(t *testing.T) {
	var c Counts

	c.onRequest()
	if c.Requests != 1 {
		t.Errorf("Requests: got %d, want 1", c.Requests)
	}

	c.onSuccess()
	if c.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses: got %d, want 1", c.TotalSuccesses)
	}
	if c.ConsecutiveSuccesses != 1 {
		t.Errorf("ConsecutiveSuccesses: got %d, want 1", c.ConsecutiveSuccesses)
	}

	c.onFailure()
	if c.TotalFailures != 1 {
		t.Errorf("TotalFailures: got %d, want 1", c.TotalFailures)
	}
	if c.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures: got %d, want 1", c.ConsecutiveFailures)
	}
	if c.ConsecutiveSuccesses != 0 {
		t.Errorf("ConsecutiveSuccesses should be 0 after failure, got %d", c.ConsecutiveSuccesses)
	}

	c.clear()
	if c.Requests != 0 || c.TotalSuccesses != 0 || c.TotalFailures != 0 {
		t.Error("counts not fully cleared")
	}
}

func TestCircuitBreaker_ErrorRate(t *testing.T) {
	var c Counts

	if rate := c.ErrorRate(); rate != 0.0 {
		t.Errorf("empty counts error rate: got %f, want 0.0", rate)
	}

	c.onRequest()
	c.onSuccess()
	c.onRequest()
	c.onFailure()

	rate := c.ErrorRate()
	expectedRate := 50.0
	if rate != expectedRate {
		t.Errorf("error rate: got %f, want %f", rate, expectedRate)
	}
}

func TestCircuitBreaker_ConcurrentRequests(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:                  "test",
		MaxConcurrentRequests: 5,
	})

	var wg sync.WaitGroup
	results := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cb.Execute(func() (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return nil, nil
			})
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	for err := range results {
		if err != nil && err != ErrTooManyRequests {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestCircuitBreaker_GetMetrics(t *testing.T) {
	cb := NewCircuitBreaker(Config{Name: "test"})

	cb.Execute(func() (interface{}, error) { return nil, nil })
	cb.Execute(func() (interface{}, error) { return nil, errors.New("err") })

	metrics := cb.GetMetrics()

	if metrics.Name != "test" {
		t.Errorf("Name: got %s, want test", metrics.Name)
	}
	if metrics.State != "CLOSED" {
		t.Errorf("State: got %s, want CLOSED", metrics.State)
	}
	if metrics.TotalRequests != 2 {
		t.Errorf("TotalRequests: got %d, want 2", metrics.TotalRequests)
	}
	if metrics.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses: got %d, want 1", metrics.TotalSuccesses)
	}
	if metrics.TotalFailures != 1 {
		t.Errorf("TotalFailures: got %d, want 1", metrics.TotalFailures)
	}
}
