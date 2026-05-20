package app

import (
	"context"
	"testing"

	"elydelva/one/internal/adapters/scopestore"
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

func TestListCapabilities_RejectsNonScopeMode(t *testing.T) {
	s := scopestore.NewFileScopeStore()
	uc := NewListCapabilities(nil, s)
	_, err := uc.Run(context.Background(), CapabilitiesInput{ScopeOnly: false, ProjectDir: t.TempDir()})
	if err == nil {
		t.Errorf("expected error for non-scope-only mode in v0.1")
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
