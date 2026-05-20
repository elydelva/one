//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"elydelva/one/internal/testing/fakeapi"
)

func TestE2E_HappyAndExitCodes(t *testing.T) {
	bin := buildBinary(t)
	fixtureRoot := findFixtureRoot(t)

	srv := fakeapi.New(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues/1", Status: 200, Body: map[string]any{"number": 1, "title": "Hi"}},
		{Method: "GET", Path: "/repos/x/y/issues/404", Status: 404, Body: map[string]any{"message": "nope"}},
	})

	cwd := setupProject(t, fixtureRoot, srv.URL)

	t.Run("happy_path", func(t *testing.T) {
		stdout, _, code := runOne(t, bin, cwd, fixtureRoot, "github", "issues.read", "--owner", "x", "--repo", "y", "--issue_number", "1")
		if code != 0 {
			t.Fatalf("exit %d, stdout=%s", code, stdout)
		}
		var env map[string]any
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, stdout)
		}
		if _, ok := env["data"]; !ok {
			t.Errorf("missing data: %s", stdout)
		}
		if _, ok := env["trace_id"]; !ok {
			t.Errorf("missing trace_id: %s", stdout)
		}
	})

	t.Run("not_in_scope", func(t *testing.T) {
		// repos.delete not in scope (we only added issues.*).
		_, stderr, code := runOne(t, bin, cwd, fixtureRoot, "can", "github", "repos.delete")
		if code != 3 {
			t.Fatalf("exit %d (want 3), stderr=%s", code, stderr)
		}
	})

	t.Run("not_authenticated", func(t *testing.T) {
		_, _, code := runOneNoCred(t, bin, cwd, fixtureRoot, "github", "issues.read", "--owner", "x", "--repo", "y", "--issue_number", "1")
		if code != 2 {
			t.Fatalf("exit %d (want 2)", code)
		}
	})

	t.Run("unknown_service", func(t *testing.T) {
		_, _, code := runOne(t, bin, cwd, fixtureRoot, "info", "nope-service")
		if code != 5 {
			t.Fatalf("exit %d (want 5)", code)
		}
	})

	t.Run("input_validation", func(t *testing.T) {
		_, stderr, code := runOne(t, bin, cwd, fixtureRoot, "github", "issues.read", "--owner", "x")
		if code != 1 {
			t.Fatalf("exit %d (want 1), stderr=%s", code, stderr)
		}
		if !strings.Contains(stderr, "validation") {
			t.Errorf("stderr missing 'validation': %s", stderr)
		}
	})

	t.Run("not_found_404", func(t *testing.T) {
		_, _, code := runOne(t, bin, cwd, fixtureRoot, "github", "issues.read", "--owner", "x", "--repo", "y", "--issue_number", "404")
		if code != 5 {
			t.Fatalf("exit %d (want 5), got %d", code, code)
		}
	})
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "one")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/one")
	cmd.Dir = findRepoRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found")
		}
		dir = parent
	}
}

func findFixtureRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(findRepoRoot(t), "internal", "testing", "fixture", "catalog", "v1-minimal-e2e")
}

// setupProject creates a project dir with .onerc.yaml + a copy of the v1-minimal
// fixture catalog rewritten to point at the fakeapi server.
func setupProject(t *testing.T, fixtureRoot, baseURL string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(fixtureRoot, "github", "actions"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Write a service.yaml pointing at the fakeapi.
	svcYAML := "version: 1\nid: github\nname: GitHub\nbase_url: " + baseURL + "\nauth:\n  providers: [pat]\n  injection:\n    pat:\n      header: Authorization\n      format: \"Bearer {access_token}\"\n"
	if err := os.WriteFile(filepath.Join(fixtureRoot, "github", "service.yaml"), []byte(svcYAML), 0o644); err != nil {
		t.Fatalf("write svc: %v", err)
	}
	// Copy the action def.
	actionSrc := filepath.Join(findRepoRoot(t), "internal", "testing", "fixture", "catalog", "v1-minimal", "github", "actions", "issues.read.yaml")
	raw, err := os.ReadFile(actionSrc)
	if err != nil {
		t.Fatalf("read action: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureRoot, "github", "actions", "issues.read.yaml"), raw, 0o644); err != nil {
		t.Fatalf("write action: %v", err)
	}

	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, ".onerc.yaml"), []byte("version: 1\nservices:\n  github:\n    allow:\n      - \"issues.*\"\n"), 0o644); err != nil {
		t.Fatalf("scope: %v", err)
	}
	return cwd
}

func runOne(t *testing.T, bin, cwd, catalogRoot string, args ...string) (string, string, int) {
	return run(t, bin, cwd, catalogRoot, true, args...)
}

func runOneNoCred(t *testing.T, bin, cwd, catalogRoot string, args ...string) (string, string, int) {
	return run(t, bin, cwd, catalogRoot, false, args...)
}

func run(t *testing.T, bin, cwd, catalogRoot string, withCred bool, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	env := append(os.Environ(),
		"ONE_CATALOG_ROOT="+catalogRoot,
		"ONE_TRANSPORT_ALLOW_HTTP=1",
		"ONE_TRANSPORT_ALLOWED_HOSTS=127.0.0.1",
		"HOME="+cwd,
	)
	if withCred {
		env = append(env, `ONE_CREDS_GITHUB_DEFAULT={"provider":"pat","service":"github","account":"default","access_token":"ghp_test"}`)
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run: %v", err)
	}
	_ = http.StatusOK // keep import alive in case we extend later
	return stdout.String(), stderr.String(), code
}
