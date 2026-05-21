//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
)

// TestE2E_InstallGuide covers `one install <svc> --list` and
// `one install <svc> <guide>` against the FS catalog with a markdown guide
// that carries YAML frontmatter (title + verify).
func TestE2E_InstallGuide(t *testing.T) {
	bin := buildBinary(t)
	fixtureRoot := findFixtureRoot(t)
	cwd := setupProject(t, fixtureRoot, "http://127.0.0.1:1")

	t.Run("list_enumerates_guides", func(t *testing.T) {
		stdout, stderr, code := runOne(t, bin, cwd, fixtureRoot, "install", "github", "--list")
		if code != 0 {
			t.Fatalf("exit %d, stderr=%s", code, stderr)
		}
		if !strings.Contains(stdout, "getting-started") {
			t.Errorf("missing getting-started in list:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Set up a GitHub personal access token") {
			t.Errorf("missing title in list:\n%s", stdout)
		}
	})

	t.Run("renders_title_and_body", func(t *testing.T) {
		stdout, stderr, code := runOne(t, bin, cwd, fixtureRoot, "install", "github", "getting-started")
		if code != 0 {
			t.Fatalf("exit %d, stderr=%s", code, stderr)
		}
		if !strings.HasPrefix(stdout, "# Set up a GitHub personal access token") {
			t.Errorf("expected markdown title header, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "github.com/settings/tokens") {
			t.Errorf("expected body content, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "Verify: one github issues.read") {
			t.Errorf("expected verify hint, got:\n%s", stdout)
		}
	})

	t.Run("unknown_guide_exits_nonzero", func(t *testing.T) {
		_, stderr, code := runOne(t, bin, cwd, fixtureRoot, "install", "github", "no-such-guide")
		if code == 0 {
			t.Fatalf("expected failure, stderr=%s", stderr)
		}
	})
}
