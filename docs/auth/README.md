# Authorization Strategies

This directory contains comprehensive documentation for all authorization strategies supported by the MCP server.

## Overview

The MCP server supports multiple authorization strategies that allow you to control how authorization headers are handled when making requests to backend services. Each strategy serves different use cases and can be configured at both the handler and service levels.

## Available Strategies

### 1. [Pass-Through](./pass-through.md)
Forwards the incoming `Authorization` header from the client request directly to the backend service without modification. This is the default strategy.

**Use when:** You want to transparently forward client authorization to backend services.

### 2. [Transform](./transform.md)
Modifies the format of the incoming authorization header before sending it to the backend. Useful for converting between different authorization formats.

**Use when:** Your backend expects a different authorization format than what the client provides.

### 3. [Static](./static.md)
Uses a fixed authorization value from configuration or environment variables. The value doesn't depend on the incoming request.

**Use when:** You need service-to-service authentication with fixed API keys or tokens.

### 4. [Basic](./basic.md)
Implements HTTP Basic Authentication by encoding credentials as `Basic base64(username:password)`. Supports extracting credentials from request headers or using static values.

**Use when:** Your backend requires Basic Authentication, or you need to extract credentials from custom headers and encode them.

### 5. [None](./none.md)
Explicitly disables authorization header forwarding. No authorization header is sent to the backend.

**Use when:** You have public endpoints or want to explicitly disable authentication for specific handlers.

## Quick Reference

| Strategy | Description | Configuration Priority |
|----------|-------------|------------------------|
| `pass-through` | Forward client auth as-is | Default if no config |
| `transform` | Modify auth header format | Handler > Service |
| `static` | Use fixed auth value | Handler > Service |
| `basic` | Basic Auth encoding | Handler > Service |
| `none` | No auth header | Handler > Service |

## Configuration Resolution

Authorization configuration is resolved in the following order (highest to lowest priority):

1. **Handler-level** authorization config
2. **Service-level** authorization config
3. **Default** pass-through behavior (if Authorization header is available)

## Common Patterns

### Pattern 1: Service-Level Default with Handler Overrides

```yaml
serviceConfig:
  api-service:
    baseURL: https://api.example.com
    authorization:
      strategy: pass-through  # Default for all handlers

handlers:
  # Inherits service-level pass-through
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
  
  # Overrides with transform
  create_user:
    type: http
    method: POST
    path: /api/v1/users
    authorization:
      strategy: transform
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
  
  # Overrides with none (public endpoint)
  get_status:
    type: http
    method: GET
    path: /api/v1/status
    authorization:
      strategy: none
```

### Pattern 2: Different Strategies for Different Services

```yaml
serviceConfig:
  # Modern API - pass-through
  modern-api:
    baseURL: https://api.example.com
    authorization:
      strategy: pass-through
  
  # Legacy API - Basic Auth
  legacy-api:
    baseURL: https://legacy.example.com
    authorization:
      strategy: basic
      basicAuth:
        usernameEnv: LEGACY_USERNAME
        passwordEnv: LEGACY_PASSWORD
  
  # Internal API - static token
  internal-api:
    baseURL: https://internal.example.com
    authorization:
      strategy: static
      staticValueEnv: INTERNAL_API_KEY
```

### Pattern 3: Header Extraction for Basic Auth

```yaml
handlers:
  backend_call:
    type: http
    method: POST
    path: /api/v1/endpoint
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: SECREAT_KEY
        passwordHeader: PASSWORD
```

## Security Best Practices

1. **Use Environment Variables**: Never store sensitive tokens or credentials directly in configuration files
2. **HTTPS Required**: Always use HTTPS to protect authorization headers in transit
3. **Secret Management**: Use secret management systems (Kubernetes secrets, AWS Secrets Manager, etc.) to inject environment variables
4. **Token Rotation**: Implement processes for rotating credentials regularly
5. **Least Privilege**: Use credentials with minimal required permissions
6. **Validation**: Always validate authorization tokens on backend services

## How Authorization Works

1. **HTTP Transport** extracts the `Authorization` header from incoming requests
2. **Context Propagation** passes the header through the request context
3. **Handler Generation** applies authorization based on configuration
4. **Backend Request** includes the authorization header in the HTTP call

## Getting Started

1. Choose the appropriate strategy for your use case
2. Read the detailed documentation for that strategy
3. Configure the strategy in your `handlers.yaml` or `handlers.json` file
4. Test the configuration to ensure it works as expected

## Documentation Index

- [Pass-Through Strategy](./pass-through.md) - Forward client authorization as-is
- [Transform Strategy](./transform.md) - Modify authorization header format
- [Static Strategy](./static.md) - Use fixed authorization values
- [Basic Strategy](./basic.md) - Basic Authentication encoding
- [None Strategy](./none.md) - Explicitly disable authorization

## Examples

See the [examples](../examples/) directory for complete working examples of authorization strategies in action.

