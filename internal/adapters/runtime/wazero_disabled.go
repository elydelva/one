//go:build nowasm

package runtime

import (
	"context"
	"fmt"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// HostAPIVersion is published for catalog tools even when the runtime is disabled.
const HostAPIVersion = 1

// HandlerResolver kept for type parity with the wasm build.
type HandlerResolver interface {
	ReadHandler(ctx context.Context, svc core.ServiceID, file string) ([]byte, error)
}

// WazeroRuntime in nowasm builds always refuses execution.
type WazeroRuntime struct{}

// NewWazeroRuntime returns a no-op runtime that rejects handler-backed actions.
func NewWazeroRuntime(_ any, _ any, _ any, _ any, _ HandlerResolver, _ string) *WazeroRuntime {
	return &WazeroRuntime{}
}

func (r *WazeroRuntime) Execute(_ context.Context, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
	return ports.ExecuteResult{}, core.ErrUnsupportedRuntime{
		Action: req.Action.ID,
		Reason: fmt.Sprintf("binary built with -tags=nowasm"),
	}
}

var _ ports.Runtime = (*WazeroRuntime)(nil)

// FSHandlerResolver kept as no-op in nowasm builds.
type FSHandlerResolver struct{}

func NewFSHandlerResolver(_ string) *FSHandlerResolver { return &FSHandlerResolver{} }

func (r *FSHandlerResolver) ReadHandler(_ context.Context, _ core.ServiceID, _ string) ([]byte, error) {
	return nil, fmt.Errorf("nowasm build: no handler resolver")
}
