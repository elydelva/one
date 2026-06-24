package cli

import (
	"testing"

	"elydelva/one/internal/core"
)

func TestSkipLeadingGlobalFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int // index of the service token
	}{
		{"no flags", []string{"github", "issues.list"}, 0},
		{"json first", []string{"--json", "github", "issues.list"}, 1},
		{"json eq", []string{"--json=true", "github", "issues.list"}, 1},
		{"account value", []string{"--account", "work", "github", "issues.list"}, 2},
		{"project eq", []string{"--project=/tmp", "github", "issues.list"}, 1},
		{"mixed", []string{"--json", "--account", "work", "github", "x"}, 3},
		{"all consumed", []string{"--json"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skipLeadingGlobalFlags(tc.args); got != tc.want {
				t.Errorf("skipLeadingGlobalFlags(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}

func TestParseActionFlagsAcceptsJSON(t *testing.T) {
	schema := core.InputSchema{}
	// --json after the action must not error nor be treated as input.
	for _, args := range [][]string{
		{"--json"},
		{"--json=true"},
		{"--json", "--dry-run"},
	} {
		inputs, _, _, _, _, err := parseActionFlags(args, schema)
		if err != nil {
			t.Errorf("parseActionFlags(%v) unexpected error: %v", args, err)
		}
		if len(inputs) != 0 {
			t.Errorf("parseActionFlags(%v) leaked inputs: %v", args, inputs)
		}
	}
}
