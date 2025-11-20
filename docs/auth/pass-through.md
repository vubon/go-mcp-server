# Pass-Through Authorization Strategy

The **pass-through** strategy is the default authorization behavior. It forwards the incoming `Authorization` header from the client request directly to the backend service without any modification.

## Overview

When using the pass-through strategy, the MCP server acts as a transparent proxy for authorization headers. The exact header value received from the client is sent to the backend service.

## Configuration

### Basic Usage

The pass-through strategy is the default, so you don't need to explicitly configure it:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # No authorization config = pass-through by default
```

### Explicit Configuration

You can also explicitly specify the pass-through strategy:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: pass-through
```

### Custom Header Name

By default, the `Authorization` header is used. You can specify a different header name:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: pass-through
      headerName: X-Custom-Auth
```

## How It Works

1. **Client Request**: Client sends a request with an `Authorization` header (e.g., `Bearer token-123`)
2. **Extraction**: HTTP transport extracts the header value
3. **Context Propagation**: Header value is stored in request context
4. **Backend Request**: The exact same header value is sent to the backend service

## Use Cases

- **OAuth2/JWT Tokens**: When you want to pass Bearer tokens directly to backend services
- **API Keys**: When clients provide API keys that should be forwarded as-is
- **Custom Auth Schemes**: Any custom authorization header format that doesn't need transformation

## Examples

### Example 1: Bearer Token Pass-Through

**Client Request:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Backend Request:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Configuration:**
```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: pass-through
```

### Example 2: Service-Level Configuration

```yaml
serviceConfig:
  user-service:
    baseURL: https://api.example.com
    authorization:
      strategy: pass-through
      headerName: Authorization

handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # Inherits service-level pass-through config
```

## Behavior When No Authorization Header is Present

If the client request doesn't include an `Authorization` header:
- The pass-through strategy returns an empty string
- No authorization header is sent to the backend
- This is the expected behavior for unauthenticated requests

## Configuration Resolution

The pass-through strategy respects the configuration resolution order:

1. **Handler-level** authorization config (highest priority)
2. **Service-level** authorization config
3. **Default** pass-through behavior (if Authorization header is available)

## Security Considerations

- **No Validation**: The pass-through strategy does not validate or verify the authorization header. It simply forwards it.
- **Header Exposure**: Ensure your transport layer (HTTPS) is properly configured to prevent header interception
- **Backend Validation**: Always validate authorization tokens on the backend service

## Related Strategies

- **[Transform](./transform.md)**: Use when you need to change the header format
- **[Static](./static.md)**: Use when you need a fixed authorization value
- **[Basic](./basic.md)**: Use for Basic Authentication encoding
- **[None](./none.md)**: Use to explicitly disable authorization

