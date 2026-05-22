//go:build !nowasm

package runtime

import (
	"errors"
	"testing"

	"elydelva/one/internal/core"
)

func TestCheckSourceTrust_OfficialAllowed(t *testing.T) {
	if err := checkSourceTrust(core.Action{ID: "a", Source: ""}); err != nil {
		t.Errorf("official refused: %v", err)
	}
}

func TestCheckSourceTrust_TapRefusedByDefault(t *testing.T) {
	t.Setenv("ONE_TAP_ALLOW_HANDLERS", "")
	err := checkSourceTrust(core.Action{ID: "a", Source: "tap:x/y"})
	var v core.ErrSandboxViolation
	if !errors.As(err, &v) {
		t.Fatalf("expected ErrSandboxViolation, got %T %v", err, err)
	}
	if v.Kind != "untrusted_source" {
		t.Errorf("kind = %q", v.Kind)
	}
}

func TestCheckSourceTrust_AllowlistOptIn(t *testing.T) {
	t.Setenv("ONE_TAP_ALLOW_HANDLERS", "tap:other/z, tap:x/y ,tap:more")
	if err := checkSourceTrust(core.Action{ID: "a", Source: "tap:x/y"}); err != nil {
		t.Errorf("allowlisted tap refused: %v", err)
	}
}

func TestCheckSourceTrust_AllowlistMiss(t *testing.T) {
	t.Setenv("ONE_TAP_ALLOW_HANDLERS", "tap:other/z")
	err := checkSourceTrust(core.Action{ID: "a", Source: "tap:x/y"})
	if err == nil {
		t.Fatal("expected refusal")
	}
}
