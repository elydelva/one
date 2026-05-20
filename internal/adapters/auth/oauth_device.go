package auth

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// OAuthDeviceProvider handles the OAuth 2.0 device code flow (headless-friendly).
type OAuthDeviceProvider struct{ transport ports.Transport }

// NewOAuthDeviceProvider creates a provider for the device code OAuth flow.
func NewOAuthDeviceProvider(transport ports.Transport) *OAuthDeviceProvider {
	return &OAuthDeviceProvider{transport: transport}
}

func (p *OAuthDeviceProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthDevice
}

func (p *OAuthDeviceProvider) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (p *OAuthDeviceProvider) Refresh(_ context.Context, _ core.Credential) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

var _ ports.AuthProvider = (*OAuthDeviceProvider)(nil)
