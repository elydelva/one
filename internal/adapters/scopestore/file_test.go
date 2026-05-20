package scopestore

import (
	"os"
	"path/filepath"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/portstest"
)

func TestFileScopeStore_Contract(t *testing.T) {
	portstest.RunScopeStoreTests(t, "FileScopeStore", func(_ *testing.T) ports.ScopeStore {
		return NewFileScopeStore()
	})
}

func TestFileScopeStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewFileScopeStore()

	in := core.Scope{
		Version: 1,
		Services: map[core.ServiceID]core.ServiceScope{
			"github": {
				Account: "work",
				Allow:   []core.PermissionPattern{"issues.read", "issues.*"},
				Deny:    []core.PermissionPattern{"repos.delete"},
			},
		},
	}
	if err := s.Save(dir, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".onerc.yaml"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("expected mode 0644, got %v", info.Mode().Perm())
	}

	got, err := s.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	gh := got.Services["github"]
	if gh.Account != "work" {
		t.Errorf("account = %q", gh.Account)
	}
	if len(gh.Allow) != 2 || len(gh.Deny) != 1 {
		t.Errorf("counts: allow=%d deny=%d", len(gh.Allow), len(gh.Deny))
	}
}

func TestFileScopeStore_LoadMissingReturnsEmptyScope(t *testing.T) {
	got, err := NewFileScopeStore().Load(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Version != 1 || len(got.Services) != 0 {
		t.Errorf("expected empty scope v1, got %+v", got)
	}
}
