# Configuration Validation

The MCP server automatically validates your `tools.json`/`tools.yaml` and `handlers.json`/`handlers.yaml` configuration files to catch errors before they cause runtime failures.

## Current Validation Features

### ✅ Tool Validation
- Tool name is required and must be a valid identifier
- InputSchema is required and must have valid structure
- InputSchema type must be valid (object, array, string, number, integer, boolean, null)
- InputSchema properties must be valid objects with type fields
- Endpoint format must be valid (starts with `/`)

### ✅ Handler Validation
- Handler type is required and must be "http"
- HTTP method is required and must be valid (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- Path or endpoint is required
- Path format must be valid (starts with `/`)
- **Path parameters must exist in InputSchema properties** (required)
- **Query parameters must exist in InputSchema properties** (required)
- Timeout format must be valid duration (e.g., "30s", "1m")
- Authorization configuration must be valid

### ✅ Service Validation
- baseURL is required
- baseURL must be a valid URL with http/https scheme
- baseURL must have a host
- Timeout format must be valid duration
- Authorization configuration must be valid

### ✅ Cross-Reference Validation
- Every tool must have a corresponding handler
- Every handler must have a corresponding tool
- Every tool's ServiceName must reference an existing service
- Service name is required for tools

### ✅ Version Validation
- If both tools and handlers specify versions, they must match

## How It Works

### Automatic Validation

Validation happens automatically when you register tools from files:

```go
server := mcpserver.New(&mcpserver.Config{
    Name:    "my-server",
    Version: "1.0.0",
})

// Validation happens automatically here
err := server.RegisterToolsFromFiles("tools.json", "handlers.json")
if err != nil {
    // Error includes validation failures
    log.Fatal(err)
}
```

**What happens:**
1. Files are parsed (JSON/YAML)
2. **Configuration is validated** ← Automatic step
3. If validation fails, registration fails with clear error messages
4. If validation passes, tools are registered

### Validation Errors

When validation fails, you'll get clear error messages:

```
configuration validation failed:
tool "getUser": handler not found for tool; 
tool "listUsers": service "user-service" not found; 
tool "createUser": invalid HTTP method "PATCH"
```

Each error includes:
- **Tool name** (if applicable)
- **Service name** (if applicable)
- **Field name** (if applicable)
- **Clear error message**

## Common Validation Errors

### 1. Missing Handler

**Error:**
```
tool "getUser": handler not found for tool
```

**Cause:** Tool exists in `tools.json` but no corresponding handler in `handlers.json`

**Fix:** Add handler configuration:
```json
{
  "handlers": {
    "getUser": {
      "type": "http",
      "method": "GET",
      "path": "/api/v1/users/{userId}"
    }
  }
}
```

### 2. Missing Service

**Error:**
```
tool "getUser": service "user-service" not found
```

**Cause:** Tool references a service that doesn't exist in `serviceConfig`

**Fix:** Add service configuration:
```json
{
  "serviceConfig": {
    "user-service": {
      "baseURL": "https://api.example.com"
    }
  }
}
```

### 3. Invalid HTTP Method

**Error:**
```
tool "createUser": invalid HTTP method "PATCH"
```

**Cause:** HTTP method is not one of the valid methods

**Fix:** Use valid HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS):
```json
{
  "handlers": {
    "createUser": {
      "type": "http",
      "method": "POST",  // ✅ Valid
      "path": "/api/v1/users"
    }
  }
}
```

### 4. Path Parameter Not in InputSchema

**Error:**
```
tool "getUser": path: path parameters [userId] not found in InputSchema properties
```

**Cause:** Path has `{userId}` parameter but InputSchema doesn't have `userId` property

**Fix:** Add parameter to InputSchema:
```json
{
  "Name": "getUser",
  "InputSchema": {
    "type": "object",
    "properties": {
      "userId": {  // ✅ Must match path parameter
        "type": "string"
      }
    }
  }
}
```

### 5. Query Parameter Not in InputSchema

**Error:**
```
tool "searchUsers": queryParams: query parameters [searchQuery] not found in InputSchema properties
```

**Cause:** Query params use `{searchQuery}` but InputSchema doesn't have it

**Fix:** Add parameter to InputSchema:
```json
{
  "Name": "searchUsers",
  "InputSchema": {
    "type": "object",
    "properties": {
      "searchQuery": {  // ✅ Must match query parameter
        "type": "string"
      }
    }
  }
}
```

### 6. Invalid baseURL

**Error:**
```
service "user-service": baseURL: URL must have scheme (http/https)
```

**Cause:** baseURL is missing scheme or invalid format

**Fix:** Use valid URL:
```json
{
  "serviceConfig": {
    "user-service": {
      "baseURL": "https://api.example.com"  // ✅ Valid URL
    }
  }
}
```

### 7. Invalid Timeout Format

**Error:**
```
tool "getUser": timeout: invalid timeout format "30": time: missing unit in duration "30"
```

**Cause:** Timeout doesn't have a unit (s, m, h)

**Fix:** Use valid duration format:
```json
{
  "handlers": {
    "getUser": {
      "timeout": "30s"  // ✅ Valid (30 seconds)
    }
  }
}
```

Valid formats: `"30s"`, `"1m"`, `"2h"`, etc.

### 8. Invalid Authorization Strategy

**Error:**
```
tool "getUser": authorization: invalid authorization strategy "invalid" (must be one of: pass-through, transform, static, basic, none)
```

**Cause:** Authorization strategy is not valid

**Fix:** Use valid strategy:
```yaml
authorization:
  strategy: pass-through  # ✅ Valid
```

### 9. Missing InputSchema

**Error:**
```
tool "getUser": InputSchema: InputSchema is required
```

**Cause:** Tool doesn't have InputSchema defined

**Fix:** Add InputSchema:
```json
{
  "Name": "getUser",
  "InputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

### 10. Invalid InputSchema Structure

**Error:**
```
tool "getUser": InputSchema: InputSchema must have 'type' field
```

**Cause:** InputSchema is missing required `type` field

**Fix:** Add type field:
```json
{
  "InputSchema": {
    "type": "object",  // ✅ Required
    "properties": {}
  }
}
```

## Examples

### Example 1: Valid Configuration

**tools.json:**
```json
[
  {
    "Name": "getUser",
    "Description": "Get user by ID",
    "ServiceName": "user-service",
    "Endpoint": "/api/v1/users/{userId}",
    "InputSchema": {
      "type": "object",
      "properties": {
        "userId": {
          "type": "string"
        }
      },
      "required": ["userId"]
    }
  }
]
```

**handlers.json:**
```json
{
  "serviceConfig": {
    "user-service": {
      "baseURL": "https://api.example.com",
      "timeout": "30s"
    }
  },
  "handlers": {
    "getUser": {
      "type": "http",
      "method": "GET",
      "path": "/api/v1/users/{userId}"
    }
  }
}
```

✅ **Result:** Validation passes, tool is registered

### Example 2: Invalid Configuration

**tools.json:**
```json
[
  {
    "Name": "getUser",
    "ServiceName": "user-service",
    "Endpoint": "/api/v1/users/{userId}",
    "InputSchema": {
      "type": "object",
      "properties": {
        "name": {  // ❌ Wrong parameter name
          "type": "string"
        }
      }
    }
  }
]
```

**handlers.json:**
```json
{
  "serviceConfig": {
    "user-service": {
      "baseURL": "api.example.com"  // ❌ Missing scheme
    }
  },
  "handlers": {
    "getUser": {
      "type": "http",
      "method": "INVALID",  // ❌ Invalid method
      "path": "/api/v1/users/{userId}"
    }
  }
}
```

❌ **Result:** Validation fails with errors:
```
configuration validation failed:
tool "getUser": path: path parameters [userId] not found in InputSchema properties; 
service "user-service": baseURL: URL must have scheme (http/https); 
tool "getUser": method: invalid HTTP method "INVALID"
```

## Manual Validation

You can also validate configuration programmatically without registering:

```go
import "github.com/vubon/go-mcp-server"

// Parse files
tools, _, _ := mcpserver.ParseToolsFromJSON(toolsData)
handlers, _ := mcpserver.ParseHandlersFromJSON(handlersData)

// Validate
result := mcpserver.ValidateConfiguration(tools, handlers)
if !result.Valid() {
    for _, err := range result.Errors {
        fmt.Printf("Error: %s\n", err)
    }
    return
}

fmt.Println("✅ Configuration is valid")
```

## Validation Rules Summary

### Path Parameters

**Rule:** All path parameters (e.g., `{userId}`) must exist in InputSchema properties

**Example:**
```yaml
# Path
path: /api/v1/users/{userId}

# InputSchema must have userId
InputSchema:
  type: object
  properties:
    userId:  # ✅ Required
      type: string
```

### Query Parameters

**Rule:** All dynamic query parameters (e.g., `{searchQuery}`) must exist in InputSchema properties

**Example:**
```yaml
# Query params
queryParams:
  q: "{searchQuery}"

# InputSchema must have searchQuery
InputSchema:
  type: object
  properties:
    searchQuery:  # ✅ Required
      type: string
```

### URL Format

**Rule:** baseURL must be a valid URL with http/https scheme

**Valid:**
- `https://api.example.com`
- `http://localhost:8080`
- `https://api.example.com/v1`

**Invalid:**
- `api.example.com` (missing scheme)
- `ftp://api.example.com` (invalid scheme)
- `https://` (missing host)

### Timeout Format

**Rule:** Timeout must be a valid Go duration string

**Valid:**
- `"30s"` (30 seconds)
- `"1m"` (1 minute)
- `"2h"` (2 hours)
- `"500ms"` (500 milliseconds)

**Invalid:**
- `"30"` (missing unit)
- `"1min"` (invalid unit, use "m")
- `"abc"` (not a duration)

### HTTP Methods

**Valid methods:**
- `GET`
- `POST`
- `PUT`
- `PATCH`
- `DELETE`
- `HEAD`
- `OPTIONS`

**Case-insensitive:** `"get"`, `"Get"`, `"GET"` are all valid

### Authorization Strategies

**Valid strategies:**
- `pass-through` - Forward client authorization as-is
- `transform` - Transform authorization header format
- `static` - Use fixed authorization value
- `basic` - Basic Authentication encoding
- `none` - No authorization header

See [Authorization Documentation](./auth/README.md) for details.

## Best Practices

1. **Validate Early**: Let validation catch errors during development
2. **Fix All Errors**: Don't ignore validation errors - they prevent runtime failures
3. **Match Parameters**: Ensure path/query parameters match InputSchema properties
4. **Use Valid URLs**: Always use full URLs with http/https scheme
5. **Valid Durations**: Use proper duration format for timeouts (e.g., "30s", not "30")

## Troubleshooting

### Validation passes but tool doesn't work

If validation passes but the tool fails at runtime:
- Check backend service is accessible
- Verify authorization credentials are correct
- Check network connectivity
- Review backend API documentation

### Too many validation errors

If you have many validation errors:
1. Fix one error at a time
2. Start with missing handlers/services (most critical)
3. Then fix parameter mismatches
4. Finally fix format issues (URLs, timeouts)

### Need to bypass validation

**Note:** Validation cannot be bypassed - it's mandatory for safety. If you need to test without validation, you can:
1. Fix the validation errors (recommended)
2. Use programmatic tool registration (bypasses file-based validation)

## Related Documentation

- [File-Based Registration](./file-based-registration.md) - How to configure tools and handlers
- [Authorization Strategies](./auth/README.md) - Authorization configuration
- [API Reference](../README.md#api-reference) - Programmatic API usage

