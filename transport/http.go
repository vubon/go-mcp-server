package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/vubon/go-mcp-server"
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
func (t *HTTPTransport) Run(ctx context.Context) error {
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
	log.Printf("📥 JSON-RPC request: method=%s, id=%v", req.Method, req.ID)

	// Extract Authorization header and add to context
	ctx := r.Context()
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = auth.WithAuthorization(ctx, authHeader)
	}

	// Handle request with enhanced context
	resp := t.server.HandleRequest(ctx, &req)

	// Notifications (like "initialized") don't require a response
	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Log response
	log.Printf("📤 JSON-RPC response: id=%v, error=%v", resp.ID, resp.Error != nil)

	// Send response
	w.Header().Set("Content-Type", mcpserver.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}
