package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"elydelva/one/internal/adapters/scopestore"
)

func TestInit_CreatesScopeAndGitignore(t *testing.T) {
	dir := t.TempDir()
	s := scopestore.NewFileScopeStore()
	uc := NewInit(s)

	out, err := uc.Run(InitInput{ProjectDir: dir})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !out.ScopeCreated {
		t.Errorf("expected ScopeCreated=true")
	}
	if !out.GitignoreAppended {
		t.Errorf("expected GitignoreAppended=true")
	}
	if _, err := os.Stat(filepath.Join(dir, ".onerc.yaml")); err != nil {
		t.Errorf(".onerc.yaml not created: %v", err)
	}
	gi, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(gi), ".onerc.local.yaml") {
		t.Errorf(".gitignore missing entry: %q", gi)
	}
}

func TestInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	s := scopestore.NewFileScopeStore()
	uc := NewInit(s)
	_, _ = uc.Run(InitInput{ProjectDir: dir})
	out, err := uc.Run(InitInput{ProjectDir: dir})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if out.ScopeCreated {
		t.Errorf("expected ScopeCreated=false on second run")
	}
	if out.GitignoreAppended {
		t.Errorf("expected GitignoreAppended=false on second run")
	}
}

func TestInit_PreservesExistingGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	uc := NewInit(scopestore.NewFileScopeStore())
	_, _ = uc.Run(InitInput{ProjectDir: dir})

	gi, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(gi), "node_modules") {
		t.Errorf("existing entry lost: %q", gi)
	}
	if !strings.Contains(string(gi), ".onerc.local.yaml") {
		t.Errorf("new entry missing: %q", gi)
	}
}
