package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/vubon/go-mcp-server/auth"
)

// Authorization strategy constants
const (
	StrategyPassThrough = "pass-through"
	StrategyTransform   = "transform"
	StrategyStatic      = "static"
	StrategyBasic       = "basic"
	StrategyNone        = "none"
)

// Handler type constants
const (
	HandlerTypeHTTP = "http"
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

// BasicAuthConfig represents configuration for Basic Authentication
type BasicAuthConfig struct {
	// Header name in incoming request to extract username value from
	// If specified, extracts the header value from incoming request and passes it to backend
	UsernameHeader string `json:"usernameHeader,omitempty" yaml:"usernameHeader,omitempty"`

	// Header name in incoming request to extract password value from
	// If specified, extracts the header value from incoming request and passes it to backend
	// Can be "None" or empty to indicate password is not required
	PasswordHeader string `json:"passwordHeader,omitempty" yaml:"passwordHeader,omitempty"`

	// Username for Basic Auth (legacy, use usernameHeader instead)
	Username string `json:"username,omitempty" yaml:"username,omitempty"`

	// Password for Basic Auth (legacy, use passwordHeader instead)
	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	// Environment variable for username (legacy)
	UsernameEnv string `json:"usernameEnv,omitempty" yaml:"usernameEnv,omitempty"`

	// Environment variable for password (legacy)
	PasswordEnv string `json:"passwordEnv,omitempty" yaml:"passwordEnv,omitempty"`

	// Pre-encoded Basic Auth value (e.g., "Basic base64(username:password)")
	// If provided, this takes precedence over username/password
	EncodedValue string `json:"encodedValue,omitempty" yaml:"encodedValue,omitempty"`

	// Environment variable for pre-encoded Basic Auth value
	EncodedValueEnv string `json:"encodedValueEnv,omitempty" yaml:"encodedValueEnv,omitempty"`
}

// AuthorizationConfig represents authorization configuration for handlers or services
type AuthorizationConfig struct {
	// Strategy: StrategyPassThrough, StrategyTransform, StrategyStatic, StrategyBasic, StrategyNone
	// - StrategyPassThrough: use incoming Authorization header as-is
	// - StrategyTransform: transform the header (e.g., Bearer -> API-Key)
	// - StrategyStatic: use static value from config
	// - StrategyBasic: use Basic Authentication (username:password encoded as base64)
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

	// For StrategyBasic: Basic Authentication configuration
	BasicAuth *BasicAuthConfig `json:"basicAuth,omitempty" yaml:"basicAuth,omitempty"`
}

// RetryConfig represents configuration for retry logic
type RetryConfig struct {
	// Maximum number of retry attempts (including initial attempt)
	// Default: 1 (no retries)
	MaxAttempts int `json:"maxAttempts,omitempty" yaml:"maxAttempts,omitempty"`

	// Initial delay before first retry
	// Default: 100ms
	InitialDelay string `json:"initialDelay,omitempty" yaml:"initialDelay,omitempty"`

	// Maximum delay between retries
	// Default: 5s
	MaxDelay string `json:"maxDelay,omitempty" yaml:"maxDelay,omitempty"`

	// Exponential backoff multiplier
	// Default: 2.0
	Multiplier float64 `json:"multiplier,omitempty" yaml:"multiplier,omitempty"`

	// Add jitter to prevent thundering herd
	// Default: true
	Jitter bool `json:"jitter,omitempty" yaml:"jitter,omitempty"`

	// HTTP status codes that should trigger retry
	// Default: [500, 502, 503, 504]
	RetryableStatusCodes []int `json:"retryableStatusCodes,omitempty" yaml:"retryableStatusCodes,omitempty"`

	// Error types that should trigger retry
	// Options: "timeout", "connection_refused", "temporary", "network"
	// Default: ["timeout", "connection_refused", "temporary"]
	RetryableErrors []string `json:"retryableErrors,omitempty" yaml:"retryableErrors,omitempty"`
}

// CircuitBreakerConfig represents configuration for circuit breaker
type CircuitBreakerConfig struct {
	// Maximum number of consecutive failures before opening circuit
	// Default: 5
	MaxFailures int `json:"maxFailures,omitempty" yaml:"maxFailures,omitempty"`

	// Duration to keep circuit open before attempting half-open
	// Default: 60s
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Maximum number of calls in half-open state
	// Default: 3
	HalfOpenMaxCalls int `json:"halfOpenMaxCalls,omitempty" yaml:"halfOpenMaxCalls,omitempty"`

	// Number of successful calls needed to close circuit from half-open
	// Default: 2
	SuccessThreshold int `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`
}

// ServiceConfig represents configuration for a service
type ServiceConfig struct {
	BaseURL        string                `json:"baseURL" yaml:"baseURL"`
	Timeout        string                `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Headers        map[string]string     `json:"headers,omitempty" yaml:"headers,omitempty"`
	Authorization  *AuthorizationConfig  `json:"authorization,omitempty" yaml:"authorization,omitempty"`
	Retry          *RetryConfig          `json:"retry,omitempty" yaml:"retry,omitempty"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty" yaml:"circuitBreaker,omitempty"`
}

// HandlerConfig represents configuration for a handler
type HandlerConfig struct {
	Type          string               `json:"type" yaml:"type"`
	Method        string               `json:"method" yaml:"method"`
	Path          string               `json:"path" yaml:"path"`
	QueryParams   map[string]string    `json:"queryParams,omitempty" yaml:"queryParams,omitempty"`
	Headers       map[string]string    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Timeout       string               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Authorization *AuthorizationConfig `json:"authorization,omitempty" yaml:"authorization,omitempty"`
	Retry         *RetryConfig         `json:"retry,omitempty" yaml:"retry,omitempty"`
}

// HandlersConfig represents the complete handlers configuration
type HandlersConfig struct {
	Version       string                   `json:"version,omitempty" yaml:"version,omitempty"` // Optional version string
	ConfigHash    string                   `json:"hash,omitempty" yaml:"hash,omitempty"`       // Optional hash of config
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

// ToolsConfig represents a tools configuration file (with optional version)
type ToolsConfig struct {
	Version string     `json:"version,omitempty" yaml:"version,omitempty"` // Optional version string
	Tools   []ToolFile `json:"tools,omitempty" yaml:"tools,omitempty"`     // Tools array (when wrapped)
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
// Supports both formats:
// 1. Array format: [{"Name": "tool1", ...}, ...]
// 2. Wrapped format: {"version": "1.0.0", "tools": [{"Name": "tool1", ...}, ...]}
func parseToolsFromJSON(data []byte) ([]ToolFile, string, error) {
	// Try wrapped format first
	var wrappedConfig ToolsConfig
	if err := json.Unmarshal(data, &wrappedConfig); err == nil {
		// Check if it's actually wrapped format (has tools field)
		if wrappedConfig.Tools != nil {
			return wrappedConfig.Tools, wrappedConfig.Version, nil
		}
	}

	// Fall back to array format
	var tools []ToolFile
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, "", err
	}
	return tools, "", nil
}

// parseToolsFromYAML parses tools from YAML bytes
// Supports both formats:
// 1. Array format: - Name: tool1 ...
// 2. Wrapped format: version: "1.0.0" tools: - Name: tool1 ...
func parseToolsFromYAML(data []byte) ([]ToolFile, string, error) {
	// Try wrapped format first
	var wrappedConfig ToolsConfig
	if err := yaml.Unmarshal(data, &wrappedConfig); err == nil {
		// Check if it's actually wrapped format (has tools field)
		if wrappedConfig.Tools != nil {
			return wrappedConfig.Tools, wrappedConfig.Version, nil
		}
	}

	// Fall back to array format
	var tools []ToolFile
	if err := yaml.Unmarshal(data, &tools); err != nil {
		return nil, "", err
	}
	return tools, "", nil
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

// substituteQueryParams substitutes query parameter templates with values from args.
// Returns the query string and a map of removed parameters.
// Query parameter templates can be:
//   - "{paramName}" - Dynamic parameter from args
//   - "staticValue" - Static value (not a placeholder)
func substituteQueryParams(
	queryParams map[string]string, args map[string]interface{},
) (queryString string, removedParams map[string]interface{}) {
	if len(queryParams) == 0 {
		return "", make(map[string]interface{})
	}

	removedParams = make(map[string]interface{})
	var values []string

	for key, template := range queryParams {
		// Check if template is a placeholder {paramName}
		if strings.HasPrefix(template, "{") && strings.HasSuffix(template, "}") {
			paramName := template[1 : len(template)-1]

			// Get value from args
			if value, exists := args[paramName]; exists {
				valueStr := valueToString(value)
				// URL encode the value
				encodedValue := url.QueryEscape(valueStr)
				values = append(values, key+"="+encodedValue)
				removedParams[paramName] = value
			}
			// If not found, skip this query parameter
		} else {
			// Static value (not a placeholder)
			encodedValue := url.QueryEscape(template)
			values = append(values, key+"="+encodedValue)
		}
	}

	if len(values) > 0 {
		queryString = "?" + strings.Join(values, "&")
	}

	return queryString, removedParams
}

// substitutePathParams substitutes path parameters like {param_name} with values from args.
// Returns the substituted path and a map of removed parameters.
func substitutePathParams(
	path string, args map[string]interface{},
) (substitutedPath string, removedParams map[string]interface{}) {
	if len(args) == 0 {
		return path, make(map[string]interface{})
	}

	removedParams = make(map[string]interface{})

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
			removedParams[paramName] = value
			start = paramEnd + 1
		} else {
			// Parameter not found, keep placeholder as-is
			builder.WriteString(path[paramStart : paramEnd+1])
			start = paramEnd + 1
		}
	}

	substitutedPath = builder.String()
	return substitutedPath, removedParams
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

// extractBasicAuthFromHeaders extracts header values from incoming request when
// usernameHeader/passwordHeader are specified.
// Returns a Basic Auth encoded string (e.g., "Basic base64(username:password)").
// If usernameHeader/passwordHeader are not specified or values not found, returns empty string.
func extractBasicAuthFromHeaders(ctx context.Context, basicAuth *BasicAuthConfig) string {
	if basicAuth == nil {
		return ""
	}

	// If usernameHeader is specified, extract header values from incoming request and encode as Basic Auth
	if basicAuth.UsernameHeader != "" {
		// Extract username from incoming request header (case-insensitive lookup)
		usernameValue, usernameOk := auth.GetHeaderFromContext(ctx, basicAuth.UsernameHeader)
		if !usernameOk || usernameValue == "" {
			return ""
		}

		// Extract password from incoming request header (if specified and not "None")
		passwordValue := ""
		if basicAuth.PasswordHeader != "" && !strings.EqualFold(basicAuth.PasswordHeader, "none") {
			if pwd, ok := auth.GetHeaderFromContext(ctx, basicAuth.PasswordHeader); ok {
				passwordValue = pwd
			}
		}

		// Encode as Basic Auth: base64(username:password)
		credentials := usernameValue
		if passwordValue != "" {
			credentials = usernameValue + ":" + passwordValue
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		return "Basic " + encoded
	}

	return ""
}

// buildBasicAuth builds a Basic Authentication header value from configuration.
// Returns "Basic base64(username:password)" format.
// Resolution order: header extraction > encodedValue/encodedValueEnv > username/password (with env var support)
func buildBasicAuth(ctx context.Context, basicAuth *BasicAuthConfig) string {
	if basicAuth == nil {
		return ""
	}

	// If usernameHeader is specified, extract from incoming request headers and encode as Basic Auth
	if basicAuth.UsernameHeader != "" {
		return extractBasicAuthFromHeaders(ctx, basicAuth)
	}

	// Check for pre-encoded value first (highest priority)
	if basicAuth.EncodedValue != "" {
		// If it already has "Basic " prefix, return as-is, otherwise add it
		if strings.HasPrefix(basicAuth.EncodedValue, "Basic ") {
			return basicAuth.EncodedValue
		}
		return "Basic " + basicAuth.EncodedValue
	}

	if basicAuth.EncodedValueEnv != "" {
		encodedValue := os.Getenv(basicAuth.EncodedValueEnv)
		if encodedValue != "" {
			if strings.HasPrefix(encodedValue, "Basic ") {
				return encodedValue
			}
			return "Basic " + encodedValue
		}
	}

	// Get username and password (with env var support)
	username := basicAuth.Username
	if username == "" && basicAuth.UsernameEnv != "" {
		username = os.Getenv(basicAuth.UsernameEnv)
	}

	password := basicAuth.Password
	if password == "" && basicAuth.PasswordEnv != "" {
		password = os.Getenv(basicAuth.PasswordEnv)
	}

	// If both username and password are available, encode them
	if username != "" && password != "" {
		credentials := username + ":" + password
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		return "Basic " + encoded
	}

	return ""
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

	case StrategyBasic:
		return buildBasicAuth(ctx, config.BasicAuth)

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
	baseURL        string
	pathTemplate   string
	queryParams    map[string]string
	method         string
	timeout        time.Duration
	headers        map[string]string
	authConfig     *AuthorizationConfig
	serviceAuth    *AuthorizationConfig
	client         *http.Client
	retryConfig    *RetryConfig
	circuitBreaker *CircuitBreaker
}

// validateHandlerConfig validates handler and service configuration.
func validateHandlerConfig(tool *ToolFile, handlerConfig *HandlerConfig, serviceConfig ServiceConfig) error {
	if handlerConfig.Type != HandlerTypeHTTP {
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

	// Copy query params from handler config
	queryParams := make(map[string]string)
	if handlerConfig.QueryParams != nil {
		for k, v := range handlerConfig.QueryParams {
			queryParams[k] = v
		}
	}

	// Resolve retry config: handler > service > default (no retry)
	retryConfig := resolveRetryConfig(handlerConfig, serviceConfig)

	// Get or create circuit breaker
	var circuitBreaker *CircuitBreaker
	if serviceConfig.CircuitBreaker != nil {
		circuitBreaker = globalCBManager.GetCircuitBreaker(
			tool.ServiceName, serviceConfig.CircuitBreaker,
		)
	}

	return &httpHandlerConfig{
		baseURL:        serviceConfig.BaseURL,
		pathTemplate:   pathTemplate,
		queryParams:    queryParams,
		method:         handlerConfig.Method,
		timeout:        handlerTimeout,
		headers:        mergedHeaders,
		authConfig:     handlerConfig.Authorization,
		serviceAuth:    serviceConfig.Authorization,
		client:         client,
		retryConfig:    retryConfig,
		circuitBreaker: circuitBreaker,
	}
}

// buildHTTPRequest builds an HTTP request from context and arguments
func (cfg *httpHandlerConfig) buildHTTPRequest(
	ctx context.Context, args map[string]interface{},
) (*http.Request, error) {
	// Substitute path parameters from args
	path, pathRemovedParams := substitutePathParams(cfg.pathTemplate, args)

	// Substitute query parameters from args
	queryString, queryRemovedParams := substituteQueryParams(cfg.queryParams, args)

	// Combine removed parameters (path + query)
	allRemovedParams := make(map[string]interface{})
	for k, v := range pathRemovedParams {
		allRemovedParams[k] = v
	}
	for k, v := range queryRemovedParams {
		allRemovedParams[k] = v
	}

	// Build full URL with substituted path and query string
	fullURL := cfg.baseURL + path + queryString

	// Create a copy of args without path/query parameters (they're now in the URL)
	bodyArgs := make(map[string]interface{})
	for k, v := range args {
		// Skip parameters that were used in the path or query
		if _, wasRemoved := allRemovedParams[k]; !wasRemoved {
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

		// Make HTTP request with retry and circuit breaker
		resp, err := cfg.executeWithRetry(ctx, req)
		if err != nil {
			// Record failure for circuit breaker
			if cfg.circuitBreaker != nil {
				cfg.circuitBreaker.OnFailure()
			}
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		// Record success for circuit breaker
		if cfg.circuitBreaker != nil {
			cfg.circuitBreaker.OnSuccess()
		}

		// Handle response
		return handleHTTPResponse(resp)
	}, nil
}
