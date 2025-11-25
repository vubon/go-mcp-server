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

	// Extract tool name once (avoid parsing JSON twice)
	toolName := extractToolName(&req)

	logRequest(t.server, &req, toolName)
	ctx := t.buildContext(r)
	resp := t.server.HandleRequest(ctx, &req)

	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	logResponse(t.server, resp, toolName)
	t.sendResponse(w, resp)
}

// buildContext builds the request context with authorization and headers
func (t *HTTPTransport) buildContext(r *http.Request) context.Context {
	ctx := r.Context()

	// Add authorization header if present
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = auth.WithAuthorization(ctx, authHeader)
	}

	// Extract all request headers for flexible header matching
	// Store headers with their original case for flexible matching
	requestHeaders := t.extractHeaders(r)
	if len(requestHeaders) > 0 {
		ctx = auth.WithRequestHeaders(ctx, requestHeaders)
	}

	return ctx
}

// extractHeaders extracts all headers from the request, preserving case
func (t *HTTPTransport) extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// sendResponse sends the JSON-RPC response to the client
func (t *HTTPTransport) sendResponse(w http.ResponseWriter, resp *mcpserver.Response) {
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
