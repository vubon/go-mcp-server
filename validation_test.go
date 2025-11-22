package mcpserver

import (
	"strings"
	"testing"
)

const testToolName = "getUser"

func TestValidateConfiguration(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				Description: "Get user by ID",
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"userId": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/v1/users/{userId}",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if !result.Valid() {
			t.Errorf("Expected valid configuration, got errors: %v", result.Errors)
		}
	})

	t.Run("Missing handler", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		if len(result.Errors) == 0 {
			t.Error("Expected at least one error")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "handler not found") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'handler not found' error, got: %v", result.Errors)
		}
	})

	t.Run("Missing service", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "missing-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Service == "missing-service" && strings.Contains(err.Message, "not found") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'service not found' error, got: %v", result.Errors)
		}
	})

	t.Run("Path parameter not in InputSchema", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/v1/users/{userId}",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "path parameters") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected path parameter error, got: %v", result.Errors)
		}
	})

	t.Run("Query parameter not in InputSchema", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        "searchUsers",
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				"searchUsers": {
					Type:   "http",
					Method: "GET",
					Path:   "/api/v1/users",
					QueryParams: map[string]string{
						"q": "{searchQuery}",
					},
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == "searchUsers" && strings.Contains(err.Message, "query parameters") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected query parameter error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid HTTP method", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "INVALID",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "invalid HTTP method") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid HTTP method error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid baseURL", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "invalid-url",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Service == "user-service" && strings.Contains(err.Message, "URL") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid URL error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid timeout format", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:    "http",
					Method:  "GET",
					Path:    "/api/users",
					Timeout: "invalid",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "timeout") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid timeout error, got: %v", result.Errors)
		}
	})

	t.Run("Missing InputSchema", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "InputSchema is required") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected InputSchema required error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid InputSchema type", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "invalid-type",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "invalid schema type") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid schema type error, got: %v", result.Errors)
		}
	})

	t.Run("Orphaned handler", func(t *testing.T) {
		tools := []ToolFile{}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers: map[string]HandlerConfig{
				"orphanedHandler": {
					Type:   "http",
					Method: "GET",
					Path:   "/api/test",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == "orphanedHandler" && strings.Contains(err.Message, "handler has no corresponding tool") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected orphaned handler error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid authorization strategy", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
					Authorization: &AuthorizationConfig{
						Strategy: "invalid-strategy",
					},
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "invalid authorization strategy") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid authorization strategy error, got: %v", result.Errors)
		}
	})

	t.Run("Missing service name", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "service name is required") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected service name required error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid path format", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "GET",
					Path:   "api/users", // Missing leading /
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if err.Tool == testToolName && strings.Contains(err.Message, "path must start with") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid path format error, got: %v", result.Errors)
		}
	})

	t.Run("Invalid tool name", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        "invalid tool name!",
				ServiceName: "user-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "https://api.example.com",
				},
			},
			Handlers: map[string]HandlerConfig{
				"invalid tool name!": {
					Type:   "http",
					Method: "GET",
					Path:   "/api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		found := false
		for _, err := range result.Errors {
			if strings.Contains(err.Message, "not a valid identifier") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid identifier error, got: %v", result.Errors)
		}
	})

	t.Run("Multiple errors", func(t *testing.T) {
		tools := []ToolFile{
			{
				Name:        testToolName,
				ServiceName: "missing-service",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
			{
				Name:        "createUser",
				ServiceName: "user-service",
				// Missing InputSchema
			},
		}

		handlers := &HandlersConfig{
			ServiceConfig: map[string]ServiceConfig{
				"user-service": {
					BaseURL: "invalid-url",
				},
			},
			Handlers: map[string]HandlerConfig{
				testToolName: {
					Type:   "http",
					Method: "INVALID",
					Path:   "api/users",
				},
			},
		}

		result := ValidateConfiguration(tools, handlers)
		if result.Valid() {
			t.Error("Expected validation to fail")
		}
		if len(result.Errors) < 3 {
			t.Errorf("Expected multiple errors, got %d: %v", len(result.Errors), result.Errors)
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("Error with tool", func(t *testing.T) {
		err := ValidationError{
			Tool:    testToolName,
			Message: "test error",
		}
		expected := `tool "getUser": test error`
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error with service", func(t *testing.T) {
		err := ValidationError{
			Service: "user-service",
			Message: "test error",
		}
		expected := `service "user-service": test error`
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without tool or service", func(t *testing.T) {
		err := ValidationError{
			Message: "test error",
		}
		if err.Error() != "test error" {
			t.Errorf("Expected %q, got %q", "test error", err.Error())
		}
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("Valid result", func(t *testing.T) {
		result := &ValidationResult{}
		if !result.Valid() {
			t.Error("Expected valid result")
		}
		if result.Error() != "validation passed" {
			t.Errorf("Expected 'validation passed', got %q", result.Error())
		}
	})

	t.Run("Invalid result", func(t *testing.T) {
		result := &ValidationResult{}
		result.AddError(ValidationError{
			Tool:    testToolName,
			Message: "handler not found",
		})
		result.AddError(ValidationError{
			Service: "user-service",
			Message: "baseURL is required",
		})

		if result.Valid() {
			t.Error("Expected invalid result")
		}

		errMsg := result.Error()
		if !strings.Contains(errMsg, "handler not found") {
			t.Errorf("Expected error message to contain 'handler not found', got %q", errMsg)
		}
		if !strings.Contains(errMsg, "baseURL is required") {
			t.Errorf("Expected error message to contain 'baseURL is required', got %q", errMsg)
		}
	})
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "Single parameter",
			path:     "/api/v1/users/{userId}",
			expected: []string{"userId"},
		},
		{
			name:     "Multiple parameters",
			path:     "/api/v1/users/{userId}/posts/{postId}",
			expected: []string{"userId", "postId"},
		},
		{
			name:     "No parameters",
			path:     "/api/v1/users",
			expected: []string{},
		},
		{
			name:     "Nested parameters",
			path:     "/api/{version}/users/{userId}",
			expected: []string{"version", "userId"},
		},
		{
			name:     "Empty path",
			path:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPathParams(tt.path)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parameters, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Expected parameter %d to be %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid: alphanumeric", "getUser123", true},
		{"Valid: with underscore", "get_user", true},
		{"Valid: with hyphen", "get-user", true},
		{"Valid: mixed", "get-User_123", true},
		{"Invalid: empty", "", false},
		{"Invalid: with space", "get user", false},
		{"Invalid: with special chars", "get@user", false},
		{"Invalid: with dot", "get.user", false},
		{"Valid: single char", "a", true},
		{"Valid: numbers only", "123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIdentifier(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidHTTPMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"Valid: GET", "GET", true},
		{"Valid: POST", "POST", true},
		{"Valid: PUT", "PUT", true},
		{"Valid: PATCH", "PATCH", true},
		{"Valid: DELETE", "DELETE", true},
		{"Valid: HEAD", "HEAD", true},
		{"Valid: OPTIONS", "OPTIONS", true},
		{"Valid: lowercase", "get", true},
		{"Valid: mixed case", "Get", true},
		{"Invalid: INVALID", "INVALID", false},
		{"Invalid: empty", "", false},
		{"Invalid: TRACE", "TRACE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidHTTPMethod(tt.method)
			if result != tt.expected {
				t.Errorf("isValidHTTPMethod(%q) = %v, expected %v", tt.method, result, tt.expected)
			}
		})
	}
}

func TestIsValidHeaderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid: standard header", "Authorization", true},
		{"Valid: with hyphen", "X-API-Key", true},
		{"Valid: with underscore", "X_API_Key", true},
		{"Valid: lowercase", "authorization", true},
		{"Invalid: empty", "", false},
		{"Invalid: with space", "X API Key", false},
		{"Invalid: with special chars", "X@API", false},
		{"Valid: numbers", "X-API-123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidHeaderName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidHeaderName(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"Valid: https", "https://api.example.com", false},
		{"Valid: http", "http://api.example.com", false},
		{"Valid: with path", "https://api.example.com/v1", false},
		{"Valid: with port", "http://localhost:8080", false},
		{"Invalid: no scheme", "api.example.com", true},
		{"Invalid: invalid scheme", "ftp://api.example.com", true},
		{"Invalid: no host", "https://", true},
		{"Invalid: empty", "", true},
		{"Invalid: malformed", "://api.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"Valid: simple path", "/api/users", false},
		{"Valid: with parameter", "/api/users/{userId}", false},
		{"Valid: multiple parameters", "/api/users/{userId}/posts/{postId}", false},
		{"Invalid: no leading slash", "api/users", true},
		{"Invalid: empty", "", true},
		{"Invalid: duplicate parameter", "/api/users/{userId}/{userId}", true},
		{"Valid: nested path", "/api/v1/users/{userId}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInputSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid: object type",
			schema: map[string]interface{}{
				"type": "object",
			},
			wantErr: false,
		},
		{
			name: "Valid: with properties",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid: with required",
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"userId"},
			},
			wantErr: false,
		},
		{
			name: "Invalid: missing type",
			schema: map[string]interface{}{
				"properties": map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Invalid: invalid type",
			schema: map[string]interface{}{
				"type": "invalid",
			},
			wantErr: true,
		},
		{
			name: "Invalid: property without type",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]interface{}{},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid: property not object",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": "string",
				},
			},
			wantErr: true,
		},
		{
			name: "Valid: array type",
			schema: map[string]interface{}{
				"type": "array",
			},
			wantErr: false,
		},
		{
			name: "Valid: string type",
			schema: map[string]interface{}{
				"type": "string",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInputSchema(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInputSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAuthorizationConfig(t *testing.T) {
	t.Run("Valid: pass-through", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyPassThrough,
		}
		if err := validateAuthorizationConfig(config); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Valid: static with value", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy:    StrategyStatic,
			StaticValue: "Bearer token",
		}
		if err := validateAuthorizationConfig(config); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Valid: static with env", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy:       StrategyStatic,
			StaticValueEnv: "TOKEN_ENV",
		}
		if err := validateAuthorizationConfig(config); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Valid: transform", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyTransform,
			Transform: &TransformConfig{
				FromPrefix: "Bearer",
				ToPrefix:   "ApiKey",
			},
		}
		if err := validateAuthorizationConfig(config); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Valid: basic with username", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyBasic,
			BasicAuth: &BasicAuthConfig{
				Username: "user",
				Password: "pass",
			},
		}
		if err := validateAuthorizationConfig(config); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Valid: basic with header", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyBasic,
			BasicAuth: &BasicAuthConfig{
				UsernameHeader: "X-Username",
				PasswordHeader: "X-Password",
			},
		}
		if err := validateAuthorizationConfig(config); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Invalid: unknown strategy", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: "invalid",
		}
		if err := validateAuthorizationConfig(config); err == nil {
			t.Error("Expected error for invalid strategy")
		}
	})

	t.Run("Invalid: transform without config", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyTransform,
		}
		if err := validateAuthorizationConfig(config); err == nil {
			t.Error("Expected error for transform without config")
		}
	})

	t.Run("Invalid: static without value", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyStatic,
		}
		if err := validateAuthorizationConfig(config); err == nil {
			t.Error("Expected error for static without value")
		}
	})

	t.Run("Invalid: basic without config", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy: StrategyBasic,
		}
		if err := validateAuthorizationConfig(config); err == nil {
			t.Error("Expected error for basic without config")
		}
	})

	t.Run("Invalid: basic without credentials", func(t *testing.T) {
		config := &AuthorizationConfig{
			Strategy:  StrategyBasic,
			BasicAuth: &BasicAuthConfig{},
		}
		if err := validateAuthorizationConfig(config); err == nil {
			t.Error("Expected error for basic without credentials")
		}
	})
}

func TestValidatePathParameters(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		inputSchema map[string]interface{}
		wantErr     bool
	}{
		{
			name: "Valid: parameter exists",
			path: "/api/users/{userId}",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid: parameter missing",
			path: "/api/users/{userId}",
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name:        "Valid: no parameters",
			path:        "/api/users",
			inputSchema: nil,
			wantErr:     false,
		},
		{
			name: "Invalid: no properties",
			path: "/api/users/{userId}",
			inputSchema: map[string]interface{}{
				"type": "object",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathParameters(tt.path, tt.inputSchema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePathParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateQueryParameters(t *testing.T) {
	tests := []struct {
		name        string
		queryParams map[string]string
		inputSchema map[string]interface{}
		wantErr     bool
	}{
		{
			name: "Valid: parameter exists",
			queryParams: map[string]string{
				"q": "{searchQuery}",
			},
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"searchQuery": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid: parameter missing",
			queryParams: map[string]string{
				"q": "{searchQuery}",
			},
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Valid: static query param",
			queryParams: map[string]string{
				"api_key": "static-value",
			},
			inputSchema: map[string]interface{}{
				"type": "object",
			},
			wantErr: false,
		},
		{
			name:        "Valid: no query params",
			queryParams: nil,
			inputSchema: nil,
			wantErr:     false,
		},
		{
			name: "Invalid: no properties",
			queryParams: map[string]string{
				"q": "{searchQuery}",
			},
			inputSchema: map[string]interface{}{
				"type": "object",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryParameters(tt.queryParams, tt.inputSchema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateQueryParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
