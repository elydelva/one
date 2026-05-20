package vault

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ChainVault tries each vault in order for reads; writes go to the first writable vault.
// Default chain: EnvVar (read-only) → Keyring (read/write).
type ChainVault struct{ vaults []ports.Vault }

// NewChainVault creates a chain from the given vaults (tried left to right for reads).
func NewChainVault(vaults ...ports.Vault) *ChainVault {
	return &ChainVault{vaults: vaults}
}

// Store writes to the first vault that doesn't return ErrReadOnly.
func (v *ChainVault) Store(ctx context.Context, ref core.AccountRef, cred core.Credential) error {
	var lastErr error
	for _, inner := range v.vaults {
		err := inner.Store(ctx, ref, cred)
		if err == nil {
			return nil
		}
		var ro core.ErrReadOnly
		if errors.As(err, &ro) {
			lastErr = err
			continue
		}
		return err
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("chain vault: no writable backend")
}

// Fetch returns the first hit. ErrNotAuthenticated and ErrNotSupported fall through; other errors propagate.
func (v *ChainVault) Fetch(ctx context.Context, ref core.AccountRef) (core.Credential, error) {
	for _, inner := range v.vaults {
		cred, err := inner.Fetch(ctx, ref)
		if err == nil {
			return cred, nil
		}
		var notAuth core.ErrNotAuthenticated
		var notSupp core.ErrNotSupported
		if errors.As(err, &notAuth) || errors.As(err, &notSupp) {
			continue
		}
		return core.Credential{}, err
	}
	return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
}

// Delete tries every writable vault, ignoring read-only and not-found errors.
func (v *ChainVault) Delete(ctx context.Context, ref core.AccountRef) error {
	deleted := false
	for _, inner := range v.vaults {
		err := inner.Delete(ctx, ref)
		if err == nil {
			deleted = true
			continue
		}
		var ro core.ErrReadOnly
		var notAuth core.ErrNotAuthenticated
		if errors.As(err, &ro) || errors.As(err, &notAuth) {
			continue
		}
		return err
	}
	if !deleted {
		return core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	return nil
}

// List unions results from all vaults, deduped. ErrNotSupported is ignored.
func (v *ChainVault) List(ctx context.Context, svc core.ServiceID) ([]core.AccountRef, error) {
	seen := map[core.AccountRef]struct{}{}
	var out []core.AccountRef
	for _, inner := range v.vaults {
		refs, err := inner.List(ctx, svc)
		if err != nil {
			var notSupp core.ErrNotSupported
			if errors.As(err, &notSupp) {
				continue
			}
			return nil, err
		}
		for _, r := range refs {
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			out = append(out, r)
		}
	}
	return out, nil
}

var _ ports.Vault = (*ChainVault)(nil)
