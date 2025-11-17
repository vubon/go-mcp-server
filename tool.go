package mcpserver

import "context"

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolHandler is a function that handles a tool call
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

