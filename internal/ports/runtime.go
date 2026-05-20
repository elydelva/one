package ports

import (
	"context"
	"encoding/json"

	"elydelva/one/internal/core"
)

// HTTPCall records an outbound HTTP request made during action execution (for audit).
type HTTPCall struct {
	Method string
	URL    string
	Status int
}

// ExecuteRequest carries everything the runtime needs to run an action.
type ExecuteRequest struct {
	Action     core.Action
	Inputs     core.Inputs
	Credential core.Credential
	DryRun     bool
	TraceID    string
}

// ExecuteResult holds the action output and audit trail.
type ExecuteResult struct {
	Output json.RawMessage
	Calls  []HTTPCall
}

// Runtime executes an action (declarative or WASM).
type Runtime interface {
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
}
