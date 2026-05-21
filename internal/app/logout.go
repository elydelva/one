package app

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// LogoutInput holds parameters for the Logout use case.
type LogoutInput struct {
	Service string
	Account string
}

// Logout removes credentials for a service account from the vault.
type Logout struct {
	vault ports.Vault
	log   ports.Logger
	audit ports.Audit
}

// NewLogout creates a Logout use case.
func NewLogout(vault ports.Vault, log ports.Logger) *Logout {
	return &Logout{vault: vault, log: log}
}

// WithAudit installs an audit recorder.
func (uc *Logout) WithAudit(a ports.Audit) *Logout { uc.audit = a; return uc }

// Run removes the credential. Idempotent: missing credentials are not an error.
func (uc *Logout) Run(ctx context.Context, in LogoutInput) (rerr error) {
	if in.Service == "" {
		return core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	alias := core.AccountAlias(in.Account)
	if alias == "" {
		alias = "default"
	}
	defer func() {
		emit(ctx, uc.audit, core.AuditEvent{
			Kind: core.AuditLogout, Service: in.Service, Account: string(alias),
			Outcome: outcomeOf(rerr), Err: errMsg(rerr),
		})
	}()
	ref := core.AccountRef{Service: core.ServiceID(in.Service), Alias: alias}
	err := uc.vault.Delete(ctx, ref)
	if err != nil {
		var notAuth core.ErrNotAuthenticated
		if errors.As(err, &notAuth) {
			uc.log.Info("logout no-op (not authenticated)", "service", in.Service, "account", string(alias))
			return nil
		}
		return err
	}
	uc.log.Info("logout succeeded", "service", in.Service, "account", string(alias))
	return nil
}
