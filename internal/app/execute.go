package app

import (
	"context"
	"encoding/json"
	"errors"

	"elydelva/one/internal/ports"
)

// ExecuteInput holds the parameters for the ExecuteAction use case.
type ExecuteInput struct {
	Service    string
	Action     string
	Inputs     map[string]any
	Account    string
	DryRun     bool
	ProjectDir string
}

// ExecuteOutput holds the result of a successful action execution.
type ExecuteOutput struct {
	Result  json.RawMessage
	TraceID string
	Calls   []ports.HTTPCall
}

// ExecuteAction orchestrates a full action execution: scope check, auth, run, audit.
type ExecuteAction struct {
	catalog ports.Catalog
	vault   ports.Vault
	runtime ports.Runtime
	scope   ports.ScopeStore
	auth    []ports.AuthProvider
	log     ports.Logger
	clock   ports.Clock
}

// NewExecuteAction creates an ExecuteAction use case.
func NewExecuteAction(
	catalog ports.Catalog,
	vault ports.Vault,
	runtime ports.Runtime,
	scope ports.ScopeStore,
	auth []ports.AuthProvider,
	log ports.Logger,
	clock ports.Clock,
) *ExecuteAction {
	return &ExecuteAction{
		catalog: catalog,
		vault:   vault,
		runtime: runtime,
		scope:   scope,
		auth:    auth,
		log:     log,
		clock:   clock,
	}
}

// Run executes the action.
func (uc *ExecuteAction) Run(_ context.Context, _ ExecuteInput) (ExecuteOutput, error) {
	return ExecuteOutput{}, errors.New("not implemented")
}
