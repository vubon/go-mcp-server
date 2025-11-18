package auth

import "context"

// authKey is a private type used as context key to avoid collisions
type authKey struct{}

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
