package mcpserver

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int32

const (
	// StateClosed represents a closed circuit (normal operation)
	StateClosed CircuitBreakerState = iota
	// StateOpen represents an open circuit (backend is down)
	StateOpen
	// StateHalfOpen represents a half-open circuit (testing recovery)
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	maxFailures      int
	timeout          time.Duration
	halfOpenMaxCalls int
	successThreshold int

	// State management (atomic operations for thread safety)
	state             int32 // atomic: CircuitBreakerState
	failures          int32 // atomic: consecutive failures
	halfOpenCalls     int32 // atomic: calls in half-open state
	halfOpenSuccesses int32 // atomic: successes in half-open state
	lastFailureTime   int64 // atomic: unix timestamp of last failure
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	maxFailures := 5
	if config.MaxFailures > 0 {
		maxFailures = config.MaxFailures
	}

	timeout := 60 * time.Second
	if config.Timeout != "" {
		if d, err := time.ParseDuration(config.Timeout); err == nil {
			timeout = d
		}
	}

	halfOpenMaxCalls := 3
	if config.HalfOpenMaxCalls > 0 {
		halfOpenMaxCalls = config.HalfOpenMaxCalls
	}

	successThreshold := 2
	if config.SuccessThreshold > 0 {
		successThreshold = config.SuccessThreshold
	}

	return &CircuitBreaker{
		name:             name,
		maxFailures:      maxFailures,
		timeout:          timeout,
		halfOpenMaxCalls: halfOpenMaxCalls,
		successThreshold: successThreshold,
		state:            int32(StateClosed),
	}
}

// Allow checks if the circuit breaker allows the request
func (cb *CircuitBreaker) Allow() bool {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		lastFailureTime := atomic.LoadInt64(&cb.lastFailureTime)
		if lastFailureTime > 0 {
			elapsed := time.Since(time.Unix(0, lastFailureTime))
			if elapsed >= cb.timeout {
				// Move to half-open state
				if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
					atomic.StoreInt32(&cb.halfOpenCalls, 1) // Count the transition call
					atomic.StoreInt32(&cb.halfOpenSuccesses, 0)
					return true
				}
			}
		}
		return false

	case StateHalfOpen:
		// Allow limited calls in half-open state
		calls := atomic.LoadInt32(&cb.halfOpenCalls)
		// Safe conversion: halfOpenMaxCalls is validated to be > 0 and reasonable
		//nolint:gosec // Integer conversion is safe (validated config value)
		if calls < int32(cb.halfOpenMaxCalls) {
			atomic.AddInt32(&cb.halfOpenCalls, 1)
			return true
		}
		return false

	default:
		return false
	}
}

// OnSuccess records a successful call
func (cb *CircuitBreaker) OnSuccess() {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt32(&cb.failures, 0)

	case StateHalfOpen:
		// Record success in half-open state
		successes := atomic.AddInt32(&cb.halfOpenSuccesses, 1)
		// Safe conversion: successThreshold is validated to be > 0 and reasonable
		//nolint:gosec // Integer conversion is safe (validated config value)
		if successes >= int32(cb.successThreshold) {
			// Close circuit - backend is back up
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateClosed)) {
				atomic.StoreInt32(&cb.failures, 0)
				atomic.StoreInt32(&cb.halfOpenCalls, 0)
				atomic.StoreInt32(&cb.halfOpenSuccesses, 0)
				atomic.StoreInt64(&cb.lastFailureTime, 0)
			}
		}
	}
}

// OnFailure records a failed call
func (cb *CircuitBreaker) OnFailure() {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		// Increment failure count
		failures := atomic.AddInt32(&cb.failures, 1)
		atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

		// Open circuit if threshold reached
		// Safe conversion: maxFailures is validated to be > 0 and reasonable
		//nolint:gosec // Integer conversion is safe (validated config value)
		if failures >= int32(cb.maxFailures) {
			atomic.CompareAndSwapInt32(&cb.state, int32(StateClosed), int32(StateOpen))
		}

	case StateHalfOpen:
		// Failure in half-open state - open circuit again
		atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateOpen))
		atomic.StoreInt32(&cb.halfOpenCalls, 0)
		atomic.StoreInt32(&cb.halfOpenSuccesses, 0)
		atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())
	}
}

// CircuitBreakerManager manages circuit breakers per service
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

var globalCBManager = &CircuitBreakerManager{
	breakers: make(map[string]*CircuitBreaker),
}

// GetCircuitBreaker gets or creates circuit breaker for a service
func (m *CircuitBreakerManager) GetCircuitBreaker(
	serviceName string, config *CircuitBreakerConfig,
) *CircuitBreaker {
	if config == nil {
		return nil // No circuit breaker configured
	}

	m.mu.RLock()
	if cb, exists := m.breakers[serviceName]; exists {
		m.mu.RUnlock()
		return cb
	}
	m.mu.RUnlock()

	// Create new circuit breaker
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check
	if cb, exists := m.breakers[serviceName]; exists {
		return cb
	}

	cb := NewCircuitBreaker(serviceName, config)
	m.breakers[serviceName] = cb
	return cb
}
