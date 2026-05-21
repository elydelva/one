package app_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"elydelva/one/internal/adapters/scopestore"
	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
)

type fakeCat struct {
	services map[core.ServiceID]*core.Service
}

func (f *fakeCat) GetService(_ context.Context, id core.ServiceID) (*core.Service, error) {
	if s, ok := f.services[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, core.ErrUnknownService{Service: id}
}
func (f *fakeCat) GetAction(_ context.Context, svc core.ServiceID, a core.ActionID) (*core.Action, error) {
	return nil, core.ErrUnknownAction{Service: svc, Action: a}
}
func (f *fakeCat) ListServices(_ context.Context) ([]core.Service, error) { return nil, nil }
func (f *fakeCat) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", nil
}
func (f *fakeCat) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, nil
}
func (f *fakeCat) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, nil
}

func TestLock_GenerateAndCheckPass(t *testing.T) {
	dir := t.TempDir()
	scope := scopestore.NewFileScopeStore()
	if err := scope.Save(dir, core.Scope{Version: 1, Services: map[core.ServiceID]core.ServiceScope{
		"github": {},
	}}); err != nil {
		t.Fatal(err)
	}
	cat := &fakeCat{services: map[core.ServiceID]*core.Service{
		"github": {ID: "github", Version: "0.4.2", Sha256: "abc"},
	}}
	uc := app.NewLockScope(cat, scope)
	if err := uc.Run(context.Background(), app.LockInput{ProjectDir: dir, Mode: app.LockUpdateAll}); err != nil {
		t.Fatal(err)
	}
	if _, err := readFile(t, filepath.Join(dir, app.LockFileName)); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
	if err := uc.Run(context.Background(), app.LockInput{ProjectDir: dir, Mode: app.LockCheck}); err != nil {
		t.Fatalf("check should pass: %v", err)
	}
}

func TestLock_CheckDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	scope := scopestore.NewFileScopeStore()
	_ = scope.Save(dir, core.Scope{Version: 1, Services: map[core.ServiceID]core.ServiceScope{
		"github": {},
	}})
	cat := &fakeCat{services: map[core.ServiceID]*core.Service{
		"github": {ID: "github", Version: "0.4.2", Sha256: "abc"},
	}}
	uc := app.NewLockScope(cat, scope)
	if err := uc.Run(context.Background(), app.LockInput{ProjectDir: dir, Mode: app.LockUpdateAll}); err != nil {
		t.Fatal(err)
	}
	cat.services["github"] = &core.Service{ID: "github", Version: "0.5.0", Sha256: "def"}
	err := uc.Run(context.Background(), app.LockInput{ProjectDir: dir, Mode: app.LockCheck})
	var drift app.ErrLockDrift
	if !errors.As(err, &drift) {
		t.Fatalf("expected ErrLockDrift, got %v", err)
	}
	if len(drift.Drifts) != 1 || drift.Drifts[0].Service != "github" {
		t.Errorf("got %+v", drift.Drifts)
	}
}

func readFile(t *testing.T, path string) ([]byte, error) { t.Helper(); return os.ReadFile(path) }
