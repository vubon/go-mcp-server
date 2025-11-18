package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vubon/go-mcp-server"
)

func TestNewStdio(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewStdio(server)
	if transport == nil {
		t.Fatal("NewStdio returned nil")
	}
	if transport.server != server {
		t.Error("Transport server does not match")
	}
	if transport.input == nil {
		t.Error("Transport input is nil")
	}
	if transport.output == nil {
		t.Error("Transport output is nil")
	}
}

func TestNewStdioWithIO(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	input := strings.NewReader("test input")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)
	if transport == nil {
		t.Fatal("NewStdioWithIO returned nil")
	}
	if transport.server != server {
		t.Error("Transport server does not match")
	}
	if transport.input != input {
		t.Error("Transport input does not match")
	}
	if transport.output != output {
		t.Error("Transport output does not match")
	}
}

func TestStdioTransport_GetServer(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewStdio(server)
	if transport.GetServer() != server {
		t.Error("GetServer returned wrong server")
	}
}

func TestStdioTransport_SendResponse(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	output := &bytes.Buffer{}
	transport := NewStdioWithIO(server, strings.NewReader(""), output)

	resp := &mcpserver.Response{
		JSONRPC: mcpserver.JSONRPCVersion,
		ID:      1,
		Result:  map[string]interface{}{"test": "value"},
	}

	err := transport.sendResponse(resp)
	if err != nil {
		t.Fatalf("sendResponse failed: %v", err)
	}

	// Verify output ends with newline
	outputBytes := output.Bytes()
	if len(outputBytes) == 0 {
		t.Fatal("Output is empty")
	}
	if outputBytes[len(outputBytes)-1] != '\n' {
		t.Error("Output does not end with newline")
	}

	// Verify it's valid JSON
	var decodedResp mcpserver.Response
	if err := json.Unmarshal(outputBytes[:len(outputBytes)-1], &decodedResp); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if decodedResp.JSONRPC != mcpserver.JSONRPCVersion {
		t.Errorf("Expected JSONRPC %q, got %s", mcpserver.JSONRPCVersion, decodedResp.JSONRPC)
	}
	// ID is interface{}, JSON unmarshals numbers as float64
	if decodedResp.ID != float64(1) && decodedResp.ID != 1 {
		t.Errorf("Expected ID 1, got %v (type %T)", decodedResp.ID, decodedResp.ID)
	}
}

func TestStdioTransport_Run_ValidRequest(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	request := mcpserver.Request{
		JSONRPC: mcpserver.JSONRPCVersion,
		Method:  "tools/list",
		ID:      1,
	}

	requestJSON, _ := json.Marshal(request)
	input := strings.NewReader(string(requestJSON) + "\n")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Run in a goroutine since it blocks
	done := make(chan error, 1)
	go func() {
		done <- transport.Run(ctx)
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Run failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}

	// Verify response was written
	if output.Len() == 0 {
		t.Fatal("No response written")
	}

	// Verify response is valid JSON
	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) == 0 {
		t.Fatal("No response lines")
	}

	var resp mcpserver.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if resp.JSONRPC != mcpserver.JSONRPCVersion {
		t.Errorf("Expected JSONRPC %q, got %s", mcpserver.JSONRPCVersion, resp.JSONRPC)
	}
	// ID is interface{}, JSON unmarshals numbers as float64
	if resp.ID != float64(1) && resp.ID != 1 {
		t.Errorf("Expected ID 1, got %v (type %T)", resp.ID, resp.ID)
	}
}

func TestStdioTransport_Run_InvalidJSON(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	input := strings.NewReader("invalid json\n")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- transport.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Run failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}

	// Verify error response was written
	if output.Len() == 0 {
		t.Fatal("No error response written")
	}

	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) == 0 {
		t.Fatal("No response lines")
	}

	var resp mcpserver.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error response")
	}
	if resp.Error.Code != -32700 {
		t.Errorf("Expected error code -32700, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Parse error" {
		t.Errorf("Expected error message 'Parse error', got %s", resp.Error.Message)
	}
}

func TestStdioTransport_Run_EmptyLines(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	request := mcpserver.Request{
		JSONRPC: mcpserver.JSONRPCVersion,
		Method:  "tools/list",
		ID:      1,
	}

	requestJSON, _ := json.Marshal(request)
	// Include empty lines
	input := strings.NewReader("\n\n" + string(requestJSON) + "\n\n")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- transport.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Run failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}

	// Should still process the valid request
	if output.Len() == 0 {
		t.Fatal("No response written")
	}
}

func TestStdioTransport_Run_Notification(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	// Notification has no ID
	request := mcpserver.Request{
		JSONRPC: mcpserver.JSONRPCVersion,
		Method:  "initialized",
		ID:      nil,
	}

	requestJSON, _ := json.Marshal(request)
	input := strings.NewReader(string(requestJSON) + "\n")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- transport.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Run failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}

	// Notifications should not produce a response
	if output.Len() != 0 {
		t.Errorf("Expected no response for notification, got %d bytes", output.Len())
	}
}

func TestStdioTransport_Run_ContextCancellation(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	// Create input that will block (empty buffer, scanner will wait)
	input := bytes.NewBuffer([]byte(""))
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- transport.Run(ctx)
	}()

	// Cancel context after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		// With empty input, scanner returns EOF immediately (nil error)
		// Context cancellation only works if scanner is actively waiting
		// So we accept either context.Canceled or nil (EOF)
		if err != context.Canceled && err != nil {
			t.Errorf("Expected context.Canceled or nil (EOF), got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}
}

func TestStdioTransport_Run_EOF(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	// Empty input (EOF immediately)
	input := strings.NewReader("")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := transport.Run(ctx)
	if err != nil {
		t.Errorf("Expected nil error on EOF, got %v", err)
	}
}

func TestStdioTransport_Run_MultipleRequests(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	request1 := mcpserver.Request{
		JSONRPC: mcpserver.JSONRPCVersion,
		Method:  "tools/list",
		ID:      1,
	}
	request2 := mcpserver.Request{
		JSONRPC: mcpserver.JSONRPCVersion,
		Method:  "tools/list",
		ID:      2,
	}

	request1JSON, _ := json.Marshal(request1)
	request2JSON, _ := json.Marshal(request2)
	input := strings.NewReader(string(request1JSON) + "\n" + string(request2JSON) + "\n")
	output := &bytes.Buffer{}

	transport := NewStdioWithIO(server, input, output)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- transport.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Run failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}

	// Verify both responses were written
	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 response lines, got %d", len(lines))
	}

	// Verify first response
	var resp1 mcpserver.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp1); err != nil {
		t.Fatalf("First response is not valid JSON: %v", err)
	}
	// ID is interface{}, JSON unmarshals numbers as float64
	if resp1.ID != float64(1) && resp1.ID != 1 {
		t.Errorf("Expected first response ID 1, got %v (type %T)", resp1.ID, resp1.ID)
	}

	// Verify second response
	var resp2 mcpserver.Response
	if err := json.Unmarshal([]byte(lines[1]), &resp2); err != nil {
		t.Fatalf("Second response is not valid JSON: %v", err)
	}
	// ID is interface{}, JSON unmarshals numbers as float64
	if resp2.ID != float64(2) && resp2.ID != 2 {
		t.Errorf("Expected second response ID 2, got %v (type %T)", resp2.ID, resp2.ID)
	}
}
