package runtime

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// DeclarativeRuntime executes actions defined purely in YAML (HTTP requests, no WASM).
type DeclarativeRuntime struct {
	transport ports.Transport
	clock     ports.Clock
}

// NewDeclarativeRuntime creates a runtime that executes declarative HTTP actions.
func NewDeclarativeRuntime(transport ports.Transport, clock ports.Clock) *DeclarativeRuntime {
	return &DeclarativeRuntime{transport: transport, clock: clock}
}

func (r *DeclarativeRuntime) Execute(_ context.Context, _ ports.ExecuteRequest) (ports.ExecuteResult, error) {
	return ports.ExecuteResult{}, errors.New("not implemented")
}

var _ ports.Runtime = (*DeclarativeRuntime)(nil)
