package app

import (
	"context"
	"errors"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// TraceInput holds parameters for the ShowTrace use case.
type TraceInput struct {
	Since   time.Duration
	Service string
	Kind    core.AuditKind
	TraceID string
	Limit   int
}

// ShowTrace reads audit log entries via the Audit port.
type ShowTrace struct {
	audit ports.Audit
	clock ports.Clock
}

// NewShowTrace creates a ShowTrace use case. audit may be nil (returns error on Run).
func NewShowTrace(audit ports.Audit, clock ports.Clock) *ShowTrace {
	return &ShowTrace{audit: audit, clock: clock}
}

// Run returns trace entries newest-first.
func (uc *ShowTrace) Run(ctx context.Context, in TraceInput) ([]core.AuditEvent, error) {
	if uc.audit == nil {
		return nil, errors.New("trace: no audit backend configured")
	}
	f := core.AuditFilter{
		Service: in.Service,
		Kind:    in.Kind,
		TraceID: in.TraceID,
		Limit:   in.Limit,
	}
	if in.Since > 0 {
		now := time.Now().UTC()
		if uc.clock != nil {
			now = uc.clock.Now().UTC()
		}
		f.Since = now.Add(-in.Since)
	}
	return uc.audit.Read(ctx, f)
}
