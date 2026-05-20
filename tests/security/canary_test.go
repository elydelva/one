//go:build security

package security_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCanaryNoCredentialLeak runs the binary across all Phase 1 commands with a
// unique canary in ONE_CREDS_*, then asserts the canary appears in no output stream.
//
// A leak would mean the Secret type, a logger, or a renderer accidentally
// printed the plaintext token.
func TestCanaryNoCredentialLeak(t *testing.T) {
	bin := buildBinary(t)

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand: %v", err)
	}
	canary := "CANARY_DO_NOT_LEAK_" + hex.EncodeToString(buf)

	envCred := fmt.Sprintf(`{"provider":"pat","service":"github","account":"default","access_token":%q}`, canary)

	cwd := t.TempDir()

	// init the project so scope commands have a file to work against.
	run(t, bin, cwd, envCred, "init")

	cases := [][]string{
		{"scope", "add", "github", "issues.read"},
		{"scope", "add", "github", "issues.*"},
		{"scope", "show"},
		{"capabilities", "--scope-only"},
		{"can", "github", "issues.read"},
	}
	for _, args := range cases {
		stdout, stderr := run(t, bin, cwd, envCred, args...)
		if strings.Contains(stdout, canary) {
			t.Errorf("CANARY LEAKED in stdout of `one %s`:\n%s", strings.Join(args, " "), stdout)
		}
		if strings.Contains(stderr, canary) {
			t.Errorf("CANARY LEAKED in stderr of `one %s`:\n%s", strings.Join(args, " "), stderr)
		}
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	bin := filepath.Join(t.TempDir(), "one")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/one")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, out)
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
			t.Fatalf("could not find go.mod")
		}
		dir = parent
	}
}

func run(t *testing.T, bin, cwd, envCred string, args ...string) (string, string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"ONE_CREDS_GITHUB_DEFAULT="+envCred,
		"HOME="+cwd, // isolate keychain access on Linux (best-effort)
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // exit codes are part of the contract; we don't assert here.
	return stdout.String(), stderr.String()
}
