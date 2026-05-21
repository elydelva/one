package app

import (
	"context"
	"fmt"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// LoginInput holds parameters for the Login use case.
type LoginInput struct {
	Service  string
	Account  string
	Provider core.ProviderKind // empty = default (PAT)
}

// Login authenticates a user for a service and stores the credential in the vault.
type Login struct {
	vault   ports.Vault
	catalog ports.Catalog
	auth    []ports.AuthProvider
	log     ports.Logger
	audit   ports.Audit
}

// NewLogin creates a Login use case.
func NewLogin(vault ports.Vault, catalog ports.Catalog, auth []ports.AuthProvider, log ports.Logger) *Login {
	return &Login{vault: vault, catalog: catalog, auth: auth, log: log}
}

// WithAudit installs an audit recorder. Events: LOGIN ok/error.
func (uc *Login) WithAudit(a ports.Audit) *Login { uc.audit = a; return uc }

// Run starts the login flow.
func (uc *Login) Run(ctx context.Context, in LoginInput) (err error) {
	if in.Service == "" {
		return core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	alias := core.AccountAlias(in.Account)
	if alias == "" {
		alias = "default"
	}
	kind := in.Provider
	if kind == "" {
		kind = core.ProviderPAT
	}
	defer func() {
		emit(ctx, uc.audit, core.AuditEvent{
			Kind: core.AuditLogin, Service: in.Service, Account: string(alias),
			Outcome: outcomeOf(err), Err: errMsg(err),
		})
	}()

	var provider ports.AuthProvider
	for _, p := range uc.auth {
		if p.Supports(kind) {
			provider = p
			break
		}
	}
	if provider == nil {
		return fmt.Errorf("no auth provider supports %s", kind)
	}

	cred, err := provider.Login(ctx, core.ServiceID(in.Service), alias)
	if err != nil {
		return err
	}
	if err := uc.vault.Store(ctx, cred.Ref(), cred); err != nil {
		return err
	}
	uc.log.Info("login succeeded", "service", in.Service, "account", string(alias), "provider", string(kind))
	return nil
}
