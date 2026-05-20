package runtime

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// Router dispatches to DeclarativeRuntime or WazeroRuntime based on whether the action has a handler.
type Router struct {
	declarative ports.Runtime
	wasm        ports.Runtime
}

// NewRouter creates a routing runtime (WASM if handler present, declarative otherwise).
func NewRouter(declarative, wasm ports.Runtime) *Router {
	return &Router{declarative: declarative, wasm: wasm}
}

func (r *Router) Execute(_ context.Context, _ ports.ExecuteRequest) (ports.ExecuteResult, error) {
	return ports.ExecuteResult{}, errors.New("not implemented")
}

var _ ports.Runtime = (*Router)(nil)
