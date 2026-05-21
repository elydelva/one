package audit

import (
	"context"
	"testing"
	"time"

	"elydelva/one/internal/core"
)

func TestNDJSONAudit_RecordRead(t *testing.T) {
	dir := t.TempDir()
	a := NewNDJSONAudit(dir, time.Hour)
	ctx := context.Background()
	t0 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

	evs := []core.AuditEvent{
		{At: t0, Kind: core.AuditLogin, Service: "github", Account: "default", Outcome: core.OutcomeOK},
		{At: t0.Add(1 * time.Minute), Kind: core.AuditExec, Service: "github", Action: "issues.list", Outcome: core.OutcomeOK},
		{At: t0.Add(2 * time.Minute), Kind: core.AuditExec, Service: "linear", Action: "issues.create", Outcome: core.OutcomeError, Err: "scope denied"},
	}
	for _, e := range evs {
		if err := a.Record(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	got, err := a.Read(ctx, core.AuditFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events got %d", len(got))
	}
	// newest first
	if got[0].Service != "linear" {
		t.Fatalf("expected newest first, got %+v", got[0])
	}
}

func TestNDJSONAudit_Filter(t *testing.T) {
	dir := t.TempDir()
	a := NewNDJSONAudit(dir, time.Hour)
	ctx := context.Background()
	now := time.Now().UTC()
	_ = a.Record(ctx, core.AuditEvent{At: now, Kind: core.AuditExec, Service: "github", Outcome: core.OutcomeOK, TraceID: "t1"})
	_ = a.Record(ctx, core.AuditEvent{At: now, Kind: core.AuditLogin, Service: "linear", Outcome: core.OutcomeOK, TraceID: "t2"})

	got, _ := a.Read(ctx, core.AuditFilter{Service: "github"})
	if len(got) != 1 || got[0].Service != "github" {
		t.Fatalf("service filter failed: %+v", got)
	}
	got, _ = a.Read(ctx, core.AuditFilter{Kind: core.AuditLogin})
	if len(got) != 1 || got[0].Kind != core.AuditLogin {
		t.Fatalf("kind filter failed: %+v", got)
	}
	got, _ = a.Read(ctx, core.AuditFilter{TraceID: "t1"})
	if len(got) != 1 || got[0].TraceID != "t1" {
		t.Fatalf("traceID filter failed: %+v", got)
	}
	got, _ = a.Read(ctx, core.AuditFilter{Limit: 1})
	if len(got) != 1 {
		t.Fatalf("limit failed: %+v", got)
	}
}

func TestNDJSONAudit_DefaultsAt(t *testing.T) {
	dir := t.TempDir()
	a := NewNDJSONAudit(dir, 0)
	if err := a.Record(context.Background(), core.AuditEvent{Kind: core.AuditLogin}); err != nil {
		t.Fatal(err)
	}
	got, _ := a.Read(context.Background(), core.AuditFilter{})
	if len(got) != 1 || got[0].At.IsZero() {
		t.Fatalf("expected At populated: %+v", got)
	}
}
