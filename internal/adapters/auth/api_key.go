package auth

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// APIKeyProvider prompts for an API key (no expiry, no refresh).
type APIKeyProvider struct{}

// NewAPIKeyProvider creates a provider for API key authentication.
func NewAPIKeyProvider() *APIKeyProvider { return &APIKeyProvider{} }

func (p *APIKeyProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderAPIKey
}

func (p *APIKeyProvider) Login(_ context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	tok, err := promptSecret("Paste your API key for " + string(svc) + " (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	return core.Credential{
		Provider:    core.ProviderAPIKey,
		Service:     svc,
		Account:     alias,
		AccessToken: tok,
	}, nil
}

func (p *APIKeyProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil // API keys don't expire
}

var _ ports.AuthProvider = (*APIKeyProvider)(nil)
