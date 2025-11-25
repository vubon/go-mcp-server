package transport

import (
	"encoding/json"

	mcpserver "github.com/vubon/go-mcp-server"
)

// methodToolsCall is the JSON-RPC method name for tool calls
const methodToolsCall = "tools/call"

// extractToolName extracts the tool name from request params if method is tools/call
func extractToolName(req *mcpserver.Request) string {
	if req.Method != methodToolsCall || len(req.Params) == 0 {
		return ""
	}

	var params mcpserver.ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ""
	}

	return params.Name
}

// logRequest logs the incoming JSON-RPC request (shared by all transports)
func logRequest(server *mcpserver.Server, req *mcpserver.Request, toolName string) {
	logger := server.GetLogger()
	if logger == nil {
		return
	}

	fields := buildRequestLogFields(req, toolName)
	logger.Info("JSON-RPC request received", fields...)
}

// logResponse logs the JSON-RPC response (shared by all transports)
func logResponse(server *mcpserver.Server, resp *mcpserver.Response, toolName string) {
	logger := server.GetLogger()
	if logger == nil {
		return
	}

	fields := buildResponseLogFields(resp, toolName)
	logger.Info("JSON-RPC response sent", fields...)
}

// buildRequestLogFields builds log fields for request logging (shared by all transports)
func buildRequestLogFields(req *mcpserver.Request, toolName string) []mcpserver.Field {
	fields := []mcpserver.Field{
		{Key: "method", Value: req.Method},
		{Key: "request_id", Value: req.ID},
		{Key: "instance_id", Value: mcpserver.GetInstanceID()},
	}

	if toolName != "" {
		fields = append(fields, mcpserver.Field{Key: "tool", Value: toolName})
	}

	return fields
}

// buildResponseLogFields builds log fields for response logging (shared by all transports)
func buildResponseLogFields(resp *mcpserver.Response, toolName string) []mcpserver.Field {
	fields := []mcpserver.Field{
		{Key: "request_id", Value: resp.ID},
		{Key: "has_error", Value: resp.Error != nil},
	}

	if toolName != "" {
		fields = append(fields, mcpserver.Field{Key: "tool", Value: toolName})
	}

	if resp.Error != nil {
		fields = append(fields,
			mcpserver.Field{Key: "error_code", Value: resp.Error.Code},
			mcpserver.Field{Key: "error_message", Value: resp.Error.Message},
		)
	}

	return fields
}
