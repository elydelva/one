package vault

import (
	"context"
	"errors"

	"github.com/zalando/go-keyring"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// KeyringVault stores credentials in the OS native keychain (macOS Keychain, Linux Secret Service, Windows CredMgr).
// Keychain layout: service = constructor arg (e.g. "one"), account = "<service>:<alias>" (e.g. "github:work").
type KeyringVault struct {
	service string
	// backend overrides for testing; nil = real keyring.
	set    func(service, user, password string) error
	get    func(service, user string) (string, error)
	delete func(service, user string) error
}

// NewKeyringVault creates a vault using the given keychain service name.
func NewKeyringVault(service string) *KeyringVault {
	return &KeyringVault{
		service: service,
		set:     keyring.Set,
		get:     keyring.Get,
		delete:  keyring.Delete,
	}
}

func (v *KeyringVault) keyringUser(ref core.AccountRef) string {
	return string(ref.Service) + ":" + string(ref.Alias)
}

func (v *KeyringVault) Store(_ context.Context, ref core.AccountRef, cred core.Credential) error {
	data, err := cred.MarshalForStorage()
	if err != nil {
		return err
	}
	return v.set(v.service, v.keyringUser(ref), string(data))
}

func (v *KeyringVault) Fetch(_ context.Context, ref core.AccountRef) (core.Credential, error) {
	raw, err := v.get(v.service, v.keyringUser(ref))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
		}
		return core.Credential{}, err
	}
	return core.UnmarshalCredentialFromStorage([]byte(raw))
}

func (v *KeyringVault) Delete(_ context.Context, ref core.AccountRef) error {
	err := v.delete(v.service, v.keyringUser(ref))
	if err != nil && errors.Is(err, keyring.ErrNotFound) {
		return core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	return err
}

// List is not supported by all keyring backends; we return ErrNotSupported and let
// callers union with EnvVarVault.List instead.
func (v *KeyringVault) List(_ context.Context, _ core.ServiceID) ([]core.AccountRef, error) {
	return nil, core.ErrNotSupported{Source: "keyring", Op: "list"}
}

var _ ports.Vault = (*KeyringVault)(nil)
