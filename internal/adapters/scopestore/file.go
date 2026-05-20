package scopestore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ScopeFileName is the canonical scope file at the root of a project.
const ScopeFileName = ".onerc.yaml"

// FileScopeStore loads and saves .onerc.yaml in a project directory.
type FileScopeStore struct{}

// NewFileScopeStore creates a scope store that reads and writes .onerc.yaml files.
func NewFileScopeStore() *FileScopeStore { return &FileScopeStore{} }

// yamlScope is the on-disk shape (map order preserved).
type yamlScope struct {
	Version  int                       `yaml:"version"`
	Services map[string]yamlServiceDoc `yaml:"services"`
}

type yamlServiceDoc struct {
	Account string   `yaml:"account,omitempty"`
	Allow   []string `yaml:"allow,omitempty"`
	Deny    []string `yaml:"deny,omitempty"`
}

func (s *FileScopeStore) Load(dir string) (core.Scope, error) {
	path := filepath.Join(dir, ScopeFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return core.Scope{Version: 1, Services: map[core.ServiceID]core.ServiceScope{}}, nil
		}
		return core.Scope{}, err
	}

	var doc yamlScope
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return core.Scope{}, fmt.Errorf("parse %s: %w", path, err)
	}

	if doc.Version != 1 {
		return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: fmt.Sprintf("unsupported version %d (expected 1)", doc.Version)}
	}

	out := core.Scope{Version: 1, Services: make(map[core.ServiceID]core.ServiceScope, len(doc.Services))}
	for svc, sdoc := range doc.Services {
		ss := core.ServiceScope{Account: core.AccountAlias(sdoc.Account)}
		for _, p := range sdoc.Allow {
			pat, err := core.ParsePermissionPattern(p)
			if err != nil {
				return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: fmt.Sprintf("services.%s.allow[%q]: %s", svc, p, err.Error())}
			}
			ss.Allow = append(ss.Allow, pat)
		}
		for _, p := range sdoc.Deny {
			pat, err := core.ParsePermissionPattern(p)
			if err != nil {
				return core.Scope{}, core.ErrInvalidScopeFile{Path: path, Reason: fmt.Sprintf("services.%s.deny[%q]: %s", svc, p, err.Error())}
			}
			ss.Deny = append(ss.Deny, pat)
		}
		out.Services[core.ServiceID(svc)] = ss
	}
	return out, nil
}

// Save writes the scope to .onerc.yaml atomically (tmp + rename).
// File mode is 0644 since the file is meant to be committed.
func (s *FileScopeStore) Save(dir string, scope core.Scope) error {
	if scope.Version == 0 {
		scope.Version = 1
	}
	doc := yamlScope{
		Version:  scope.Version,
		Services: map[string]yamlServiceDoc{},
	}
	for svc, ss := range scope.Services {
		sdoc := yamlServiceDoc{Account: string(ss.Account)}
		for _, p := range ss.Allow {
			sdoc.Allow = append(sdoc.Allow, string(p))
		}
		for _, p := range ss.Deny {
			sdoc.Deny = append(sdoc.Deny, string(p))
		}
		doc.Services[string(svc)] = sdoc
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, ScopeFileName)
	tmp, err := os.CreateTemp(dir, ".onerc.yaml.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

var (
	_ ports.ScopeStore  = (*FileScopeStore)(nil)
	_ ports.ScopeWriter = (*FileScopeStore)(nil)
)
