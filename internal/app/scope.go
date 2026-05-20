package app

import (
	"context"
	"errors"

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
	Permission string
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
	Action     string
	ProjectDir string
}

// CheckScopeOutput holds the permission check result.
type CheckScopeOutput struct {
	Allowed bool
	Reason  string
}

// ShowScope returns the current scope for a project.
type ShowScope struct{ scope ports.ScopeStore }

// NewShowScope creates a ShowScope use case.
func NewShowScope(scope ports.ScopeStore) *ShowScope { return &ShowScope{scope: scope} }

// Run returns the scope.
func (uc *ShowScope) Run(_ context.Context, _ ShowScopeInput) (core.Scope, error) {
	return core.Scope{}, errors.New("not implemented")
}

// AddScope adds a permission to .onerc.yaml.
type AddScope struct{ scope ports.ScopeStore }

// NewAddScope creates an AddScope use case.
func NewAddScope(scope ports.ScopeStore) *AddScope { return &AddScope{scope: scope} }

// Run adds the permission.
func (uc *AddScope) Run(_ context.Context, _ AddScopeInput) error {
	return errors.New("not implemented")
}

// RemoveScope removes a permission from .onerc.yaml.
type RemoveScope struct{ scope ports.ScopeStore }

// NewRemoveScope creates a RemoveScope use case.
func NewRemoveScope(scope ports.ScopeStore) *RemoveScope { return &RemoveScope{scope: scope} }

// Run removes the permission.
func (uc *RemoveScope) Run(_ context.Context, _ RemoveScopeInput) error {
	return errors.New("not implemented")
}

// CheckScope checks whether a permission is granted.
type CheckScope struct{ scope ports.ScopeStore }

// NewCheckScope creates a CheckScope use case.
func NewCheckScope(scope ports.ScopeStore) *CheckScope { return &CheckScope{scope: scope} }

// Run checks the permission.
func (uc *CheckScope) Run(_ context.Context, _ CheckScopeInput) (CheckScopeOutput, error) {
	return CheckScopeOutput{}, errors.New("not implemented")
}
