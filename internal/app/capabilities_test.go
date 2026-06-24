package app

import (
	"context"
	"testing"

	"elydelva/one/internal/adapters/scopestore"
	"elydelva/one/internal/core"
	"elydelva/one/internal/testing/fake"
)

func TestListCapabilities_ScopeOnly(t *testing.T) {
	s := scopestore.NewFileScopeStore()
	dir := t.TempDir()
	add := NewAddScope(s, s)
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir})
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.*", ProjectDir: dir})
	_ = add.Run(context.Background(), AddScopeInput{Service: "notion", Permission: "pages.read", ProjectDir: dir})

	uc := NewListCapabilities(nil, s)
	got, err := uc.Run(context.Background(), CapabilitiesInput{ScopeOnly: true, ProjectDir: dir})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(got.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(got.Services))
	}
	if got.Services[0].ID != "github" || got.Services[1].ID != "notion" {
		t.Errorf("sorted order broken: %+v", got.Services)
	}
	if len(got.Services[0].Actions) != 2 {
		t.Errorf("github actions = %+v", got.Services[0].Actions)
	}
}

func TestListCapabilities_CatalogBacked(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{
		{ID: "github", Name: "GitHub", Actions: []core.Action{
			{ID: "issues.read", Permission: "issues.read"},
			{ID: "issues.create", Permission: "issues.create"},
		}},
		{ID: "notion", Name: "Notion", Actions: []core.Action{
			{ID: "pages.read", Permission: "pages.read"},
		}},
	})
	uc := NewListCapabilities(cat, scopestore.NewFileScopeStore())
	got, err := uc.Run(context.Background(), CapabilitiesInput{ScopeOnly: false})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(got.Services) != 2 {
		t.Fatalf("expected 2 services, got %d: %+v", len(got.Services), got.Services)
	}
	if got.Services[0].ID != "github" || len(got.Services[0].Actions) != 2 {
		t.Errorf("github actions = %+v", got.Services[0])
	}
}

func TestListCapabilities_CatalogBacked_FilterByService(t *testing.T) {
	cat := fake.NewCatalog([]core.Service{
		{ID: "github", Name: "GitHub", Actions: []core.Action{{ID: "issues.read", Permission: "issues.read"}}},
		{ID: "notion", Name: "Notion", Actions: []core.Action{{ID: "pages.read", Permission: "pages.read"}}},
	})
	uc := NewListCapabilities(cat, scopestore.NewFileScopeStore())
	got, err := uc.Run(context.Background(), CapabilitiesInput{ScopeOnly: false, Service: "notion"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(got.Services) != 1 || got.Services[0].ID != "notion" {
		t.Errorf("filter failed: %+v", got.Services)
	}
}

func TestListCapabilities_FilterByService(t *testing.T) {
	s := scopestore.NewFileScopeStore()
	dir := t.TempDir()
	add := NewAddScope(s, s)
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir})
	_ = add.Run(context.Background(), AddScopeInput{Service: "notion", Permission: "pages.read", ProjectDir: dir})

	uc := NewListCapabilities(nil, s)
	got, _ := uc.Run(context.Background(), CapabilitiesInput{ScopeOnly: true, Service: "github", ProjectDir: dir})
	if len(got.Services) != 1 || got.Services[0].ID != "github" {
		t.Errorf("filter failed: %+v", got.Services)
	}
}
