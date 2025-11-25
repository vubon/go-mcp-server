package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	mcpserver "github.com/vubon/go-mcp-server"
)

// StdioTransport handles MCP communication over stdin/stdout
type StdioTransport struct {
	server *mcpserver.Server
	input  io.Reader
	output io.Writer
}

// NewStdio creates a new stdio transport
func NewStdio(server *mcpserver.Server) *StdioTransport {
	return &StdioTransport{
		server: server,
		input:  os.Stdin,
		output: os.Stdout,
	}
}

// NewStdioWithIO creates a new stdio transport with custom input/output
// Useful for testing
func NewStdioWithIO(server *mcpserver.Server, input io.Reader, output io.Writer) *StdioTransport {
	return &StdioTransport{
		server: server,
		input:  input,
		output: output,
	}
}

// GetServer returns the underlying MCP server
func (t *StdioTransport) GetServer() *mcpserver.Server {
	return t.server
}

// Run starts the stdio transport and processes requests from stdin
func (t *StdioTransport) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(t.input)

	// Use a custom split function to handle JSON-RPC messages
	// JSON-RPC messages are typically newline-delimited JSON
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					if err == io.EOF {
						return nil // Normal EOF
					}
					return fmt.Errorf("error reading from stdin: %w", err)
				}
				return nil // No more input
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue // Skip empty lines
			}

			// Parse JSON-RPC request
			var req mcpserver.Request
			if err := json.Unmarshal(line, &req); err != nil {
				// Send error response
				resp := &mcpserver.Response{
					JSONRPC: mcpserver.JSONRPCVersion,
					Error: &mcpserver.Error{
						Code:    -32700,
						Message: "Parse error",
						Data:    err.Error(),
					},
					ID: nil,
				}
				if err := t.sendResponse(resp); err != nil {
					if logger := t.server.GetLogger(); logger != nil {
						logger.Error("Error sending parse error response",
							mcpserver.Field{Key: "error", Value: err.Error()},
						)
					}
				}
				continue
			}

			// Extract tool name once (avoid parsing JSON twice)
			toolName := extractToolName(&req)

			// Log request
			logRequest(t.server, &req, toolName)

			// Handle request
			resp := t.server.HandleRequest(ctx, &req)

			// Notifications don't require a response
			if resp == nil {
				continue
			}

			// Log response
			logResponse(t.server, resp, toolName)

			// Send response
			if err := t.sendResponse(resp); err != nil {
				if logger := t.server.GetLogger(); logger != nil {
					logger.Error("Error sending response",
						mcpserver.Field{Key: "error", Value: err.Error()},
					)
				}
				return err
			}
		}
	}
}

// sendResponse sends a JSON-RPC response to stdout
func (t *StdioTransport) sendResponse(resp *mcpserver.Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("error marshaling response: %w", err)
	}

	// Write response followed by newline (newline-delimited JSON)
	data = append(data, '\n')
	if _, err := t.output.Write(data); err != nil {
		return fmt.Errorf("error writing response: %w", err)
	}

	return nil
}
