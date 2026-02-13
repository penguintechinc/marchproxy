package circuitbreaker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

func TestCircuitBreakerStates(t *testing.T) {
	config := Config{
		Name:        "test",
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}

	cb := NewCircuitBreaker(config)

	if cb.State() != StateClosed {
		t.Errorf("expected initial state to be Closed, got %v", cb.State())
	}

	for i := 0; i < 3; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
		if err == nil {
			t.Error("expected error but got none")
		}
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state to be Open after failures, got %v", cb.State())
	}

	_, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != ErrCircuitBreakerOpen {
		t.Errorf("expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	config := Config{
		Name:        "test-recovery",
		MaxRequests: 2,
		Interval:    time.Millisecond * 100,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
		SleepWindow: time.Millisecond * 50,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state to be Open, got %v", cb.State())
	}

	time.Sleep(time.Millisecond * 60)

	if cb.State() != StateHalfOpen {
		t.Errorf("expected state to be HalfOpen after sleep window, got %v", cb.State())
	}

	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != nil {
		t.Errorf("expected success in half-open state, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state to be Closed after successful request, got %v", cb.State())
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	config := Config{
		Name:                  "test-concurrency",
		MaxRequests:          10,
		MaxConcurrentRequests: 5,
		Interval:             time.Minute,
		Timeout:              time.Second,
	}

	cb := NewCircuitBreaker(config)

	done := make(chan struct{})
	var errorCount int64

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, err := cb.Execute(func() (interface{}, error) {
				time.Sleep(time.Millisecond * 100)
				return "success", nil
			})
			if err == ErrTooManyRequests {
				atomic.AddInt64(&errorCount, 1)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if atomic.LoadInt64(&errorCount) == 0 {
		t.Error("expected some requests to be rejected due to concurrency limit")
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	config := Config{
		Name:        "test-metrics",
		MaxRequests: 10,
		Interval:    time.Minute,
		Timeout:     time.Second,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 5; i++ {
		cb.Execute(func() (interface{}, error) {
			return "success", nil
		})
	}

	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	metrics := cb.GetMetrics()
	if metrics.TotalRequests != 8 {
		t.Errorf("expected 8 total requests, got %d", metrics.TotalRequests)
	}
	if metrics.TotalSuccesses != 5 {
		t.Errorf("expected 5 successes, got %d", metrics.TotalSuccesses)
	}
	if metrics.TotalFailures != 3 {
		t.Errorf("expected 3 failures, got %d", metrics.TotalFailures)
	}

	expectedErrorRate := 3.0 / 8.0 * 100
	if metrics.ErrorRate != expectedErrorRate {
		t.Errorf("expected error rate %.2f%%, got %.2f%%", expectedErrorRate, metrics.ErrorRate)
	}
}

func TestServiceCircuitBreaker(t *testing.T) {
	config := Config{
		Name:        "service-test",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}

	scb := NewServiceCircuitBreaker(config)

	service1 := &manager.Service{
		Host: "service1.example.com",
		Port: 8080,
	}

	service2 := &manager.Service{
		Host: "service2.example.com",
		Port: 8080,
	}

	breaker1 := scb.GetBreaker(service1)
	breaker2 := scb.GetBreaker(service2)

	if breaker1 == breaker2 {
		t.Error("expected different breakers for different services")
	}

	if breaker1.name != "service1.example.com:8080" {
		t.Errorf("expected breaker name 'service1.example.com:8080', got %s", breaker1.name)
	}

	result, err := scb.ExecuteRequest(service1, func() (interface{}, error) {
		return "service1-response", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "service1-response" {
		t.Errorf("expected 'service1-response', got %v", result)
	}

	for i := 0; i < 2; i++ {
		scb.ExecuteRequest(service1, func() (interface{}, error) {
			return nil, errors.New("service1 failure")
		})
	}

	if breaker1.State() != StateOpen {
		t.Errorf("expected service1 breaker to be Open, got %v", breaker1.State())
	}

	if breaker2.State() != StateClosed {
		t.Errorf("expected service2 breaker to be Closed, got %v", breaker2.State())
	}
}

func TestCircuitBreakerWithContext(t *testing.T) {
	config := Config{
		Name:        "test-context",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Millisecond * 500,
	}

	cb := NewCircuitBreaker(config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	_, err := cb.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		// Sleep longer than the context timeout to ensure ctx expires
		select {
		case <-time.After(time.Second):
			return "success", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context deadline exceeded, got %v", err)
	}
}

func TestCircuitBreakerFallback(t *testing.T) {
	fallbackCalled := false
	config := Config{
		Name:        "test-fallback",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
		FallbackFunc: func(ctx context.Context, err error) (interface{}, error) {
			fallbackCalled = true
			return "fallback-response", nil
		},
	}

	cb := NewCircuitBreaker(config)

	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected fallback to handle error, got %v", err)
	}
	if result != "fallback-response" {
		t.Errorf("expected 'fallback-response', got %v", result)
	}
	if !fallbackCalled {
		t.Error("expected fallback function to be called")
	}

	metrics := cb.GetMetrics()
	if metrics.TotalFallbacks == 0 {
		t.Error("expected fallback to be recorded in metrics")
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	config := Config{
		Name:        "test-reset",
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 2; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state to be Open, got %v", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected state to be Closed after reset, got %v", cb.State())
	}

	counts := cb.Counts()
	if counts.Requests != 0 || counts.TotalFailures != 0 || counts.ConsecutiveFailures != 0 {
		t.Error("expected counts to be reset to zero")
	}
}

func TestAdvancedCircuitBreakerConfig(t *testing.T) {
	config := AdvancedCircuitBreakerConfig()

	if config.MaxRequests != 5 {
		t.Errorf("expected MaxRequests 5, got %d", config.MaxRequests)
	}
	if config.ErrorPercentThreshold != 30.0 {
		t.Errorf("expected ErrorPercentThreshold 30.0, got %.1f", config.ErrorPercentThreshold)
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 10; i++ {
		cb.Execute(func() (interface{}, error) {
			if i < 7 {
				return "success", nil
			}
			return nil, errors.New("failure")
		})
	}

	counts := cb.Counts()
	if counts.ErrorRate() < 30.0 && cb.State() != StateOpen {
		t.Error("expected circuit breaker to trip with 30% error rate")
	}
}

func BenchmarkCircuitBreakerExecution(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(func() (interface{}, error) {
				return "success", nil
			})
		}
	})
}

func BenchmarkServiceCircuitBreaker(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	scb := NewServiceCircuitBreaker(config)

	service := &manager.Service{
		Host: "test.example.com",
		Port: 8080,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			scb.ExecuteRequest(service, func() (interface{}, error) {
				return "success", nil
			})
		}
	})
}