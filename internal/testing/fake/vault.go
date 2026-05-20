package fake

import (
	"context"
	"sync"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// Vault is an in-memory implementation of ports.Vault for testing.
type Vault struct {
	mu    sync.Mutex
	creds map[core.AccountRef]core.Credential
}

// NewVault creates an empty in-memory vault.
func NewVault() *Vault {
	return &Vault{creds: map[core.AccountRef]core.Credential{}}
}

func (v *Vault) Store(_ context.Context, ref core.AccountRef, cred core.Credential) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.creds[ref] = cred
	return nil
}

func (v *Vault) Fetch(_ context.Context, ref core.AccountRef) (core.Credential, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if cred, ok := v.creds[ref]; ok {
		return cred, nil
	}
	return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
}

func (v *Vault) Delete(_ context.Context, ref core.AccountRef) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, ok := v.creds[ref]; !ok {
		return core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	delete(v.creds, ref)
	return nil
}

func (v *Vault) List(_ context.Context, svc core.ServiceID) ([]core.AccountRef, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	var refs []core.AccountRef
	for ref := range v.creds {
		if ref.Service == svc {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

var _ ports.Vault = (*Vault)(nil)
