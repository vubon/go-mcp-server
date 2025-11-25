# MCP Server Package

A reusable Go package for building MCP (Model Context Protocol) servers.

## Features

- ✅ **MCP Protocol Compliant** - Full support for MCP schema versions (2024-11-05, 2025-03-26, 2025-06-18)
- ✅ **Structured Logging** - JSON format, Datadog compatible, async, zero latency
- ✅ **File-Based & Programmatic Configuration** - Declarative tool registration via JSON/YAML or programmatic API
- ✅ **Multiple Transports** - HTTP and Stdio support
- ✅ **Authorization Strategies** - Flexible auth handling (pass-through, transform, static, basic, none)

📚 **[View Full Documentation](./docs/README.md)**

## Quick Start

### 1. Create a Server with Structured Logging

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/vubon/go-mcp-server"
    "github.com/vubon/go-mcp-server/transport"
)

func main() {
    // Create structured logger
    logger := mcpserver.NewLogger(&mcpserver.LoggerConfig{
        Level:  "info",
        Format: "json",
    })

    // Create server with logger
    server := mcpserver.New(&mcpserver.Config{
        Name:    "greeter",
        Version: "1.0.0",
        Logger:  logger,
    })
    
    // Register tools from JSON/YAML files (recommended)
    err := server.RegisterToolsFromFiles("tools.json", "handlers.json")
    if err != nil {
        log.Fatalf("Failed to register tools: %v", err)
    }
    
    // Create HTTP transport and start server
    httpHandler := transport.NewHTTP(server)
    http.Handle("/jsonrpc", httpHandler)
    
    if logger := server.GetLogger(); logger != nil {
        logger.Info("Server starting",
            mcpserver.F("port", "8080"),
        )
    }
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## API Reference

### Server

```go
// Create structured logger (optional, but recommended)
logger := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    Level:  "info",
    Format: "json",
})

// New creates a new MCP server
server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: mcpserver.SchemaVersion2025_06_18, // optional, defaults to latest (2025-06-18)
    Logger:          logger, // optional, but recommended for production
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
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/vubon/go-mcp-server"
    "github.com/vubon/go-mcp-server/transport"
)

func main() {
    // Create structured logger
    logger := mcpserver.NewLogger(&mcpserver.LoggerConfig{
        Level:  "info",
        Format: "json",
    })

    // Create server with logger
    server := mcpserver.New(&mcpserver.Config{
        Name:    "my-server",
        Version: "1.0.0",
        Logger:  logger,
    })
    
    // Register tools programmatically or from files
    greetTool := mcpserver.Tool{
        Name:        "greet",
        Description: "Greets a person by name",
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
        name, ok := args["name"].(string)
        if !ok {
            return nil, fmt.Errorf("invalid parameter: name must be a string")
        }
        
        // Handlers can make HTTP API calls, access databases, call external services, etc.
        // The ctx parameter can be used for cancellation, timeouts, and request tracing
        
        return map[string]string{
            "greeting": "Hello, " + name + "!",
        }, nil
    })
    
    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        if logger := server.GetLogger(); logger != nil {
            logger.Info("Shutting down...")
        }
        cancel()
    }()
    
    // Create stdio transport
    stdioTransport := transport.NewStdio(server)
    if logger := server.GetLogger(); logger != nil {
        logger.Info("MCP Server (stdio) starting",
            mcpserver.F("transport", "stdio"),
        )
    }
    
    if err := stdioTransport.Run(ctx); err != nil && err != context.Canceled {
        log.Fatalf("Transport error: %v", err)
    }
    
    // Gracefully shutdown logger (flushes pending logs)
    if closeLogger, ok := logger.(interface{ Close() error }); ok {
        _ = closeLogger.Close()
    }
}
```

## License

MIT

