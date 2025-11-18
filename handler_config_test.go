package mcpserver

import (
	"context"
	"testing"

	"github.com/vubon/go-mcp-server/auth"
)

func TestValueToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "String value",
			input:    "test",
			expected: "test",
		},
		{
			name:     "Integer value",
			input:    123,
			expected: "123",
		},
		{
			name:     "Int64 value",
			input:    int64(456),
			expected: "456",
		},
		{
			name:     "Float64 value",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "Float32 value",
			input:    float32(2.5),
			expected: "2.5",
		},
		{
			name:     "Boolean true",
			input:    true,
			expected: "true",
		},
		{
			name:     "Boolean false",
			input:    false,
			expected: "false",
		},
		{
			name:     "Uint value",
			input:    uint(789),
			expected: "789",
		},
		{
			name:     "Slice value",
			input:    []int{1, 2, 3},
			expected: "[1 2 3]",
		},
		{
			name:     "Map value",
			input:    map[string]int{"a": 1},
			expected: "map[a:1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueToString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSubstitutePathParams(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		args           map[string]interface{}
		expectedPath   string
		expectedRemoved map[string]interface{}
	}{
		{
			name:           "Single parameter",
			path:           "/api/users/{userId}",
			args:           map[string]interface{}{"userId": "123"},
			expectedPath:   "/api/users/123",
			expectedRemoved: map[string]interface{}{"userId": "123"},
		},
		{
			name:           "Multiple parameters",
			path:           "/api/users/{userId}/posts/{postId}",
			args:           map[string]interface{}{"userId": "123", "postId": "456"},
			expectedPath:   "/api/users/123/posts/456",
			expectedRemoved: map[string]interface{}{"userId": "123", "postId": "456"},
		},
		{
			name:           "Integer parameter",
			path:           "/api/users/{userId}",
			args:           map[string]interface{}{"userId": 123},
			expectedPath:   "/api/users/123",
			expectedRemoved: map[string]interface{}{"userId": 123},
		},
		{
			name:           "Float parameter",
			path:           "/api/values/{value}",
			args:           map[string]interface{}{"value": 3.14},
			expectedPath:   "/api/values/3.14",
			expectedRemoved: map[string]interface{}{"value": 3.14},
		},
		{
			name:           "Parameter not in args",
			path:           "/api/users/{userId}",
			args:           map[string]interface{}{"other": "value"},
			expectedPath:   "/api/users/{userId}",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:           "No parameters",
			path:           "/api/users",
			args:           map[string]interface{}{"userId": "123"},
			expectedPath:   "/api/users",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:           "Empty args",
			path:           "/api/users/{userId}",
			args:           map[string]interface{}{},
			expectedPath:   "/api/users/{userId}",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:           "Nil args",
			path:           "/api/users/{userId}",
			args:           nil,
			expectedPath:   "/api/users/{userId}",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:           "Unclosed brace",
			path:           "/api/users/{userId",
			args:           map[string]interface{}{"userId": "123"},
			expectedPath:   "/api/users/{userId",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:           "Adjacent parameters",
			path:           "/api/{a}{b}",
			args:           map[string]interface{}{"a": "x", "b": "y"},
			expectedPath:   "/api/xy",
			expectedRemoved: map[string]interface{}{"a": "x", "b": "y"},
		},
		{
			name:           "Parameter at start",
			path:           "{userId}/posts",
			args:           map[string]interface{}{"userId": "123"},
			expectedPath:   "123/posts",
			expectedRemoved: map[string]interface{}{"userId": "123"},
		},
		{
			name:           "Parameter at end",
			path:           "/api/users/{userId}",
			args:           map[string]interface{}{"userId": "123"},
			expectedPath:   "/api/users/123",
			expectedRemoved: map[string]interface{}{"userId": "123"},
		},
		{
			name:           "Mixed found and not found",
			path:           "/api/{found}/items/{notFound}",
			args:           map[string]interface{}{"found": "value"},
			expectedPath:   "/api/value/items/{notFound}",
			expectedRemoved: map[string]interface{}{"found": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultPath, resultRemoved := substitutePathParams(tt.path, tt.args)
			
			if resultPath != tt.expectedPath {
				t.Errorf("Path: Expected %q, got %q", tt.expectedPath, resultPath)
			}
			
			if len(resultRemoved) != len(tt.expectedRemoved) {
				t.Errorf("Removed: Expected %d items, got %d", len(tt.expectedRemoved), len(resultRemoved))
			}
			
			for k, v := range tt.expectedRemoved {
				if resultRemoved[k] != v {
					t.Errorf("Removed[%s]: Expected %v, got %v", k, v, resultRemoved[k])
				}
			}
		})
	}
}

func TestGetAuthHeaderName(t *testing.T) {
	tests := []struct {
		name          string
		handlerConfig *AuthorizationConfig
		serviceConfig *AuthorizationConfig
		expected      string
	}{
		{
			name: "Handler config with header name",
			handlerConfig: &AuthorizationConfig{
				HeaderName: "X-API-Key",
			},
			serviceConfig: &AuthorizationConfig{
				HeaderName: "Authorization",
			},
			expected: "X-API-Key",
		},
		{
			name:          "Service config with header name",
			handlerConfig: nil,
			serviceConfig: &AuthorizationConfig{
				HeaderName: "X-API-Key",
			},
			expected: "X-API-Key",
		},
		{
			name:          "No config - default",
			handlerConfig: nil,
			serviceConfig: nil,
			expected:      DefaultAuthHeaderName,
		},
		{
			name: "Handler config without header name falls back to service",
			handlerConfig: &AuthorizationConfig{
				Strategy: StrategyPassThrough,
			},
			serviceConfig: &AuthorizationConfig{
				HeaderName: "X-API-Key",
			},
			expected: "X-API-Key",
		},
		{
			name: "Service config without header name",
			handlerConfig: nil,
			serviceConfig: &AuthorizationConfig{
				Strategy: StrategyPassThrough,
			},
			expected: DefaultAuthHeaderName,
		},
		{
			name: "Handler config empty header name",
			handlerConfig: &AuthorizationConfig{
				HeaderName: "",
			},
			serviceConfig: &AuthorizationConfig{
				HeaderName: "X-API-Key",
			},
			expected: "X-API-Key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAuthHeaderName(tt.handlerConfig, tt.serviceConfig)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetAuthFromContext(t *testing.T) {
	t.Run("With authorization in context", func(t *testing.T) {
		ctx := context.Background()
		ctx = auth.WithAuthorization(ctx, "Bearer test-token")
		
		result, ok := getAuthFromContext(ctx)
		if !ok {
			t.Error("Expected authorization to be found")
		}
		if result != "Bearer test-token" {
			t.Errorf("Expected %q, got %q", "Bearer test-token", result)
		}
	})
	
	t.Run("Without authorization in context", func(t *testing.T) {
		ctx := context.Background()
		
		_, ok := getAuthFromContext(ctx)
		if ok {
			t.Error("Expected authorization not to be found")
		}
	})
}

