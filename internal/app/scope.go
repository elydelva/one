package app

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ShowScopeInput holds parameters for ShowScope.
type ShowScopeInput struct {
	Service    string
	ProjectDir string
}

// AddScopeInput holds parameters for AddScope.
type AddScopeInput struct {
	Service    string
	Permission string // pattern (e.g. "issues.*")
	Deny       bool   // when true, adds to deny instead of allow
	ProjectDir string
}

// RemoveScopeInput holds parameters for RemoveScope.
type RemoveScopeInput struct {
	Service    string
	Permission string
	ProjectDir string
}

// CheckScopeInput holds parameters for CheckScope.
type CheckScopeInput struct {
	Service    string
	Action     string // permission path (e.g. "issues.read")
	ProjectDir string
}

// CheckScopeOutput holds the permission check result.
type CheckScopeOutput struct {
	Allowed bool
	Reason  string
}

// ShowScope returns the current scope for a project.
type ShowScope struct{ store ports.ScopeStore }

// NewShowScope creates a ShowScope use case.
func NewShowScope(store ports.ScopeStore) *ShowScope { return &ShowScope{store: store} }

// Run returns the scope, optionally filtered to a single service.
func (uc *ShowScope) Run(_ context.Context, in ShowScopeInput) (core.Scope, error) {
	scope, err := uc.store.Load(in.ProjectDir)
	if err != nil {
		return core.Scope{}, err
	}
	if in.Service == "" {
		return scope, nil
	}
	svc := core.ServiceID(in.Service)
	filtered := core.Scope{Version: scope.Version, Services: map[core.ServiceID]core.ServiceScope{}}
	if ss, ok := scope.Services[svc]; ok {
		filtered.Services[svc] = ss
	}
	return filtered, nil
}

// AddScope adds a permission pattern to .onerc.yaml.
type AddScope struct {
	store  ports.ScopeStore
	writer ports.ScopeWriter
}

// NewAddScope creates an AddScope use case.
func NewAddScope(store ports.ScopeStore, writer ports.ScopeWriter) *AddScope {
	return &AddScope{store: store, writer: writer}
}

// Run adds the permission to allow (or deny if Deny=true). Idempotent.
func (uc *AddScope) Run(_ context.Context, in AddScopeInput) error {
	if in.Service == "" {
		return core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	pat, err := core.ParsePermissionPattern(in.Permission)
	if err != nil {
		return err
	}
	scope, err := uc.store.Load(in.ProjectDir)
	if err != nil {
		return err
	}
	if scope.Services == nil {
		scope.Services = map[core.ServiceID]core.ServiceScope{}
	}
	svc := core.ServiceID(in.Service)
	ss := scope.Services[svc]
	if in.Deny {
		if !containsPattern(ss.Deny, pat) {
			ss.Deny = append(ss.Deny, pat)
		}
	} else {
		if !containsPattern(ss.Allow, pat) {
			ss.Allow = append(ss.Allow, pat)
		}
	}
	scope.Services[svc] = ss
	return uc.writer.Save(in.ProjectDir, scope)
}

// RemoveScope removes a permission pattern from .onerc.yaml (both allow and deny).
type RemoveScope struct {
	store  ports.ScopeStore
	writer ports.ScopeWriter
}

// NewRemoveScope creates a RemoveScope use case.
func NewRemoveScope(store ports.ScopeStore, writer ports.ScopeWriter) *RemoveScope {
	return &RemoveScope{store: store, writer: writer}
}

// Run removes the permission. Idempotent.
func (uc *RemoveScope) Run(_ context.Context, in RemoveScopeInput) error {
	if in.Service == "" {
		return core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	pat, err := core.ParsePermissionPattern(in.Permission)
	if err != nil {
		return err
	}
	scope, err := uc.store.Load(in.ProjectDir)
	if err != nil {
		return err
	}
	svc := core.ServiceID(in.Service)
	ss, ok := scope.Services[svc]
	if !ok {
		return nil
	}
	ss.Allow = removePattern(ss.Allow, pat)
	ss.Deny = removePattern(ss.Deny, pat)
	scope.Services[svc] = ss
	return uc.writer.Save(in.ProjectDir, scope)
}

// CheckScope checks whether a permission is granted.
type CheckScope struct{ store ports.ScopeStore }

// NewCheckScope creates a CheckScope use case.
func NewCheckScope(store ports.ScopeStore) *CheckScope { return &CheckScope{store: store} }

// Run checks the permission and returns the rule that fired.
func (uc *CheckScope) Run(_ context.Context, in CheckScopeInput) (CheckScopeOutput, error) {
	if in.Service == "" {
		return CheckScopeOutput{}, core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	path, err := core.ParsePermissionPath(in.Action)
	if err != nil {
		return CheckScopeOutput{}, err
	}
	scope, err := uc.store.Load(in.ProjectDir)
	if err != nil {
		return CheckScopeOutput{}, err
	}
	perm := core.Permission{Service: core.ServiceID(in.Service), Path: path}
	allowed, reason := scope.AllowsWithReason(perm)
	return CheckScopeOutput{Allowed: allowed, Reason: reason}, nil
}

func containsPattern(haystack []core.PermissionPattern, needle core.PermissionPattern) bool {
	for _, p := range haystack {
		if p == needle {
			return true
		}
	}
	return false
}

func removePattern(haystack []core.PermissionPattern, needle core.PermissionPattern) []core.PermissionPattern {
	out := haystack[:0]
	for _, p := range haystack {
		if p != needle {
			out = append(out, p)
		}
	}
	return out
}
