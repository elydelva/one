package app

import (
	"context"
	"errors"

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
}

// NewLogout creates a Logout use case.
func NewLogout(vault ports.Vault, log ports.Logger) *Logout {
	return &Logout{vault: vault, log: log}
}

// Run removes the credential.
func (uc *Logout) Run(_ context.Context, _ LogoutInput) error {
	return errors.New("not implemented")
}
