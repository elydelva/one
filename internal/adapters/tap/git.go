// Package tap provides the git-backed TapFetcher used by app.TapOps.
//
// Operations shell out to the `git` binary on PATH. Reasons over adding a
// go-git dependency: zero new Go deps, predictable behavior matching what
// users see when they run git themselves, and well-understood security
// properties.
package tap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Git is a TapFetcher implementation that shells out to the git binary.
type Git struct{}

// New returns a Git fetcher.
func New() *Git { return &Git{} }

// Clone runs `git clone --depth=1` and returns the HEAD SHA on success.
func (Git) Clone(ctx context.Context, url, dir string) (string, error) {
	if err := run(ctx, "", "clone", "--depth=1", "--", url, dir); err != nil {
		return "", err
	}
	return revParse(ctx, dir)
}

// HeadSHA returns the current HEAD SHA of the clone at dir.
func (Git) HeadSHA(ctx context.Context, dir string) (string, error) {
	return revParse(ctx, dir)
}

// FetchHead updates dir to upstream's default branch with a shallow fetch and
// returns the new HEAD SHA.
func (Git) FetchHead(ctx context.Context, dir string) (string, error) {
	if err := run(ctx, dir, "fetch", "--depth=1", "origin", "HEAD"); err != nil {
		return "", err
	}
	if err := run(ctx, dir, "reset", "--hard", "FETCH_HEAD"); err != nil {
		return "", err
	}
	return revParse(ctx, dir)
}

func revParse(ctx context.Context, dir string) (string, error) {
	out, err := output(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func output(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
