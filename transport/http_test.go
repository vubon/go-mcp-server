package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vubon/go-mcp-server"
	"github.com/vubon/go-mcp-server/auth"
)

func TestNewHTTP(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)
	if transport == nil {
		t.Fatal("NewHTTP returned nil")
	}
	if transport.server != server {
		t.Error("Transport server does not match")
	}
}

func TestHTTPTransport_GetServer(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)
	if transport.GetServer() != server {
		t.Error("GetServer returned wrong server")
	}
}

func TestHTTPTransport_Run(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)
	ctx := context.Background()
	
	err := transport.Run(ctx)
	if err == nil {
		t.Error("Expected Run to return an error for HTTP transport")
	}
	if err.Error() != "HTTP transport does not support Run(), use ServeHTTP() instead" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestHTTPTransport_ServeHTTP_InvalidMethod(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	tests := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range tests {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/jsonrpc", nil)
			w := httptest.NewRecorder()

			transport.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var resp mcpserver.Response
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Error == nil {
				t.Error("Expected error response")
			}
			if resp.Error.Code != -32600 {
				t.Errorf("Expected error code -32600, got %d", resp.Error.Code)
			}
			if resp.Error.Message != "Method not allowed" {
				t.Errorf("Expected error message 'Method not allowed', got %s", resp.Error.Message)
			}
		})
	}
}

func TestHTTPTransport_ServeHTTP_InvalidJSON(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp mcpserver.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error response")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Expected error code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Expected error message 'Invalid Request', got %s", resp.Error.Message)
	}
}

func TestHTTPTransport_ServeHTTP_ValidRequest(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	// Create a valid JSON-RPC request
	request := mcpserver.Request{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp mcpserver.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %s", resp.JSONRPC)
	}
	// ID is interface{}, JSON unmarshals numbers as float64
	if resp.ID != float64(1) && resp.ID != 1 {
		t.Errorf("Expected ID 1, got %v (type %T)", resp.ID, resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}

func TestHTTPTransport_ServeHTTP_WithAuthorization(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	// Create a valid JSON-RPC request
	request := mcpserver.Request{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token-123")
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the authorization was passed to context
	// We can't directly test this, but we can verify the request was handled
	var resp mcpserver.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}

func TestHTTPTransport_ServeHTTP_Notification(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	// Create a notification request (no ID)
	request := mcpserver.Request{
		JSONRPC: "2.0",
		Method:  "initialized",
		ID:      nil,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Notifications should return empty body
	if w.Body.Len() != 0 {
		t.Errorf("Expected empty body for notification, got %d bytes", w.Body.Len())
	}
}

func TestHTTPTransport_ServeHTTP_ErrorResponse(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	// Create a request with invalid method
	request := mcpserver.Request{
		JSONRPC: "2.0",
		Method:  "invalid/method",
		ID:      1,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp mcpserver.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error response")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestHTTPTransport_ServeHTTP_ContentType(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	transport := NewHTTP(server)

	request := mcpserver.Request{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, -32600, "Test Error", "test data", 123)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}

	var resp mcpserver.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %s", resp.JSONRPC)
	}
	if resp.Error == nil {
		t.Error("Expected error in response")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Expected error code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Test Error" {
		t.Errorf("Expected error message 'Test Error', got %s", resp.Error.Message)
	}
	if resp.Error.Data != "test data" {
		t.Errorf("Expected error data 'test data', got %s", resp.Error.Data)
	}
	// ID is interface{}, JSON unmarshals numbers as float64
	if resp.ID != float64(123) && resp.ID != 123 {
		t.Errorf("Expected ID 123, got %v (type %T)", resp.ID, resp.ID)
	}
}

func TestHTTPTransport_AuthorizationContext(t *testing.T) {
	server := mcpserver.New(&mcpserver.Config{
		Name:    "test",
		Version: "1.0.0",
	})

	// Register a tool that checks for authorization in context
	server.RegisterTool("test_auth", mcpserver.Tool{
		Name:        "test_auth",
		Description: "Test auth",
		InputSchema: map[string]interface{}{"type": "object"},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		auth, ok := auth.AuthorizationFromContext(ctx)
		if !ok {
			return map[string]interface{}{"auth": "not found"}, nil
		}
		return map[string]interface{}{"auth": auth}, nil
	})

	transport := NewHTTP(server)

	request := mcpserver.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      1,
		Params: json.RawMessage(`{"name": "test_auth", "arguments": {}}`),
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token-456")
	w := httptest.NewRecorder()

	transport.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp mcpserver.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Verify the authorization was passed through
	// The response should contain the auth token
	resultBytes, _ := json.Marshal(resp.Result)
	if !bytes.Contains(resultBytes, []byte("test-token-456")) {
		t.Errorf("Expected authorization token in response, got: %s", string(resultBytes))
	}
}

