package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/vubon/go-mcp-server"
	"github.com/vubon/go-mcp-server/transport"
)

func main() {
	// Create server
	server := mcpserver.New(&mcpserver.Config{
		Name:    "greeter",
		Version: "1.0.0",
	})

	// Register a simple greet tool
	server.RegisterTool("greet", mcpserver.Tool{
		Name:        "greet",
		Description: "Greets a person by name",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the person to greet",
				},
			},
			"required": []string{"name"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, ok := args["name"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid parameter: name must be a string")
		}
		return map[string]string{
			"greeting": "Hello, " + name + "!",
		}, nil
	})

	// Create HTTP transport
	httpHandler := transport.NewHTTP(server)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc", httpHandler.ServeHTTP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := "8080"
	log.Printf("🚀 MCP Greeter Server starting on port %s", port)
	log.Printf("📡 JSON-RPC endpoint: http://localhost:%s/jsonrpc", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

