package mcpserver

import "context"

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	ServiceName string                 `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	APIVersion  string                 `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Endpoint    string                 `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema" yaml:"inputSchema"`
}

// ToolHandler is a function that handles a tool call
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

