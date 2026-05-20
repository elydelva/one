package app

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// LockInput holds parameters for the LockScope use case.
type LockInput struct{ ProjectDir string }

// LockScope freezes the catalog service versions in .onerc.lock.
type LockScope struct {
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewLockScope creates a LockScope use case.
func NewLockScope(catalog ports.Catalog, scope ports.ScopeStore) *LockScope {
	return &LockScope{catalog: catalog, scope: scope}
}

// Run writes the lock file.
func (uc *LockScope) Run(_ context.Context, _ LockInput) error {
	return errors.New("not implemented")
}
