package auth

import (
	"context"
	"testing"
)

func TestWithAuthorization(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token-123"

	ctxWithAuth := WithAuthorization(ctx, authHeader)

	// Extract and verify
	extracted, ok := AuthorizationFromContext(ctxWithAuth)
	if !ok {
		t.Fatal("Expected authorization to be found in context")
	}
	if extracted != authHeader {
		t.Errorf("Expected %q, got %q", authHeader, extracted)
	}
}

func TestWithAuthorization_Empty(t *testing.T) {
	ctx := context.Background()
	ctxWithAuth := WithAuthorization(ctx, "")

	// Should not add empty authorization
	_, ok := AuthorizationFromContext(ctxWithAuth)
	if ok {
		t.Error("Expected no authorization in context for empty header")
	}
}

func TestAuthorizationFromContext_NotFound(t *testing.T) {
	ctx := context.Background()
	_, ok := AuthorizationFromContext(ctx)
	if ok {
		t.Error("Expected no authorization in context")
	}
}

func TestAuthorizationFromContext_MultipleCalls(t *testing.T) {
	ctx := context.Background()
	authHeader := "Bearer test-token-123"

	ctxWithAuth := WithAuthorization(ctx, authHeader)

	// Extract multiple times
	for i := 0; i < 3; i++ {
		extracted, ok := AuthorizationFromContext(ctxWithAuth)
		if !ok {
			t.Fatalf("Expected authorization to be found on call %d", i+1)
		}
		if extracted != authHeader {
			t.Errorf("Call %d: Expected %q, got %q", i+1, authHeader, extracted)
		}
	}
}
