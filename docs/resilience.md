# Retry and Circuit Breaker

This guide explains how to configure retry logic and circuit breaker patterns for backend HTTP calls in the MCP server.

## Overview

The MCP server supports two resilience patterns to handle backend service failures:

1. **Retry Logic**: Automatically retries failed requests with exponential backoff and jitter
2. **Circuit Breaker**: Prevents cascading failures by stopping requests to failing backends

## Retry Configuration

Retry configuration can be set at the **service level** (applies to all handlers) or **handler level** (overrides service-level for specific handlers).

### Resolution Order

Retry configuration follows the same pattern as `timeout`, `headers`, and `authorization`:
- **Handler-level retry** (if specified) → Use handler config
- **Service-level retry** (if specified) → Use service config
- **Default** (no retry) → No retry logic

### Configuration Options

```yaml
retry:
  maxAttempts: 3              # Maximum retry attempts (including initial attempt)
  initialDelay: "100ms"       # Initial delay before first retry
  maxDelay: "5s"              # Maximum delay between retries
  multiplier: 2.0             # Exponential backoff multiplier
  jitter: true                # Add random variation to prevent thundering herd
  retryableStatusCodes:       # HTTP status codes that trigger retry
    - 500
    - 502
    - 503
    - 504
  retryableErrors:            # Error types that trigger retry
    - "timeout"
    - "connection_refused"
    - "temporary"
    - "network"
```

### Default Values

- `maxAttempts`: 1 (no retries)
- `initialDelay`: 100ms
- `maxDelay`: 5s
- `multiplier`: 2.0
- `jitter`: true
- `retryableStatusCodes`: [500, 502, 503, 504]
- `retryableErrors`: ["timeout", "connection_refused", "temporary"]

### Example: Service-Level Retry

```yaml
serviceConfig:
  user-service:
    baseURL: "https://api.example.com"
    retry:
      maxAttempts: 3
      initialDelay: "100ms"
      maxDelay: "5s"
      multiplier: 2.0
      jitter: true
      retryableStatusCodes: [500, 502, 503, 504]

handlers:
  getUser:
    type: http
    method: GET
    path: "/users/{userId}"
    # Uses service-level retry config
```

### Example: Handler-Level Override

```yaml
serviceConfig:
  user-service:
    baseURL: "https://api.example.com"
    retry:
      maxAttempts: 3
      initialDelay: "100ms"
      # Default retry for all handlers

handlers:
  getUser:
    type: http
    method: GET
    path: "/users/{userId}"
    # Uses service-level retry (default)

  createUser:
    type: http
    method: POST
    path: "/users"
    retry:
      maxAttempts: 5
      initialDelay: "200ms"
      # Overrides service-level retry for this handler only
```

## Circuit Breaker Configuration

Circuit breaker configuration is set at the **service level** only (applies to all handlers for that service).

### Configuration Options

```yaml
circuitBreaker:
  maxFailures: 5        # Maximum consecutive failures before opening circuit
  timeout: "60s"        # Duration to keep circuit open before attempting half-open
  halfOpenMaxCalls: 3   # Maximum calls allowed in half-open state
  successThreshold: 2   # Successful calls needed to close circuit from half-open
```

### Default Values

- `maxFailures`: 5
- `timeout`: 60s
- `halfOpenMaxCalls`: 3
- `successThreshold`: 2

### Circuit Breaker States

1. **CLOSED** (Normal): Requests go through normally
2. **OPEN** (Backend Down): Requests are rejected immediately (no HTTP calls)
3. **HALF-OPEN** (Testing): Allows limited test requests to check if backend recovered

### Example: Circuit Breaker

```yaml
serviceConfig:
  user-service:
    baseURL: "https://api.example.com"
    circuitBreaker:
      maxFailures: 5
      timeout: "60s"
      halfOpenMaxCalls: 3
      successThreshold: 2

handlers:
  getUser:
    type: http
    method: GET
    path: "/users/{userId}"
    # Uses service-level circuit breaker
```

## Combined Configuration

You can use retry and circuit breaker together:

```yaml
serviceConfig:
  user-service:
    baseURL: "https://api.example.com"
    retry:
      maxAttempts: 3
      initialDelay: "100ms"
      maxDelay: "5s"
      retryableStatusCodes: [500, 502, 503, 504]
    circuitBreaker:
      maxFailures: 5
      timeout: "60s"

handlers:
  getUser:
    type: http
    method: GET
    path: "/users/{userId}"
    # Uses both service-level retry and circuit breaker
```

### How They Work Together

1. **Circuit breaker check**: Before making HTTP request, check if circuit is open
2. **HTTP request**: Make request with retry logic
3. **Record result**: Record success/failure for circuit breaker state

**Flow:**
```
Request → Circuit Breaker Check
  ├─ Circuit OPEN → Return error immediately (no retry)
  └─ Circuit CLOSED → Make HTTP request
         ├─ Success → Record success, return result
         └─ Failure → Retry (if retryable)
                ├─ Retry succeeds → Record success
                └─ All retries fail → Record failure
```

## Best Practices

### Retry Configuration

1. **Conservative retry settings**: Use maxAttempts of 3-5 for most cases
2. **Appropriate delays**: Start with 100ms initialDelay, cap at 5s maxDelay
3. **Enable jitter**: Always use jitter to prevent thundering herd
4. **Retryable errors**: Only retry on transient errors (timeouts, 5xx status codes)

### Circuit Breaker Configuration

1. **Reasonable failure threshold**: Use 5-10 failures before opening circuit
2. **Appropriate timeout**: Use 60s timeout for most services
3. **Test recovery**: Use 2-3 test calls in half-open state
4. **Per-service isolation**: Each service has its own circuit breaker

### When to Use

**Use Retry When:**
- Backend has transient failures
- Network issues are common
- You want automatic recovery

**Use Circuit Breaker When:**
- Backend can go down completely
- You have multiple requests per service
- You want fast failure response
- You want automatic recovery

**Use Both When:**
- You want maximum resilience
- Backend has both transient and persistent failures
- You want to prevent cascading failures

## Error Handling

### Retry Errors

**Retryable errors:**
- Network errors (connection refused, timeout)
- 5xx status codes (server errors)
- Temporary errors

**Non-retryable errors:**
- 4xx status codes (client errors)
- Permanent errors
- Context cancellation

### Circuit Breaker Errors

**Circuit opens when:**
- Consecutive failures >= maxFailures

**Circuit closes when:**
- Half-open state: successThreshold successful calls

**Circuit rejects requests when:**
- Circuit is open (not in half-open state)

## Troubleshooting

### Retry Not Working

1. **Check configuration**: Verify retry config is set at service or handler level
2. **Check retryable errors**: Verify error type is in retryableErrors list
3. **Check status codes**: Verify status code is in retryableStatusCodes list
4. **Check maxAttempts**: Ensure maxAttempts > 1

### Circuit Breaker Not Opening

1. **Check configuration**: Verify circuitBreaker config is set at service level
2. **Check failure threshold**: Verify maxFailures is reached
3. **Check failures**: Verify failures are consecutive (not spread out)

### Circuit Breaker Not Closing

1. **Check timeout**: Verify timeout has passed
2. **Check test calls**: Verify test calls are succeeding
3. **Check success threshold**: Verify successThreshold is reached

## Examples

See the [examples directory](../examples/) for complete working examples with retry and circuit breaker configuration.

