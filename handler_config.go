package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vubon/go-mcp-server/auth"
	"gopkg.in/yaml.v3"
)

// TransformConfig represents configuration for authorization header transformation
type TransformConfig struct {
	// Transform Bearer token to different format
	FromPrefix string `json:"fromPrefix,omitempty" yaml:"fromPrefix,omitempty"` // e.g., "Bearer"
	ToPrefix   string `json:"toPrefix,omitempty" yaml:"toPrefix,omitempty"`     // e.g., "ApiKey"
	
	// Or custom transformation function name
	Function string `json:"function,omitempty" yaml:"function,omitempty"`
}

// AuthorizationConfig represents authorization configuration for handlers or services
type AuthorizationConfig struct {
	// Strategy: "pass-through", "transform", "static", "none"
	// - "pass-through": use incoming Authorization header as-is
	// - "transform": transform the header (e.g., Bearer -> API-Key)
	// - "static": use static value from config
	// - "none": don't add Authorization header
	Strategy string `json:"strategy" yaml:"strategy"`
	
	// Header name to use (default: "Authorization")
	HeaderName string `json:"headerName,omitempty" yaml:"headerName,omitempty"`
	
	// For "transform" strategy: transformation rules
	Transform *TransformConfig `json:"transform,omitempty" yaml:"transform,omitempty"`
	
	// For "static" strategy: static value
	StaticValue string `json:"staticValue,omitempty" yaml:"staticValue,omitempty"`
	
	// Environment variable for static value
	StaticValueEnv string `json:"staticValueEnv,omitempty" yaml:"staticValueEnv,omitempty"`
}

// ServiceConfig represents configuration for a service
type ServiceConfig struct {
	BaseURL       string                 `json:"baseURL" yaml:"baseURL"`
	Timeout       string                 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty" yaml:"headers,omitempty"`
	Authorization *AuthorizationConfig   `json:"authorization,omitempty" yaml:"authorization,omitempty"`
}

// HandlerConfig represents configuration for a handler
type HandlerConfig struct {
	Type          string                 `json:"type" yaml:"type"`
	Method        string                 `json:"method" yaml:"method"`
	Path          string                 `json:"path" yaml:"path"`
	Headers       map[string]string      `json:"headers,omitempty" yaml:"headers,omitempty"`
	Timeout       string                 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Authorization *AuthorizationConfig   `json:"authorization,omitempty" yaml:"authorization,omitempty"`
}

// HandlersConfig represents the complete handlers configuration
type HandlersConfig struct {
	ServiceConfig map[string]ServiceConfig `json:"serviceConfig" yaml:"serviceConfig"`
	Handlers      map[string]HandlerConfig `json:"handlers" yaml:"handlers"`
}

// ToolFile represents a tool definition from file (with capitalized fields)
type ToolFile struct {
	Name        string                 `json:"Name" yaml:"Name"`
	Description string                 `json:"Description" yaml:"Description"`
	ServiceName string                 `json:"ServiceName,omitempty" yaml:"ServiceName,omitempty"`
	APIVersion  string                 `json:"APIVersion,omitempty" yaml:"APIVersion,omitempty"`
	Endpoint    string                 `json:"Endpoint,omitempty" yaml:"Endpoint,omitempty"`
	InputSchema map[string]interface{} `json:"InputSchema" yaml:"InputSchema"`
}

// ToTool converts ToolFile to Tool (with lowercase JSON tags for API compatibility)
func (tf *ToolFile) ToTool() Tool {
	return Tool{
		Name:        tf.Name,
		Description: tf.Description,
		ServiceName: tf.ServiceName,
		APIVersion:  tf.APIVersion,
		Endpoint:    tf.Endpoint,
		InputSchema: tf.InputSchema,
	}
}

// parseToolsFromJSON parses tools from JSON bytes
func parseToolsFromJSON(data []byte) ([]ToolFile, error) {
	var tools []ToolFile
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}
	return tools, nil
}

// parseToolsFromYAML parses tools from YAML bytes
func parseToolsFromYAML(data []byte) ([]ToolFile, error) {
	var tools []ToolFile
	if err := yaml.Unmarshal(data, &tools); err != nil {
		return nil, err
	}
	return tools, nil
}

// parseHandlersFromJSON parses handlers configuration from JSON bytes
func parseHandlersFromJSON(data []byte) (*HandlersConfig, error) {
	var config HandlersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// parseHandlersFromYAML parses handlers configuration from YAML bytes
func parseHandlersFromYAML(data []byte) (*HandlersConfig, error) {
	var config HandlersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// substituteEnvVars replaces ${VAR} patterns with environment variable values
func substituteEnvVars(s string) string {
	// os.ExpandEnv supports $VAR and ${VAR} formats
	return os.ExpandEnv(s)
}

// substituteHeaders replaces environment variables in header values
func substituteHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range headers {
		result[k] = substituteEnvVars(v)
	}
	return result
}

// mergeHeaders merges service headers with handler headers (handler overrides service)
func mergeHeaders(serviceHeaders, handlerHeaders map[string]string) map[string]string {
	result := make(map[string]string)
	
	// First, add service headers
	for k, v := range serviceHeaders {
		result[k] = v
	}
	
	// Then, override/add handler headers
	for k, v := range handlerHeaders {
		result[k] = v
	}
	
	// Substitute environment variables
	return substituteHeaders(result)
}

// parseTimeout parses timeout string (e.g., "30s", "5s") to time.Duration
func parseTimeout(timeoutStr string, defaultTimeout time.Duration) time.Duration {
	if timeoutStr == "" {
		return defaultTimeout
	}
	
	duration, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return defaultTimeout
	}
	return duration
}

// transformAuthorization transforms an authorization header based on transform configuration
func transformAuthorization(authHeader string, transform *TransformConfig) string {
	if transform == nil {
		return authHeader
	}
	
	// Simple prefix transformation
	if transform.FromPrefix != "" {
		prefix := transform.FromPrefix + " "
		if strings.HasPrefix(authHeader, prefix) {
			token := strings.TrimPrefix(authHeader, prefix)
			// If ToPrefix is empty, return just the token (no prefix)
			if transform.ToPrefix == "" {
				return token
			}
			// Otherwise, add the ToPrefix
			return transform.ToPrefix + " " + token
		}
	}
	
	return authHeader
}

// extractAuthorization extracts and processes authorization header based on configuration
// Resolution order: handler config > service config > default (pass-through)
func extractAuthorization(ctx context.Context, handlerConfig *AuthorizationConfig, serviceConfig *AuthorizationConfig) string {
	// Determine which config to use (handler overrides service)
	config := handlerConfig
	if config == nil {
		config = serviceConfig
	}
	
	// If no config, default to pass-through if available
	if config == nil {
		if auth, ok := auth.AuthorizationFromContext(ctx); ok {
			return auth
		}
		return ""
	}
	
	switch config.Strategy {
	case "pass-through":
		if auth, ok := auth.AuthorizationFromContext(ctx); ok {
			return auth
		}
		return ""
		
	case "transform":
		if auth, ok := auth.AuthorizationFromContext(ctx); ok {
			return transformAuthorization(auth, config.Transform)
		}
		return ""
		
	case "static":
		if config.StaticValue != "" {
			return config.StaticValue
		}
		if config.StaticValueEnv != "" {
			return os.Getenv(config.StaticValueEnv)
		}
		return ""
		
	case "none":
		return ""
		
	default:
		// Unknown strategy, fallback to pass-through
		if auth, ok := auth.AuthorizationFromContext(ctx); ok {
			return auth
		}
		return ""
	}
}

// generateHTTPHandler creates an HTTP handler function from tool and handler configuration
func generateHTTPHandler(tool ToolFile, handlerConfig HandlerConfig, serviceConfig ServiceConfig) (ToolHandler, error) {
	// Validate handler type
	if handlerConfig.Type != "http" {
		return nil, fmt.Errorf("unsupported handler type: %s", handlerConfig.Type)
	}

	// Determine base URL and path
	baseURL := serviceConfig.BaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required for service %s", tool.ServiceName)
	}

	// Use handler path if provided, otherwise use tool endpoint
	path := handlerConfig.Path
	if path == "" {
		path = tool.Endpoint
	}
	if path == "" {
		return nil, fmt.Errorf("path or endpoint is required for tool %s", tool.Name)
	}

	// Build full URL
	fullURL := baseURL + path

	// Resolve timeout: handler > service > default (30s)
	defaultTimeout := 30 * time.Second
	serviceTimeout := parseTimeout(serviceConfig.Timeout, defaultTimeout)
	handlerTimeout := parseTimeout(handlerConfig.Timeout, serviceTimeout)

	// Merge headers: service + handler (handler overrides)
	serviceHeaders := serviceConfig.Headers
	if serviceHeaders == nil {
		serviceHeaders = make(map[string]string)
	}
	handlerHeaders := handlerConfig.Headers
	if handlerHeaders == nil {
		handlerHeaders = make(map[string]string)
	}
	mergedHeaders := mergeHeaders(serviceHeaders, handlerHeaders)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: handlerTimeout,
	}

	// Return handler function
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Create request body from args
		bodyBytes, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, handlerConfig.Method, fullURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for k, v := range mergedHeaders {
			req.Header.Set(k, v)
		}

		// Extract and apply authorization header
		authHeader := extractAuthorization(ctx, handlerConfig.Authorization, serviceConfig.Authorization)
		if authHeader != "" {
			headerName := "Authorization"
			// Determine which config to use for header name
			config := handlerConfig.Authorization
			if config == nil {
				config = serviceConfig.Authorization
			}
			if config != nil && config.HeaderName != "" {
				headerName = config.HeaderName
			}
			req.Header.Set(headerName, authHeader)
		}

		// Make HTTP request
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Check for HTTP errors
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(respBody))
		}

		// Try to parse as JSON, otherwise return as string
		var result interface{}
		if err := json.Unmarshal(respBody, &result); err != nil {
			// If not JSON, return as string
			return map[string]interface{}{
				"response": string(respBody),
			}, nil
		}

		return result, nil
	}, nil
}

