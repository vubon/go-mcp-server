package mcpserver

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:      5,
		Timeout:          "60s",
		HalfOpenMaxCalls: 3,
		SuccessThreshold: 2,
	}

	cb := NewCircuitBreaker("test-service", config)

	if cb == nil {
		t.Fatal("Expected circuit breaker, got nil")
	}
	if cb.name != "test-service" {
		t.Errorf("Expected name 'test-service', got %q", cb.name)
	}
	if cb.maxFailures != 5 {
		t.Errorf("Expected maxFailures 5, got %d", cb.maxFailures)
	}
	if cb.timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", cb.timeout)
	}
	if cb.halfOpenMaxCalls != 3 {
		t.Errorf("Expected halfOpenMaxCalls 3, got %d", cb.halfOpenMaxCalls)
	}
	if cb.successThreshold != 2 {
		t.Errorf("Expected successThreshold 2, got %d", cb.successThreshold)
	}
}

func TestNewCircuitBreaker_Defaults(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{})

	if cb.maxFailures != 5 {
		t.Errorf("Expected default maxFailures 5, got %d", cb.maxFailures)
	}
	if cb.timeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", cb.timeout)
	}
	if cb.halfOpenMaxCalls != 3 {
		t.Errorf("Expected default halfOpenMaxCalls 3, got %d", cb.halfOpenMaxCalls)
	}
	if cb.successThreshold != 2 {
		t.Errorf("Expected default successThreshold 2, got %d", cb.successThreshold)
	}
}

func TestCircuitBreaker_Allow_Closed(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures: 5,
	})

	// Circuit should be closed initially
	if !cb.Allow() {
		t.Error("Expected circuit to allow requests when closed")
	}
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures: 3,
	})

	// Circuit should be closed initially
	if !cb.Allow() {
		t.Error("Expected circuit to allow requests when closed")
	}

	// Record failures
	cb.OnFailure()
	cb.OnFailure()
	cb.OnFailure()

	// Circuit should now be open
	if cb.Allow() {
		t.Error("Expected circuit to reject requests when open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures: 2,
		Timeout:     "100ms",
	})

	// Open circuit
	cb.OnFailure()
	cb.OnFailure()

	// Circuit should be open
	if cb.Allow() {
		t.Error("Expected circuit to reject requests when open")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should be half-open now
	if !cb.Allow() {
		t.Error("Expected circuit to allow requests when half-open")
	}
}

func TestCircuitBreaker_HalfOpenMaxCalls(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          "100ms",
		HalfOpenMaxCalls: 2,
	})

	// Open circuit
	cb.OnFailure()
	cb.OnFailure()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should allow up to HalfOpenMaxCalls (2 calls)
	// First call should be allowed
	if !cb.Allow() {
		t.Error("Expected first call in half-open to be allowed")
	}
	// Second call should be allowed
	if !cb.Allow() {
		t.Error("Expected second call in half-open to be allowed")
	}
	// Third call should be rejected (exceeds HalfOpenMaxCalls of 2)
	// Note: The check is `calls < int32(cb.halfOpenMaxCalls)`, so when calls == 2, it should reject
	if cb.Allow() {
		t.Error("Expected third call in half-open to be rejected (exceeds HalfOpenMaxCalls)")
	}
}

func TestCircuitBreaker_CloseAfterSuccessThreshold(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          "100ms",
		HalfOpenMaxCalls: 3,
		SuccessThreshold: 2,
	})

	// Open circuit
	cb.OnFailure()
	cb.OnFailure()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Record successes in half-open state
	cb.Allow() // First call
	cb.OnSuccess()
	cb.Allow() // Second call
	cb.OnSuccess()

	// Circuit should be closed now
	state := CircuitBreakerState(cb.state)
	if state != StateClosed {
		t.Errorf("Expected circuit to be closed, got state %d", state)
	}

	// Should allow requests
	if !cb.Allow() {
		t.Error("Expected circuit to allow requests when closed")
	}
}

func TestCircuitBreaker_ReopenOnFailureInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          "100ms",
		HalfOpenMaxCalls: 3,
		SuccessThreshold: 2,
	})

	// Open circuit
	cb.OnFailure()
	cb.OnFailure()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Record failure in half-open state
	cb.Allow()
	cb.OnFailure()

	// Circuit should be open again
	state := CircuitBreakerState(cb.state)
	if state != StateOpen {
		t.Errorf("Expected circuit to be open, got state %d", state)
	}

	// Should reject requests
	if cb.Allow() {
		t.Error("Expected circuit to reject requests when open")
	}
}

func TestCircuitBreaker_ResetFailuresOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures: 3,
	})

	// Record some failures
	cb.OnFailure()
	cb.OnFailure()

	// Record success
	cb.OnSuccess()

	// Failures should be reset
	failures := atomic.LoadInt32(&cb.failures)
	if failures != 0 {
		t.Errorf("Expected failures to be reset to 0, got %d", failures)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker("test-service", &CircuitBreakerConfig{
		MaxFailures: 10,
	})

	var wg sync.WaitGroup
	concurrency := 100

	// Concurrently call Allow, OnSuccess, OnFailure
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Allow()
			if i%2 == 0 {
				cb.OnSuccess()
			} else {
				cb.OnFailure()
			}
		}()
	}

	wg.Wait()

	// Should not panic and should have consistent state
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))
	if state != StateClosed && state != StateOpen {
		t.Errorf("Expected valid state, got %d", state)
	}
}

func TestCircuitBreakerManager_GetCircuitBreaker(t *testing.T) {
	manager := &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}

	config := &CircuitBreakerConfig{
		MaxFailures: 5,
	}

	// Get circuit breaker for service
	cb1 := manager.GetCircuitBreaker("service1", config)
	if cb1 == nil {
		t.Fatal("Expected circuit breaker, got nil")
	}

	// Get again - should return same instance
	cb2 := manager.GetCircuitBreaker("service1", config)
	if cb1 != cb2 {
		t.Error("Expected same circuit breaker instance")
	}

	// Get for different service - should return different instance
	cb3 := manager.GetCircuitBreaker("service2", config)
	if cb1 == cb3 {
		t.Error("Expected different circuit breaker instance for different service")
	}
}

func TestCircuitBreakerManager_NilConfig(t *testing.T) {
	manager := &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}

	cb := manager.GetCircuitBreaker("service1", nil)
	if cb != nil {
		t.Error("Expected nil circuit breaker when config is nil")
	}
}

func TestGlobalCBManager(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures: 5,
	}

	cb1 := globalCBManager.GetCircuitBreaker("test-service", config)
	if cb1 == nil {
		t.Fatal("Expected circuit breaker, got nil")
	}

	cb2 := globalCBManager.GetCircuitBreaker("test-service", config)
	if cb1 != cb2 {
		t.Error("Expected same circuit breaker instance from global manager")
	}
}
