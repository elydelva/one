package scopestore

import (
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// MergedScopeStore loads and merges a base scope with an optional local override (.onerc.local.yaml).
// Local overrides can only restrict (never expand) the base scope.
type MergedScopeStore struct {
	base  ports.ScopeStore
	local ports.ScopeStore
}

// NewMergedScopeStore creates a scope store that merges base and local scopes.
func NewMergedScopeStore(base, local ports.ScopeStore) *MergedScopeStore {
	return &MergedScopeStore{base: base, local: local}
}

func (s *MergedScopeStore) Load(_ string) (core.Scope, error) {
	return core.Scope{}, errors.New("not implemented")
}

var _ ports.ScopeStore = (*MergedScopeStore)(nil)
