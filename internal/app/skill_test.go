package app

import (
	"strings"
	"testing"
)

// TestSkillContentExitCodes guards against drift between the embedded skill's
// documented exit codes and the real mapping in internal/cli/exit.go.
// The codes are asserted as exact substrings so a future change to either side
// forces a conscious update here.
func TestSkillContentExitCodes(t *testing.T) {
	wantLines := []string{
		"- 0 success",
		"- 1 input/validation",
		"- 2 not authenticated",
		"- 3 not in scope",
		"- 4 setup required (surface install.command)",
		"- 5 unknown service/action",
	}
	for _, line := range wantLines {
		if !strings.Contains(SkillContent, line) {
			t.Errorf("embedded skill missing exit-code line %q", line)
		}
	}
	// The old, wrong mappings must not reappear.
	forbidden := []string{
		"2 setup required",
		"4 upstream API error",
		"5 transport / runtime error",
	}
	for _, bad := range forbidden {
		if strings.Contains(SkillContent, bad) {
			t.Errorf("embedded skill still contains stale exit-code text %q", bad)
		}
	}
}
