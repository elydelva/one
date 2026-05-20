package vault

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// EnvVarVault reads credentials from environment variables (ONE_CREDS_<SVC>_<ACCOUNT>=<json>).
// Read-only: Store/Delete/List are unsupported.
type EnvVarVault struct{}

// NewEnvVarVault creates a vault that reads from environment variables.
func NewEnvVarVault() *EnvVarVault { return &EnvVarVault{} }

func (v *EnvVarVault) Store(_ context.Context, _ core.AccountRef, _ core.Credential) error {
	return errors.New("not implemented")
}

func (v *EnvVarVault) Fetch(_ context.Context, _ core.AccountRef) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (v *EnvVarVault) Delete(_ context.Context, _ core.AccountRef) error {
	return errors.New("env var vault is read-only")
}

func (v *EnvVarVault) List(_ context.Context, _ core.ServiceID) ([]core.AccountRef, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Vault = (*EnvVarVault)(nil)
