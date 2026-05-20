package runtime

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// Router dispatches to DeclarativeRuntime or WazeroRuntime based on whether the action has a handler.
// In v0.2 the WASM runtime is unavailable; passing nil here is OK — the router
// returns ErrUnsupportedRuntime for any handler-backed action.
type Router struct {
	declarative ports.Runtime
	wasm        ports.Runtime
}

// NewRouter creates a routing runtime (WASM if handler present, declarative otherwise).
func NewRouter(declarative, wasm ports.Runtime) *Router {
	return &Router{declarative: declarative, wasm: wasm}
}

func (r *Router) Execute(ctx context.Context, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
	if req.Action.IsDeclarative() {
		return r.declarative.Execute(ctx, req)
	}
	if r.wasm == nil {
		return ports.ExecuteResult{}, core.ErrUnsupportedRuntime{
			Action: req.Action.ID,
			Reason: "WASM handler runtime is not enabled in this build (v0.2)",
		}
	}
	return r.wasm.Execute(ctx, req)
}

var _ ports.Runtime = (*Router)(nil)
