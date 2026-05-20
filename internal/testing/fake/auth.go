package fake

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// AuthProvider is a configurable fake auth provider for testing.
type AuthProvider struct {
	Kind         core.ProviderKind
	LoginFunc    func(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error)
	RefreshFunc  func(ctx context.Context, cred core.Credential) (core.Credential, error)
	RefreshCount int
}

// NewAuthProvider creates a fake that supports the given provider kind.
func NewAuthProvider(kind core.ProviderKind) *AuthProvider {
	return &AuthProvider{Kind: kind}
}

func (p *AuthProvider) Supports(kind core.ProviderKind) bool { return p.Kind == kind }

func (p *AuthProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	if p.LoginFunc != nil {
		return p.LoginFunc(ctx, svc, alias)
	}
	return core.Credential{}, errors.New("fake auth: no LoginFunc configured")
}

func (p *AuthProvider) Refresh(ctx context.Context, cred core.Credential) (core.Credential, error) {
	p.RefreshCount++
	if p.RefreshFunc != nil {
		return p.RefreshFunc(ctx, cred)
	}
	return cred, nil
}

var _ ports.AuthProvider = (*AuthProvider)(nil)
