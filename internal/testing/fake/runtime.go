package fake

import (
	"context"
	"encoding/json"
	"fmt"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

type runtimeKey struct {
	svc    core.ServiceID
	action core.ActionID
}

// Runtime is an in-memory implementation of ports.Runtime for testing.
type Runtime struct {
	responses map[runtimeKey]json.RawMessage
	errors    map[runtimeKey]error
	calls     map[runtimeKey][]ports.HTTPCall
}

// NewRuntime creates an empty fake runtime.
func NewRuntime() *Runtime {
	return &Runtime{
		responses: map[runtimeKey]json.RawMessage{},
		errors:    map[runtimeKey]error{},
		calls:     map[runtimeKey][]ports.HTTPCall{},
	}
}

// SetResponse configures a canned response for the given service+action.
func (r *Runtime) SetResponse(svc core.ServiceID, action core.ActionID, output json.RawMessage) {
	r.responses[runtimeKey{svc, action}] = output
}

// SetError configures a canned error for the given service+action.
func (r *Runtime) SetError(svc core.ServiceID, action core.ActionID, err error) {
	r.errors[runtimeKey{svc, action}] = err
}

// SetCalls configures the HTTP calls reported for the given service+action.
func (r *Runtime) SetCalls(svc core.ServiceID, action core.ActionID, calls ...ports.HTTPCall) {
	r.calls[runtimeKey{svc, action}] = calls
}

func (r *Runtime) Execute(_ context.Context, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
	k := runtimeKey{req.Action.Service, req.Action.ID}
	if err, ok := r.errors[k]; ok {
		return ports.ExecuteResult{Calls: r.calls[k]}, err
	}
	if out, ok := r.responses[k]; ok {
		return ports.ExecuteResult{Output: out, Calls: r.calls[k]}, nil
	}
	return ports.ExecuteResult{}, fmt.Errorf("fake runtime: no response configured for %s.%s", req.Action.Service, req.Action.ID)
}

var _ ports.Runtime = (*Runtime)(nil)
