//go:build !nowasm

package runtime

import (
	"os"
	"strings"

	"elydelva/one/internal/core"
)

// checkSourceTrust gates WASM handler execution by the action's Source tag.
//
// Default-deny: any non-empty Source (e.g. "tap:user/repo") is refused with
// ErrSandboxViolation. The official catalog leaves Source empty so its
// handlers run unaffected.
//
// Operators can opt in per-tap by setting:
//
//	ONE_TAP_ALLOW_HANDLERS=tap:user/repo,tap:other/x
//
// This is intentionally an env knob and not a scope-file directive: the
// policy decision (run untrusted code) deserves an explicit step that
// doesn't blend into a normal scope grant. A full scope-level model lives
// in the v1.3 RFC.
func checkSourceTrust(action core.Action) error {
	if action.Source == "" {
		return nil
	}
	allow := os.Getenv("ONE_TAP_ALLOW_HANDLERS")
	if allow == "" {
		return core.ErrSandboxViolation{
			Action: action.ID,
			Kind:   "untrusted_source",
			Detail: "WASM handler from " + action.Source + " refused (set ONE_TAP_ALLOW_HANDLERS to opt in)",
		}
	}
	for _, allowed := range strings.Split(allow, ",") {
		if strings.TrimSpace(allowed) == action.Source {
			return nil
		}
	}
	return core.ErrSandboxViolation{
		Action: action.ID,
		Kind:   "untrusted_source",
		Detail: action.Source + " not present in ONE_TAP_ALLOW_HANDLERS",
	}
}
