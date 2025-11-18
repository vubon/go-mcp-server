package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("With protocol version", func(t *testing.T) {
		config := &Config{
			Name:            "test-server",
			Version:         "1.0.0",
			ProtocolVersion: "2024-11-05",
		}

		server := New(config)
		if server == nil {
			t.Fatal("New returned nil")
		}
		if server.config != config {
			t.Error("Server config does not match")
		}
		if server.config.ProtocolVersion != "2024-11-05" {
			t.Errorf("Expected protocol version '2024-11-05', got %s", server.config.ProtocolVersion)
		}
	})

	t.Run("Without protocol version (default)", func(t *testing.T) {
		config := &Config{
			Name:    "test-server",
			Version: "1.0.0",
		}

		server := New(config)
		if server == nil {
			t.Fatal("New returned nil")
		}
		if server.config.ProtocolVersion != "2024-11-05" {
			t.Errorf("Expected default protocol version '2024-11-05', got %s", server.config.ProtocolVersion)
		}
	})

	t.Run("Empty protocol version (default)", func(t *testing.T) {
		config := &Config{
			Name:            "test-server",
			Version:         "1.0.0",
			ProtocolVersion: "",
		}

		server := New(config)
		if server == nil {
			t.Fatal("New returned nil")
		}
		if server.config.ProtocolVersion != "2024-11-05" {
			t.Errorf("Expected default protocol version '2024-11-05', got %s", server.config.ProtocolVersion)
		}
	})
}

func TestServer_RegisterTool(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	t.Run("Valid tool registration", func(t *testing.T) {
		tool := Tool{
			Name:        "test_tool",
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}

		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return map[string]string{"result": "success"}, nil
		}

		err := server.RegisterTool("test_tool", tool, handler)
		if err != nil {
			t.Fatalf("RegisterTool failed: %v", err)
		}

		// Verify tool was registered
		tools := server.ListTools()
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
		if tools[0].Name != "test_tool" {
			t.Errorf("Expected tool name 'test_tool', got %s", tools[0].Name)
		}
	})

	t.Run("Empty tool name", func(t *testing.T) {
		tool := Tool{
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}

		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, nil
		}

		err := server.RegisterTool("", tool, handler)
		if err == nil {
			t.Error("Expected error for empty tool name")
		}
		if err.Error() != "tool name cannot be empty" {
			t.Errorf("Expected error 'tool name cannot be empty', got %v", err)
		}
	})

	t.Run("Nil handler", func(t *testing.T) {
		tool := Tool{
			Name:        "test_tool",
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}

		err := server.RegisterTool("test_tool", tool, nil)
		if err == nil {
			t.Error("Expected error for nil handler")
		}
		if err.Error() != "tool handler cannot be nil" {
			t.Errorf("Expected error 'tool handler cannot be nil', got %v", err)
		}
	})

	t.Run("Tool name auto-filled from parameter", func(t *testing.T) {
		tool := Tool{
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}

		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, nil
		}

		err := server.RegisterTool("auto_name", tool, handler)
		if err != nil {
			t.Fatalf("RegisterTool failed: %v", err)
		}

		// Verify tool name was set
		tools := server.ListTools()
		found := false
		for _, t := range tools {
			if t.Name == "auto_name" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Tool name was not auto-filled")
		}
	})
}

func TestServer_ListTools(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	t.Run("Empty tools list", func(t *testing.T) {
		tools := server.ListTools()
		if len(tools) != 0 {
			t.Errorf("Expected empty tools list, got %d tools", len(tools))
		}
	})

	t.Run("Multiple tools", func(t *testing.T) {
		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, nil
		}

		server.RegisterTool("tool1", Tool{Name: "tool1", InputSchema: map[string]interface{}{}}, handler)
		server.RegisterTool("tool2", Tool{Name: "tool2", InputSchema: map[string]interface{}{}}, handler)
		server.RegisterTool("tool3", Tool{Name: "tool3", InputSchema: map[string]interface{}{}}, handler)

		tools := server.ListTools()
		if len(tools) != 3 {
			t.Errorf("Expected 3 tools, got %d", len(tools))
		}
	})
}

func TestServer_HandleRequest(t *testing.T) {
	server := New(&Config{
		Name:    "test-server",
		Version: "1.0.0",
	})

	t.Run("Invalid JSON-RPC version", func(t *testing.T) {
		req := &Request{
			JSONRPC: "1.0",
			Method:  "initialize",
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error == nil {
			t.Error("Expected error response")
		}
		if resp.Error.Code != -32600 {
			t.Errorf("Expected error code -32600, got %d", resp.Error.Code)
		}
		if resp.Error.Message != "Invalid Request" {
			t.Errorf("Expected error message 'Invalid Request', got %s", resp.Error.Message)
		}
	})

	t.Run("Initialize method", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "initialize",
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC '2.0', got %s", resp.JSONRPC)
		}
		if resp.Error != nil {
			t.Errorf("Unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		if result["protocolVersion"] != "2024-11-05" {
			t.Errorf("Expected protocol version '2024-11-05', got %v", result["protocolVersion"])
		}

		serverInfo, ok := result["serverInfo"].(map[string]interface{})
		if !ok {
			t.Fatal("serverInfo is not a map")
		}
		if serverInfo["name"] != "test-server" {
			t.Errorf("Expected server name 'test-server', got %v", serverInfo["name"])
		}
		if serverInfo["version"] != "1.0.0" {
			t.Errorf("Expected server version '1.0.0', got %v", serverInfo["version"])
		}
	})

	t.Run("Initialized notification", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "initialized",
			ID:      nil,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp != nil {
			t.Error("Expected nil response for notification")
		}
	})

	t.Run("Tools list method", func(t *testing.T) {
		// Register a tool first
		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, nil
		}
		server.RegisterTool("test_tool", Tool{
			Name:        "test_tool",
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}, handler)

		req := &Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error != nil {
			t.Errorf("Unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		tools, ok := result["tools"].([]Tool)
		if !ok {
			t.Fatal("Tools is not a slice")
		}
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("Tools call method", func(t *testing.T) {
		// Register a tool
		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return map[string]string{"result": "success"}, nil
		}
		server.RegisterTool("test_tool", Tool{
			Name:        "test_tool",
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}, handler)

		params := ToolCallParams{
			Name:      "test_tool",
			Arguments: map[string]interface{}{"key": "value"},
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error != nil {
			t.Errorf("Unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		content, ok := result["content"].([]map[string]interface{})
		if !ok {
			t.Fatal("Content is not a slice")
		}
		if len(content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(content))
		}
		if content[0]["type"] != "text" {
			t.Errorf("Expected content type 'text', got %v", content[0]["type"])
		}
	})

	t.Run("Unknown method", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "unknown/method",
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error == nil {
			t.Error("Expected error response")
		}
		if resp.Error.Code != -32601 {
			t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
		}
		if resp.Error.Message != "Method not found" {
			t.Errorf("Expected error message 'Method not found', got %s", resp.Error.Message)
		}
	})
}

func TestServer_HandleToolCall(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	t.Run("Invalid params", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  json.RawMessage("invalid json"),
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error == nil {
			t.Error("Expected error response")
		}
		if resp.Error.Code != -32602 {
			t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
		}
		if resp.Error.Message != "Invalid params" {
			t.Errorf("Expected error message 'Invalid params', got %s", resp.Error.Message)
		}
	})

	t.Run("Tool not found", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "nonexistent_tool",
			Arguments: map[string]interface{}{},
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error == nil {
			t.Error("Expected error response")
		}
		if resp.Error.Code != -32601 {
			t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
		}
		if resp.Error.Message != "Tool not found: nonexistent_tool" {
			t.Errorf("Expected error message 'Tool not found: nonexistent_tool', got %s", resp.Error.Message)
		}
	})

	t.Run("Handler error", func(t *testing.T) {
		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("handler error")
		}
		server.RegisterTool("error_tool", Tool{
			Name:        "error_tool",
			Description: "Error tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}, handler)

		params := ToolCallParams{
			Name:      "error_tool",
			Arguments: map[string]interface{}{},
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error == nil {
			t.Error("Expected error response")
		}
		if resp.Error.Code != -32000 {
			t.Errorf("Expected error code -32000, got %d", resp.Error.Code)
		}
		if resp.Error.Message != "Server error" {
			t.Errorf("Expected error message 'Server error', got %s", resp.Error.Message)
		}
		if resp.Error.Data != "handler error" {
			t.Errorf("Expected error data 'handler error', got %v", resp.Error.Data)
		}
	})

	t.Run("Success", func(t *testing.T) {
		handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return map[string]string{"status": "ok"}, nil
		}
		server.RegisterTool("success_tool", Tool{
			Name:        "success_tool",
			Description: "Success tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}, handler)

		params := ToolCallParams{
			Name:      "success_tool",
			Arguments: map[string]interface{}{"key": "value"},
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      1,
		}

		resp := server.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Error != nil {
			t.Errorf("Unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		content, ok := result["content"].([]map[string]interface{})
		if !ok {
			t.Fatal("Content is not a slice")
		}
		if len(content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(content))
		}
		if content[0]["type"] != "text" {
			t.Errorf("Expected content type 'text', got %v", content[0]["type"])
		}
	})
}

func TestFormatResultAsContent(t *testing.T) {
	t.Run("Valid result", func(t *testing.T) {
		result := map[string]interface{}{
			"key": "value",
			"num": 123,
		}

		content := formatResultAsContent(result)
		if len(content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(content))
		}
		if content[0]["type"] != "text" {
			t.Errorf("Expected type 'text', got %v", content[0]["type"])
		}
		if _, ok := content[0]["text"].(string); !ok {
			t.Error("Text is not a string")
		}
	})

	t.Run("Complex result", func(t *testing.T) {
		result := map[string]interface{}{
			"nested": map[string]interface{}{
				"key": "value",
			},
			"array": []interface{}{1, 2, 3},
		}

		content := formatResultAsContent(result)
		if len(content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(content))
		}
		text := content[0]["text"].(string)
		if len(text) == 0 {
			t.Error("Text is empty")
		}
	})
}

func TestServer_RegisterToolsFromJSON(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	t.Run("Valid JSON", func(t *testing.T) {
		toolsJSON := []byte(`[
			{
				"Name": "test_tool",
				"Description": "Test tool",
				"ServiceName": "test-service",
				"InputSchema": {"type": "object"}
			}
		]`)

		handlersJSON := []byte(`{
			"serviceConfig": {
				"test-service": {
					"baseURL": "https://api.example.com"
				}
			},
			"handlers": {
				"test_tool": {
					"type": "http",
					"method": "POST",
					"path": "/api/test"
				}
			}
		}`)

		err := server.RegisterToolsFromJSON(toolsJSON, handlersJSON)
		if err != nil {
			t.Fatalf("RegisterToolsFromJSON failed: %v", err)
		}

		tools := server.ListTools()
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("Invalid tools JSON", func(t *testing.T) {
		toolsJSON := []byte(`invalid json`)
		handlersJSON := []byte(`{}`)

		err := server.RegisterToolsFromJSON(toolsJSON, handlersJSON)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("Invalid handlers JSON", func(t *testing.T) {
		toolsJSON := []byte(`[]`)
		handlersJSON := []byte(`invalid json`)

		err := server.RegisterToolsFromJSON(toolsJSON, handlersJSON)
		if err == nil {
			t.Error("Expected error for invalid handlers JSON")
		}
	})
}

func TestServer_RegisterToolsFromYAML(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	t.Run("Valid YAML", func(t *testing.T) {
		toolsYAML := []byte(`
- Name: test_tool
  Description: Test tool
  ServiceName: test-service
  InputSchema:
    type: object
`)

		handlersYAML := []byte(`
serviceConfig:
  test-service:
    baseURL: https://api.example.com
handlers:
  test_tool:
    type: http
    method: POST
    path: /api/test
`)

		err := server.RegisterToolsFromYAML(toolsYAML, handlersYAML)
		if err != nil {
			t.Fatalf("RegisterToolsFromYAML failed: %v", err)
		}

		tools := server.ListTools()
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("Invalid tools YAML", func(t *testing.T) {
		toolsYAML := []byte(`invalid: yaml: [`)
		handlersYAML := []byte(`{}`)

		err := server.RegisterToolsFromYAML(toolsYAML, handlersYAML)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})
}

func TestServer_RegisterToolsFromFiles(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	// Create temporary files
	tmpDir := t.TempDir()
	toolsFile := filepath.Join(tmpDir, "tools.json")
	handlersFile := filepath.Join(tmpDir, "handlers.json")

	t.Run("Valid JSON files", func(t *testing.T) {
		toolsJSON := `[
			{
				"Name": "test_tool",
				"Description": "Test tool",
				"ServiceName": "test-service",
				"InputSchema": {"type": "object"}
			}
		]`
		os.WriteFile(toolsFile, []byte(toolsJSON), 0644)

		handlersJSON := `{
			"serviceConfig": {
				"test-service": {
					"baseURL": "https://api.example.com"
				}
			},
			"handlers": {
				"test_tool": {
					"type": "http",
					"method": "POST",
					"path": "/api/test"
				}
			}
		}`
		os.WriteFile(handlersFile, []byte(handlersJSON), 0644)

		err := server.RegisterToolsFromFiles(toolsFile, handlersFile)
		if err != nil {
			t.Fatalf("RegisterToolsFromFiles failed: %v", err)
		}

		tools := server.ListTools()
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("Valid YAML files", func(t *testing.T) {
		toolsYAMLFile := filepath.Join(tmpDir, "tools.yaml")
		handlersYAMLFile := filepath.Join(tmpDir, "handlers.yaml")

		toolsYAML := `
- Name: test_tool
  Description: Test tool
  ServiceName: test-service
  InputSchema:
    type: object
`
		os.WriteFile(toolsYAMLFile, []byte(toolsYAML), 0644)

		handlersYAML := `
serviceConfig:
  test-service:
    baseURL: https://api.example.com
handlers:
  test_tool:
    type: http
    method: POST
    path: /api/test
`
		os.WriteFile(handlersYAMLFile, []byte(handlersYAML), 0644)

		err := server.RegisterToolsFromFiles(toolsYAMLFile, handlersYAMLFile)
		if err != nil {
			t.Fatalf("RegisterToolsFromFiles failed: %v", err)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		err := server.RegisterToolsFromFiles("nonexistent.json", "nonexistent.json")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("Mixed JSON and YAML", func(t *testing.T) {
		toolsJSONFile := filepath.Join(tmpDir, "tools.json")
		handlersYAMLFile := filepath.Join(tmpDir, "handlers.yaml")

		toolsJSON := `[
			{
				"Name": "test_tool",
				"Description": "Test tool",
				"ServiceName": "test-service",
				"InputSchema": {"type": "object"}
			}
		]`
		os.WriteFile(toolsJSONFile, []byte(toolsJSON), 0644)

		handlersYAML := `
serviceConfig:
  test-service:
    baseURL: https://api.example.com
handlers:
  test_tool:
    type: http
    method: POST
    path: /api/test
`
		os.WriteFile(handlersYAMLFile, []byte(handlersYAML), 0644)

		err := server.RegisterToolsFromFiles(toolsJSONFile, handlersYAMLFile)
		if err != nil {
			t.Fatalf("RegisterToolsFromFiles failed with mixed formats: %v", err)
		}
	})
}

func TestServer_RegisterToolsFromConfig_Validation(t *testing.T) {
	server := New(&Config{
		Name:    "test",
		Version: "1.0.0",
	})

	t.Run("Nil handlers config", func(t *testing.T) {
		tools := []ToolFile{
			{Name: "test_tool", ServiceName: "test-service"},
		}

		err := server.registerToolsFromConfig(tools, nil)
		if err == nil {
			t.Error("Expected error for nil handlers config")
		}
	})

	t.Run("Empty tool name", func(t *testing.T) {
		tools := []ToolFile{
			{Name: "", ServiceName: "test-service"},
		}

		handlersConfig := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers:      map[string]HandlerConfig{},
		}

		err := server.registerToolsFromConfig(tools, handlersConfig)
		if err == nil {
			t.Error("Expected error for empty tool name")
		}
	})

	t.Run("Missing handler config", func(t *testing.T) {
		tools := []ToolFile{
			{Name: "test_tool", ServiceName: "test-service"},
		}

		handlersConfig := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers:      map[string]HandlerConfig{},
		}

		err := server.registerToolsFromConfig(tools, handlersConfig)
		if err == nil {
			t.Error("Expected error for missing handler config")
		}
	})

	t.Run("Missing service name", func(t *testing.T) {
		tools := []ToolFile{
			{Name: "test_tool", ServiceName: ""},
		}

		handlersConfig := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers: map[string]HandlerConfig{
				"test_tool": {Type: "http", Method: "POST", Path: "/api/test"},
			},
		}

		err := server.registerToolsFromConfig(tools, handlersConfig)
		if err == nil {
			t.Error("Expected error for missing service name")
		}
	})

	t.Run("Missing service config", func(t *testing.T) {
		tools := []ToolFile{
			{Name: "test_tool", ServiceName: "test-service"},
		}

		handlersConfig := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers: map[string]HandlerConfig{
				"test_tool": {Type: "http", Method: "POST", Path: "/api/test"},
			},
		}

		err := server.registerToolsFromConfig(tools, handlersConfig)
		if err == nil {
			t.Error("Expected error for missing service config")
		}
	})
}

