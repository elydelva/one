package app

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// LoginInput holds parameters for the Login use case.
type LoginInput struct {
	Service string
	Account string
}

// Login authenticates a user for a service and stores the credential in the vault.
type Login struct {
	vault   ports.Vault
	catalog ports.Catalog
	auth    []ports.AuthProvider
	log     ports.Logger
}

// NewLogin creates a Login use case.
func NewLogin(vault ports.Vault, catalog ports.Catalog, auth []ports.AuthProvider, log ports.Logger) *Login {
	return &Login{vault: vault, catalog: catalog, auth: auth, log: log}
}

// Run starts the login flow.
func (uc *Login) Run(_ context.Context, _ LoginInput) error {
	return errors.New("not implemented")
}
