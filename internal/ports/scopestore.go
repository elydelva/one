package ports

import "elydelva/one/internal/core"

// ScopeStore loads the effective scope for a project directory.
// It merges .onerc.yaml and .onerc.local.yaml when both exist.
type ScopeStore interface {
	Load(dir string) (core.Scope, error)
}
