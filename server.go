package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Constants for MCP protocol
const (
	DefaultProtocolVersion = "2024-11-05"
	JSONRPCVersion         = "2.0"
	ContentTypeText        = "text"
	ContentTypeJSON        = "application/json"
	YAMLExtension          = ".yaml"
	YMLExtension           = ".yml"
)

// Config configures an MCP server
type Config struct {
	Name            string
	Version         string
	ProtocolVersion string // default: DefaultProtocolVersion
}

// Server handles MCP tool calls
type Server struct {
	config   *Config
	tools    map[string]Tool
	handlers map[string]ToolHandler
}

// New creates a new MCP server
func New(config *Config) *Server {
	if config.ProtocolVersion == "" {
		config.ProtocolVersion = DefaultProtocolVersion
	}

	return &Server{
		config:   config,
		tools:    make(map[string]Tool),
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
	if req.JSONRPC != JSONRPCVersion {
		return &Response{
			JSONRPC: JSONRPCVersion,
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
			JSONRPC: JSONRPCVersion,
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
		JSONRPC: JSONRPCVersion,
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
		JSONRPC: JSONRPCVersion,
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
			JSONRPC: JSONRPCVersion,
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
			JSONRPC: JSONRPCVersion,
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
			JSONRPC: JSONRPCVersion,
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
		JSONRPC: JSONRPCVersion,
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
			"type": ContentTypeText,
			"text": string(jsonBytes),
		},
	}
}

// RegisterToolsFromJSON registers tools from JSON data
func (s *Server) RegisterToolsFromJSON(toolsData, handlersData []byte) error {
	// Parse tools
	tools, err := parseToolsFromJSON(toolsData)
	if err != nil {
		return fmt.Errorf("failed to parse tools JSON: %w", err)
	}

	// Parse handlers
	handlersConfig, err := parseHandlersFromJSON(handlersData)
	if err != nil {
		return fmt.Errorf("failed to parse handlers JSON: %w", err)
	}

	return s.registerToolsFromConfig(tools, handlersConfig)
}

// RegisterToolsFromYAML registers tools from YAML data
func (s *Server) RegisterToolsFromYAML(toolsData, handlersData []byte) error {
	// Parse tools
	tools, err := parseToolsFromYAML(toolsData)
	if err != nil {
		return fmt.Errorf("failed to parse tools YAML: %w", err)
	}

	// Parse handlers
	handlersConfig, err := parseHandlersFromYAML(handlersData)
	if err != nil {
		return fmt.Errorf("failed to parse handlers YAML: %w", err)
	}

	return s.registerToolsFromConfig(tools, handlersConfig)
}

// RegisterToolsFromFiles registers tools from files (auto-detects format)
func (s *Server) RegisterToolsFromFiles(toolsFile, handlersFile string) error {
	// Read tools file
	toolsData, err := os.ReadFile(toolsFile)
	if err != nil {
		return fmt.Errorf("failed to read tools file %s: %w", toolsFile, err)
	}

	// Read handlers file
	handlersData, err := os.ReadFile(handlersFile)
	if err != nil {
		return fmt.Errorf("failed to read handlers file %s: %w", handlersFile, err)
	}

	// Detect format from file extension
	toolsExt := strings.ToLower(filepath.Ext(toolsFile))
	handlersExt := strings.ToLower(filepath.Ext(handlersFile))

	// Parse tools based on format
	var tools []ToolFile
	if toolsExt == YAMLExtension || toolsExt == YMLExtension {
		tools, err = parseToolsFromYAML(toolsData)
	} else {
		tools, err = parseToolsFromJSON(toolsData)
	}
	if err != nil {
		return fmt.Errorf("failed to parse tools file: %w", err)
	}

	// Parse handlers based on format
	var handlersConfig *HandlersConfig
	if handlersExt == YAMLExtension || handlersExt == YMLExtension {
		handlersConfig, err = parseHandlersFromYAML(handlersData)
	} else {
		handlersConfig, err = parseHandlersFromJSON(handlersData)
	}
	if err != nil {
		return fmt.Errorf("failed to parse handlers file: %w", err)
	}

	return s.registerToolsFromConfig(tools, handlersConfig)
}

// registerToolsFromConfig registers tools from parsed configuration.
func (s *Server) registerToolsFromConfig(tools []ToolFile, handlersConfig *HandlersConfig) error {
	// Validate handlers config
	if handlersConfig == nil {
		return fmt.Errorf("handlers configuration is nil")
	}

	// Register each tool
	for _, toolFile := range tools {
		// Validate tool
		if toolFile.Name == "" {
			return fmt.Errorf("tool name cannot be empty")
		}

		// Get handler config
		handlerConfig, exists := handlersConfig.Handlers[toolFile.Name]
		if !exists {
			return fmt.Errorf("handler configuration not found for tool: %s", toolFile.Name)
		}

		// Get service config
		if toolFile.ServiceName == "" {
			return fmt.Errorf("service name is required for tool: %s", toolFile.Name)
		}

		serviceConfig, exists := handlersConfig.ServiceConfig[toolFile.ServiceName]
		if !exists {
			return fmt.Errorf(
				"service configuration not found for service: %s (tool: %s)",
				toolFile.ServiceName, toolFile.Name,
			)
		}

		// Generate HTTP handler
		handler, err := generateHTTPHandler(&toolFile, &handlerConfig, serviceConfig)
		if err != nil {
			return fmt.Errorf("failed to generate handler for tool %s: %w", toolFile.Name, err)
		}

		// Convert ToolFile to Tool
		tool := toolFile.ToTool()

		// Register tool
		if err := s.RegisterTool(toolFile.Name, tool, handler); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", toolFile.Name, err)
		}
	}

	return nil
}
