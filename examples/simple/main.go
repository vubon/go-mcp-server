package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/vubon/go-mcp-server"
	"github.com/vubon/go-mcp-server/transport"
)

func main() {
	// Create structured logger
	logger := mcpserver.NewLogger(&mcpserver.LoggerConfig{
		Level:  "info",
		Format: "json",
	})

	// Create server with logger
	server := mcpserver.New(&mcpserver.Config{
		Name:    "greeter",
		Version: "1.0.0",
		Logger:  logger,
	})

	// Register tools from JSON/YAML files
	// This automatically generates HTTP handlers based on handlers.json configuration
	err := server.RegisterToolsFromFiles("tools.json", "handlers.json")
	if err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Log configuration version info
	configInfo := server.GetConfigInfo()
	if logger := server.GetLogger(); logger != nil {
		logger.Info("Configuration loaded",
			mcpserver.F("version", configInfo["version"]),
			mcpserver.F("hash", configInfo["hash"]),
			mcpserver.F("tool_count", configInfo["toolCount"]),
		)
	}

	// Alternative: Manual tool registration (commented out)
	// server.RegisterTool("greet", mcpserver.Tool{
	// 	Name:        "greet",
	// 	Description: "Greets a person by name",
	// 	InputSchema: map[string]interface{}{
	// 		"type": "object",
	// 		"properties": map[string]interface{}{
	// 			"name": map[string]interface{}{
	// 				"type":        "string",
	// 				"description": "Name of the person to greet",
	// 			},
	// 		},
	// 		"required": []string{"name"},
	// 	},
	// }, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// 	name, ok := args["name"].(string)
	// 	if !ok {
	// 		return nil, fmt.Errorf("invalid parameter: name must be a string")
	// 	}
	// 	return map[string]string{
	// 		"greeting": "Hello, " + name + "!",
	// 	}, nil
	// })

	// Create HTTP transport
	httpHandler := transport.NewHTTP(server)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc", httpHandler.ServeHTTP)
	
	// Health check endpoint with configuration info
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		configInfo := server.GetConfigInfo()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"config": configInfo,
		})
	})

	port := "8080"
	if logger := server.GetLogger(); logger != nil {
		logger.Info("MCP Greeter Server starting",
			mcpserver.F("port", port),
			mcpserver.F("endpoint", "http://localhost:"+port+"/jsonrpc"),
		)
	}
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
