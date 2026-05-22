package catalog

import (
	"context"
	"testing"

	"elydelva/one/internal/core"
)

func TestTaggedCatalog_StampsSource(t *testing.T) {
	embed := NewCatalogEmbed()
	tagged := NewTaggedCatalog(embed, "tap:acme/extras")

	svc, err := tagged.GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if svc.Source != "tap:acme/extras" {
		t.Errorf("service.Source = %q", svc.Source)
	}
	if len(svc.Actions) == 0 {
		t.Fatal("no actions")
	}
	for _, a := range svc.Actions {
		if a.Source != "tap:acme/extras" {
			t.Errorf("action %s Source = %q", a.ID, a.Source)
		}
	}

	act, err := tagged.GetAction(context.Background(), "github", "issues.list")
	if err != nil {
		t.Fatalf("get action: %v", err)
	}
	if act.Source != "tap:acme/extras" {
		t.Errorf("GetAction Source = %q", act.Source)
	}

	list, err := tagged.ListServices(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, s := range list {
		if s.Source != "tap:acme/extras" {
			t.Errorf("list svc %s Source = %q", s.ID, s.Source)
		}
	}
}

func TestTaggedCatalog_DoesNotLeakOnError(t *testing.T) {
	embed := NewCatalogEmbed()
	tagged := NewTaggedCatalog(embed, "tap:x/y")
	_, err := tagged.GetService(context.Background(), "nope")
	if _, ok := err.(core.ErrUnknownService); !ok {
		t.Errorf("expected ErrUnknownService, got %T", err)
	}
}
