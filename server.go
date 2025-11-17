package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// Config configures an MCP server
type Config struct {
	Name            string
	Version         string
	ProtocolVersion string // default: "2024-11-05"
}

// Server handles MCP tool calls
type Server struct {
	config  *Config
	tools   map[string]Tool
	handlers map[string]ToolHandler
}

// New creates a new MCP server
func New(config *Config) *Server {
	if config.ProtocolVersion == "" {
		config.ProtocolVersion = "2024-11-05"
	}

	return &Server{
		config:  config,
		tools:   make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
	}
}

// RegisterTool registers a tool with the server
func (s *Server) RegisterTool(name string, tool Tool, handler ToolHandler) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool.Name == "" {
		tool.Name = name
	}
	if handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}

	s.tools[name] = tool
	s.handlers[name] = handler
	return nil
}

// ListTools returns all registered tools
func (s *Server) ListTools() []Tool {
	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

// HandleRequest processes a JSON-RPC request
func (s *Server) HandleRequest(ctx context.Context, req *Request) *Response {
	if req.JSONRPC != "2.0" {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32600,
				Message: "Invalid Request",
			},
			ID: req.ID,
		}
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID)
	case "initialized", "notifications/initialized":
		// MCP notification - no response needed
		return nil
	case "tools/list":
		return s.handleListTools(req.ID)
	case "tools/call":
		return s.handleToolCall(ctx, req)
	default:
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32601,
				Message: "Method not found",
			},
			ID: req.ID,
		}
	}
}

func (s *Server) handleInitialize(id interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"protocolVersion": s.config.ProtocolVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.config.Name,
				"version": s.config.Version,
			},
		},
		ID: id,
	}
}

func (s *Server) handleListTools(id interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"tools": s.ListTools(),
		},
		ID: id,
	}
}

func (s *Server) handleToolCall(ctx context.Context, req *Request) *Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
			ID: req.ID,
		}
	}

	handler, exists := s.handlers[params.Name]
	if !exists {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32601,
				Message: fmt.Sprintf("Tool not found: %s", params.Name),
			},
			ID: req.ID,
		}
	}

	result, err := handler(ctx, params.Arguments)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32000,
				Message: "Server error",
				Data:    err.Error(),
			},
			ID: req.ID,
		}
	}

	// Format result as MCP content array
	content := formatResultAsContent(result)
	return &Response{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"content": content,
		},
		ID: req.ID,
	}
}

// formatResultAsContent formats the result as MCP content array
func formatResultAsContent(result interface{}) []map[string]interface{} {
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		jsonBytes = []byte(fmt.Sprintf("Error formatting result: %v", err))
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": string(jsonBytes),
		},
	}
}

