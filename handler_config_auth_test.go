package mcpserver

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
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
	authHeader := testBearerToken
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
	authHeader := testBearerToken
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
		Strategy:    StrategyStatic,
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
		Strategy:       "static",
		StaticValueEnv: "TEST_AUTH_TOKEN",
	}

	result = extractAuthorization(ctx, config, nil)
	if result != "Bearer env-token" {
		t.Errorf("Expected env token, got %q", result)
	}
}

func TestExtractAuthorization_None(t *testing.T) {
	ctx := context.Background()
	authHeader := testBearerToken
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
	authHeader := testBearerToken
	ctx = auth.WithAuthorization(ctx, authHeader)

	// No config - should default to pass-through
	result := extractAuthorization(ctx, nil, nil)
	if result != authHeader {
		t.Errorf("Expected %q, got %q", authHeader, result)
	}
}

func TestExtractAuthorization_ServiceConfig(t *testing.T) {
	ctx := context.Background()
	authHeader := testBearerToken
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
	authHeader := testBearerToken
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
	authHeader := testBearerToken
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

func TestBuildBasicAuth(t *testing.T) {
	tests := []struct {
		name      string
		basicAuth *BasicAuthConfig
		expected  string
		expectErr bool
	}{
		{
			name: "Username and password",
			basicAuth: &BasicAuthConfig{
				Username: "user",
				Password: "pass",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name: "Username and password from env",
			basicAuth: &BasicAuthConfig{
				UsernameEnv: "TEST_USER",
				PasswordEnv: "TEST_PASS",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("envuser:envpass")),
		},
		{
			name: "Pre-encoded value",
			basicAuth: &BasicAuthConfig{
				EncodedValue: base64.StdEncoding.EncodeToString([]byte("user:pass")),
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name: "Pre-encoded value with Basic prefix",
			basicAuth: &BasicAuthConfig{
				EncodedValue: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name: "Pre-encoded value from env",
			basicAuth: &BasicAuthConfig{
				EncodedValueEnv: "TEST_BASIC_AUTH",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("envuser:envpass")),
		},
		{
			name: "Encoded value takes precedence over username/password",
			basicAuth: &BasicAuthConfig{
				EncodedValue: base64.StdEncoding.EncodeToString([]byte("encoded:value")),
				Username:     "user",
				Password:     "pass",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("encoded:value")),
		},
		{
			name: "Missing username",
			basicAuth: &BasicAuthConfig{
				Password: "pass",
			},
			expected: "",
		},
		{
			name: "Missing password",
			basicAuth: &BasicAuthConfig{
				Username: "user",
			},
			expected: "",
		},
		{
			name:      "Nil config",
			basicAuth: nil,
			expected:  "",
		},
		{
			name:      "Empty config",
			basicAuth: &BasicAuthConfig{},
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables if needed
			if tt.basicAuth != nil {
				if tt.basicAuth.UsernameEnv != "" {
					os.Setenv(tt.basicAuth.UsernameEnv, "envuser")
					defer os.Unsetenv(tt.basicAuth.UsernameEnv)
				}
				if tt.basicAuth.PasswordEnv != "" {
					os.Setenv(tt.basicAuth.PasswordEnv, "envpass")
					defer os.Unsetenv(tt.basicAuth.PasswordEnv)
				}
				if tt.basicAuth.EncodedValueEnv != "" {
					encoded := base64.StdEncoding.EncodeToString([]byte("envuser:envpass"))
					os.Setenv(tt.basicAuth.EncodedValueEnv, encoded)
					defer os.Unsetenv(tt.basicAuth.EncodedValueEnv)
				}
			}

			ctx := context.Background()
			result := buildBasicAuth(ctx, tt.basicAuth)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}

			// Verify the format is correct if we got a result
			if result != "" {
				if !strings.HasPrefix(result, "Basic ") {
					t.Errorf("Expected result to start with 'Basic ', got %q", result)
				}
				// Verify it's valid base64
				encodedPart := strings.TrimPrefix(result, "Basic ")
				decoded, err := base64.StdEncoding.DecodeString(encodedPart)
				if err != nil {
					t.Errorf("Expected valid base64, got error: %v", err)
				}
				if !strings.Contains(string(decoded), ":") {
					t.Errorf("Expected decoded value to contain ':', got %q", string(decoded))
				}
			}
		})
	}
}

func TestExtractAuthorization_Basic(t *testing.T) {
	ctx := context.Background()

	// Test with username and password
	config := &AuthorizationConfig{
		Strategy: StrategyBasic,
		BasicAuth: &BasicAuthConfig{
			Username: "testuser",
			Password: "testpass",
		},
	}

	result := extractAuthorization(ctx, config, nil)
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test with environment variables
	os.Setenv("BASIC_USER", "envuser")
	os.Setenv("BASIC_PASS", "envpass")
	defer os.Unsetenv("BASIC_USER")
	defer os.Unsetenv("BASIC_PASS")

	config = &AuthorizationConfig{
		Strategy: StrategyBasic,
		BasicAuth: &BasicAuthConfig{
			UsernameEnv: "BASIC_USER",
			PasswordEnv: "BASIC_PASS",
		},
	}

	result = extractAuthorization(ctx, config, nil)
	expected = "Basic " + base64.StdEncoding.EncodeToString([]byte("envuser:envpass"))
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test with pre-encoded value
	encodedValue := base64.StdEncoding.EncodeToString([]byte("preencoded:value"))
	config = &AuthorizationConfig{
		Strategy: StrategyBasic,
		BasicAuth: &BasicAuthConfig{
			EncodedValue: encodedValue,
		},
	}

	result = extractAuthorization(ctx, config, nil)
	expected = "Basic " + encodedValue
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test that Basic Auth doesn't require context auth
	config = &AuthorizationConfig{
		Strategy: StrategyBasic,
		BasicAuth: &BasicAuthConfig{
			Username: "user",
			Password: "pass",
		},
	}

	result = extractAuthorization(ctx, config, nil)
	if result == "" {
		t.Error("Expected Basic Auth to work without context auth")
	}
}

func TestExtractBasicAuthFromHeaders(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		basicAuth *BasicAuthConfig
		expected  string
	}{
		{
			name: "Extract username header and encode as Basic Auth",
			ctx: func() context.Context {
				ctx := context.Background()
				headers := map[string]string{
					"SECREAT_KEY": "secret-value",
				}
				return auth.WithRequestHeaders(ctx, headers)
			}(),
			basicAuth: &BasicAuthConfig{
				UsernameHeader: "SECREAT_KEY",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("secret-value")),
		},
		{
			name: "Extract username and password headers and encode as Basic Auth",
			ctx: func() context.Context {
				ctx := context.Background()
				headers := map[string]string{
					"SECREAT_KEY": "secret-value",
					"PASSWORD":    "password-value",
				}
				return auth.WithRequestHeaders(ctx, headers)
			}(),
			basicAuth: &BasicAuthConfig{
				UsernameHeader: "SECREAT_KEY",
				PasswordHeader: "PASSWORD",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("secret-value:password-value")),
		},
		{
			name: "Password header None - only username",
			ctx: func() context.Context {
				ctx := context.Background()
				headers := map[string]string{
					"SECREAT_KEY": "secret-value",
				}
				return auth.WithRequestHeaders(ctx, headers)
			}(),
			basicAuth: &BasicAuthConfig{
				UsernameHeader: "SECREAT_KEY",
				PasswordHeader: "None",
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("secret-value")),
		},
		{
			name: "Case insensitive header matching",
			ctx: func() context.Context {
				ctx := context.Background()
				// Test case-insensitive matching - header stored as "Secreat-Key" but looked up as "Secreat-Key" (same format, different case)
				headers := map[string]string{
					"Secreat-Key": "secret-value",
				}
				return auth.WithRequestHeaders(ctx, headers)
			}(),
			basicAuth: &BasicAuthConfig{
				UsernameHeader: "secreat-key", // Different case, same format
			},
			expected: "Basic " + base64.StdEncoding.EncodeToString([]byte("secret-value")),
		},
		{
			name:      "No usernameHeader specified",
			ctx:       context.Background(),
			basicAuth: &BasicAuthConfig{},
			expected:  "",
		},
		{
			name: "Header not found in request",
			ctx: func() context.Context {
				ctx := context.Background()
				headers := map[string]string{
					"Other-Header": "value",
				}
				return auth.WithRequestHeaders(ctx, headers)
			}(),
			basicAuth: &BasicAuthConfig{
				UsernameHeader: "SECREAT_KEY",
			},
			expected: "",
		},
		{
			name:      "Nil basicAuth",
			ctx:       context.Background(),
			basicAuth: nil,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBasicAuthFromHeaders(tt.ctx, tt.basicAuth)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
			// Verify the format is correct if we got a result
			if result != "" {
				if !strings.HasPrefix(result, "Basic ") {
					t.Errorf("Expected result to start with 'Basic ', got %q", result)
				}
				// Verify it's valid base64
				encodedPart := strings.TrimPrefix(result, "Basic ")
				decoded, err := base64.StdEncoding.DecodeString(encodedPart)
				if err != nil {
					t.Errorf("Expected valid base64, got error: %v", err)
				}
				// For username:password format, verify it contains the expected values
				if strings.Contains(tt.expected, ":") {
					if !strings.Contains(string(decoded), ":") {
						t.Errorf("Expected decoded value to contain ':', got %q", string(decoded))
					}
				}
			}
		})
	}
}
