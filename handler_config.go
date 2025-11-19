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

// Authorization strategy constants
const (
	StrategyPassThrough = "pass-through"
	StrategyTransform   = "transform"
	StrategyStatic      = "static"
	StrategyNone        = "none"
)

// Default header name for authorization
const DefaultAuthHeaderName = "Authorization"

// Default HTTP timeout
const DefaultHTTPTimeout = 30 * time.Second

// TransformConfig represents configuration for authorization header transformation
type TransformConfig struct {
	// Transform Bearer token to different format
	FromPrefix string `json:"fromPrefix,omitempty" yaml:"fromPrefix,omitempty"` // e.g., "Bearer"
	ToPrefix   string `json:"toPrefix,omitempty" yaml:"toPrefix,omitempty"`     // e.g., "ApiKey"
}

// AuthorizationConfig represents authorization configuration for handlers or services
type AuthorizationConfig struct {
	// Strategy: StrategyPassThrough, StrategyTransform, StrategyStatic, StrategyNone
	// - StrategyPassThrough: use incoming Authorization header as-is
	// - StrategyTransform: transform the header (e.g., Bearer -> API-Key)
	// - StrategyStatic: use static value from config
	// - StrategyNone: don't add Authorization header
	Strategy string `json:"strategy" yaml:"strategy"`

	// Header name to use (default: DefaultAuthHeaderName)
	HeaderName string `json:"headerName,omitempty" yaml:"headerName,omitempty"`

	// For StrategyTransform: transformation rules
	Transform *TransformConfig `json:"transform,omitempty" yaml:"transform,omitempty"`

	// For StrategyStatic: static value
	StaticValue string `json:"staticValue,omitempty" yaml:"staticValue,omitempty"`

	// Environment variable for static value
	StaticValueEnv string `json:"staticValueEnv,omitempty" yaml:"staticValueEnv,omitempty"`
}

// ServiceConfig represents configuration for a service
type ServiceConfig struct {
	BaseURL       string               `json:"baseURL" yaml:"baseURL"`
	Timeout       string               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Headers       map[string]string    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Authorization *AuthorizationConfig `json:"authorization,omitempty" yaml:"authorization,omitempty"`
}

// HandlerConfig represents configuration for a handler
type HandlerConfig struct {
	Type          string               `json:"type" yaml:"type"`
	Method        string               `json:"method" yaml:"method"`
	Path          string               `json:"path" yaml:"path"`
	Headers       map[string]string    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Timeout       string               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Authorization *AuthorizationConfig `json:"authorization,omitempty" yaml:"authorization,omitempty"`
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

// valueToString converts a value to string representation
func valueToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// substitutePathParams substitutes path parameters like {param_name} with values from args
// Returns the substituted path and a map of removed parameters
func substitutePathParams(path string, args map[string]interface{}) (string, map[string]interface{}) {
	if len(args) == 0 {
		return path, make(map[string]interface{})
	}

	removed := make(map[string]interface{})

	// Use strings.Builder for efficient string building
	var builder strings.Builder
	start := 0

	for {
		// Find next parameter placeholder
		paramStart := strings.Index(path[start:], "{")
		if paramStart == -1 {
			// No more parameters, append remaining path
			if start < len(path) {
				builder.WriteString(path[start:])
			}
			break
		}
		paramStart += start

		// Find closing brace
		paramEnd := strings.Index(path[paramStart:], "}")
		if paramEnd == -1 {
			// Unclosed brace, append remaining path
			builder.WriteString(path[start:])
			break
		}
		paramEnd += paramStart

		// Extract parameter name (without braces)
		paramName := path[paramStart+1 : paramEnd]

		// Append text before parameter
		builder.WriteString(path[start:paramStart])

		// Get value from args and substitute
		if value, exists := args[paramName]; exists {
			valueStr := valueToString(value)
			builder.WriteString(valueStr)
			removed[paramName] = value
			start = paramEnd + 1
		} else {
			// Parameter not found, keep placeholder as-is
			builder.WriteString(path[paramStart : paramEnd+1])
			start = paramEnd + 1
		}
	}

	return builder.String(), removed
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

// getAuthFromContext extracts authorization from context if available
func getAuthFromContext(ctx context.Context) (string, bool) {
	return auth.AuthorizationFromContext(ctx)
}

// extractAuthorization extracts and processes authorization header based on configuration.
// Resolution order: handler config > service config > default (pass-through)
func extractAuthorization(ctx context.Context, handlerConfig, serviceConfig *AuthorizationConfig) string {
	// Determine which config to use (handler overrides service)
	config := handlerConfig
	if config == nil {
		config = serviceConfig
	}

	// If no config, default to pass-through if available
	if config == nil {
		if auth, ok := getAuthFromContext(ctx); ok {
			return auth
		}
		return ""
	}

	// Get auth from context once for strategies that need it
	authFromCtx, hasAuth := getAuthFromContext(ctx)

	switch config.Strategy {
	case StrategyPassThrough:
		if hasAuth {
			return authFromCtx
		}
		return ""

	case StrategyTransform:
		if hasAuth {
			return transformAuthorization(authFromCtx, config.Transform)
		}
		return ""

	case StrategyStatic:
		if config.StaticValue != "" {
			return config.StaticValue
		}
		if config.StaticValueEnv != "" {
			return os.Getenv(config.StaticValueEnv)
		}
		return ""

	case StrategyNone:
		return ""

	default:
		// Unknown strategy, fallback to pass-through
		if hasAuth {
			return authFromCtx
		}
		return ""
	}
}

// getAuthHeaderName returns the header name to use for authorization.
// Resolution order: handler config > service config > default
func getAuthHeaderName(handlerConfig, serviceConfig *AuthorizationConfig) string {
	// Check handler config first
	if handlerConfig != nil && handlerConfig.HeaderName != "" {
		return handlerConfig.HeaderName
	}

	// Fall back to service config
	if serviceConfig != nil && serviceConfig.HeaderName != "" {
		return serviceConfig.HeaderName
	}

	// Default
	return DefaultAuthHeaderName
}

// httpHandlerConfig holds resolved configuration for HTTP handler
type httpHandlerConfig struct {
	baseURL      string
	pathTemplate string
	method       string
	timeout      time.Duration
	headers      map[string]string
	authConfig   *AuthorizationConfig
	serviceAuth  *AuthorizationConfig
	client       *http.Client
}

// validateHandlerConfig validates handler and service configuration.
func validateHandlerConfig(tool *ToolFile, handlerConfig *HandlerConfig, serviceConfig ServiceConfig) error {
	if handlerConfig.Type != "http" {
		return fmt.Errorf("unsupported handler type: %s", handlerConfig.Type)
	}

	if serviceConfig.BaseURL == "" {
		return fmt.Errorf("baseURL is required for service %s", tool.ServiceName)
	}

	pathTemplate := handlerConfig.Path
	if pathTemplate == "" {
		pathTemplate = tool.Endpoint
	}
	if pathTemplate == "" {
		return fmt.Errorf("path or endpoint is required for tool %s", tool.Name)
	}

	return nil
}

// resolveHandlerConfig resolves all configuration values.
func resolveHandlerConfig(
	tool *ToolFile, handlerConfig *HandlerConfig, serviceConfig ServiceConfig,
) *httpHandlerConfig {
	// Resolve path template
	pathTemplate := handlerConfig.Path
	if pathTemplate == "" {
		pathTemplate = tool.Endpoint
	}

	// Resolve timeout: handler > service > default
	serviceTimeout := parseTimeout(serviceConfig.Timeout, DefaultHTTPTimeout)
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

	return &httpHandlerConfig{
		baseURL:      serviceConfig.BaseURL,
		pathTemplate: pathTemplate,
		method:       handlerConfig.Method,
		timeout:      handlerTimeout,
		headers:      mergedHeaders,
		authConfig:   handlerConfig.Authorization,
		serviceAuth:  serviceConfig.Authorization,
		client:       client,
	}
}

// buildHTTPRequest builds an HTTP request from context and arguments
func (cfg *httpHandlerConfig) buildHTTPRequest(
	ctx context.Context, args map[string]interface{},
) (*http.Request, error) {
	// Substitute path parameters from args
	path, removedParams := substitutePathParams(cfg.pathTemplate, args)

	// Build full URL with substituted path
	fullURL := cfg.baseURL + path

	// Create a copy of args without path parameters (they're now in the URL)
	bodyArgs := make(map[string]interface{})
	for k, v := range args {
		// Skip parameters that were used in the path
		if _, wasRemoved := removedParams[k]; !wasRemoved {
			bodyArgs[k] = v
		}
	}

	// Create request body from remaining args
	bodyBytes, err := json.Marshal(bodyArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = http.NoBody
	}
	req, err := http.NewRequestWithContext(ctx, cfg.method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	// Extract and apply authorization header
	authHeader := extractAuthorization(ctx, cfg.authConfig, cfg.serviceAuth)
	if authHeader != "" {
		headerName := getAuthHeaderName(cfg.authConfig, cfg.serviceAuth)
		req.Header.Set(headerName, authHeader)
	}

	return req, nil
}

// handleHTTPResponse processes the HTTP response and returns the result.
func handleHTTPResponse(resp *http.Response) (interface{}, error) {
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
}

// generateHTTPHandler creates an HTTP handler function from tool and handler configuration.
func generateHTTPHandler(
	tool *ToolFile, handlerConfig *HandlerConfig, serviceConfig ServiceConfig,
) (ToolHandler, error) {
	// Validate configuration
	if err := validateHandlerConfig(tool, handlerConfig, serviceConfig); err != nil {
		return nil, err
	}

	// Resolve configuration
	cfg := resolveHandlerConfig(tool, handlerConfig, serviceConfig)

	// Return handler function
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Build HTTP request
		req, err := cfg.buildHTTPRequest(ctx, args)
		if err != nil {
			return nil, err
		}

		// Make HTTP request
		resp, err := cfg.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		// Handle response
		return handleHTTPResponse(resp)
	}, nil
}
