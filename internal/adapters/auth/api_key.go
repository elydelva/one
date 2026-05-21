package auth

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// APIKeyProvider prompts for an API key (no expiry, no refresh).
// Optionally validates via AuthConfig.ValidateURL.
type APIKeyProvider struct {
	transport ports.Transport
	catalog   ports.Catalog
}

// NewAPIKeyProvider creates a provider for API key authentication.
func NewAPIKeyProvider() *APIKeyProvider { return &APIKeyProvider{} }

// WithValidation enables post-prompt validation via the service's validate_url.
func (p *APIKeyProvider) WithValidation(transport ports.Transport, catalog ports.Catalog) *APIKeyProvider {
	p.transport = transport
	p.catalog = catalog
	return p
}

func (p *APIKeyProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderAPIKey
}

func (p *APIKeyProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	tok, err := promptSecret("Paste your API key for " + string(svc) + " (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	cred := core.Credential{
		Provider:    core.ProviderAPIKey,
		Service:     svc,
		Account:     alias,
		AccessToken: tok,
	}
	if err := validateToken(ctx, p.transport, p.catalog, svc, core.ProviderAPIKey, cred); err != nil {
		return core.Credential{}, err
	}
	return cred, nil
}

func (p *APIKeyProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil
}

var _ ports.AuthProvider = (*APIKeyProvider)(nil)
