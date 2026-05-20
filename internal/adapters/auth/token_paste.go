package auth

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// TokenPasteProvider prompts the user to paste a Personal Access Token.
type TokenPasteProvider struct{}

// NewTokenPasteProvider creates a PAT provider.
func NewTokenPasteProvider() *TokenPasteProvider { return &TokenPasteProvider{} }

func (p *TokenPasteProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderPAT
}

func (p *TokenPasteProvider) Login(_ context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	tok, err := promptSecret("Paste your token for " + string(svc) + " (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	return core.Credential{
		Provider:    core.ProviderPAT,
		Service:     svc,
		Account:     alias,
		AccessToken: tok,
	}, nil
}

func (p *TokenPasteProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil // PATs don't expire
}

var _ ports.AuthProvider = (*TokenPasteProvider)(nil)
