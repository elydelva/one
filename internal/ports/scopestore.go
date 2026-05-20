package ports

import "elydelva/one/internal/core"

// ScopeStore loads the effective scope for a project directory.
// It merges .onerc.yaml and .onerc.local.yaml when both exist.
type ScopeStore interface {
	Load(dir string) (core.Scope, error)
}

// ScopeWriter persists a scope back to disk (for `one scope add/remove/init`).
// Adapters that are read-only (env, merged view) need not implement this.
type ScopeWriter interface {
	Save(dir string, scope core.Scope) error
}
