package runtime

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// WazeroRuntime executes WASM handlers in a sandboxed wazero environment.
type WazeroRuntime struct {
	transport ports.Transport
	crypto    ports.Crypto
	clock     ports.Clock
	log       ports.Logger
}

// NewWazeroRuntime creates a WASM runtime with the given host function implementations.
func NewWazeroRuntime(transport ports.Transport, crypto ports.Crypto, clock ports.Clock, log ports.Logger) *WazeroRuntime {
	return &WazeroRuntime{transport: transport, crypto: crypto, clock: clock, log: log}
}

func (r *WazeroRuntime) Execute(_ context.Context, _ ports.ExecuteRequest) (ports.ExecuteResult, error) {
	return ports.ExecuteResult{}, errors.New("not implemented")
}

var _ ports.Runtime = (*WazeroRuntime)(nil)
