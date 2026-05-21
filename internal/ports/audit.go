package ports

import (
	"context"

	"elydelva/one/internal/core"
)

// Audit records and reads audit events. Implementations must never persist
// secrets — callers are expected to redact at the call site.
type Audit interface {
	Record(ctx context.Context, ev core.AuditEvent) error
	Read(ctx context.Context, f core.AuditFilter) ([]core.AuditEvent, error)
}
