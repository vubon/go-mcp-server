package mcpserver

import (
	"context"
	"net/url"
	"reflect"
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
		name            string
		path            string
		args            map[string]interface{}
		expectedPath    string
		expectedRemoved map[string]interface{}
	}{
		{
			name:            "Single parameter",
			path:            "/api/users/{userId}",
			args:            map[string]interface{}{"userId": "123"},
			expectedPath:    "/api/users/123",
			expectedRemoved: map[string]interface{}{"userId": "123"},
		},
		{
			name:            "Multiple parameters",
			path:            "/api/users/{userId}/posts/{postId}",
			args:            map[string]interface{}{"userId": "123", "postId": "456"},
			expectedPath:    "/api/users/123/posts/456",
			expectedRemoved: map[string]interface{}{"userId": "123", "postId": "456"},
		},
		{
			name:            "Integer parameter",
			path:            "/api/users/{userId}",
			args:            map[string]interface{}{"userId": 123},
			expectedPath:    "/api/users/123",
			expectedRemoved: map[string]interface{}{"userId": 123},
		},
		{
			name:            "Float parameter",
			path:            "/api/values/{value}",
			args:            map[string]interface{}{"value": 3.14},
			expectedPath:    "/api/values/3.14",
			expectedRemoved: map[string]interface{}{"value": 3.14},
		},
		{
			name:            "Parameter not in args",
			path:            "/api/users/{userId}",
			args:            map[string]interface{}{"other": "value"},
			expectedPath:    "/api/users/{userId}",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "No parameters",
			path:            "/api/users",
			args:            map[string]interface{}{"userId": "123"},
			expectedPath:    "/api/users",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Empty args",
			path:            "/api/users/{userId}",
			args:            map[string]interface{}{},
			expectedPath:    "/api/users/{userId}",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Nil args",
			path:            "/api/users/{userId}",
			args:            nil,
			expectedPath:    "/api/users/{userId}",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Unclosed brace",
			path:            "/api/users/{userId",
			args:            map[string]interface{}{"userId": "123"},
			expectedPath:    "/api/users/{userId",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Adjacent parameters",
			path:            "/api/{a}{b}",
			args:            map[string]interface{}{"a": "x", "b": "y"},
			expectedPath:    "/api/xy",
			expectedRemoved: map[string]interface{}{"a": "x", "b": "y"},
		},
		{
			name:            "Parameter at start",
			path:            "{userId}/posts",
			args:            map[string]interface{}{"userId": "123"},
			expectedPath:    "123/posts",
			expectedRemoved: map[string]interface{}{"userId": "123"},
		},
		{
			name:            "Parameter at end",
			path:            "/api/users/{userId}",
			args:            map[string]interface{}{"userId": "123"},
			expectedPath:    "/api/users/123",
			expectedRemoved: map[string]interface{}{"userId": "123"},
		},
		{
			name:            "Mixed found and not found",
			path:            "/api/{found}/items/{notFound}",
			args:            map[string]interface{}{"found": "value"},
			expectedPath:    "/api/value/items/{notFound}",
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
			name:          "Service config without header name",
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

func TestSubstituteQueryParams(t *testing.T) {
	tests := []struct {
		name            string
		queryParams     map[string]string
		args            map[string]interface{}
		expectedQuery   string
		expectedRemoved map[string]interface{}
	}{
		{
			name:            "Empty query params",
			queryParams:     nil,
			args:            map[string]interface{}{"page": 1},
			expectedQuery:   "",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Empty query params map",
			queryParams:     map[string]string{},
			args:            map[string]interface{}{"page": 1},
			expectedQuery:   "",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Single dynamic parameter",
			queryParams:     map[string]string{"page": "{page}"},
			args:            map[string]interface{}{"page": 1},
			expectedQuery:   "?page=1",
			expectedRemoved: map[string]interface{}{"page": 1},
		},
		{
			name: "Multiple dynamic parameters",
			queryParams: map[string]string{
				"page":  "{page}",
				"limit": "{limit}",
			},
			args:            map[string]interface{}{"page": 1, "limit": 10},
			expectedQuery:   "?limit=10&page=1", // Order may vary
			expectedRemoved: map[string]interface{}{"page": 1, "limit": 10},
		},
		{
			name: "Mixed static and dynamic",
			queryParams: map[string]string{
				"page":   "{page}",
				"format": "json", // Static
			},
			args:            map[string]interface{}{"page": 1},
			expectedQuery:   "?format=json&page=1", // Order may vary
			expectedRemoved: map[string]interface{}{"page": 1},
		},
		{
			name:            "URL encoding",
			queryParams:     map[string]string{"q": "{query}"},
			args:            map[string]interface{}{"query": "hello world"},
			expectedQuery:   "?q=hello+world",
			expectedRemoved: map[string]interface{}{"query": "hello world"},
		},
		{
			name:            "URL encoding special characters",
			queryParams:     map[string]string{"q": "{query}"},
			args:            map[string]interface{}{"query": "hello&world=test"},
			expectedQuery:   "?q=hello%26world%3Dtest",
			expectedRemoved: map[string]interface{}{"query": "hello&world=test"},
		},
		{
			name:            "Missing parameter",
			queryParams:     map[string]string{"page": "{page}"},
			args:            map[string]interface{}{"other": "value"},
			expectedQuery:   "",
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "String value",
			queryParams:     map[string]string{"name": "{name}"},
			args:            map[string]interface{}{"name": "John"},
			expectedQuery:   "?name=John",
			expectedRemoved: map[string]interface{}{"name": "John"},
		},
		{
			name:            "Integer value",
			queryParams:     map[string]string{"id": "{id}"},
			args:            map[string]interface{}{"id": 123},
			expectedQuery:   "?id=123",
			expectedRemoved: map[string]interface{}{"id": 123},
		},
		{
			name:            "Float value",
			queryParams:     map[string]string{"price": "{price}"},
			args:            map[string]interface{}{"price": 99.99},
			expectedQuery:   "?price=99.99",
			expectedRemoved: map[string]interface{}{"price": 99.99},
		},
		{
			name:            "Boolean value",
			queryParams:     map[string]string{"active": "{active}"},
			args:            map[string]interface{}{"active": true},
			expectedQuery:   "?active=true",
			expectedRemoved: map[string]interface{}{"active": true},
		},
		{
			name: "All static values",
			queryParams: map[string]string{
				"format":  "json",
				"version": "v1",
			},
			args:            map[string]interface{}{},
			expectedQuery:   "?format=json&version=v1", // Order may vary
			expectedRemoved: map[string]interface{}{},
		},
		{
			name:            "Empty string value",
			queryParams:     map[string]string{"filter": "{filter}"},
			args:            map[string]interface{}{"filter": ""},
			expectedQuery:   "?filter=",
			expectedRemoved: map[string]interface{}{"filter": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, removed := substituteQueryParams(tt.queryParams, tt.args)

			// Parse expected and actual query strings to compare (order may vary)
			if tt.expectedQuery != "" {
				expectedParams, _ := url.ParseQuery(tt.expectedQuery[1:]) // Remove "?"
				actualParams, _ := url.ParseQuery(query[1:])              // Remove "?"

				if !reflect.DeepEqual(expectedParams, actualParams) {
					t.Errorf("Query params mismatch.\nExpected: %v\nGot: %v", expectedParams, actualParams)
				}
			} else if query != "" {
				t.Errorf("Expected empty query string, got %q", query)
			}

			// Check removed parameters
			if !reflect.DeepEqual(removed, tt.expectedRemoved) {
				t.Errorf("Removed params mismatch.\nExpected: %v\nGot: %v", tt.expectedRemoved, removed)
			}
		})
	}
}
