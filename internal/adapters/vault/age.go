package vault

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// AgeVault stores credentials in an age-encrypted file at ~/.one/vault.age.
type AgeVault struct{ path string }

// NewAgeVault creates a vault backed by the given age-encrypted file path.
func NewAgeVault(path string) *AgeVault { return &AgeVault{path: path} }

func (v *AgeVault) Store(_ context.Context, _ core.AccountRef, _ core.Credential) error {
	return errors.New("not implemented")
}

func (v *AgeVault) Fetch(_ context.Context, _ core.AccountRef) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (v *AgeVault) Delete(_ context.Context, _ core.AccountRef) error {
	return errors.New("not implemented")
}

func (v *AgeVault) List(_ context.Context, _ core.ServiceID) ([]core.AccountRef, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Vault = (*AgeVault)(nil)
