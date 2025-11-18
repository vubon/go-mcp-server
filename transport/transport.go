package transport

import (
	"context"

	"github.com/vubon/go-mcp-server"
)

// Transport is the interface that all MCP transports must implement
type Transport interface {
	// Run starts the transport and blocks until the context is cancelled
	// For HTTP transport, this may not be applicable (use ServeHTTP instead)
	Run(ctx context.Context) error
}

// ServerTransport is a transport that wraps an MCP server
type ServerTransport interface {
	Transport
	// GetServer returns the underlying MCP server
	GetServer() *mcpserver.Server
}
