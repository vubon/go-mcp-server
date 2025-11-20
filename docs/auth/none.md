# None Authorization Strategy

The **none** strategy explicitly disables authorization header forwarding. No authorization header will be sent to the backend service, regardless of what the client sends.

## Overview

The none strategy is used when you want to explicitly ensure that no authorization header is sent to the backend. This is useful for public endpoints, unauthenticated services, or when authorization is handled through other means.

## Configuration

### Basic Usage

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: none
```

## How It Works

1. **Client Request**: Client may or may not send an `Authorization` header
2. **Strategy Processing**: The none strategy explicitly ignores any incoming authorization
3. **Backend Request**: No authorization header is sent to the backend service

## Use Cases

- **Public Endpoints**: APIs that don't require authentication
- **Alternative Auth Methods**: When authentication is handled via query parameters, cookies, or other mechanisms
- **Testing/Development**: Disabling auth for local development
- **Explicit Override**: Overriding service-level authorization configuration for specific handlers

## Examples

### Example 1: Public Endpoint

```yaml
handlers:
  # Public endpoint - no auth required
  get_public_data:
    type: http
    method: GET
    path: /api/v1/public/data
    authorization:
      strategy: none
```

### Example 2: Override Service-Level Auth

```yaml
serviceConfig:
  api-service:
    baseURL: https://api.example.com
    authorization:
      strategy: pass-through  # Default for all handlers

handlers:
  # Most handlers inherit pass-through
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
  
  # But this public endpoint explicitly disables auth
  get_public_info:
    type: http
    method: GET
    path: /api/v1/public/info
    authorization:
      strategy: none  # Overrides service-level config
```

### Example 3: Multiple Strategies

```yaml
handlers:
  # Authenticated endpoint
  get_user:
    authorization:
      strategy: pass-through
  
  # Public endpoint
  get_status:
    authorization:
      strategy: none
  
  # Service-to-service endpoint
  internal_call:
    authorization:
      strategy: static
      staticValueEnv: INTERNAL_API_KEY
```

## Behavior

### When Client Sends Authorization Header

Even if the client sends an `Authorization` header, it will be ignored:

**Client Request:**
```
Authorization: Bearer token-123
```

**Backend Request:**
```
(No Authorization header)
```

### When Client Doesn't Send Authorization Header

If the client doesn't send an authorization header, the behavior is the same:

**Client Request:**
```
(No Authorization header)
```

**Backend Request:**
```
(No Authorization header)
```

## Configuration Resolution

The none strategy respects the configuration resolution order:

1. **Handler-level** authorization config (highest priority)
2. **Service-level** authorization config
3. **Default** pass-through behavior (if Authorization header is available)

**Important**: When `strategy: none` is specified at the handler level, it overrides any service-level configuration.

## Use Cases by Scenario

### Scenario 1: Public API Endpoints

```yaml
handlers:
  # Public health check endpoint
  health:
    type: http
    method: GET
    path: /health
    authorization:
      strategy: none
  
  # Public status endpoint
  status:
    type: http
    method: GET
    path: /status
    authorization:
      strategy: none
```

### Scenario 2: Mixed Authentication

```yaml
serviceConfig:
  api-service:
    baseURL: https://api.example.com
    authorization:
      strategy: pass-through  # Default: forward client auth

handlers:
  # Authenticated endpoints
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # Uses pass-through (inherited)
  
  # Public endpoints
  get_public_data:
    type: http
    method: GET
    path: /api/v1/public/data
    authorization:
      strategy: none  # Explicitly no auth
```

### Scenario 3: Development/Testing

```yaml
# Development configuration
handlers:
  test_endpoint:
    type: http
    method: POST
    path: /api/v1/test
    authorization:
      strategy: none  # Disable auth for local testing
```

## Security Considerations

1. **Explicit Intent**: Using `strategy: none` makes it clear that no authentication is required
2. **Backend Validation**: Ensure your backend service properly handles unauthenticated requests
3. **Endpoint Security**: Review endpoints using the none strategy to ensure they should truly be public
4. **Rate Limiting**: Consider implementing rate limiting for public endpoints

## Comparison with Other Strategies

| Strategy | Behavior | Use Case |
|----------|----------|----------|
| **none** | No auth header sent | Public endpoints, explicit override |
| **pass-through** | Forward client auth | Standard auth forwarding |
| **static** | Fixed auth value | Service-to-service auth |
| **transform** | Modify auth format | Format conversion |
| **basic** | Basic Auth encoding | Basic Auth required |

## Common Patterns

### Pattern 1: Public + Private Endpoints

```yaml
handlers:
  # Public endpoints
  public_data:
    authorization:
      strategy: none
  
  # Private endpoints
  private_data:
    authorization:
      strategy: pass-through
```

### Pattern 2: Service-Level Override

```yaml
serviceConfig:
  api-service:
    authorization:
      strategy: pass-through

handlers:
  # Override for specific public endpoint
  public_endpoint:
    authorization:
      strategy: none
```

## Related Strategies

- **[Pass-Through](./pass-through.md)**: Use when you want to forward client authorization
- **[Transform](./transform.md)**: Use when you need to modify authorization format
- **[Static](./static.md)**: Use when you need a fixed authorization value
- **[Basic](./basic.md)**: Use for Basic Authentication encoding

