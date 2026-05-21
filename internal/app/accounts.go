package app

import (
	"context"
	"fmt"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ListAccounts returns the accounts registered in the vault for a service.
type ListAccounts struct{ vault ports.Vault }

// NewListAccounts creates the use case.
func NewListAccounts(v ports.Vault) *ListAccounts { return &ListAccounts{vault: v} }

// Run lists accounts for the given service.
func (uc *ListAccounts) Run(ctx context.Context, svc string) ([]core.AccountRef, error) {
	if svc == "" {
		return nil, core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	return uc.vault.List(ctx, core.ServiceID(svc))
}

// Rotate re-runs the login flow for an account, replacing its credential.
type Rotate struct {
	login *Login
}

// NewRotate creates the rotate use case.
func NewRotate(login *Login) *Rotate { return &Rotate{login: login} }

// Run rotates the credential for the given account.
func (uc *Rotate) Run(ctx context.Context, in LoginInput) error {
	if in.Service == "" || in.Account == "" {
		return fmt.Errorf("rotate: service+account required")
	}
	return uc.login.Run(ctx, in)
}

// ForceRefresh runs the refresh path unconditionally for the given account.
type ForceRefresh struct {
	vault   ports.Vault
	refresh *RefreshIfNeeded
}

// NewForceRefresh creates the use case.
func NewForceRefresh(v ports.Vault, r *RefreshIfNeeded) *ForceRefresh {
	return &ForceRefresh{vault: v, refresh: r}
}

// Run fetches the credential, drops its expiry to force a refresh, then runs RefreshIfNeeded.
func (uc *ForceRefresh) Run(ctx context.Context, svc, alias string) error {
	ref := core.AccountRef{Service: core.ServiceID(svc), Alias: core.AccountAlias(alias)}
	cred, err := uc.vault.Fetch(ctx, ref)
	if err != nil {
		return err
	}
	if cred.ExpiresAt == nil {
		return fmt.Errorf("refresh: credential has no expiry (provider %s)", cred.Provider)
	}
	// Force refresh by pretending it expired 1h ago.
	past := uc.refresh.clock.Now().Add(-time.Hour)
	cred.ExpiresAt = &past
	_, err = uc.refresh.Run(ctx, cred)
	return err
}
