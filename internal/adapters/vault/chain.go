package vault

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ChainVault tries each vault in order for reads; writes go to all vaults.
// Default chain: EnvVar (read) → Keyring (read/write) → Age (read/write).
type ChainVault struct{ vaults []ports.Vault }

// NewChainVault creates a chain from the given vaults (tried left to right for reads).
func NewChainVault(vaults ...ports.Vault) *ChainVault {
	return &ChainVault{vaults: vaults}
}

func (v *ChainVault) Store(_ context.Context, _ core.AccountRef, _ core.Credential) error {
	return errors.New("not implemented")
}

func (v *ChainVault) Fetch(_ context.Context, _ core.AccountRef) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (v *ChainVault) Delete(_ context.Context, _ core.AccountRef) error {
	return errors.New("not implemented")
}

func (v *ChainVault) List(_ context.Context, _ core.ServiceID) ([]core.AccountRef, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Vault = (*ChainVault)(nil)
