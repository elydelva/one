package scopestore

import (
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// FileScopeStore loads the scope from .onerc.yaml in the given project directory.
type FileScopeStore struct{}

// NewFileScopeStore creates a scope store that reads .onerc.yaml files.
func NewFileScopeStore() *FileScopeStore { return &FileScopeStore{} }

func (s *FileScopeStore) Load(_ string) (core.Scope, error) {
	return core.Scope{}, errors.New("not implemented")
}

var _ ports.ScopeStore = (*FileScopeStore)(nil)
