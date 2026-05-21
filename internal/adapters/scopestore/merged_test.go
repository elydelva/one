package scopestore

import (
	"os"
	"path/filepath"
	"testing"

	"elydelva/one/internal/core"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMergedScopeStore_LocalRestricts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".onerc.yaml"), `version: 1
services:
  github:
    allow:
      - issues.read
      - issues.write
`)
	writeFile(t, filepath.Join(dir, ".onerc.local.yaml"), `version: 1
services:
  github:
    allow:
      - issues.read
`)
	store := NewMergedScopeStore(NewFileScopeStore())
	sc, err := store.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sc.Services["github"]; !ok {
		t.Fatal("github missing")
	}
	if !sc.Allows(core.Permission{Service: "github", Path: "issues.read"}) {
		t.Error("issues.read should remain allowed")
	}
	if sc.Allows(core.Permission{Service: "github", Path: "issues.write"}) {
		t.Error("issues.write should be restricted away by local override")
	}
}
