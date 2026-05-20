package auth

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// AWSKeysProvider handles AWS credential-based authentication (access key + secret).
type AWSKeysProvider struct{}

// NewAWSKeysProvider creates a provider for AWS key authentication.
func NewAWSKeysProvider() *AWSKeysProvider { return &AWSKeysProvider{} }

func (p *AWSKeysProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderAWSKeys
}

func (p *AWSKeysProvider) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (p *AWSKeysProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil
}

var _ ports.AuthProvider = (*AWSKeysProvider)(nil)
