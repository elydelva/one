package auth

import (
	"context"
	"errors"

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

func (p *APIKeyProvider) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (p *APIKeyProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil // API keys don't expire
}

var _ ports.AuthProvider = (*APIKeyProvider)(nil)
