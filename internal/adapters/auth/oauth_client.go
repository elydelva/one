package auth

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// OAuthClientProvider handles the OAuth 2.0 client credentials flow.
type OAuthClientProvider struct{ transport ports.Transport }

// NewOAuthClientProvider creates a provider for client credentials flow.
func NewOAuthClientProvider(transport ports.Transport) *OAuthClientProvider {
	return &OAuthClientProvider{transport: transport}
}

func (p *OAuthClientProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthClient
}

func (p *OAuthClientProvider) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (p *OAuthClientProvider) Refresh(_ context.Context, _ core.Credential) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

var _ ports.AuthProvider = (*OAuthClientProvider)(nil)
