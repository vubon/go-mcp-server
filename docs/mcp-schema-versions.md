# MCP Schema Version Support

This package supports multiple MCP (Model Context Protocol) schema versions, allowing you to choose the version that best fits your needs.

## Supported Versions

The package supports the following official MCP schema versions:

1. **`2024-11-05`** - Initial/legacy version
   - First stable MCP schema release
   - Basic tool and protocol support

2. **`2025-03-26`** - First major update
   - Initial release of formal schema definitions
   - Enhanced protocol structure

3. **`2025-06-18`** - Latest stable version (default)
   - Introduced JSON Schema 2020-12 as default dialect
   - Enhanced schema validation
   - Most up-to-date specification

## Default Version

The package defaults to **`2025-06-18`** (latest stable version) when no protocol version is specified.

```go
server := mcpserver.New(&mcpserver.Config{
    Name:    "my-server",
    Version: "1.0.0",
    // ProtocolVersion not specified - uses default (2025-06-18)
})
```

## Specifying a Version

You can explicitly specify which MCP schema version to use:

### Using Constants (Recommended)

```go
import "github.com/vubon/go-mcp-server"

server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: mcpserver.SchemaVersion2025_06_18, // Latest stable
})

// Or use an older version
server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: mcpserver.SchemaVersion2024_11_05, // Legacy version
})
```

### Available Constants

- `mcpserver.SchemaVersion2024_11_05` - Initial/legacy version
- `mcpserver.SchemaVersion2025_03_26` - First major update
- `mcpserver.SchemaVersion2025_06_18` - Latest stable (default)

### Using String Literals

You can also use string literals directly:

```go
server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: "2025-06-18", // String literal
})
```

## Version in Initialize Response

The protocol version you specify is returned in the `initialize` response:

```json
{
  "jsonrpc": "2.0",
  "result": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "tools": {
        "listChanged": false
      }
    },
    "serverInfo": {
      "name": "my-server",
      "version": "1.0.0"
    }
  },
  "id": 1
}
```

## Version Compatibility

### Backward Compatibility

All supported versions are backward compatible with the core MCP protocol:
- JSON-RPC 2.0 messaging
- Tool registration and listing
- Tool execution
- HTTP and Stdio transports

### Version Differences

The main differences between versions are:
- **JSON Schema dialect**: 2025-06-18 uses JSON Schema 2020-12
- **Schema validation**: Enhanced validation in newer versions
- **Protocol capabilities**: Extended capabilities in newer versions

### Choosing a Version

**Use latest (2025-06-18) if:**
- You're starting a new project
- You want the most up-to-date features
- You need JSON Schema 2020-12 support

**Use older versions if:**
- You need compatibility with older MCP clients
- You have existing integrations that require specific versions
- You're maintaining legacy systems

## Migration Guide

### Updating from 2024-11-05 to 2025-06-18

If you're currently using the old default (`2024-11-05`), you can update:

**Before:**
```go
server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: "2024-11-05", // Old default
})
```

**After:**
```go
server := mcpserver.New(&mcpserver.Config{
    Name:            "my-server",
    Version:         "1.0.0",
    ProtocolVersion: mcpserver.SchemaVersion2025_06_18, // New default
})
```

**Or simply remove the ProtocolVersion** (it will default to 2025-06-18):
```go
server := mcpserver.New(&mcpserver.Config{
    Name:    "my-server",
    Version: "1.0.0",
    // ProtocolVersion omitted - uses default (2025-06-18)
})
```

## Best Practices

1. **Use constants instead of strings** - Prevents typos and makes code more maintainable
   ```go
   // ✅ Good
   ProtocolVersion: mcpserver.SchemaVersion2025_06_18
   
   // ❌ Avoid
   ProtocolVersion: "2025-06-18"
   ```

2. **Use latest version for new projects** - Default to `2025-06-18` unless you have specific requirements

3. **Document version requirements** - If your project requires a specific version, document it clearly

4. **Test with your MCP clients** - Verify that your chosen version works with your MCP clients

## References

- [MCP Schema Repository](https://github.com/modelcontextprotocol/modelcontextprotocol/tree/main/schema)
- [MCP Specification](https://modelcontextprotocol.io/)
- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/schema)

## Related Documentation

- [File-Based Registration](./file-based-registration.md) - Configure tools and handlers
- [Configuration Validation](./configuration-validation.md) - Validate configuration files
- [Authorization Strategies](./auth/README.md) - Configure authorization

