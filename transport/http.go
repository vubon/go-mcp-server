package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	mcpserver "github.com/vubon/go-mcp-server"
	"github.com/vubon/go-mcp-server/auth"
)

// HTTPTransport handles HTTP requests for MCP server
type HTTPTransport struct {
	server *mcpserver.Server
}

// GetServer returns the underlying MCP server
func (t *HTTPTransport) GetServer() *mcpserver.Server {
	return t.server
}

// Run is not applicable for HTTP transport (use ServeHTTP instead)
// This implements the Transport interface but returns an error
func (t *HTTPTransport) Run(_ context.Context) error {
	return fmt.Errorf("HTTP transport does not support Run(), use ServeHTTP() instead")
}

// NewHTTP creates a new HTTP transport
func NewHTTP(server *mcpserver.Server) *HTTPTransport {
	return &HTTPTransport{
		server: server,
	}
}

// ServeHTTP implements http.Handler
func (t *HTTPTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, -32600, "Method not allowed", "", nil)
		return
	}

	var req mcpserver.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, -32600, "Invalid Request", err.Error(), nil)
		return
	}

	// Log request (excluding sensitive data)
	if logger := t.server.GetLogger(); logger != nil {
		fields := []mcpserver.Field{
			{Key: "method", Value: req.Method},
			{Key: "request_id", Value: req.ID},
			{Key: "instance_id", Value: mcpserver.GetInstanceID()},
		}
		logger.Info("JSON-RPC request received", fields...)
	}

	// Extract Authorization header and add to context
	ctx := r.Context()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = auth.WithAuthorization(ctx, authHeader)
	}

	// Extract all request headers and add to context (for Basic Auth header extraction)
	// Store headers with their original case for flexible matching
	requestHeaders := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			// Store with original key name (preserve case for flexible matching)
			requestHeaders[key] = values[0]
		}
	}
	if len(requestHeaders) > 0 {
		ctx = auth.WithRequestHeaders(ctx, requestHeaders)
	}

	// Handle request with enhanced context
	resp := t.server.HandleRequest(ctx, &req)

	// Notifications (like "initialized") don't require a response
	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Log response
	if logger := t.server.GetLogger(); logger != nil {
		fields := []mcpserver.Field{
			{Key: "request_id", Value: resp.ID},
			{Key: "has_error", Value: resp.Error != nil},
		}
		if resp.Error != nil {
			fields = append(fields,
				mcpserver.Field{Key: "error_code", Value: resp.Error.Code},
				mcpserver.Field{Key: "error_message", Value: resp.Error.Message},
			)
		}
		logger.Info("JSON-RPC response sent", fields...)
	}

	// Send response
	w.Header().Set("Content-Type", mcpserver.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		if logger := t.server.GetLogger(); logger != nil {
			logger.Error("Error encoding response",
				mcpserver.Field{Key: "error", Value: err.Error()},
			)
		}
	}
}

func respondError(w http.ResponseWriter, code int, message, data string, id interface{}) {
	resp := mcpserver.Response{
		JSONRPC: mcpserver.JSONRPCVersion,
		Error: &mcpserver.Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
	w.Header().Set("Content-Type", mcpserver.ContentTypeJSON)
	w.WriteHeader(http.StatusOK) // JSON-RPC uses 200 OK even for errors
	// Ignore encoding errors in error handler (rare case, can't log it)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Error encoding error response - nothing we can do
		_ = err
	}
}
