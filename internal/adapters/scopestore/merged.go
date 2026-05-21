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

// LocalScopeFileName is the developer-local override read alongside .onerc.yaml.
const LocalScopeFileName = ".onerc.local.yaml"

// MergedScopeStore loads .onerc.yaml (base) and merges an optional
// .onerc.local.yaml (per-developer override). Local can only restrict the
// base scope: any allow not present in base is dropped; deny entries are unioned.
type MergedScopeStore struct {
	base ports.ScopeStore
}

// NewMergedScopeStore wraps a base scope store with local-override merging.
func NewMergedScopeStore(base ports.ScopeStore) *MergedScopeStore {
	return &MergedScopeStore{base: base}
}

func (s *MergedScopeStore) Load(dir string) (core.Scope, error) {
	baseScope, err := s.base.Load(dir)
	if err != nil {
		return core.Scope{}, err
	}
	local, err := loadLocal(dir)
	if err != nil {
		return core.Scope{}, err
	}
	if local == nil {
		return baseScope, nil
	}
	return mergeRestrictive(baseScope, *local), nil
}

func loadLocal(dir string) (*core.Scope, error) {
	path := filepath.Join(dir, LocalScopeFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var doc yamlScope
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, core.ErrInvalidScopeFile{Path: path, Reason: err.Error()}
	}
	if doc.Version != 1 {
		return nil, core.ErrInvalidScopeFile{Path: path, Reason: "unsupported version"}
	}
	out := core.Scope{Version: 1, Services: make(map[core.ServiceID]core.ServiceScope, len(doc.Services))}
	for svc, sdoc := range doc.Services {
		ss := core.ServiceScope{Account: core.AccountAlias(sdoc.Account)}
		for _, p := range sdoc.Allow {
			pat, err := core.ParsePermissionPattern(p)
			if err != nil {
				return nil, core.ErrInvalidScopeFile{Path: path, Reason: err.Error()}
			}
			ss.Allow = append(ss.Allow, pat)
		}
		for _, p := range sdoc.Deny {
			pat, err := core.ParsePermissionPattern(p)
			if err != nil {
				return nil, core.ErrInvalidScopeFile{Path: path, Reason: err.Error()}
			}
			ss.Deny = append(ss.Deny, pat)
		}
		out.Services[core.ServiceID(svc)] = ss
	}
	return &out, nil
}

// mergeRestrictive applies local on top of base: keep only allows present in base,
// union deny entries, override account when local sets one.
func mergeRestrictive(base, local core.Scope) core.Scope {
	out := core.Scope{Version: 1, Services: make(map[core.ServiceID]core.ServiceScope, len(base.Services))}
	for id, bss := range base.Services {
		lss, hasLocal := local.Services[id]
		merged := core.ServiceScope{
			Account: bss.Account,
			Allow:   append([]core.PermissionPattern(nil), bss.Allow...),
			Deny:    append([]core.PermissionPattern(nil), bss.Deny...),
		}
		if hasLocal {
			if lss.Account != "" {
				merged.Account = lss.Account
			}
			merged.Allow = intersectPaths(bss.Allow, lss.Allow)
			merged.Deny = unionPaths(bss.Deny, lss.Deny)
		}
		out.Services[id] = merged
	}
	return out
}

func intersectPaths(a, b []core.PermissionPattern) []core.PermissionPattern {
	if len(b) == 0 {
		return a
	}
	set := map[core.PermissionPattern]struct{}{}
	for _, p := range a {
		set[p] = struct{}{}
	}
	var out []core.PermissionPattern
	for _, p := range b {
		if _, ok := set[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func unionPaths(a, b []core.PermissionPattern) []core.PermissionPattern {
	seen := map[core.PermissionPattern]struct{}{}
	out := make([]core.PermissionPattern, 0, len(a)+len(b))
	for _, p := range a {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range b {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

var _ ports.ScopeStore = (*MergedScopeStore)(nil)
