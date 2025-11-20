# File-Based Tool Registration

You can register tools from JSON or YAML files without writing Go code. This allows you to declaratively define tools and their HTTP handlers.

## Quick Example

```go
server := mcpserver.New(&mcpserver.Config{
    Name:    "my-server",
    Version: "1.0.0",
})

// Register tools from files (auto-detects JSON/YAML format)
err := server.RegisterToolsFromFiles("tools.json", "handlers.json")
if err != nil {
    log.Fatal(err)
}
```

## Tools Configuration File (`tools.json` or `tools.yaml`)

The tools file defines your MCP tools as an array. Each tool must have:

**Required Fields:**
- `Name` (string): Unique tool identifier
- `Description` (string): Human-readable description for AI agents
- `InputSchema` (object): JSON Schema defining the tool's parameters

**Optional Fields:**
- `ServiceName` (string): Name of the service this tool calls (must match a service in handlers file)
- `APIVersion` (string): API version for the service
- `Endpoint` (string): Default endpoint path (can be overridden in handlers file)

**Example: `tools.json`**
```json
[
  {
    "Name": "greet",
    "Description": "Greets a person by name",
    "ServiceName": "greeting-service",
    "APIVersion": "v1",
    "Endpoint": "/api/v1/greet",
    "InputSchema": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "description": "Name of the person to greet"
        }
      },
      "required": ["name"]
    }
  }
]
```

**Example: `tools.yaml`**
```yaml
- Name: greet
  Description: Greets a person by name
  ServiceName: greeting-service
  APIVersion: v1
  Endpoint: /api/v1/greet
  InputSchema:
    type: object
    properties:
      name:
        type: string
        description: Name of the person to greet
    required:
      - name
```

## Handlers Configuration File (`handlers.json` or `handlers.yaml`)

The handlers file defines how to call your services. It has two main sections:

### 1. `serviceConfig` (Service-Level Configuration)

Defines base configuration for each service. All tools using the same service share these settings.

**Fields:**
- `baseURL` (string, required): Base URL of the service (e.g., `https://api.example.com`)
- `timeout` (string, optional): Default timeout for all endpoints (e.g., `"30s"`, `"1m"`)
- `headers` (object, optional): Default headers applied to all requests for this service
- `authorization` (object, optional): Authorization configuration (see [Authorization Strategies](./auth/README.md))

### 2. `handlers` (Tool-Specific Configuration)

Defines HTTP call configuration for each tool. Tool name must match the `Name` in tools file.

**Fields:**
- `type` (string, required): Handler type (currently only `"http"` is supported)
- `method` (string, required): HTTP method (`GET`, `POST`, `PUT`, `DELETE`, etc.)
- `path` (string, optional): Endpoint path (overrides `Endpoint` from tools file if provided)
- `queryParams` (object, optional): Query parameters (supports dynamic substitution with `{paramName}`)
- `headers` (object, optional): Tool-specific headers (merged with service headers, tool headers override)
- `timeout` (string, optional): Tool-specific timeout (overrides service timeout)
- `authorization` (object, optional): Tool-specific authorization configuration (overrides service config)

**Example: `handlers.json`**
```json
{
  "serviceConfig": {
    "greeting-service": {
      "baseURL": "https://api.example.com",
      "timeout": "30s",
      "headers": {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "X-API-Key": "${API_KEY}"
      }
    }
  },
  "handlers": {
    "greet": {
      "type": "http",
      "method": "POST",
      "path": "/api/v1/greet",
      "headers": {
        "X-Request-ID": "${REQUEST_ID}",
        "X-Client-Version": "1.0.0"
      },
      "timeout": "5s"
    }
  }
}
```

**Example: `handlers.yaml`**
```yaml
serviceConfig:
  greeting-service:
    baseURL: https://api.example.com
    timeout: 30s
    headers:
      Content-Type: application/json
      Accept: application/json
      X-API-Key: ${API_KEY}

handlers:
  greet:
    type: http
    method: POST
    path: /api/v1/greet
    headers:
      X-Request-ID: ${REQUEST_ID}
      X-Client-Version: "1.0.0"
    timeout: 5s
```

## How It Works

1. **Service Configuration**: All tools using `greeting-service` get the base URL, default timeout, and default headers
2. **Handler Configuration**: Each tool can override/add headers and timeout
3. **Header Merging**: Final headers = service headers + handler headers (handler overrides service)
4. **Timeout Resolution**: Handler timeout > Service timeout > Default (30s)
5. **Environment Variables**: Use `${VAR_NAME}` syntax in headers (e.g., `${API_KEY}`)
6. **Path Parameters**: Use `{paramName}` in paths for dynamic substitution
7. **Query Parameters**: Use `{paramName}` in query params for dynamic substitution

## Complete Example

**`tools.json`:**
```json
[
  {
    "Name": "get_user",
    "Description": "Retrieves user information by ID",
    "ServiceName": "user-service",
    "APIVersion": "v1",
    "Endpoint": "/api/v1/users",
    "InputSchema": {
      "type": "object",
      "properties": {
        "userId": {
          "type": "string",
          "description": "Unique user identifier"
        }
      },
      "required": ["userId"]
    }
  },
  {
    "Name": "create_user",
    "Description": "Creates a new user",
    "ServiceName": "user-service",
    "APIVersion": "v1",
    "Endpoint": "/api/v1/users",
    "InputSchema": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "email": {"type": "string", "format": "email"}
      },
      "required": ["name", "email"]
    }
  }
]
```

**`handlers.json`:**
```json
{
  "serviceConfig": {
    "user-service": {
      "baseURL": "https://api.example.com",
      "timeout": "30s",
      "headers": {
        "Content-Type": "application/json"
      }
    }
  },
  "handlers": {
    "get_user": {
      "type": "http",
      "method": "GET",
      "path": "/api/v1/users/{userId}",
      "timeout": "10s"
    },
    "create_user": {
      "type": "http",
      "method": "POST",
      "path": "/api/v1/users",
      "headers": {
        "X-Request-ID": "${REQUEST_ID}"
      },
      "timeout": "15s"
    }
  }
}
```

## Path and Query Parameter Substitution

### Path Parameters

Use `{paramName}` in the path to substitute values from tool arguments:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
```

When called with `{"userId": "123"}`, the path becomes `/api/v1/users/123`.

### Query Parameters

Use `{paramName}` in query parameters for dynamic substitution:

```yaml
handlers:
  search_users:
    type: http
    method: GET
    path: /api/v1/users
    queryParams:
      q: "{searchQuery}"
      page: "{pageNumber}"
```

When called with `{"searchQuery": "john", "pageNumber": 1}`, the URL becomes `/api/v1/users?q=john&page=1`.

Parameters used in path or query are automatically removed from the request body.

## Best Practices

1. **Service Organization**: Group related tools under the same service name
2. **Shared Headers**: Put common headers (API keys, auth tokens) in `serviceConfig`
3. **Tool-Specific Headers**: Use handler-level headers for endpoint-specific needs
4. **Environment Variables**: Use `${VAR}` for sensitive values (API keys, tokens)
5. **Timeout Strategy**: Set service-level default, override per tool if needed
6. **Path Override**: Use handler `path` to override tool `Endpoint` when needed
7. **Authorization**: Configure authorization at service level, override per handler if needed (see [Authorization Strategies](./auth/README.md))

## Error Handling

The registration will fail with clear error messages if:
- Tool name is empty
- Handler configuration not found for a tool
- Service configuration not found for a tool's `ServiceName`
- Required fields are missing
- Invalid JSON/YAML syntax

## Related Documentation

- [Authorization Strategies](./auth/README.md) - Configure authentication and authorization
- [API Reference](../README.md#api-reference) - Programmatic API usage

