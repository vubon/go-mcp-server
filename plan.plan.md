# JSON/YAML Tool Registration with Auto-Generated HTTP Handlers

## Overview

Add functionality to register MCP tools from JSON or YAML files with automatic HTTP handler generation. Tools and handlers are defined declaratively in separate files, allowing users to configure service endpoints without writing Go code.

## Implementation Details

### 1. Extend Tool Structure

- Update `Tool` struct in `tool.go` to include optional service metadata:
  - `ServiceName` (string, optional): Name of the service to call
  - `APIVersion` (string, optional): API version for the service
  - `Endpoint` (string, optional): Service endpoint path
  - Update JSON tags to use capitalized field names to match file format (`Name`, `Description`, etc.)

### 2. Create Handler Configuration Structs

- Create new file `handler_config.go` with:
  - `ServiceConfig` struct: baseURL, timeout, headers
  - `HandlerConfig` struct: type, method, path, headers, timeout
  - `HandlersConfig` struct: serviceConfig map, handlers map
  - Unmarshaling logic for JSON/YAML

### 3. Add File-Based Registration

- Create `RegisterToolsFromFiles(toolsFile, handlersFile string) error` method in `server.go`:
  - Auto-detects format (JSON vs YAML) from file extensions (.json, .yaml, .yml)
  - Reads and parses both tools and handlers files
  - Validates tool definitions
  - Auto-generates HTTP handlers from configuration
  - Registers tools with generated handlers

### 4. Add Format Parsing Methods

- Create `RegisterToolsFromJSON(toolsData, handlersData []byte) error` for JSON parsing
- Create `RegisterToolsFromYAML(toolsData, handlersData []byte) error` for YAML parsing
- Both methods parse tools and handlers, then generate HTTP handlers

### 5. Auto-Generate HTTP Handlers

- Create `generateHTTPHandler()` function that:
  - Takes tool definition and handler config
  - Creates HTTP client with service baseURL
  - Merges headers: service headers + handler headers (handler overrides service)
  - Resolves timeout: handler > service > default (30s)
  - Substitutes environment variables in headers (e.g., `${API_KEY}`)
  - Makes HTTP request with proper method, path, body (from tool args)
  - Parses and returns response

### 6. Header Merging Strategy

- Service-level headers (in `serviceConfig`) apply to all endpoints
- Handler-level headers (in `handlers`) are endpoint-specific
- Final headers = service headers merged with handler headers (handler overrides)
- Support environment variable substitution: `${VAR_NAME}`

### 7. Timeout Resolution

- Priority order: Handler timeout > Service timeout > Default (30s)
- Each endpoint can have different timeout based on its needs

### 8. Dependencies

- Add YAML parsing library: `gopkg.in/yaml.v3`

### Files to Modify/Create:

- `tool.go`: Extend `Tool` struct with service metadata fields (ServiceName, APIVersion, Endpoint)
- `server.go`: Add `RegisterToolsFromFiles()`, `RegisterToolsFromJSON()`, `RegisterToolsFromYAML()` methods and HTTP handler generation
- `handler_config.go` (NEW): Handler configuration structs and parsing logic
- `go.mod`: Add YAML parsing dependency

### Example Usage:

```go
// In main.go
server := mcpserver.New(&mcpserver.Config{
    Name:    "greeter",
    Version: "1.0.0",
})

err := server.RegisterToolsFromFiles("tools.json", "handlers.json")
if err != nil {
    log.Fatal(err)
}
```

### File Structure:

**tools.json/tools.yaml:**
- Array of tool definitions
- Each tool: Name, Description, ServiceName, APIVersion, Endpoint, InputSchema

**handlers.json/handlers.yaml:**
- `serviceConfig`: Map of service name to config (baseURL, timeout, headers)
- `handlers`: Map of tool name to handler config (type, method, path, headers, timeout)

### To-dos

- [ ] Add YAML dependency to go.mod
- [ ] Extend Tool struct with ServiceName, APIVersion, Endpoint fields
- [ ] Create handler_config.go with configuration structs
- [ ] Add RegisterToolsFromJSON method that parses tools and handlers, generates HTTP handlers
- [ ] Add RegisterToolsFromYAML method that parses tools and handlers, generates HTTP handlers
- [ ] Add RegisterToolsFromFiles method that auto-detects format and calls appropriate parser
- [ ] Implement HTTP handler generation with header merging and timeout resolution
- [ ] Add environment variable substitution for headers
- [ ] Add comprehensive error handling and validation
- [ ] Update example main.go to use new file-based registration

