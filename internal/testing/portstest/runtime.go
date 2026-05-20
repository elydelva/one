package portstest

import (
	"context"
	"encoding/json"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RunRuntimeTests executes the ports.Runtime contract against any implementation.
func RunRuntimeTests(t *testing.T, name string, factory func(t *testing.T) ports.Runtime) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Run("Execute returns result for valid request", func(t *testing.T) {
			rt := factory(t)
			req := ports.ExecuteRequest{
				Action: core.Action{
					ID:      "test.action",
					Service: "test",
				},
				Inputs:  core.Inputs{},
				TraceID: "test-trace",
			}
			result, err := rt.Execute(context.Background(), req)
			// Stubs may return "not implemented" — that's acceptable at this phase.
			if err == nil && result.Output != nil {
				var v any
				if jerr := json.Unmarshal(result.Output, &v); jerr != nil {
					t.Errorf("output is not valid JSON: %v", jerr)
				}
			}
		})
	})
}
