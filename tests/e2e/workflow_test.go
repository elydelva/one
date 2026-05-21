//go:build e2e

package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"elydelva/one/internal/testing/fakeapi"
)

// TestE2E_FullWorkflow exercises the v1.0 happy path: scope load → exec → trace.
// init/login are exercised indirectly: scope file is pre-written and credential
// is injected via ONE_CREDS_* env (the EnvVarVault path), matching the simplest
// agent setup.
func TestE2E_FullWorkflow(t *testing.T) {
	bin := buildBinary(t)
	fixtureRoot := findFixtureRoot(t)

	srv := fakeapi.New(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues/1", Status: 200, Body: map[string]any{"number": 1, "title": "Hi"}},
	})
	cwd := setupProject(t, fixtureRoot, srv.URL)

	t.Run("capabilities_lists_in_scope_actions", func(t *testing.T) {
		stdout, _, code := runOne(t, bin, cwd, fixtureRoot, "capabilities", "--scope-only")
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(stdout, "issues") {
			t.Errorf("expected issues.* in capabilities, got: %s", stdout)
		}
	})

	t.Run("exec_succeeds_and_records_trace", func(t *testing.T) {
		stdout, _, code := runOne(t, bin, cwd, fixtureRoot, "github", "issues.read", "--owner", "x", "--repo", "y", "--issue_number", "1")
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		var env map[string]any
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("invalid JSON envelope: %v", err)
		}
		traceID, _ := env["trace_id"].(string)
		if traceID == "" {
			t.Fatal("missing trace_id")
		}
		// Audit file should now exist under $HOME/.one/audit/audit-YYYY-MM.ndjson
		matches, err := filepath.Glob(filepath.Join(cwd, ".one", "audit", "audit-*.ndjson"))
		if err != nil || len(matches) == 0 {
			t.Fatalf("audit file missing: %v", err)
		}
		raw, _ := os.ReadFile(matches[0])
		if !strings.Contains(string(raw), traceID) {
			t.Errorf("audit log missing trace_id %s:\n%s", traceID, raw)
		}
		if !strings.Contains(string(raw), `"kind":"EXEC"`) {
			t.Errorf("audit log missing EXEC kind:\n%s", raw)
		}
	})

	t.Run("trace_lists_recent_events", func(t *testing.T) {
		// First, ensure at least one EXEC was logged above.
		stdout, _, code := runOne(t, bin, cwd, fixtureRoot, "trace", "--limit", "5")
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(stdout, "EXEC") {
			t.Errorf("trace output missing EXEC: %s", stdout)
		}
		if !strings.Contains(stdout, "github") {
			t.Errorf("trace output missing service: %s", stdout)
		}
	})

	t.Run("trace_json_output_is_parseable", func(t *testing.T) {
		stdout, _, code := runOne(t, bin, cwd, fixtureRoot, "trace", "--limit", "5", "--json")
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		var events []map[string]any
		if err := json.Unmarshal([]byte(stdout), &events); err != nil {
			t.Fatalf("trace --json not valid JSON: %v\n%s", err, stdout)
		}
		if len(events) == 0 {
			t.Fatal("expected at least 1 event")
		}
	})
}
