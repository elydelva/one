package fake

import (
	"context"
	"sort"
	"sync"
	"time"

	"elydelva/one/internal/core"
)

// Audit is an in-memory fake Audit implementation for tests.
type Audit struct {
	mu     sync.Mutex
	Events []core.AuditEvent
}

// NewAudit creates a fake audit log.
func NewAudit() *Audit { return &Audit{} }

// Record appends an event in memory.
func (a *Audit) Record(_ context.Context, ev core.AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	a.Events = append(a.Events, ev)
	return nil
}

// Read returns matching events newest-first.
func (a *Audit) Read(_ context.Context, f core.AuditFilter) ([]core.AuditEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := append([]core.AuditEvent(nil), a.Events...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].At.After(cp[j].At) })
	var out []core.AuditEvent
	for _, ev := range cp {
		if !f.Since.IsZero() && ev.At.Before(f.Since) {
			continue
		}
		if f.Service != "" && ev.Service != f.Service {
			continue
		}
		if f.Kind != "" && ev.Kind != f.Kind {
			continue
		}
		if f.TraceID != "" && ev.TraceID != f.TraceID {
			continue
		}
		out = append(out, ev)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	return out, nil
}
