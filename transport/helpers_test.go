package transport

import (
	"bytes"
	"encoding/json"
	"testing"

	mcpserver "github.com/vubon/go-mcp-server"
)

const (
	testToolName  = "test_tool"
	parseErrorMsg = "Parse error"
)

func TestExtractToolName(t *testing.T) {
	tests := []struct {
		name     string
		request  *mcpserver.Request
		expected string
	}{
		{
			name: "valid tools/call request",
			request: &mcpserver.Request{
				Method: methodToolsCall,
				Params: json.RawMessage(`{"name": "test_tool", "arguments": {}}`),
			},
			expected: "test_tool",
		},
		{
			name: "tools/call with different tool name",
			request: &mcpserver.Request{
				Method: methodToolsCall,
				Params: json.RawMessage(`{"name": "another_tool", "arguments": {"key": "value"}}`),
			},
			expected: "another_tool",
		},
		{
			name: "non-tools/call method",
			request: &mcpserver.Request{
				Method: "tools/list",
				Params: json.RawMessage(`{}`),
			},
			expected: "",
		},
		{
			name: "tools/call with empty params",
			request: &mcpserver.Request{
				Method: methodToolsCall,
				Params: json.RawMessage(``),
			},
			expected: "",
		},
		{
			name: "tools/call with nil params",
			request: &mcpserver.Request{
				Method: methodToolsCall,
				Params: nil,
			},
			expected: "",
		},
		{
			name: "tools/call with invalid JSON params",
			request: &mcpserver.Request{
				Method: methodToolsCall,
				Params: json.RawMessage(`invalid json`),
			},
			expected: "",
		},
		{
			name: "tools/call with params missing name field",
			request: &mcpserver.Request{
				Method: methodToolsCall,
				Params: json.RawMessage(`{"arguments": {}}`),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolName(tt.request)
			if result != tt.expected {
				t.Errorf("extractToolName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildRequestLogFields(t *testing.T) {
	tests := []struct {
		name     string
		request  *mcpserver.Request
		toolName string
		check    func([]mcpserver.Field) bool
	}{
		{
			name: "request with tool name",
			request: &mcpserver.Request{
				Method: "tools/call",
				ID:     123,
			},
			toolName: "test_tool",
			check: func(fields []mcpserver.Field) bool {
				if len(fields) != 4 {
					return false
				}
				fieldMap := make(map[string]interface{})
				for _, f := range fields {
					fieldMap[f.Key] = f.Value
				}
				return fieldMap["method"] == "tools/call" &&
					fieldMap["request_id"] == 123 &&
					fieldMap["tool"] == "test_tool" &&
					fieldMap["instance_id"] != nil
			},
		},
		{
			name: "request without tool name",
			request: &mcpserver.Request{
				Method: "tools/list",
				ID:     456,
			},
			toolName: "",
			check: func(fields []mcpserver.Field) bool {
				if len(fields) != 3 {
					return false
				}
				fieldMap := make(map[string]interface{})
				for _, f := range fields {
					fieldMap[f.Key] = f.Value
				}
				_, hasTool := fieldMap["tool"]
				return fieldMap["method"] == "tools/list" &&
					fieldMap["request_id"] == 456 &&
					fieldMap["instance_id"] != nil &&
					!hasTool
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := buildRequestLogFields(tt.request, tt.toolName)
			if !tt.check(fields) {
				t.Errorf("buildRequestLogFields() produced unexpected fields: %+v", fields)
			}
		})
	}
}

func TestBuildResponseLogFields(t *testing.T) {
	tests := []struct {
		name     string
		response *mcpserver.Response
		toolName string
		check    func([]mcpserver.Field) bool
	}{
		{
			name: "successful response with tool name",
			response: &mcpserver.Response{
				ID:     789,
				Result: map[string]interface{}{"key": "value"},
				Error:  nil,
			},
			toolName: "test_tool",
			check: func(fields []mcpserver.Field) bool {
				if len(fields) != 3 {
					return false
				}
				fieldMap := make(map[string]interface{})
				for _, f := range fields {
					fieldMap[f.Key] = f.Value
				}
				return fieldMap["request_id"] == 789 &&
					fieldMap["has_error"] == false &&
					fieldMap["tool"] == testToolName
			},
		},
		{
			name: "error response with tool name",
			response: &mcpserver.Response{
				ID: 101,
				Error: &mcpserver.Error{
					Code:    -32601,
					Message: "Method not found",
				},
			},
			toolName: "failing_tool",
			check: func(fields []mcpserver.Field) bool {
				if len(fields) != 5 {
					return false
				}
				fieldMap := make(map[string]interface{})
				for _, f := range fields {
					fieldMap[f.Key] = f.Value
				}
				return fieldMap["request_id"] == 101 &&
					fieldMap["has_error"] == true &&
					fieldMap["tool"] == "failing_tool" &&
					fieldMap["error_code"] == -32601 &&
					fieldMap["error_message"] == "Method not found"
			},
		},
		{
			name: "successful response without tool name",
			response: &mcpserver.Response{
				ID:     202,
				Result: map[string]interface{}{"status": "ok"},
				Error:  nil,
			},
			toolName: "",
			check: func(fields []mcpserver.Field) bool {
				if len(fields) != 2 {
					return false
				}
				fieldMap := make(map[string]interface{})
				for _, f := range fields {
					fieldMap[f.Key] = f.Value
				}
				_, hasTool := fieldMap["tool"]
				return fieldMap["request_id"] == 202 &&
					fieldMap["has_error"] == false &&
					!hasTool
			},
		},
		{
			name: "error response without tool name",
			response: &mcpserver.Response{
				ID: 303,
				Error: &mcpserver.Error{
					Code:    -32700,
					Message: "Parse error",
				},
			},
			toolName: "",
			check: func(fields []mcpserver.Field) bool {
				if len(fields) != 4 {
					return false
				}
				fieldMap := make(map[string]interface{})
				for _, f := range fields {
					fieldMap[f.Key] = f.Value
				}
				_, hasTool := fieldMap["tool"]
				return fieldMap["request_id"] == 303 &&
					fieldMap["has_error"] == true &&
					fieldMap["error_code"] == -32700 &&
					fieldMap["error_message"] == parseErrorMsg &&
					!hasTool
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := buildResponseLogFields(tt.response, tt.toolName)
			if !tt.check(fields) {
				t.Errorf("buildResponseLogFields() produced unexpected fields: %+v", fields)
			}
		})
	}
}

func TestLogRequest(t *testing.T) {
	// Create a server with a logger that writes to a buffer
	var buf bytes.Buffer
	logger := mcpserver.NewLogger(&mcpserver.LoggerConfig{
		Level:  "info",
		Writer: &buf,
	})

	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
		Logger:  logger,
	})

	request := &mcpserver.Request{
		Method: "tools/call",
		ID:     123,
	}

	// This should not panic
	logRequest(server, request, testToolName)

	// Close logger to flush all logs
	logger.(*mcpserver.JSONLogger).Close()

	// Verify log was written
	if buf.Len() == 0 {
		t.Error("Expected log to be written, but buffer is empty")
	}
}

func TestLogResponse(t *testing.T) {
	// Create a server with a logger that writes to a buffer
	var buf bytes.Buffer
	logger := mcpserver.NewLogger(&mcpserver.LoggerConfig{
		Level:  "info",
		Writer: &buf,
	})

	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
		Logger:  logger,
	})

	response := &mcpserver.Response{
		ID:     123,
		Result: map[string]interface{}{"key": "value"},
	}

	// This should not panic
	logResponse(server, response, testToolName)

	// Close logger to flush all logs
	logger.(*mcpserver.JSONLogger).Close()

	// Verify log was written
	if buf.Len() == 0 {
		t.Error("Expected log to be written, but buffer is empty")
	}
}

func TestLogRequest_NoLogger(t *testing.T) {
	t.Helper()
	// Create a server without a logger
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	request := &mcpserver.Request{
		Method: "tools/call",
		ID:     123,
	}

	// This should not panic even without a logger
	logRequest(server, request, testToolName)
}

func TestLogResponse_NoLogger(t *testing.T) {
	t.Helper()
	// Create a server without a logger
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	response := &mcpserver.Response{
		ID:     123,
		Result: map[string]interface{}{"key": "value"},
	}

	// This should not panic even without a logger
	logResponse(server, response, testToolName)
}
