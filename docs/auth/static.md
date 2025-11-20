# Static Authorization Strategy

The **static** strategy uses a fixed authorization value that doesn't depend on the incoming request. This is useful for service-to-service authentication where you need a consistent API key or token.

## Overview

The static strategy allows you to specify an authorization value directly in the configuration or via an environment variable. This value is sent to the backend service regardless of what the client sends in their request.

## Configuration

### Using Environment Variable (Recommended)

Store sensitive tokens in environment variables:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: static
      staticValueEnv: BACKEND_API_KEY
```

**Environment Variable:**
```bash
export BACKEND_API_KEY="Bearer api-key-12345"
```

### Using Direct Value

Specify the authorization value directly in the configuration:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: static
      staticValue: "Bearer static-token-456"
```

### Custom Header Name

Send the static value to a different header:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: static
      headerName: X-API-Key
      staticValueEnv: API_KEY
```

## Configuration Options

### `staticValue` (optional)
The authorization value to use directly in the configuration. **Not recommended for production** as it exposes secrets in config files.

### `staticValueEnv` (optional)
The name of an environment variable containing the authorization value. **Recommended for production**.

### `headerName` (optional)
The header name to use in the backend request. Defaults to `Authorization`.

## How It Works

1. **Configuration**: Static authorization value is specified in config or environment variable
2. **Resolution**: System reads the value from config or environment
3. **Backend Request**: The static value is sent to the backend service
4. **Client Request**: The incoming client authorization header is **ignored**

## Use Cases

- **Service-to-Service Auth**: When your MCP server needs to authenticate with backend services using a fixed API key
- **Internal APIs**: For internal service communication where client auth isn't needed
- **API Gateway Pattern**: When the MCP server acts as a gateway with its own backend credentials
- **Development/Testing**: Using fixed tokens for development environments

## Examples

### Example 1: Backend API Key

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: static
      staticValueEnv: BACKEND_API_KEY
```

**Environment Variable:**
```bash
export BACKEND_API_KEY="Bearer backend-secret-key-123"
```

**Backend Request:**
```
Authorization: Bearer backend-secret-key-123
```

### Example 2: Custom Header with Static Value

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: static
      headerName: X-API-Key
      staticValueEnv: API_KEY
```

**Environment Variable:**
```bash
export API_KEY="my-api-key-value"
```

**Backend Request:**
```
X-API-Key: my-api-key-value
```

### Example 3: Service-Level Static Auth

```yaml
serviceConfig:
  internal-api:
    baseURL: https://internal-api.example.com
    authorization:
      strategy: static
      staticValueEnv: INTERNAL_API_KEY

handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # Inherits service-level static auth
```

### Example 4: Development vs Production

**Development (direct value):**
```yaml
handlers:
  get_user:
    authorization:
      strategy: static
      staticValue: "Bearer dev-token-123"
```

**Production (environment variable):**
```yaml
handlers:
  get_user:
    authorization:
      strategy: static
      staticValueEnv: PROD_API_KEY
```

## Priority Resolution

When both `staticValue` and `staticValueEnv` are specified:
- `staticValue` takes precedence if it's not empty
- `staticValueEnv` is used as fallback

**Example:**
```yaml
authorization:
  strategy: static
  staticValue: "Bearer direct-value"
  staticValueEnv: API_KEY
```
Result: `Bearer direct-value` is used (direct value takes precedence)

## Environment Variable Substitution

The static strategy supports environment variable substitution in the value itself:

```yaml
authorization:
  strategy: static
  staticValue: "Bearer ${API_KEY}"
```

This uses Go's `os.ExpandEnv()` to substitute `${API_KEY}` with the actual environment variable value.

## Security Best Practices

1. **Never Commit Secrets**: Never put actual API keys or tokens in configuration files that are committed to version control
2. **Use Environment Variables**: Always use `staticValueEnv` in production
3. **Secret Management**: Use secret management systems (e.g., Kubernetes secrets, AWS Secrets Manager) to inject environment variables
4. **Rotation**: Implement a process for rotating static API keys
5. **Least Privilege**: Use API keys with minimal required permissions

## Common Patterns

### Pattern 1: Service Account Authentication

```yaml
serviceConfig:
  backend-service:
    baseURL: https://backend.example.com
    authorization:
      strategy: static
      staticValueEnv: SERVICE_ACCOUNT_TOKEN

handlers:
  # All handlers use the same service account token
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
  
  create_user:
    type: http
    method: POST
    path: /api/v1/users
```

### Pattern 2: Per-Handler Static Auth

```yaml
handlers:
  # Different static tokens for different handlers
  public_endpoint:
    authorization:
      strategy: static
      staticValueEnv: PUBLIC_API_KEY
  
  admin_endpoint:
    authorization:
      strategy: static
      staticValueEnv: ADMIN_API_KEY
```

## Behavior When Environment Variable is Missing

If `staticValueEnv` is specified but the environment variable doesn't exist:
- An empty string is returned
- No authorization header is sent to the backend
- This may cause authentication failures on the backend

**Best Practice**: Always validate that required environment variables are set at startup.

## Related Strategies

- **[Pass-Through](./pass-through.md)**: Use when you want to forward client authorization
- **[Transform](./transform.md)**: Use when you need to modify client authorization
- **[Basic](./basic.md)**: Use for Basic Authentication encoding
- **[None](./none.md)**: Use to explicitly disable authorization

