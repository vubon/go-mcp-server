package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestAuthorization_BasicStrategy(t *testing.T) {
	// Create a mock backend server that checks for Basic Auth
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "No authorization"})
			return
		}

		// Verify it's Basic Auth format
		if !strings.HasPrefix(authHeader, "Basic ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not Basic Auth"})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
			"status":        "success",
		})
	}))
	defer mockBackend.Close()

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
		Authorization: &AuthorizationConfig{
			Strategy: StrategyBasic,
			BasicAuth: &BasicAuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	// Generate handler
	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	// Test with Basic Auth (no context auth needed)
	ctx := context.Background()
	result, err := handler(ctx, map[string]interface{}{"test": "value"})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	authHeader, ok := resultMap["authorization"].(string)
	if !ok {
		t.Fatal("Expected authorization header in response")
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Expected Basic Auth header, got %q", authHeader)
	}

	// Verify the encoded value is correct
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	if !strings.Contains(authHeader, expectedEncoded) {
		t.Errorf("Expected Basic Auth to contain encoded credentials, got %q", authHeader)
	}
}

func TestAuthorization_BasicStrategy_WithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_BASIC_USER", "envuser")
	os.Setenv("TEST_BASIC_PASS", "envpass")
	defer os.Unsetenv("TEST_BASIC_USER")
	defer os.Unsetenv("TEST_BASIC_PASS")

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test tool",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		Authorization: &AuthorizationConfig{
			Strategy: StrategyBasic,
			BasicAuth: &BasicAuthConfig{
				UsernameEnv: "TEST_BASIC_USER",
				PasswordEnv: "TEST_BASIC_PASS",
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
	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	authHeader, ok := resultMap["authorization"].(string)
	if !ok {
		t.Fatal("Expected authorization header in response")
	}

	expectedEncoded := base64.StdEncoding.EncodeToString([]byte("envuser:envpass"))
	if !strings.Contains(authHeader, expectedEncoded) {
		t.Errorf("Expected Basic Auth with env credentials, got %q", authHeader)
	}
}

func TestAuthorization_BasicStrategy_HeaderExtraction(t *testing.T) {
	// Create a mock backend server that checks for Basic Auth header
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Basic Auth Authorization header
		authHeader := r.Header.Get("Authorization")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
			"status":        "success",
		})
	}))
	defer mockBackend.Close()

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
		Authorization: &AuthorizationConfig{
			Strategy: StrategyBasic,
			BasicAuth: &BasicAuthConfig{
				UsernameHeader: "SECREAT_KEY",
				PasswordHeader: "PASSWORD",
			},
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	// Generate handler
	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	// Create context with request headers
	ctx := context.Background()
	requestHeaders := map[string]string{
		"SECREAT_KEY": "my-secret-value",
		"PASSWORD":    "my-password-value",
	}
	ctx = auth.WithRequestHeaders(ctx, requestHeaders)

	// Execute handler
	result, err := handler(ctx, map[string]interface{}{"test": "value"})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	// Verify Basic Auth header was set correctly
	authHeader, ok := resultMap["authorization"].(string)
	if !ok {
		t.Fatal("Expected authorization header in response")
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Expected Basic Auth header, got %q", authHeader)
	}

	// Verify the encoded value is correct
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte("my-secret-value:my-password-value"))
	if !strings.Contains(authHeader, expectedEncoded) {
		t.Errorf("Expected Basic Auth to contain encoded credentials, got %q", authHeader)
	}
}

func TestAuthorization_BasicStrategy_HeaderExtraction_PasswordNone(t *testing.T) {
	// Test with passwordHeader set to "None" - should only encode username
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": authHeader,
		})
	}))
	defer mockBackend.Close()

	tool := ToolFile{
		Name:        "test_tool",
		Description: "Test tool",
		ServiceName: "test-service",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "POST",
		Path:   "/test",
		Authorization: &AuthorizationConfig{
			Strategy: StrategyBasic,
			BasicAuth: &BasicAuthConfig{
				UsernameHeader: "SECREAT_KEY",
				PasswordHeader: "None",
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
	requestHeaders := map[string]string{
		"SECREAT_KEY": "my-secret-value",
	}
	ctx = auth.WithRequestHeaders(ctx, requestHeaders)

	result, err := handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	authHeader, ok := resultMap["authorization"].(string)
	if !ok {
		t.Fatal("Expected authorization header in response")
	}

	// Should be Basic Auth with only username (no password)
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte("my-secret-value"))
	if !strings.Contains(authHeader, expectedEncoded) {
		t.Errorf("Expected Basic Auth with only username, got %q", authHeader)
	}
	// Should not contain colon (no password)
	if strings.Contains(authHeader, base64.StdEncoding.EncodeToString([]byte("my-secret-value:"))) {
		t.Errorf("Expected no password in Basic Auth, got %q", authHeader)
	}
}

func TestIntegration_QueryParameters(t *testing.T) {
	// Create a mock backend server that captures query parameters
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture query parameters
		queryParams := r.URL.Query()

		// Capture request body
		var body map[string]interface{}
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&body)
		}

		// Respond with captured data
		response := map[string]interface{}{
			"queryParams": queryParams,
			"requestPath": r.URL.Path,
			"requestBody": body,
			"method":      r.Method,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockBackend.Close()

	// Create tool and handler config with query parameters
	tool := ToolFile{
		Name:        "getUsers",
		Description: "Get users with pagination",
		ServiceName: "user-service",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page":  map[string]interface{}{"type": "integer"},
				"limit": map[string]interface{}{"type": "integer"},
				"sort":  map[string]interface{}{"type": "string"},
				"data":  map[string]interface{}{"type": "string"}, // Should be in body
			},
		},
	}

	handlerConfig := HandlerConfig{
		Type:   "http",
		Method: "GET",
		Path:   "/api/v1/users",
		QueryParams: map[string]string{
			"page":   "{page}",
			"limit":  "{limit}",
			"sort":   "{sort}",
			"format": "json", // Static query param
		},
	}

	serviceConfig := ServiceConfig{
		BaseURL: mockBackend.URL,
	}

	// Generate handler
	handler, err := generateHTTPHandler(&tool, &handlerConfig, serviceConfig)
	if err != nil {
		t.Fatalf("Failed to generate handler: %v", err)
	}

	tests := []struct {
		name           string
		args           map[string]interface{}
		expectedParams map[string][]string
		expectedBody   map[string]interface{}
	}{
		{
			name: "All query parameters provided",
			args: map[string]interface{}{
				"page":  1,
				"limit": 10,
				"sort":  "name",
				"data":  "should-be-in-body",
			},
			expectedParams: map[string][]string{
				"page":   {"1"},
				"limit":  {"10"},
				"sort":   {"name"},
				"format": {"json"},
			},
			expectedBody: map[string]interface{}{
				"data": "should-be-in-body",
			},
		},
		{
			name: "Partial query parameters",
			args: map[string]interface{}{
				"page": 2,
				"data": "body-data",
			},
			expectedParams: map[string][]string{
				"page":   {"2"},
				"format": {"json"},
			},
			expectedBody: map[string]interface{}{
				"data": "body-data",
			},
		},
		{
			name: "URL encoding in query parameters",
			args: map[string]interface{}{
				"sort": "name&order=asc",
				"page": 1,
			},
			expectedParams: map[string][]string{
				"sort":   {"name&order=asc"},
				"page":   {"1"},
				"format": {"json"},
			},
			expectedBody: map[string]interface{}{},
		},
		{
			name: "Only static query parameters",
			args: map[string]interface{}{
				"data": "body-only",
			},
			expectedParams: map[string][]string{
				"format": {"json"},
			},
			expectedBody: map[string]interface{}{
				"data": "body-only",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := handler(ctx, tt.args)
			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("Expected map result, got %T", result)
			}

			// Verify query parameters
			queryParams, ok := resultMap["queryParams"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected queryParams in result, got %T", resultMap["queryParams"])
			}

			for key, expectedValues := range tt.expectedParams {
				actualValue, exists := queryParams[key]
				if !exists {
					t.Errorf("Expected query param %q, but it was not found", key)
					continue
				}

				// Convert to []string for comparison
				actualValues, ok2 := actualValue.([]interface{})
				if !ok2 {
					t.Errorf("Expected []interface{} for query param %q, got %T", key, actualValue)
					continue
				}

				if len(actualValues) != len(expectedValues) {
					t.Errorf("Query param %q: expected %d values, got %d", key, len(expectedValues), len(actualValues))
					continue
				}

				for i, expectedVal := range expectedValues {
					if actualValues[i] != expectedVal {
						t.Errorf("Query param %q[%d]: expected %q, got %v", key, i, expectedVal, actualValues[i])
					}
				}
			}

			// Verify request body (should not contain query params)
			body, ok := resultMap["requestBody"].(map[string]interface{})
			if !ok && len(tt.expectedBody) > 0 {
				t.Fatalf("Expected requestBody in result, got %T", resultMap["requestBody"])
			}

			if len(tt.expectedBody) > 0 {
				for key, expectedVal := range tt.expectedBody {
					if body[key] != expectedVal {
						t.Errorf("Body param %q: expected %v, got %v", key, expectedVal, body[key])
					}
				}

				// Verify query params are NOT in body
				for key := range tt.expectedParams {
					if _, exists := body[key]; exists {
						t.Errorf("Query param %q should not be in request body", key)
					}
				}
			}

			// Verify path
			if resultMap["requestPath"] != "/api/v1/users" {
				t.Errorf("Expected path /api/v1/users, got %v", resultMap["requestPath"])
			}
		})
	}
}
