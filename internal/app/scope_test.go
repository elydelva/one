package app

import (
	"context"
	"testing"

	"elydelva/one/internal/adapters/scopestore"
	"elydelva/one/internal/core"
)

func newStore(t *testing.T) (*scopestore.FileScopeStore, string) {
	t.Helper()
	return scopestore.NewFileScopeStore(), t.TempDir()
}

func TestAddScope_AppendsAllow(t *testing.T) {
	s, dir := newStore(t)
	uc := NewAddScope(s, s)
	if err := uc.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir}); err != nil {
		t.Fatalf("add: %v", err)
	}
	got, _ := s.Load(dir)
	if len(got.Services["github"].Allow) != 1 {
		t.Errorf("allow = %+v", got.Services["github"].Allow)
	}
}

func TestAddScope_Idempotent(t *testing.T) {
	s, dir := newStore(t)
	uc := NewAddScope(s, s)
	in := AddScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir}
	_ = uc.Run(context.Background(), in)
	_ = uc.Run(context.Background(), in)
	got, _ := s.Load(dir)
	if len(got.Services["github"].Allow) != 1 {
		t.Errorf("expected 1 entry, got %+v", got.Services["github"].Allow)
	}
}

func TestAddScope_RefusesInvalidPattern(t *testing.T) {
	s, dir := newStore(t)
	uc := NewAddScope(s, s)
	err := uc.Run(context.Background(), AddScopeInput{Service: "github", Permission: "**", ProjectDir: dir})
	var bad core.ErrInvalidPattern
	if err == nil {
		t.Fatalf("expected error")
	}
	if !asInvalidPattern(err, &bad) {
		t.Errorf("expected ErrInvalidPattern, got %T: %v", err, err)
	}
}

func TestRemoveScope_RemovesAllowAndDeny(t *testing.T) {
	s, dir := newStore(t)
	add := NewAddScope(s, s)
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir})
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "repos.delete", Deny: true, ProjectDir: dir})

	rm := NewRemoveScope(s, s)
	if err := rm.Run(context.Background(), RemoveScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	got, _ := s.Load(dir)
	if len(got.Services["github"].Allow) != 0 {
		t.Errorf("allow not removed: %+v", got.Services["github"].Allow)
	}
	if len(got.Services["github"].Deny) != 1 {
		t.Errorf("deny tampered: %+v", got.Services["github"].Deny)
	}
}

func TestCheckScope_Allowed(t *testing.T) {
	s, dir := newStore(t)
	add := NewAddScope(s, s)
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.*", ProjectDir: dir})

	chk := NewCheckScope(s)
	out, err := chk.Run(context.Background(), CheckScopeInput{Service: "github", Action: "issues.read", ProjectDir: dir})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !out.Allowed {
		t.Errorf("expected allowed, got %+v", out)
	}
}

func TestCheckScope_DefaultDeny(t *testing.T) {
	s, dir := newStore(t)
	chk := NewCheckScope(s)
	out, _ := chk.Run(context.Background(), CheckScopeInput{Service: "github", Action: "issues.read", ProjectDir: dir})
	if out.Allowed {
		t.Errorf("expected denied by default")
	}
}

func TestShowScope_FiltersByService(t *testing.T) {
	s, dir := newStore(t)
	add := NewAddScope(s, s)
	_ = add.Run(context.Background(), AddScopeInput{Service: "github", Permission: "issues.read", ProjectDir: dir})
	_ = add.Run(context.Background(), AddScopeInput{Service: "notion", Permission: "pages.read", ProjectDir: dir})

	show := NewShowScope(s)
	got, _ := show.Run(context.Background(), ShowScopeInput{Service: "github", ProjectDir: dir})
	if _, ok := got.Services["notion"]; ok {
		t.Errorf("notion should be filtered out")
	}
	if _, ok := got.Services["github"]; !ok {
		t.Errorf("github should remain")
	}
}

// asInvalidPattern unwraps using errors.As for testing convenience.
func asInvalidPattern(err error, target *core.ErrInvalidPattern) bool {
	for err != nil {
		if e, ok := err.(core.ErrInvalidPattern); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
