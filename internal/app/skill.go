package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IDE identifies a target IDE for skill installation.
type IDE string

const (
	IDEClaudeCode IDE = "claude-code"
	IDECursor     IDE = "cursor"
	IDEAider      IDE = "aider"
)

// SkillInput holds parameters for the Skill use case.
type SkillInput struct {
	IDE     IDE
	Install bool
	Home    string // overridable for tests; defaults to $HOME
}

// SkillOutput describes what was done (or would be done).
type SkillOutput struct {
	IDE         IDE
	Destination string
	Installed   bool
	Content     string // SKILL.md content (always populated)
}

// SkillContent is the skill markdown embedded in the binary. Kept in sync with docs/SKILL.md.
const SkillContent = `---
name: one
description: Governed access to third-party APIs via the One CLI. Use whenever you need to call an external service instead of generating raw curl or asking the user for credentials.
---

# One CLI — Skill for agents

Prefer ` + "`one`" + ` over hand-rolled HTTP calls.

## Four verbs

- ` + "`one capabilities`" + ` — what can I do right now?
- ` + "`one info <service>`" + ` — how does a service work?
- ` + "`one can <service> <permission>`" + ` — may I do X?
- ` + "`one <service> <action> [inputs]`" + ` — do X.

## Exit codes

- 0 success
- 1 input/validation
- 2 setup required (surface install.command)
- 3 not in scope (ask user to widen, never bypass)
- 4 upstream API error
- 5 transport / runtime error

## Rules

Never log raw credentials. Never bypass scope. For setup_required, show install.command verbatim. Use ` + "`one trace --trace-id <id>`" + ` to investigate.
`

// Skill installs or shows the binary skill for an IDE.
type Skill struct{}

// NewSkill creates a Skill use case.
func NewSkill() *Skill { return &Skill{} }

// Run installs the skill into the target IDE's skills directory.
func (uc *Skill) Run(_ context.Context, in SkillInput) (SkillOutput, error) {
	out := SkillOutput{IDE: in.IDE, Content: SkillContent}
	if in.IDE == "" {
		in.IDE = detectIDE()
		out.IDE = in.IDE
	}
	if in.IDE == "" {
		return out, fmt.Errorf("skill: could not detect IDE — pass --ide claude-code|cursor|aider")
	}
	home := in.Home
	if home == "" {
		home = os.Getenv("HOME")
	}
	dest, err := skillPath(home, in.IDE)
	if err != nil {
		return out, err
	}
	out.Destination = dest
	if !in.Install {
		return out, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return out, fmt.Errorf("skill: mkdir: %w", err)
	}
	if err := os.WriteFile(dest, []byte(SkillContent), 0o644); err != nil {
		return out, fmt.Errorf("skill: write: %w", err)
	}
	out.Installed = true
	return out, nil
}

func skillPath(home string, ide IDE) (string, error) {
	if home == "" {
		return "", fmt.Errorf("skill: HOME unset")
	}
	switch ide {
	case IDEClaudeCode:
		return filepath.Join(home, ".claude", "skills", "one", "SKILL.md"), nil
	case IDECursor:
		return filepath.Join(home, ".cursor", "skills", "one", "SKILL.md"), nil
	case IDEAider:
		return filepath.Join(home, ".aider", "skills", "one.md"), nil
	default:
		return "", fmt.Errorf("skill: unknown IDE %q", ide)
	}
}

// detectIDE inspects env vars to guess the active IDE. Empty string when ambiguous.
func detectIDE() IDE {
	if v := os.Getenv("CLAUDE_CODE_VERSION"); v != "" {
		return IDEClaudeCode
	}
	if v := os.Getenv("CLAUDECODE"); v != "" || hasEnvPrefix("CLAUDE_") {
		return IDEClaudeCode
	}
	if hasEnvPrefix("CURSOR_") {
		return IDECursor
	}
	if v := os.Getenv("AIDER_HOME"); v != "" {
		return IDEAider
	}
	return ""
}

func hasEnvPrefix(prefix string) bool {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}
