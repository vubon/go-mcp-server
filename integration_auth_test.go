package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vubon/go-mcp-server/auth"
)

func TestAuthorizationPassThrough_EndToEnd(t *testing.T) {
	// Create a mock backend server that checks for Authorization header
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "No authorization"})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
			"status":        "success",
		})
	}))
	defer mockBackend.Close()

	// Create tool and handler config
	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test tool",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"test": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	// Generate handler
	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	// Test with authorization in context
	ctx := context.Background()
	ctx = auth.WithAuthorization(ctx, "Bearer test-token-123")

	result, err := handler(ctx, map[string]interface{}{"test": "value"})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["authorization"] != "Bearer test-token-123" {
		t.Errorf("Expected authorization header to be passed through, got %v", resultMap["authorization"])
	}
}

func TestAuthorizationPassThrough_ServiceLevelConfig(t *testing.T) {
	// Create a mock backend server
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		// No handler-level auth config - should use service-level
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
		Authorization: &AuthorizationConfig{
			Strategy: StrategyPassThrough,
		},
	}

	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	ctx := context.Background()
	ctx = auth.WithAuthorization(ctx, "Bearer service-level-token")

	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["authorization"] != "Bearer service-level-token" {
		t.Errorf("Expected service-level auth to be used, got %v", resultMap["authorization"])
	}
}

func TestAuthorization_TransformStrategy(t *testing.T) {
	// Create a mock backend that expects X-API-Key header
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"apiKey": apiKey,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		Authorization: &AuthorizationConfig{
			Strategy:   StrategyTransform,
			HeaderName: "X-API-Key",
			Transform: &TransformConfig{
				FromPrefix: "Bearer",
				ToPrefix:   "ApiKey",
			},
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	ctx := context.Background()
	ctx = auth.WithAuthorization(ctx, "Bearer token123")

	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["apiKey"] != "ApiKey token123" {
		t.Errorf("Expected transformed auth header, got %v", resultMap["apiKey"])
	}
}

func TestAuthorization_StaticStrategy(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		Authorization: &AuthorizationConfig{
			Strategy:    StrategyStatic,
			StaticValue: "Bearer static-token-456",
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	// Context without auth - should still work with static
	ctx := context.Background()
	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["authorization"] != "Bearer static-token-456" {
		t.Errorf("Expected static token, got %v", resultMap["authorization"])
	}
}

func TestAuthorization_TransformEmptyPrefix(t *testing.T) {
	// Create a mock backend that expects just the token (no prefix)
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"apiKey": apiKey,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		Authorization: &AuthorizationConfig{
			Strategy:   StrategyTransform,
			HeaderName: "X-API-Key",
			Transform: &TransformConfig{
				FromPrefix: "Bearer",
				ToPrefix:   "", // Empty = just the token, no prefix
			},
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	ctx := context.Background()
	ctx = auth.WithAuthorization(ctx, "Bearer token123")

	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	// Should be just "token123" without any prefix
	if resultMap["apiKey"] != "token123" {
		t.Errorf("Expected just token without prefix, got %v", resultMap["apiKey"])
	}
}

func TestAuthorization_NoneStrategy(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		Authorization: &AuthorizationConfig{
			Strategy: StrategyNone,
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	// Context with auth - should be ignored
	ctx := context.Background()
	ctx = auth.WithAuthorization(ctx, "Bearer should-be-ignored")

	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	// Authorization should be empty string (not set)
	if authVal, ok := resultMap["authorization"].(string); ok && authVal != "" {
		t.Errorf("Expected no authorization header, got %v", authVal)
	}
}
