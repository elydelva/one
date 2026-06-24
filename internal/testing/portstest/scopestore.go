package portstest

import (
	"os"
	"path/filepath"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RunScopeStoreTests executes the ports.ScopeStore contract against any implementation.
// The factory must return a ScopeStore that reads .onerc.yaml from the given dir.
func RunScopeStoreTests(t *testing.T, name string, factory func(t *testing.T) ports.ScopeStore) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Run("empty dir returns empty scope", func(t *testing.T) {
			s := factory(t)
			dir := t.TempDir()
			sc, err := s.Load(dir)
			if err != nil {
				t.Fatalf("load empty dir: %v", err)
			}
			if sc.Version != 1 {
				t.Errorf("expected version 1, got %d", sc.Version)
			}
			if len(sc.Services) != 0 {
				t.Errorf("expected no services, got %d", len(sc.Services))
			}
		})

		t.Run("valid file loads services", func(t *testing.T) {
			s := factory(t)
			dir := t.TempDir()
			writeFile(t, dir, ".onerc.yaml", `version: 1
services:
  github:
    allow:
      - issues.read
      - issues.*
    deny:
      - repos.delete
`)
			sc, err := s.Load(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			gh, ok := sc.Services["github"]
			if !ok {
				t.Fatalf("github service missing")
			}
			if len(gh.Allow) != 2 {
				t.Errorf("expected 2 allow, got %d", len(gh.Allow))
			}
			if len(gh.Deny) != 1 {
				t.Errorf("expected 1 deny, got %d", len(gh.Deny))
			}
		})

		t.Run("rejects version != 1", func(t *testing.T) {
			s := factory(t)
			dir := t.TempDir()
			writeFile(t, dir, ".onerc.yaml", "version: 2\nservices: {}\n")
			_, err := s.Load(dir)
			var bad core.ErrInvalidScopeFile
			if !isErrorAs(err, &bad) {
				t.Errorf("expected ErrInvalidScopeFile, got %T: %v", err, err)
			}
		})

		t.Run("rejects invalid pattern", func(t *testing.T) {
			s := factory(t)
			dir := t.TempDir()
			writeFile(t, dir, ".onerc.yaml", "version: 1\nservices:\n  github:\n    allow:\n      - \"**\"\n")
			_, err := s.Load(dir)
			var bad core.ErrInvalidScopeFile
			if !isErrorAs(err, &bad) {
				t.Errorf("expected ErrInvalidScopeFile for **, got %T: %v", err, err)
			}
		})

		t.Run("rejects malformed YAML", func(t *testing.T) {
			s := factory(t)
			dir := t.TempDir()
			writeFile(t, dir, ".onerc.yaml", "version: 1\nservices: [not a map]\n")
			_, err := s.Load(dir)
			if err == nil {
				t.Errorf("expected error for malformed YAML, got nil")
			}
		})
	})
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
