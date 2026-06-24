package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// InitInput holds parameters for the Init use case.
type InitInput struct {
	ProjectDir string
}

// InitOutput reports what changed on disk.
type InitOutput struct {
	ScopePath         string
	ScopeCreated      bool
	GitignorePath     string
	GitignoreAppended bool
}

// Init bootstraps .onerc.yaml + .gitignore for a fresh project.
type Init struct {
	writer ports.ScopeWriter
}

// NewInit creates an Init use case.
func NewInit(writer ports.ScopeWriter) *Init { return &Init{writer: writer} }

// Run writes a minimal .onerc.yaml (idempotent) and ensures .onerc.local.yaml
// is listed in .gitignore.
func (uc *Init) Run(in InitInput) (InitOutput, error) {
	if in.ProjectDir == "" {
		return InitOutput{}, core.ErrInputValidation{Field: "project_dir", Reason: "required"}
	}
	out := InitOutput{
		ScopePath:     filepath.Join(in.ProjectDir, ".onerc.yaml"),
		GitignorePath: filepath.Join(in.ProjectDir, ".gitignore"),
	}

	if _, err := os.Stat(out.ScopePath); errors.Is(err, fs.ErrNotExist) {
		if err := uc.writer.Save(in.ProjectDir, core.Scope{Version: 1, Services: map[core.ServiceID]core.ServiceScope{}}); err != nil {
			return out, err
		}
		out.ScopeCreated = true
	} else if err != nil {
		return out, err
	}

	appended, err := ensureGitignoreEntry(out.GitignorePath, ".onerc.local.yaml")
	if err != nil {
		return out, err
	}
	out.GitignoreAppended = appended
	return out, nil
}

func ensureGitignoreEntry(path, entry string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == entry {
			return false, nil
		}
	}
	var buf strings.Builder
	buf.Write(data)
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		buf.WriteByte('\n')
	}
	buf.WriteString(entry)
	buf.WriteByte('\n')
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil { //nolint:gosec // G306: .gitignore is a non-secret, VCS-committed file
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, nil
}
