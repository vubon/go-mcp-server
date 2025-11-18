package mcpserver

import (
	"context"
	"os"
	"testing"

	"github.com/vubon/go-mcp-server/auth"
)

func TestTransformAuthorization(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		transform *TransformConfig
		expected  string
	}{
		{
			name:      "Bearer to ApiKey",
			header:    "Bearer token123",
			transform: &TransformConfig{FromPrefix: "Bearer", ToPrefix: "ApiKey"},
			expected:  "ApiKey token123",
		},
		{
			name:      "No transformation when prefix doesn't match",
			header:    "Basic token123",
			transform: &TransformConfig{FromPrefix: "Bearer", ToPrefix: "ApiKey"},
			expected:  "Basic token123",
		},
		{
			name:      "Nil transform returns original",
			header:    "Bearer token123",
			transform: nil,
			expected:  "Bearer token123",
		},
		{
			name:      "Empty transform returns original",
			header:    "Bearer token123",
			transform: &TransformConfig{},
			expected:  "Bearer token123",
		},
		{
			name:      "Empty ToPrefix removes prefix (just token)",
			header:    "Bearer token123",
			transform: &TransformConfig{FromPrefix: "Bearer", ToPrefix: ""},
			expected:  "token123",
		},
		{
			name:      "Empty ToPrefix with non-Bearer prefix",
			header:    "Basic user:pass",
			transform: &TransformConfig{FromPrefix: "Basic", ToPrefix: ""},
			expected:  "user:pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformAuthorization(tt.header, tt.transform)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractAuthorization_PassThrough(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	config := &AuthorizationConfig{
		Strategy: StrategyPassThrough,
	}

	result := extractAuthorization(ctx, config, nil)
	if result != authHeader {
		t.Errorf("Expected %q, got %q", authHeader, result)
	}
}

func TestExtractAuthorization_Transform(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	config := &AuthorizationConfig{
		Strategy: StrategyTransform,
		Transform: &TransformConfig{
			FromPrefix: "Bearer",
			ToPrefix:   "ApiKey",
		},
	}

	result := extractAuthorization(ctx, config, nil)
	expected := "ApiKey test-token"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExtractAuthorization_Static(t *testing.T) {
	ctx := context.Background()

	// Test static value
	config := &AuthorizationConfig{
		Strategy:   StrategyStatic,
		StaticValue: "Bearer static-token",
	}

	result := extractAuthorization(ctx, config, nil)
	if result != "Bearer static-token" {
		t.Errorf("Expected static token, got %q", result)
	}

	// Test static value from environment
	os.Setenv("TEST_AUTH_TOKEN", "Bearer env-token")
	defer os.Unsetenv("TEST_AUTH_TOKEN")

	config = &AuthorizationConfig{
		Strategy:      "static",
		StaticValueEnv: "TEST_AUTH_TOKEN",
	}

	result = extractAuthorization(ctx, config, nil)
	if result != "Bearer env-token" {
		t.Errorf("Expected env token, got %q", result)
	}
}

func TestExtractAuthorization_None(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	config := &AuthorizationConfig{
		Strategy: StrategyNone,
	}

	result := extractAuthorization(ctx, config, nil)
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestExtractAuthorization_DefaultPassThrough(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	// No config - should default to pass-through
	result := extractAuthorization(ctx, nil, nil)
	if result != authHeader {
		t.Errorf("Expected %q, got %q", authHeader, result)
	}
}

func TestExtractAuthorization_ServiceConfig(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	// Service config with pass-through
	serviceConfig := &AuthorizationConfig{
		Strategy: StrategyPassThrough,
	}

	// No handler config - should use service config
	result := extractAuthorization(ctx, nil, serviceConfig)
	if result != authHeader {
		t.Errorf("Expected %q, got %q", authHeader, result)
	}
}

func TestExtractAuthorization_HandlerOverridesService(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	serviceConfig := &AuthorizationConfig{
		Strategy: StrategyPassThrough,
	}

	handlerConfig := &AuthorizationConfig{
		Strategy: StrategyNone,
	}

	// Handler config should override service config
	result := extractAuthorization(ctx, handlerConfig, serviceConfig)
	if result != "" {
		t.Errorf("Expected empty string (none strategy), got %q", result)
	}
}

func TestExtractAuthorization_UnknownStrategy(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token"
	ctx = auth.WithAuthorization(ctx, authHeader)

	config := &AuthorizationConfig{
		Strategy: "unknown-strategy",
	}

	// Unknown strategy should fallback to pass-through
	result := extractAuthorization(ctx, config, nil)
	if result != authHeader {
		t.Errorf("Expected fallback to pass-through %q, got %q", authHeader, result)
	}
}

func TestExtractAuthorization_NoAuthInContext(t *testing.T) {
	ctx := context.Background()

	config := &AuthorizationConfig{
		Strategy: StrategyPassThrough,
	}

	result := extractAuthorization(ctx, config, nil)
	if result != "" {
		t.Errorf("Expected empty string when no auth in context, got %q", result)
	}
}

