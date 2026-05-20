package portstest

import (
	"context"
	"encoding/json"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RuntimeFactory returns a runtime under test plus the action it knows how to execute.
type RuntimeFactory func(t *testing.T) (ports.Runtime, core.Action, core.Credential)

// RunRuntimeTests executes the ports.Runtime contract.
// The factory must wire the runtime to a fakeapi server such that executing the
// returned action with the returned credential produces a JSON response.
func RunRuntimeTests(t *testing.T, name string, factory RuntimeFactory) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Run("Execute returns JSON output", func(t *testing.T) {
			rt, action, cred := factory(t)
			req := ports.ExecuteRequest{
				Action:     action,
				Inputs:     core.Inputs{},
				Credential: cred,
				TraceID:    "t-1",
			}
			result, err := rt.Execute(context.Background(), req)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if len(result.Output) == 0 {
				t.Fatalf("expected output, got empty")
			}
			var v any
			if err := json.Unmarshal(result.Output, &v); err != nil {
				t.Errorf("output is not valid JSON: %v\n%s", err, result.Output)
			}
		})

		t.Run("Execute records HTTP calls in audit trail", func(t *testing.T) {
			rt, action, cred := factory(t)
			result, err := rt.Execute(context.Background(), ports.ExecuteRequest{
				Action: action, Credential: cred, Inputs: core.Inputs{}, TraceID: "t-2",
			})
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if len(result.Calls) == 0 {
				t.Errorf("expected at least one Call recorded")
			}
		})

		t.Run("Execute with DryRun skips network", func(t *testing.T) {
			rt, action, cred := factory(t)
			result, err := rt.Execute(context.Background(), ports.ExecuteRequest{
				Action: action, Credential: cred, Inputs: core.Inputs{}, DryRun: true, TraceID: "t-3",
			})
			if err != nil {
				t.Fatalf("dry-run: %v", err)
			}
			if len(result.Output) == 0 {
				t.Errorf("dry-run should return a planned-request preview")
			}
		})
	})
}
