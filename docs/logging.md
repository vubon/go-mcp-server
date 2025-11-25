# Structured Logging Guide

This guide explains how to use structured logging in the MCP server package.

## Overview

The package provides structured JSON logging that is:
- **Datadog Compatible**: JSON format works seamlessly with Datadog and other log aggregators
- **Zero Latency**: Async logging prevents blocking request handlers
- **Ordered**: Logs maintain order within each server instance
- **Structured**: JSON format with contextual fields
- **Optional**: Can be disabled or replaced with custom logger

## Quick Start

### Basic Usage

```go
import (
    "github.com/vubon/go-mcp-server"
    "github.com/vubon/go-mcp-server/logger"
)

// Create logger
log := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    Level:  "info",
    Format: "json",
})

// Create server with logger
server := mcpserver.New(&mcpserver.Config{
    Name:    "my-server",
    Version: "1.0.0",
    Logger:  log,
})
```

### Log Levels

Supported log levels:
- `debug` - Detailed debugging information
- `info` - General informational messages (default)
- `warn` - Warning messages
- `error` - Error messages
- `off` - Disable logging

```go
log := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    Level: "debug", // Set log level
})
```

## Log Format

### JSON Format (Default)

Logs are output as JSON, one per line:

```json
{
  "timestamp": "2025-11-23T10:00:00.123456789Z",
  "level": "info",
  "message": "JSON-RPC request received",
  "service": "my-server",
  "version": "1.0.0",
  "instance_id": "server-01",
  "request_id": "req-123",
  "method": "tools/call",
  "tool": "getUser"
}
```

### Standard Fields

All logs automatically include:
- `timestamp` - RFC3339Nano timestamp
- `level` - Log level (debug, info, warn, error)
- `message` - Log message
- `service` - Service name (from server config)
- `version` - Service version (from server config)
- `instance_id` - Server instance identifier (hostname or pod name)

### Request Fields

HTTP transport automatically adds:
- `request_id` - JSON-RPC request ID
- `method` - JSON-RPC method name
- `tool` - Tool name (for tool calls)
- `has_error` - Whether response has error
- `error_code` - Error code (if error)
- `error_message` - Error message (if error)

## Configuration

### LoggerConfig

```go
type LoggerConfig struct {
    Level      string    // "debug", "info", "warn", "error", "off"
    Format     string    // "json" (default), "text" (for development)
    Writer     io.Writer // Output writer (default: buffered os.Stdout)
    Async      bool      // Use async logging (default: true)
    BufferSize int       // Buffer size for async logging (default: 1000)
}
```

### Example Configurations

**Production (Recommended):**
```go
log := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    Level:      "info",
    Format:     "json",
    Async:      true,
    BufferSize: 1000,
})
```

**Development:**
```go
log := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    Level:  "debug",
    Format: "json",
})
```

**Custom Writer:**
```go
file, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
log := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    Level:  "info",
    Writer: file,
})
```

## Using Your Own Logger

You can provide your own logger by implementing the `Logger` interface:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithFields(fields ...Field) Logger
    WithContext(ctx context.Context) Logger
}
```

### Example: Zap Logger Adapter

```go
import (
    "go.uber.org/zap"
    "github.com/vubon/go-mcp-server"
)

type ZapLoggerAdapter struct {
    zap *zap.Logger
}

func (l *ZapLoggerAdapter) Info(msg string, fields ...mcpserver.Field) {
    zapFields := make([]zap.Field, len(fields))
    for i, f := range fields {
        zapFields[i] = zap.Any(f.Key, f.Value)
    }
    l.zap.Info(msg, zapFields...)
}

// Implement other methods...

// Use with server
zapLogger, _ := zap.NewProduction()
server := mcpserver.New(&mcpserver.Config{
    Name:    "my-server",
    Version: "1.0.0",
    Logger:  &ZapLoggerAdapter{zap: zapLogger},
})
```

## Datadog Integration

### Automatic Integration

Datadog automatically collects JSON logs when:
1. Logs are in JSON format ✅
2. Logs are written to stdout/stderr ✅
3. Datadog agent is configured

### Recommended Fields

For distributed tracing, include:
- `dd.service` - Service name
- `dd.version` - Service version
- `dd.env` - Environment (production, staging, etc.)
- `dd.trace_id` - Distributed trace ID
- `dd.span_id` - Span ID

These can be added via custom logger or context.

### Example Datadog Log

```json
{
  "timestamp": "2025-11-23T10:00:00.123Z",
  "level": "error",
  "message": "HTTP request failed",
  "service": "mcp-server",
  "version": "1.0.0",
  "instance_id": "pod-abc123",
  "request_id": "req-xyz789",
  "dd.service": "mcp-server",
  "dd.version": "1.0.0",
  "dd.env": "production",
  "dd.trace_id": "1234567890",
  "error": "connection refused",
  "url": "https://api.example.com/users/123",
  "retry_attempt": 2
}
```

## Performance

### Async Logging

By default, logging is asynchronous:
- **Zero Latency**: Request handlers never block on log writes
- **High Throughput**: Buffered writes are more efficient
- **Ordered**: Single goroutine maintains log order
- **Graceful Degradation**: Drops logs if buffer is full (prevents blocking)

### How It Works

1. Log methods queue entries to a channel (non-blocking)
2. Background goroutine writes logs sequentially
3. If channel is full, log is dropped (prevents blocking)

### Buffer Size

Default buffer size is 1000. Adjust based on your needs:

```go
log := mcpserver.NewLogger(&mcpserver.LoggerConfig{
    BufferSize: 5000, // Larger buffer for high-traffic
})
```

## Distributed Systems

### Log Ordering

- **Within Instance**: Logs maintain perfect order
- **Across Instances**: Logs may be interleaved (normal in distributed systems)
- **Correlation**: Use `request_id`, `trace_id`, and `instance_id` to correlate logs

### Instance ID

Instance ID is automatically detected:
- Hostname (default)
- `POD_NAME` environment variable (Kubernetes)
- `HOSTNAME` environment variable (Kubernetes)

## Best Practices

1. **Use Appropriate Log Levels**
   - `debug` - Development only
   - `info` - Production (default)
   - `warn` - Important warnings
   - `error` - Errors that need attention

2. **Include Contextual Fields**
   - Request ID for request correlation
   - User ID for user-specific logs
   - Operation name for clarity

3. **Structured Fields**
   - Use consistent field names
   - Include relevant context
   - Avoid sensitive data

4. **Production Settings**
   - Use `info` level or higher
   - Enable async logging
   - Use appropriate buffer size
   - Monitor log volume

## Troubleshooting

### Logs Not Appearing

1. Check log level - lower level logs are filtered
2. Verify logger is set in server config
3. Check writer output (stdout/stderr)

### High Memory Usage

1. Reduce buffer size
2. Increase log level (fewer logs)
3. Use log sampling (future feature)

### Missing Logs

1. Buffer may be full (logs dropped)
2. Increase buffer size
3. Check for errors in log writer

## Examples

See `examples/simple/main.go` for a complete example.

## API Reference

### Functions

- `NewLogger(config *LoggerConfig) Logger` - Create a new logger
- `GetInstanceID() string` - Get instance identifier
- `F(key string, value interface{}) Field` - Helper to create Field

### Types

- `Logger` - Logging interface
- `Field` - Key-value pair for structured logging
- `Level` - Log level type
- `LoggerConfig` - Logger configuration

