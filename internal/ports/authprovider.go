package ports

import (
	"context"

	"elydelva/one/internal/core"
)

// AuthProvider handles authentication and token refresh for a specific provider kind.
type AuthProvider interface {
	// Login starts the authentication flow and returns a new credential.
	Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error)
	// Refresh exchanges a refresh token for a new access token.
	Refresh(ctx context.Context, cred core.Credential) (core.Credential, error)
	// Supports reports whether this provider handles the given kind.
	Supports(kind core.ProviderKind) bool
}
