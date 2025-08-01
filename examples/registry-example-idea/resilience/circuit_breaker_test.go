package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_BasicOperation(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  3,
		ResetTimeout: 100 * time.Millisecond,
	}
	
	cb := NewCircuitBreaker(config)
	
	// Should start in closed state
	if !cb.IsClosed() {
		t.Error("Expected circuit breaker to start in closed state")
	}
	
	// Successful operations should keep it closed
	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}
	
	if !cb.IsClosed() {
		t.Error("Expected circuit breaker to remain closed after successful operations")
	}
}

func TestCircuitBreaker_OpenOnFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}
	
	cb := NewCircuitBreaker(config)
	
	// Cause failures to open the circuit
	testError := errors.New("test error")
	
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return testError
		})
		if err != testError {
			t.Errorf("Expected test error, got %v", err)
		}
	}
	
	if !cb.IsOpen() {
		t.Error("Expected circuit breaker to be open after max failures")
	}
	
	// Further calls should be rejected
	err := cb.Execute(func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when circuit breaker is open")
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:      2,
		ResetTimeout:     50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}
	
	cb := NewCircuitBreaker(config)
	
	// Open the circuit
	testError := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return testError
		})
	}
	
	if !cb.IsOpen() {
		t.Error("Expected circuit breaker to be open")
	}
	
	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)
	
	// First call after timeout should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error in half-open state, got %v", err)
	}
	
	if !cb.IsHalfOpen() {
		t.Error("Expected circuit breaker to be in half-open state")
	}
}

func TestCircuitBreaker_HalfOpenToClosedTransition(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:      2,
		ResetTimeout:     50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}
	
	cb := NewCircuitBreaker(config)
	
	// Open the circuit
	testError := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return testError
		})
	}
	
	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)
	
	// Successful calls in half-open should close the circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}
	
	if !cb.IsClosed() {
		t.Error("Expected circuit breaker to be closed after successful half-open calls")
	}
}

func TestCircuitBreaker_HalfOpenToOpenTransition(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:      2,
		ResetTimeout:     50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}
	
	cb := NewCircuitBreaker(config)
	
	// Open the circuit
	testError := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return testError
		})
	}
	
	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)
	
	// First call succeeds (transitions to half-open)
	cb.Execute(func() error {
		return nil
	})
	
	// Second call fails (should go back to open)
	cb.Execute(func() error {
		return testError
	})
	
	if !cb.IsOpen() {
		t.Error("Expected circuit breaker to be open after failure in half-open state")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 100 * time.Millisecond,
	}
	
	cb := NewCircuitBreaker(config)
	
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	
	numGoroutines := 10
	numOperations := 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				err := cb.Execute(func() error {
					// Simulate some operations failing
					if (id*numOperations+j)%10 == 0 {
						return errors.New("simulated failure")
					}
					return nil
				})
				
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	stats := cb.GetStats()
	totalOps := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)
	
	if stats.TotalRequests != totalOps {
		t.Errorf("Expected total requests %d, got %d", totalOps, stats.TotalRequests)
	}
	
	t.Logf("Success: %d, Errors: %d, Total Requests: %d", successCount, errorCount, stats.TotalRequests)
}

func TestCircuitBreaker_Statistics(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  3,
		ResetTimeout: 100 * time.Millisecond,
	}
	
	cb := NewCircuitBreaker(config)
	
	// Execute some operations
	testError := errors.New("test error")
	
	// 2 successes
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return nil
		})
	}
	
	// 3 failures (should open circuit)
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return testError
		})
	}
	
	stats := cb.GetStats()
	
	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 total requests, got %d", stats.TotalRequests)
	}
	
	if stats.TotalSuccesses != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.TotalSuccesses)
	}
	
	if stats.TotalFailures != 3 {
		t.Errorf("Expected 3 failures, got %d", stats.TotalFailures)
	}
	
	if stats.State != "OPEN" {
		t.Errorf("Expected state OPEN, got %s", stats.State)
	}
	
	expectedSuccessRate := 2.0 / 5.0
	if stats.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected success rate %f, got %f", expectedSuccessRate, stats.SuccessRate)
	}
}

func TestCircuitBreaker_ContextTimeout(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 100 * time.Millisecond,
	}
	
	cb := NewCircuitBreaker(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	
	err := cb.ExecuteWithContext(ctx, func() error {
		// Simulate long-running operation
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	
	if err == nil {
		t.Error("Expected timeout error")
	}
	
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}
}

func TestCircuitBreaker_StateChangeCallbacks(t *testing.T) {
	var stateChanges []string
	var mu sync.Mutex
	
	config := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 50 * time.Millisecond,
		OnStateChange: func(from, to State) {
			mu.Lock()
			stateChanges = append(stateChanges, fmt.Sprintf("%s->%s", from, to))
			mu.Unlock()
		},
	}
	
	cb := NewCircuitBreaker(config)
	
	// Cause failures to open circuit
	testError := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return testError
		})
	}
	
	// Wait for reset timeout and try to transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.Execute(func() error {
		return nil
	})
	
	// Wait a bit for callbacks to execute
	time.Sleep(10 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	if len(stateChanges) < 1 {
		t.Error("Expected at least one state change")
	}
	
	// Should have seen CLOSED->OPEN transition
	expectedTransition := "CLOSED->OPEN"
	found := false
	for _, change := range stateChanges {
		if change == expectedTransition {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected to see transition %s, got %v", expectedTransition, stateChanges)
	}
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	config := CircuitBreakerConfig{
		MaxFailures:  100,
		ResetTimeout: 1 * time.Second,
	}
	
	cb := NewCircuitBreaker(config)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(func() error {
				return nil
			})
		}
	})
}