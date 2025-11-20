package auth

import (
	"context"
	"strings"
)

// authKey is a private type used as context key to avoid collisions
type authKey struct{}

// headersKey is a private type used as context key for request headers
type headersKey struct{}

// WithAuthorization adds Authorization header to context
// This allows passing the Authorization header from HTTP transport
// through to tool handlers without exposing it in function signatures
func WithAuthorization(ctx context.Context, authHeader string) context.Context {
	if authHeader == "" {
		return ctx
	}
	return context.WithValue(ctx, authKey{}, authHeader)
}

// AuthorizationFromContext extracts Authorization header from context
// Returns the header value and a boolean indicating if it was found
func AuthorizationFromContext(ctx context.Context) (string, bool) {
	auth, ok := ctx.Value(authKey{}).(string)
	if !ok || auth == "" {
		return "", false
	}
	return auth, true
}

// WithRequestHeaders adds request headers to context
// This allows passing HTTP request headers from transport layer
// through to tool handlers
func WithRequestHeaders(ctx context.Context, headers map[string]string) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	return context.WithValue(ctx, headersKey{}, headers)
}

// RequestHeadersFromContext extracts request headers from context
// Returns the headers map and a boolean indicating if it was found
func RequestHeadersFromContext(ctx context.Context) (map[string]string, bool) {
	headers, ok := ctx.Value(headersKey{}).(map[string]string)
	if !ok || headers == nil {
		return nil, false
	}
	return headers, true
}

// GetHeaderFromContext extracts a specific header value from context
// Returns the header value and a boolean indicating if it was found
// Header name matching is case-insensitive
func GetHeaderFromContext(ctx context.Context, headerName string) (string, bool) {
	headers, ok := RequestHeadersFromContext(ctx)
	if !ok {
		return "", false
	}

	// Try exact match first
	if value, exists := headers[headerName]; exists {
		return value, true
	}

	// Try case-insensitive match
	for key, value := range headers {
		if strings.EqualFold(key, headerName) {
			return value, true
		}
	}

	return "", false
}
