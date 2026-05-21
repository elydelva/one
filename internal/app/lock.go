package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"go.yaml.in/yaml/v3"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// LockFileName is the canonical lock file at the project root.
const LockFileName = ".onerc.lock"

// LockMode selects the LockScope operation.
type LockMode string

const (
	LockGenerate  LockMode = "generate"   // write the lock fresh
	LockUpdate    LockMode = "update"     // refresh a single service
	LockUpdateAll LockMode = "update-all" // refresh every service in scope
	LockCheck     LockMode = "check"      // verify lock matches current catalog
)

// LockInput configures LockScope.Run.
type LockInput struct {
	ProjectDir string
	Mode       LockMode
	Service    string // required when Mode == LockUpdate
}

// LockFile is the on-disk shape of .onerc.lock.
type LockFile struct {
	Version     int                         `yaml:"version"`
	GeneratedAt time.Time                   `yaml:"generated_at"`
	Services    map[string]LockServiceEntry `yaml:"services"`
}

// LockServiceEntry pins a single service version.
type LockServiceEntry struct {
	Version       string `yaml:"version"`
	TarballSha256 string `yaml:"tarball_sha256,omitempty"`
}

// LockDrift describes a single drift detected by LockCheck.
type LockDrift struct {
	Service        string
	LockedVersion  string
	CurrentVersion string
	LockedSha      string
	CurrentSha     string
}

func (d LockDrift) String() string {
	return fmt.Sprintf("%s: locked=%s current=%s", d.Service, d.LockedVersion, d.CurrentVersion)
}

// ErrLockDrift is returned by LockCheck when the lock no longer matches the catalog.
type ErrLockDrift struct {
	Drifts []LockDrift
}

func (e ErrLockDrift) Error() string {
	if len(e.Drifts) == 0 {
		return "lock drift detected"
	}
	out := "lock drift detected:"
	for _, d := range e.Drifts {
		out += "\n  " + d.String()
	}
	out += "\nhint: run `one lock --update-all` to refresh."
	return out
}

// LockScope freezes the catalog service versions in .onerc.lock.
type LockScope struct {
	catalog ports.Catalog
	scope   ports.ScopeStore
	clock   ports.Clock
}

// NewLockScope creates a LockScope use case.
func NewLockScope(catalog ports.Catalog, scope ports.ScopeStore) *LockScope {
	return &LockScope{catalog: catalog, scope: scope}
}

// WithClock injects a clock for the GeneratedAt timestamp (defaults to time.Now).
func (uc *LockScope) WithClock(c ports.Clock) *LockScope { uc.clock = c; return uc }

// Run executes the requested lock operation.
func (uc *LockScope) Run(ctx context.Context, in LockInput) error {
	path := filepath.Join(in.ProjectDir, LockFileName)
	switch in.Mode {
	case LockCheck:
		return uc.check(ctx, path, in.ProjectDir)
	case LockUpdate:
		if in.Service == "" {
			return errors.New("lock update: service name required")
		}
		return uc.updateOne(ctx, path, in.Service)
	case LockUpdateAll, LockGenerate, "":
		return uc.generate(ctx, path, in.ProjectDir)
	default:
		return fmt.Errorf("lock: unknown mode %q", in.Mode)
	}
}

func (uc *LockScope) now() time.Time {
	if uc.clock != nil {
		return uc.clock.Now()
	}
	return time.Now()
}

func (uc *LockScope) servicesInScope(dir string) ([]string, error) {
	sc, err := uc.scope.Load(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(sc.Services))
	for id := range sc.Services {
		out = append(out, string(id))
	}
	sort.Strings(out)
	return out, nil
}

func (uc *LockScope) generate(ctx context.Context, path, dir string) error {
	ids, err := uc.servicesInScope(dir)
	if err != nil {
		return err
	}
	lock := LockFile{
		Version:     1,
		GeneratedAt: uc.now(),
		Services:    map[string]LockServiceEntry{},
	}
	for _, id := range ids {
		svc, err := uc.catalog.GetService(ctx, core.ServiceID(id))
		if err != nil {
			return fmt.Errorf("lock: %s: %w", id, err)
		}
		lock.Services[id] = LockServiceEntry{Version: svc.Version, TarballSha256: svc.Sha256}
	}
	return writeLock(path, lock)
}

func (uc *LockScope) updateOne(ctx context.Context, path, id string) error {
	lock, _ := readLock(path)
	if lock.Services == nil {
		lock.Services = map[string]LockServiceEntry{}
	}
	svc, err := uc.catalog.GetService(ctx, core.ServiceID(id))
	if err != nil {
		return err
	}
	lock.Version = 1
	lock.GeneratedAt = uc.now()
	lock.Services[id] = LockServiceEntry{Version: svc.Version, TarballSha256: svc.Sha256}
	return writeLock(path, lock)
}

func (uc *LockScope) check(ctx context.Context, path, dir string) error {
	lock, err := readLock(path)
	if err != nil {
		return fmt.Errorf("lock check: %w", err)
	}
	ids, err := uc.servicesInScope(dir)
	if err != nil {
		return err
	}
	var drifts []LockDrift
	for _, id := range ids {
		entry, locked := lock.Services[id]
		svc, err := uc.catalog.GetService(ctx, core.ServiceID(id))
		if err != nil {
			drifts = append(drifts, LockDrift{Service: id, LockedVersion: entry.Version, CurrentVersion: "<missing>"})
			continue
		}
		if !locked {
			drifts = append(drifts, LockDrift{Service: id, LockedVersion: "<missing>", CurrentVersion: svc.Version, CurrentSha: svc.Sha256})
			continue
		}
		if entry.Version != svc.Version || (entry.TarballSha256 != "" && svc.Sha256 != "" && entry.TarballSha256 != svc.Sha256) {
			drifts = append(drifts, LockDrift{
				Service: id, LockedVersion: entry.Version, CurrentVersion: svc.Version,
				LockedSha: entry.TarballSha256, CurrentSha: svc.Sha256,
			})
		}
	}
	if len(drifts) > 0 {
		return ErrLockDrift{Drifts: drifts}
	}
	return nil
}

func readLock(path string) (LockFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LockFile{}, fmt.Errorf("%s not found (run `one lock` to generate)", path)
		}
		return LockFile{}, err
	}
	var lock LockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return LockFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if lock.Version != 1 {
		return LockFile{}, fmt.Errorf("%s: unsupported version %d", path, lock.Version)
	}
	return lock, nil
}

func writeLock(path string, lock LockFile) error {
	if lock.Version == 0 {
		lock.Version = 1
	}
	data, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
