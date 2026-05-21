package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"elydelva/one/internal/ports"
)

// DoctorCheck is the result of a single diagnostic check.
type DoctorCheck struct {
	Name    string
	Status  string // "ok", "warn", "fail"
	Message string
}

// DoctorOutput holds all diagnostic results.
type DoctorOutput struct {
	Checks []DoctorCheck
}

// OK reports whether every check passed (no "fail" entries).
func (o DoctorOutput) OK() bool {
	for _, c := range o.Checks {
		if c.Status == "fail" {
			return false
		}
	}
	return true
}

// RunDoctor performs diagnostic checks on the local setup.
type RunDoctor struct {
	vault   ports.Vault
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewRunDoctor creates a RunDoctor use case.
func NewRunDoctor(vault ports.Vault, catalog ports.Catalog, scope ports.ScopeStore) *RunDoctor {
	return &RunDoctor{vault: vault, catalog: catalog, scope: scope}
}

// Run executes all checks against the current working directory.
func (uc *RunDoctor) Run(ctx context.Context) (DoctorOutput, error) {
	wd, _ := os.Getwd()
	return uc.RunForDir(ctx, wd)
}

// RunForDir executes checks for a specific project directory.
func (uc *RunDoctor) RunForDir(ctx context.Context, projectDir string) (DoctorOutput, error) {
	out := DoctorOutput{}

	sc, err := uc.scope.Load(projectDir)
	switch {
	case err != nil:
		out.Checks = append(out.Checks, DoctorCheck{Name: "scope", Status: "fail", Message: err.Error()})
	case len(sc.Services) == 0:
		out.Checks = append(out.Checks, DoctorCheck{Name: "scope", Status: "warn", Message: "no services in .onerc.yaml — run `one scope add ...`"})
	default:
		out.Checks = append(out.Checks, DoctorCheck{Name: "scope", Status: "ok", Message: fmt.Sprintf("%d service(s) in scope", len(sc.Services))})
	}

	for id := range sc.Services {
		if _, err := uc.catalog.GetService(ctx, id); err != nil {
			out.Checks = append(out.Checks, DoctorCheck{Name: "catalog:" + string(id), Status: "fail", Message: err.Error()})
		} else {
			out.Checks = append(out.Checks, DoctorCheck{Name: "catalog:" + string(id), Status: "ok"})
		}
	}

	for id := range sc.Services {
		refs, err := uc.vault.List(ctx, id)
		switch {
		case err != nil:
			out.Checks = append(out.Checks, DoctorCheck{Name: "vault:" + string(id), Status: "warn", Message: err.Error()})
		case len(refs) == 0:
			out.Checks = append(out.Checks, DoctorCheck{Name: "vault:" + string(id), Status: "warn", Message: "no accounts (run `one login " + string(id) + "`)"})
		default:
			out.Checks = append(out.Checks, DoctorCheck{Name: "vault:" + string(id), Status: "ok", Message: fmt.Sprintf("%d account(s)", len(refs))})
		}
	}

	lockPath := filepath.Join(projectDir, LockFileName)
	if _, err := os.Stat(lockPath); err == nil {
		out.Checks = append(out.Checks, DoctorCheck{Name: "lock", Status: "ok", Message: lockPath})
	} else {
		out.Checks = append(out.Checks, DoctorCheck{Name: "lock", Status: "warn", Message: "no .onerc.lock (run `one lock`)"})
	}

	if home, _ := os.UserHomeDir(); home != "" {
		dir := filepath.Join(home, ".one")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			out.Checks = append(out.Checks, DoctorCheck{Name: "home", Status: "fail", Message: dir + ": " + err.Error()})
		} else {
			out.Checks = append(out.Checks, DoctorCheck{Name: "home", Status: "ok", Message: dir})
		}
	}

	return out, nil
}
