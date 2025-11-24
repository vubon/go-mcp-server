package mcpserver

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// executeWithRetry executes HTTP request with retry logic
//
//nolint:gocyclo // Retry logic requires multiple conditional branches
func (cfg *httpHandlerConfig) executeWithRetry(
	ctx context.Context, req *http.Request,
) (*http.Response, error) {
	if cfg.retryConfig == nil || cfg.retryConfig.MaxAttempts <= 1 {
		// No retry configured, execute once
		return cfg.client.Do(req)
	}

	// Parse retry configuration
	initialDelay := 100 * time.Millisecond
	if cfg.retryConfig.InitialDelay != "" {
		if d, err := time.ParseDuration(cfg.retryConfig.InitialDelay); err == nil {
			initialDelay = d
		}
	}

	maxDelay := 5 * time.Second
	if cfg.retryConfig.MaxDelay != "" {
		if d, err := time.ParseDuration(cfg.retryConfig.MaxDelay); err == nil {
			maxDelay = d
		}
	}

	multiplier := 2.0
	if cfg.retryConfig.Multiplier > 0 {
		multiplier = cfg.retryConfig.Multiplier
	}

	maxAttempts := cfg.retryConfig.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	delay := initialDelay

	// Retry loop
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Check circuit breaker before making request
		if cfg.circuitBreaker != nil {
			if !cfg.circuitBreaker.Allow() {
				return nil, fmt.Errorf("circuit breaker is open")
			}
		}

		// Execute request
		resp, err := cfg.client.Do(req)
		lastErr = err

		// If no error and status code is not retryable, success
		if err == nil {
			if !cfg.isRetryableStatusCode(resp.StatusCode) {
				// Success - return response
				return resp, nil
			}
			// Retryable status code - close response body and retry
			if resp != nil {
				resp.Body.Close()
			}
			lastErr = fmt.Errorf("retryable status code: %d", resp.StatusCode)
		} else if !cfg.isRetryableError(err) {
			// Non-retryable error - return immediately
			return nil, err
		}

		// If this is the last attempt, return the error
		if attempt == maxAttempts-1 {
			return nil, lastErr
		}

		// Calculate next delay with exponential backoff
		if cfg.retryConfig.Jitter {
			// Add jitter: random value between 0.5x and 1.5x of delay
			// Note: Using math/rand is acceptable for jitter (not security-sensitive)
			//nolint:gosec // Jitter doesn't require cryptographic randomness
			jitter := time.Duration(float64(delay) * (0.5 + rand.Float64()))
			delay = jitter
		} else {
			delay = time.Duration(float64(delay) * multiplier)
		}

		// Cap delay at maxDelay
		if delay > maxDelay {
			delay = maxDelay
		}

		// Wait before retry (respect context cancellation)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, lastErr
}

// isRetryableError checks if error is retryable
func (cfg *httpHandlerConfig) isRetryableError(err error) bool {
	if cfg.retryConfig == nil {
		return false
	}

	// Default retryable errors if not specified
	retryableErrors := cfg.retryConfig.RetryableErrors
	if len(retryableErrors) == 0 {
		retryableErrors = []string{"timeout", "connection_refused", "temporary"}
	}

	errStr := strings.ToLower(err.Error())
	for _, retryableErr := range retryableErrors {
		switch strings.ToLower(retryableErr) {
		case "timeout":
			if strings.Contains(errStr, "timeout") {
				return true
			}
		case "connection_refused":
			if strings.Contains(errStr, "connection refused") ||
				strings.Contains(errStr, "connectionrefused") {
				return true
			}
		case "temporary":
			// Note: net.Error.Temporary() is deprecated, check for timeout-like errors
			if netErr, ok := err.(net.Error); ok {
				// Most "temporary" errors are timeouts
				if netErr.Timeout() {
					return true
				}
			}
		case "network":
			if _, ok := err.(net.Error); ok {
				return true
			}
		}
	}
	return false
}

// isRetryableStatusCode checks if status code is retryable
func (cfg *httpHandlerConfig) isRetryableStatusCode(code int) bool {
	if cfg.retryConfig == nil {
		return false
	}

	// Default retryable status codes if not specified
	retryableCodes := cfg.retryConfig.RetryableStatusCodes
	if len(retryableCodes) == 0 {
		retryableCodes = []int{500, 502, 503, 504}
	}

	for _, retryableCode := range retryableCodes {
		if code == retryableCode {
			return true
		}
	}
	return false
}

// resolveRetryConfig resolves retry configuration: handler > service > default (no retry)
func resolveRetryConfig(handlerConfig *HandlerConfig, serviceConfig ServiceConfig) *RetryConfig {
	if handlerConfig != nil && handlerConfig.Retry != nil {
		return handlerConfig.Retry // Handler-level override
	}
	if serviceConfig.Retry != nil {
		return serviceConfig.Retry // Service-level default
	}
	return nil // No retry
}
