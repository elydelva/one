package vault

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// KeyringVault stores credentials in the OS native keychain (macOS Keychain, Linux Secret Service, Windows CredMgr).
type KeyringVault struct{ service string }

// NewKeyringVault creates a vault using the given keychain service name.
func NewKeyringVault(service string) *KeyringVault { return &KeyringVault{service: service} }

func (v *KeyringVault) Store(_ context.Context, _ core.AccountRef, _ core.Credential) error {
	return errors.New("not implemented")
}

func (v *KeyringVault) Fetch(_ context.Context, _ core.AccountRef) (core.Credential, error) {
	return core.Credential{}, errors.New("not implemented")
}

func (v *KeyringVault) Delete(_ context.Context, _ core.AccountRef) error {
	return errors.New("not implemented")
}

func (v *KeyringVault) List(_ context.Context, _ core.ServiceID) ([]core.AccountRef, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Vault = (*KeyringVault)(nil)
