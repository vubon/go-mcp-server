# MCP Server Package

A reusable Go package for building MCP (Model Context Protocol) servers.

## Features

- ✅ MCP protocol compliant
- ✅ **MCP Schema Version Support** (2024-11-05, 2025-03-26, 2025-06-18)
- ✅ **Structured Logging** - JSON format, Datadog compatible, async, zero latency
- ✅ JSON-RPC 2.0 support
- ✅ HTTP and Stdio transports
- ✅ Programmatic and file-based tool registration
- ✅ Auto-generated HTTP handlers from configuration
- ✅ Multiple authorization strategies (pass-through, transform, static, basic, none)
- ✅ Path and query parameter substitution
- ✅ Environment variable support
- ✅ Flexible timeout and header configuration

📚 **[View Full Documentation](./docs/README.md)**

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
    greetTool := mcpserver.Tool{
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
    }
    server.RegisterTool("greet", &greetTool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
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
    ProtocolVersion: mcpserver.SchemaVersion2025_06_18, // optional, defaults to latest (2025-06-18)
})
```

**MCP Schema Versions:**
- `mcpserver.SchemaVersion2024_11_05` - Initial/legacy version
- `mcpserver.SchemaVersion2025_03_26` - First major update
- `mcpserver.SchemaVersion2025_06_18` - Latest stable (default, uses JSON Schema 2020-12)

### Register Tools

**Programmatic Registration:**
```go
tool := mcpserver.Tool{
    Name:        "tool_name",
    Description: "Tool description",
    InputSchema: map[string]interface{}{
        // JSON Schema definition
    },
}
server.RegisterTool("tool_name", &tool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
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

Register tools from JSON or YAML files without writing Go code. This allows you to declaratively define tools and their HTTP handlers.

**Quick Example:**
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

📚 **[View Complete File-Based Registration Guide](./docs/file-based-registration.md)**

## Authorization

The MCP server supports multiple authorization strategies for handling authentication headers:

- **Pass-Through**: Forward client authorization as-is (default)
- **Transform**: Modify authorization header format
- **Static**: Use fixed authorization values
- **Basic**: Basic Authentication encoding
- **None**: Explicitly disable authorization

📚 **[View Complete Authorization Documentation](./docs/auth/README.md)**

## Examples

- **[HTTP Server Example](./examples/simple/)** - Complete HTTP server with file-based tool registration
- **[Stdio Server Example](./examples/stdio-server/)** - Stdio transport example

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

