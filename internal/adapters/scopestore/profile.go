package scopestore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ProfileScopeStore swaps the scope file based on the ONE_PROFILE env var.
// When ONE_PROFILE=production, it loads `.onerc.production.yaml` if present;
// otherwise it falls back to the wrapped store (which usually reads .onerc.yaml).
type ProfileScopeStore struct {
	base ports.ScopeStore
}

// NewProfileScopeStore wraps a base store with profile-aware loading.
func NewProfileScopeStore(base ports.ScopeStore) *ProfileScopeStore {
	return &ProfileScopeStore{base: base}
}

func (s *ProfileScopeStore) Load(dir string) (core.Scope, error) {
	profile := os.Getenv("ONE_PROFILE")
	if profile == "" || profile == "default" {
		return s.base.Load(dir)
	}
	path := filepath.Join(dir, ".onerc."+profile+".yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // G703: path comes from trusted local config
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s.base.Load(dir)
		}
		return core.Scope{}, err
	}
	var doc yamlScope
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: err.Error()}
	}
	if doc.Version != 1 {
		return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: "unsupported version"}
	}
	out := core.Scope{Version: 1, Services: make(map[core.ServiceID]core.ServiceScope, len(doc.Services))}
	for svc, sdoc := range doc.Services {
		ss := core.ServiceScope{Account: core.AccountAlias(sdoc.Account)}
		for _, p := range sdoc.Allow {
			pat, err := core.ParsePermissionPattern(p)
			if err != nil {
				return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: err.Error()}
			}
			ss.Allow = append(ss.Allow, pat)
		}
		for _, p := range sdoc.Deny {
			pat, err := core.ParsePermissionPattern(p)
			if err != nil {
				return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: err.Error()}
			}
			ss.Deny = append(ss.Deny, pat)
		}
		out.Services[core.ServiceID(svc)] = ss
	}
	return out, nil
}

var _ ports.ScopeStore = (*ProfileScopeStore)(nil)
