package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
		Name:    "greeter-stdio",
		Version: "1.0.0",
		Logger:  logger,
	})

	// Register a simple greet tool
	greetTool := mcpserver.Tool{
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
	}
	server.RegisterTool("greet", &greetTool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, ok := args["name"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid parameter: name must be a string")
		}
		return map[string]string{
			"greeting": "Hello, " + name + "!",
		}, nil
	})

	// Create stdio transport
	stdioTransport := transport.NewStdio(server)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		if logger := server.GetLogger(); logger != nil {
			logger.Info("Shutting down...")
		}
		cancel()
	}()

	// Run the transport (blocks until context is cancelled)
	if logger := server.GetLogger(); logger != nil {
		logger.Info("MCP Greeter Server (stdio) starting",
			mcpserver.F("transport", "stdio"),
		)
	}
	if err := stdioTransport.Run(ctx); err != nil && err != context.Canceled {
		if logger := server.GetLogger(); logger != nil {
			logger.Error("Transport error",
				mcpserver.F("error", err.Error()),
			)
		}
		log.Fatalf("Transport error: %v", err)
	}
}
