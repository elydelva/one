package app

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// emit records an audit event, swallowing errors (audit failure must never
// surface to callers — the use case already succeeded or failed on its own merit).
func emit(ctx context.Context, a ports.Audit, ev core.AuditEvent) {
	if a == nil {
		return
	}
	_ = a.Record(ctx, ev)
}

func outcomeOf(err error) core.AuditOutcome {
	if err == nil {
		return core.OutcomeOK
	}
	if _, ok := err.(core.ErrNotInScope); ok {
		return core.OutcomeDenied
	}
	return core.OutcomeError
}

func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
