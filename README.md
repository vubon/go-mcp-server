# MCP Server Package

A reusable Go package for building MCP (Model Context Protocol) servers.

## Features

- ✅ MCP protocol compliant
- ✅ JSON-RPC 2.0 support
- ✅ HTTP transport
- ✅ Stdio transport (for local MCP clients)
- ✅ Transport interface (extensible)
- ✅ Easy tool registration (programmatic or file-based)
- ✅ File-based tool registration from JSON/YAML
- ✅ Auto-generated HTTP handlers from configuration
- ✅ Type-safe handlers
- ✅ Environment variable substitution in headers
- ✅ Flexible timeout and header configuration

## Quick Start

### 1. Create a Server

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/yourusername/go-mcp-server"
    "github.com/yourusername/go-mcp-server/transport"
)

func main() {
    // Create server
    server := mcpserver.New(&mcpserver.Config{
        Name:    "greeter",
        Version: "1.0.0",
    })
    
    // Register a tool
    server.RegisterTool("greet", mcpserver.Tool{
        Name:        "greet",
        Description: "Greets a person",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "name": map[string]interface{}{
                    "type":        "string",
                    "description": "Name of the person to greet",
                },
            },
            "required": []string{"name"},
        },
    }, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        name := args["name"].(string)
        return map[string]string{
            "greeting": "Hello " + name,
        }, nil
    })
    
    // Create HTTP transport
    httpHandler := transport.NewHTTP(server)
    
    // Start server
    http.Handle("/jsonrpc", httpHandler)
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## API Reference

### Server

```go
// New creates a new MCP server
server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: "2024-11-05", // optional, defaults to "2024-11-05"
})
```

### Register Tools

**Programmatic Registration:**
```go
server.RegisterTool("tool_name", mcpserver.Tool{
    Name:        "tool_name",
    Description: "Tool description",
    InputSchema: map[string]interface{}{
        // JSON Schema definition
    },
}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    // Tool implementation
    return result, nil
})
```

**File-Based Registration:**
```go
// Register from files (auto-detects JSON/YAML)
err := server.RegisterToolsFromFiles("tools.json", "handlers.json")

// Or register from byte data
toolsData, _ := os.ReadFile("tools.json")
handlersData, _ := os.ReadFile("handlers.json")
err := server.RegisterToolsFromJSON(toolsData, handlersData)
// or
err := server.RegisterToolsFromYAML(toolsData, handlersData)
```

### HTTP Transport

```go
httpHandler := transport.NewHTTP(server)
http.Handle("/jsonrpc", httpHandler)
http.ListenAndServe(":8080", nil)
```

### Stdio Transport

```go
stdioTransport := transport.NewStdio(server)
// Run blocks until context is cancelled
stdioTransport.Run(context.Background())
```

## File-Based Tool Registration

You can register tools from JSON or YAML files without writing Go code. This allows you to declaratively define tools and their HTTP handlers.

### Quick Example

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

### Tools Configuration File (`tools.json` or `tools.yaml`)

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

### Handlers Configuration File (`handlers.json` or `handlers.yaml`)

The handlers file defines how to call your services. It has two main sections:

#### 1. `serviceConfig` (Service-Level Configuration)

Defines base configuration for each service. All tools using the same service share these settings.

**Fields:**
- `baseURL` (string, required): Base URL of the service (e.g., `https://api.example.com`)
- `timeout` (string, optional): Default timeout for all endpoints (e.g., `"30s"`, `"1m"`)
- `headers` (object, optional): Default headers applied to all requests for this service

#### 2. `handlers` (Tool-Specific Configuration)

Defines HTTP call configuration for each tool. Tool name must match the `Name` in tools file.

**Fields:**
- `type` (string, required): Handler type (currently only `"http"` is supported)
- `method` (string, required): HTTP method (`GET`, `POST`, `PUT`, `DELETE`, etc.)
- `path` (string, optional): Endpoint path (overrides `Endpoint` from tools file if provided)
- `headers` (object, optional): Tool-specific headers (merged with service headers, tool headers override)
- `timeout` (string, optional): Tool-specific timeout (overrides service timeout)

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

### How It Works

1. **Service Configuration**: All tools using `greeting-service` get the base URL, default timeout, and default headers
2. **Handler Configuration**: Each tool can override/add headers and timeout
3. **Header Merging**: Final headers = service headers + handler headers (handler overrides service)
4. **Timeout Resolution**: Handler timeout > Service timeout > Default (30s)
5. **Environment Variables**: Use `${VAR_NAME}` syntax in headers (e.g., `${API_KEY}`)

### Complete Example

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
        "Content-Type": "application/json",
        "Authorization": "Bearer ${AUTH_TOKEN}"
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

### Best Practices

1. **Service Organization**: Group related tools under the same service name
2. **Shared Headers**: Put common headers (API keys, auth tokens) in `serviceConfig`
3. **Tool-Specific Headers**: Use handler-level headers for endpoint-specific needs
4. **Environment Variables**: Use `${VAR}` for sensitive values (API keys, tokens)
5. **Timeout Strategy**: Set service-level default, override per tool if needed
6. **Path Override**: Use handler `path` to override tool `Endpoint` when needed

### Authorization Header Pass-Through

The MCP server can automatically pass Authorization headers from incoming HTTP requests to backend services. This enables seamless authentication propagation without writing custom code.

#### How It Works

1. **HTTP Transport** extracts the `Authorization` header from incoming requests
2. **Context Propagation** passes the header through the request context
3. **Handler Generation** applies authorization based on configuration
4. **Backend Request** includes the authorization header in the HTTP call

#### Authorization Strategies

**1. Pass-Through (Default)**
Passes the incoming Authorization header as-is to the backend service.

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # No authorization config = pass-through by default
```

Or explicitly:
```yaml
handlers:
  get_user:
    authorization:
      strategy: pass-through
```

**2. Transform**
Transforms the authorization header format (e.g., Bearer → ApiKey).

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: transform
      headerName: X-API-Key
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
```

**3. Static**
Uses a static authorization value from configuration or environment variable.

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: static
      staticValueEnv: BACKEND_API_KEY
      # or
      # staticValue: "Bearer static-token"
```

**4. None**
Explicitly does not add an Authorization header.

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: none
```

#### Service-Level Configuration

You can set authorization defaults at the service level:

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
    # Inherits service-level authorization config
```

#### Configuration Resolution

Authorization configuration is resolved in this order:
1. **Handler-level** authorization config (highest priority)
2. **Service-level** authorization config
3. **Default** pass-through behavior (if Authorization header is available)

#### Security Best Practices

1. **Never Log Authorization Headers**: The package automatically excludes Authorization headers from logs
2. **Use Environment Variables**: Store sensitive tokens in environment variables, not in config files
3. **Transform When Needed**: Use transform strategy to convert between different auth formats
4. **Validate Backend Responses**: Always validate responses from backend services

#### Example: Complete Authorization Setup

```yaml
serviceConfig:
  user-service:
    baseURL: https://api.example.com
    authorization:
      strategy: pass-through
      headerName: Authorization

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
      headerName: X-API-Key
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
  
  # Uses static token
  admin_action:
    type: http
    method: POST
    path: /api/v1/admin/action
    authorization:
      strategy: static
      staticValueEnv: ADMIN_API_KEY
```

### Error Handling

The registration will fail with clear error messages if:
- Tool name is empty
- Handler configuration not found for a tool
- Service configuration not found for a tool's `ServiceName`
- Required fields are missing
- Invalid JSON/YAML syntax

## Examples

### HTTP Server Example

See `examples/simple/main.go` for a complete HTTP server example with file-based tool registration.

### Stdio Server Example

See `examples/stdio-server/main.go` for a complete stdio server example.

```go
package main

import (
    "context"
    "log"
    
    "github.com/yourusername/go-mcp-server"
    "github.com/yourusername/go-mcp-server/transport"
)

func main() {
    server := mcpserver.New(&mcpserver.Config{
        Name:    "my-server",
        Version: "1.0.0",
    })
    
    // Register tools...
    
    stdioTransport := transport.NewStdio(server)
    if err := stdioTransport.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## License

MIT

