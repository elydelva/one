package auth

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// OAuthUserProvider handles the OAuth 2.0 authorization code flow with PKCE.
type OAuthUserProvider struct{ transport ports.Transport }

// NewOAuthUserProvider creates a provider for the user-consent OAuth flow.
func NewOAuthUserProvider(transport ports.Transport) *OAuthUserProvider {
	return &OAuthUserProvider{transport: transport}
}

func (p *OAuthUserProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthUser
}

func (p *OAuthUserProvider) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (p *OAuthUserProvider) Refresh(_ context.Context, _ core.Credential) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

var _ ports.AuthProvider = (*OAuthUserProvider)(nil)
