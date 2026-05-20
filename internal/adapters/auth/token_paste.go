package auth

import (
	"context"
	"errors"

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

func (p *TokenPasteProvider) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (p *TokenPasteProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil // PATs don't expire
}

var _ ports.AuthProvider = (*TokenPasteProvider)(nil)
