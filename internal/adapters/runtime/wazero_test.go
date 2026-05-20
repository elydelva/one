//go:build !nowasm

package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

func TestClampInt(t *testing.T) {
	cases := []struct{ v, def, lo, hi, want int }{
		{0, 30, 1, 120, 30},
		{-5, 30, 1, 120, 30},
		{300, 30, 1, 120, 120},
		{45, 30, 1, 120, 45},
	}
	for _, c := range cases {
		if got := clampInt(c.v, c.def, c.lo, c.hi); got != c.want {
			t.Errorf("clampInt(%d,%d,%d,%d) = %d, want %d", c.v, c.def, c.lo, c.hi, got, c.want)
		}
	}
}

func TestCompileAllowlistRejectsTooPermissive(t *testing.T) {
	h := &core.HandlerRef{Sha256: "test-1", Calls: []string{".*"}}
	if _, err := compileAllowlist(h); err == nil {
		t.Fatal("expected error for .* pattern")
	}
}

func TestCompileAllowlistAnchors(t *testing.T) {
	h := &core.HandlerRef{Sha256: "test-2", Calls: []string{`https://api\.notion\.com/v1/pages`}}
	pats, err := compileAllowlist(h)
	if err != nil {
		t.Fatal(err)
	}
	if !pats[0].MatchString("https://api.notion.com/v1/pages") {
		t.Error("expected match for declared URL")
	}
	if pats[0].MatchString("https://api.notion.com/v1/pages/extra") {
		t.Error("must not match longer URL — pattern is anchored")
	}
}

func TestCompileAllowlistRejectsOverlongPattern(t *testing.T) {
	long := strings.Repeat("a", maxPatternLength+1)
	h := &core.HandlerRef{Sha256: "test-3", Calls: []string{long}}
	if _, err := compileAllowlist(h); err == nil {
		t.Fatal("expected error for overlong pattern")
	}
}

func TestFSHandlerResolverPathTraversal(t *testing.T) {
	r := NewFSHandlerResolver(t.TempDir())
	_, err := r.ReadHandler(context.Background(), "svc", "../../etc/passwd")
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("want escape error, got %v", err)
	}
}

func TestFSHandlerResolverAbsRefused(t *testing.T) {
	r := NewFSHandlerResolver(t.TempDir())
	_, err := r.ReadHandler(context.Background(), "svc", filepath.FromSlash("/etc/passwd"))
	if err == nil {
		t.Fatal("absolute path must be refused")
	}
}

func TestCredentialValueAliasing(t *testing.T) {
	c := core.Credential{AccessToken: core.NewSecret("tk")}
	if v, ok := credentialValue(c, "api_key"); !ok || v != "tk" {
		t.Errorf("api_key alias failed: %v %v", v, ok)
	}
	if _, ok := credentialValue(c, "unknown"); ok {
		t.Error("unknown key must return false")
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"a", "b"}, "b") {
		t.Error("contains miss")
	}
	if contains([]string{"a"}, "z") {
		t.Error("contains false positive")
	}
}

func TestHostAPIVersionMismatch(t *testing.T) {
	r := NewWazeroRuntime(nil, nil, nil, nil, nil, "")
	req := ports.ExecuteRequest{
		Action: core.Action{
			ID:      "test.action",
			Service: "svc",
			Handler: &core.HandlerRef{HostAPI: 99},
		},
	}
	_, err := r.Execute(context.Background(), req)
	var v core.ErrSandboxViolation
	if !errors.As(err, &v) || v.Kind != "host_api_version" {
		t.Fatalf("want host_api_version violation, got %v", err)
	}
}
