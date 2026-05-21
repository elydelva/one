//go:build e2e

package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_Lock exercises `one lock --update-all` and `one lock --check`.
// FS catalog leaves Service.Version empty, so we force drift by writing a
// lock file with a bogus pinned version and asserting --check reports it.
func TestE2E_Lock(t *testing.T) {
	bin := buildBinary(t)
	fixtureRoot := findFixtureRoot(t)
	cwd := setupProject(t, fixtureRoot, "http://127.0.0.1:1")

	lockPath := filepath.Join(cwd, ".onerc.lock")

	t.Run("update_all_writes_lock", func(t *testing.T) {
		stdout, stderr, code := runOne(t, bin, cwd, fixtureRoot, "lock", "--update-all")
		if code != 0 {
			t.Fatalf("exit %d, stdout=%s stderr=%s", code, stdout, stderr)
		}
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("read lock: %v", err)
		}
		if !strings.Contains(string(raw), "github") {
			t.Errorf("lock missing github entry:\n%s", raw)
		}
		if !strings.Contains(string(raw), "version: 1") {
			t.Errorf("lock missing schema version:\n%s", raw)
		}
	})

	t.Run("check_passes_after_update_all", func(t *testing.T) {
		_, stderr, code := runOne(t, bin, cwd, fixtureRoot, "lock", "--check")
		if code != 0 {
			t.Fatalf("exit %d (want 0), stderr=%s", code, stderr)
		}
	})

	t.Run("check_detects_drift_with_hint", func(t *testing.T) {
		bogus := "version: 1\ngenerated_at: 2026-05-21T00:00:00Z\nservices:\n  github:\n    version: 999.999.999\n"
		if err := os.WriteFile(lockPath, []byte(bogus), 0o644); err != nil {
			t.Fatalf("write lock: %v", err)
		}
		_, stderr, code := runOne(t, bin, cwd, fixtureRoot, "lock", "--check")
		if code == 0 {
			t.Fatalf("check should fail, stderr=%s", stderr)
		}
		if !strings.Contains(stderr, "drift") {
			t.Errorf("stderr missing 'drift': %s", stderr)
		}
		if !strings.Contains(stderr, "one lock --update-all") {
			t.Errorf("stderr missing remediation hint: %s", stderr)
		}
	})
}
