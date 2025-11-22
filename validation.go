package mcpserver

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	validationPassedMsg = "validation passed"
	handlerNotFoundMsg  = "handler not found for tool"
	handlerNoToolMsg    = "handler has no corresponding tool"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Tool    string // Tool name (if applicable)
	Service string // Service name (if applicable)
	Field   string // Field name (if applicable)
	Message string // Error message
}

// Error implements error interface
func (e ValidationError) Error() string {
	if e.Tool != "" {
		return fmt.Sprintf("tool %q: %s", e.Tool, e.Message)
	}
	if e.Service != "" {
		return fmt.Sprintf("service %q: %s", e.Service, e.Message)
	}
	return e.Message
}

// ValidationResult contains all validation errors
type ValidationResult struct {
	Errors []ValidationError
}

// Valid returns true if there are no errors
func (r *ValidationResult) Valid() bool {
	return len(r.Errors) == 0
}

// AddError adds an error to the result
func (r *ValidationResult) AddError(err ValidationError) {
	r.Errors = append(r.Errors, err)
}

// Error implements error interface
func (r *ValidationResult) Error() string {
	if r.Valid() {
		return validationPassedMsg
	}

	var messages []string
	for _, err := range r.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// ValidateConfiguration validates tools and handlers configuration
// This is the main entry point that can be used by both server startup and CLI
func ValidateConfiguration(tools []ToolFile, handlersConfig *HandlersConfig) *ValidationResult {
	result := &ValidationResult{}

	// Check for nil handlers config
	if handlersConfig == nil {
		result.AddError(ValidationError{
			Message: "handlers configuration is nil",
		})
		return result
	}

	// Phase 1: Critical validations
	validateToolHandlerConsistency(tools, handlersConfig, result)
	validateServiceReferences(tools, handlersConfig, result)

	// Validate each tool
	for _, tool := range tools {
		validateTool(&tool, result)
	}

	// Validate each handler
	for name, handler := range handlersConfig.Handlers {
		tool := findToolByName(tools, name)
		if tool == nil {
			continue // Already reported in consistency check
		}

		service := handlersConfig.ServiceConfig[tool.ServiceName]
		validateHandler(&handler, tool, service, result)
	}

	// Validate services
	for name, service := range handlersConfig.ServiceConfig {
		validateService(name, service, result)
	}

	return result
}

// validateToolHandlerConsistency checks that every tool has a handler and vice versa
func validateToolHandlerConsistency(tools []ToolFile, handlersConfig *HandlersConfig, result *ValidationResult) {
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true

		// Check handler exists
		if _, exists := handlersConfig.Handlers[tool.Name]; !exists {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Message: handlerNotFoundMsg,
			})
		}
	}

	// Check handler-tool consistency (warn about orphaned handlers)
	for handlerName := range handlersConfig.Handlers {
		if !toolNames[handlerName] {
			result.AddError(ValidationError{
				Tool:    handlerName,
				Message: handlerNoToolMsg,
			})
		}
	}
}

// validateServiceReferences checks that all tool service references exist
func validateServiceReferences(tools []ToolFile, handlersConfig *HandlersConfig, result *ValidationResult) {
	for _, tool := range tools {
		if tool.ServiceName == "" {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "ServiceName",
				Message: "service name is required",
			})
			continue
		}

		if _, exists := handlersConfig.ServiceConfig[tool.ServiceName]; !exists {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Service: tool.ServiceName,
				Message: fmt.Sprintf("service %q not found", tool.ServiceName),
			})
		}
	}
}

// validateTool validates a single tool
func validateTool(tool *ToolFile, result *ValidationResult) {
	// Tool name is required
	if tool.Name == "" {
		result.AddError(ValidationError{
			Field:   "Name",
			Message: "tool name cannot be empty",
		})
		return
	}

	// Tool name must be valid identifier
	if !isValidIdentifier(tool.Name) {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "Name",
			Message: fmt.Sprintf("tool name %q is not a valid identifier", tool.Name),
		})
	}

	// InputSchema is required
	if tool.InputSchema == nil {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "InputSchema",
			Message: "InputSchema is required",
		})
		return
	}

	// Validate InputSchema structure
	if err := validateInputSchema(tool.InputSchema); err != nil {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "InputSchema",
			Message: err.Error(),
		})
	}

	// Endpoint format validation (if provided)
	if tool.Endpoint != "" {
		if err := validatePath(tool.Endpoint); err != nil {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "Endpoint",
				Message: err.Error(),
			})
		}
	}
}

// validateHandler validates a single handler
func validateHandler(handler *HandlerConfig, tool *ToolFile, _ ServiceConfig, result *ValidationResult) {
	// Handler type is required
	if handler.Type == "" {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "type",
			Message: "handler type is required",
		})
		return
	}

	// Handler type must be supported
	if handler.Type != HandlerTypeHTTP {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "type",
			Message: fmt.Sprintf("unsupported handler type %q (only 'http' is supported)", handler.Type),
		})
		return
	}

	// HTTP method is required
	if handler.Method == "" {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "method",
			Message: "HTTP method is required",
		})
		return
	}

	// HTTP method must be valid
	if !isValidHTTPMethod(handler.Method) {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "method",
			Message: fmt.Sprintf("invalid HTTP method %q", handler.Method),
		})
	}

	// Path or endpoint is required
	path := handler.Path
	if path == "" {
		path = tool.Endpoint
	}
	if path == "" {
		result.AddError(ValidationError{
			Tool:    tool.Name,
			Field:   "path",
			Message: "path or endpoint is required",
		})
	} else {
		// Validate path format
		if err := validatePath(path); err != nil {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "path",
				Message: err.Error(),
			})
		}

		// Validate path parameters are in InputSchema
		if err := validatePathParameters(path, tool.InputSchema); err != nil {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "path",
				Message: err.Error(),
			})
		}
	}

	// Validate timeout format (if provided)
	if handler.Timeout != "" {
		if _, err := time.ParseDuration(handler.Timeout); err != nil {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "timeout",
				Message: fmt.Sprintf("invalid timeout format %q: %v", handler.Timeout, err),
			})
		}
	}

	// Validate query parameters are in InputSchema
	if handler.QueryParams != nil {
		if err := validateQueryParameters(handler.QueryParams, tool.InputSchema); err != nil {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "queryParams",
				Message: err.Error(),
			})
		}
	}

	// Validate authorization config (if provided)
	if handler.Authorization != nil {
		if err := validateAuthorizationConfig(handler.Authorization); err != nil {
			result.AddError(ValidationError{
				Tool:    tool.Name,
				Field:   "authorization",
				Message: err.Error(),
			})
		}
	}
}

// validateService validates a single service configuration
func validateService(name string, service ServiceConfig, result *ValidationResult) {
	// baseURL is required
	if service.BaseURL == "" {
		result.AddError(ValidationError{
			Service: name,
			Field:   "baseURL",
			Message: "baseURL is required",
		})
		return
	}

	// baseURL must be valid URL
	if err := validateURL(service.BaseURL); err != nil {
		result.AddError(ValidationError{
			Service: name,
			Field:   "baseURL",
			Message: err.Error(),
		})
	}

	// Validate timeout format (if provided)
	if service.Timeout != "" {
		if _, err := time.ParseDuration(service.Timeout); err != nil {
			result.AddError(ValidationError{
				Service: name,
				Field:   "timeout",
				Message: fmt.Sprintf("invalid timeout format %q: %v", service.Timeout, err),
			})
		}
	}

	// Validate authorization config (if provided)
	if service.Authorization != nil {
		if err := validateAuthorizationConfig(service.Authorization); err != nil {
			result.AddError(ValidationError{
				Service: name,
				Field:   "authorization",
				Message: err.Error(),
			})
		}
	}
}

// validateInputSchema validates the structure of InputSchema (basic validation)
func validateInputSchema(schema map[string]interface{}) error {
	// Check required 'type' field
	schemaType, ok := schema["type"].(string)
	if !ok {
		return fmt.Errorf("InputSchema must have 'type' field")
	}

	// Validate type
	validTypes := []string{"object", "array", "string", "number", "integer", "boolean", "null"}
	typeValid := false
	for _, validType := range validTypes {
		if schemaType == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf(
			"invalid schema type %q (must be one of: object, array, string, number, integer, boolean, null)",
			schemaType,
		)
	}

	// Validate properties if present
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for propName, propSchema := range properties {
			if propSchemaMap, ok := propSchema.(map[string]interface{}); ok {
				if err := validatePropertySchema(propName, propSchemaMap); err != nil {
					return fmt.Errorf("property %q: %w", propName, err)
				}
			} else {
				return fmt.Errorf("property %q must be an object", propName)
			}
		}
	}

	// Validate required array if present
	if required, ok := schema["required"].([]interface{}); ok {
		for _, req := range required {
			if _, ok := req.(string); !ok {
				return fmt.Errorf("required array must contain strings")
			}
		}
	}

	return nil
}

// validatePropertySchema validates a single property schema
func validatePropertySchema(_ string, propSchema map[string]interface{}) error {
	propType, ok := propSchema["type"].(string)
	if !ok {
		return fmt.Errorf("property must have 'type' field")
	}

	validTypes := []string{"string", "number", "integer", "boolean", "array", "object", "null"}
	typeValid := false
	for _, validType := range validTypes {
		if propType == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("invalid property type %q", propType)
	}

	return nil
}

// validateURL validates that a string is a valid URL
func validateURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Scheme == "" {
		return fmt.Errorf("URL must have scheme (http/https)")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must have host")
	}

	return nil
}

// validatePath validates that a path is properly formatted
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /")
	}

	// Extract and validate path parameters
	paramNames := extractPathParams(path)
	seen := make(map[string]bool)
	for _, param := range paramNames {
		if seen[param] {
			return fmt.Errorf("duplicate path parameter %q in path %q", param, path)
		}
		seen[param] = true

		if !isValidIdentifier(param) {
			return fmt.Errorf("invalid path parameter name %q in path %q", param, path)
		}
	}

	return nil
}

// validatePathParameters checks that path parameters exist in InputSchema
func validatePathParameters(path string, inputSchema map[string]interface{}) error {
	if inputSchema == nil {
		return nil
	}

	pathParams := extractPathParams(path)
	if len(pathParams) == 0 {
		return nil
	}

	properties, ok := inputSchema["properties"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("path parameters %v found but InputSchema has no properties", pathParams)
	}

	var missing []string
	for _, param := range pathParams {
		if _, exists := properties[param]; !exists {
			missing = append(missing, param)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("path parameters %v not found in InputSchema properties", missing)
	}

	return nil
}

// validateQueryParameters checks that dynamic query parameters exist in InputSchema
func validateQueryParameters(queryParams map[string]string, inputSchema map[string]interface{}) error {
	if inputSchema == nil {
		return nil
	}

	// Extract dynamic query parameters (those with {paramName} format)
	var dynamicParams []string
	for _, template := range queryParams {
		if strings.HasPrefix(template, "{") && strings.HasSuffix(template, "}") {
			paramName := template[1 : len(template)-1]
			dynamicParams = append(dynamicParams, paramName)
		}
	}

	if len(dynamicParams) == 0 {
		return nil
	}

	properties, ok := inputSchema["properties"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("query parameters %v found but InputSchema has no properties", dynamicParams)
	}

	var missing []string
	for _, param := range dynamicParams {
		if _, exists := properties[param]; !exists {
			missing = append(missing, param)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("query parameters %v not found in InputSchema properties", missing)
	}

	return nil
}

// validateAuthorizationConfig validates authorization configuration
func validateAuthorizationConfig(config *AuthorizationConfig) error {
	validStrategies := []string{
		StrategyPassThrough,
		StrategyTransform,
		StrategyStatic,
		StrategyBasic,
		StrategyNone,
	}

	strategyValid := false
	for _, valid := range validStrategies {
		if config.Strategy == valid {
			strategyValid = true
			break
		}
	}

	if !strategyValid {
		return fmt.Errorf(
			"invalid authorization strategy %q (must be one of: pass-through, transform, static, basic, none)",
			config.Strategy,
		)
	}

	// Validate transform config
	if config.Strategy == StrategyTransform {
		if config.Transform == nil {
			return fmt.Errorf("Transform config is required for transform strategy")
		}
	}

	// Validate static config
	if config.Strategy == StrategyStatic {
		if config.StaticValue == "" && config.StaticValueEnv == "" {
			return fmt.Errorf("StaticValue or StaticValueEnv is required for static strategy")
		}
	}

	// Validate basic auth config
	if config.Strategy == StrategyBasic {
		if err := validateBasicAuthConfig(config.BasicAuth); err != nil {
			return err
		}
	}

	// Validate header name format (if provided)
	if config.HeaderName != "" {
		if !isValidHeaderName(config.HeaderName) {
			return fmt.Errorf("invalid header name %q", config.HeaderName)
		}
	}

	return nil
}

// validateBasicAuthConfig validates BasicAuth configuration
func validateBasicAuthConfig(basicAuth *BasicAuthConfig) error {
	if basicAuth == nil {
		return fmt.Errorf("BasicAuth config is required for basic strategy")
	}
	if basicAuth.UsernameHeader == "" &&
		basicAuth.Username == "" &&
		basicAuth.UsernameEnv == "" &&
		basicAuth.EncodedValue == "" &&
		basicAuth.EncodedValueEnv == "" {
		return fmt.Errorf(
			"BasicAuth must have usernameHeader, username, usernameEnv, encodedValue, or encodedValueEnv",
		)
	}
	return nil
}

// Helper functions

// extractPathParams extracts parameter names from a path like "/api/v1/users/{userId}"
func extractPathParams(path string) []string {
	var params []string
	start := 0

	for {
		paramStart := strings.Index(path[start:], "{")
		if paramStart == -1 {
			break
		}
		paramStart += start

		paramEnd := strings.Index(path[paramStart:], "}")
		if paramEnd == -1 {
			break
		}
		paramEnd += paramStart

		paramName := path[paramStart+1 : paramEnd]
		params = append(params, paramName)
		start = paramEnd + 1
	}

	return params
}

// isValidIdentifier checks if a string is a valid identifier (alphanumeric, underscore, hyphen)
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// isValidHTTPMethod checks if a string is a valid HTTP method
func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	methodUpper := strings.ToUpper(method)
	for _, valid := range validMethods {
		if methodUpper == valid {
			return true
		}
	}
	return false
}

// isValidHeaderName checks if a string is a valid HTTP header name
func isValidHeaderName(name string) bool {
	if name == "" {
		return false
	}
	// HTTP header names should be alphanumeric with hyphens
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// findToolByName finds a tool by name in the tools slice
func findToolByName(tools []ToolFile, name string) *ToolFile {
	for i := range tools {
		if tools[i].Name == name {
			return &tools[i]
		}
	}
	return nil
}
