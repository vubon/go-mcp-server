# MCP Server Package

A reusable Go package for building MCP (Model Context Protocol) servers.

## Features

- ✅ MCP protocol compliant
- ✅ JSON-RPC 2.0 support
- ✅ HTTP transport
- ✅ Stdio transport (for local MCP clients)
- ✅ Transport interface (extensible)
- ✅ Easy tool registration
- ✅ Type-safe handlers
- ✅ Zero dependencies (except standard library)

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

## Examples

### HTTP Server Example

See `examples/simple/main.go` for a complete HTTP server example.

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

