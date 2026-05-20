package app

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// DoctorCheck is the result of a single diagnostic check.
type DoctorCheck struct {
	Name    string
	Status  string
	Message string
}

// DoctorOutput holds all diagnostic results.
type DoctorOutput struct{ Checks []DoctorCheck }

// RunDoctor performs diagnostic checks on the local setup.
type RunDoctor struct {
	vault   ports.Vault
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewRunDoctor creates a RunDoctor use case.
func NewRunDoctor(vault ports.Vault, catalog ports.Catalog, scope ports.ScopeStore) *RunDoctor {
	return &RunDoctor{vault: vault, catalog: catalog, scope: scope}
}

// Run executes all checks.
func (uc *RunDoctor) Run(_ context.Context) (DoctorOutput, error) {
	return DoctorOutput{}, errors.New("not implemented")
}
