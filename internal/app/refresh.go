package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RefreshIfNeeded refreshes a credential when it's near expiry, serialized by a
// file lock at <lockDir>/<service>:<alias>.lock so concurrent processes share
// a single refresh.
//
// Behavior:
//   - If !cred.NeedsRefresh(now + Margin), return cred unchanged (fast path).
//   - Acquire flock with LockTimeout. On failure → ErrReAuthRequired.
//   - Re-fetch from vault under the lock (another worker may have refreshed
//     while we were waiting).
//   - If still expired, call the matching AuthProvider.Refresh.
//   - Persist the refreshed credential to the vault.
//   - On vault store failure: rollback (keep the old in-memory cred, return
//     ErrReAuthRequired) so we never advertise success while writes are dropped.
type RefreshIfNeeded struct {
	vault    ports.Vault
	auth     []ports.AuthProvider
	clock    ports.Clock
	log      ports.Logger
	lockDir  string
	procLock sync.Mutex
	locks    map[string]*sync.Mutex

	Margin       time.Duration
	LockTimeout  time.Duration
}

// NewRefreshIfNeeded creates the use case.
func NewRefreshIfNeeded(vault ports.Vault, auth []ports.AuthProvider, clock ports.Clock, log ports.Logger, lockDir string) *RefreshIfNeeded {
	return &RefreshIfNeeded{
		vault: vault, auth: auth, clock: clock, log: log, lockDir: lockDir,
		locks:       map[string]*sync.Mutex{},
		Margin:      60 * time.Second,
		LockTimeout: 10 * time.Second,
	}
}

// Run returns the credential, refreshed if needed.
func (uc *RefreshIfNeeded) Run(ctx context.Context, cred core.Credential) (core.Credential, error) {
	if !uc.shouldRefresh(cred) {
		return cred, nil
	}
	if uc.lockDir != "" {
		if err := os.MkdirAll(uc.lockDir, 0o700); err != nil {
			return core.Credential{}, fmt.Errorf("refresh: mkdir lock dir: %w", err)
		}
	}
	lockKey := cred.Ref().String()
	inProc := uc.inProcMutex(lockKey)
	inProc.Lock()
	defer inProc.Unlock()

	var fl *flock.Flock
	if uc.lockDir != "" {
		fl = flock.New(filepath.Join(uc.lockDir, lockKey+".lock"))
		lockCtx, cancel := context.WithTimeout(ctx, uc.LockTimeout)
		defer cancel()
		got, err := fl.TryLockContext(lockCtx, 50*time.Millisecond)
		if err != nil || !got {
			return core.Credential{}, core.ErrReAuthRequired{Service: cred.Service, Account: cred.Account}
		}
		defer fl.Unlock()
	}

	// Re-fetch: someone else may have already refreshed during our wait.
	latest, err := uc.vault.Fetch(ctx, cred.Ref())
	if err == nil && !uc.shouldRefresh(latest) {
		return latest, nil
	}
	current := cred
	if err == nil {
		current = latest
	}

	provider := uc.providerFor(current.Provider)
	if provider == nil {
		return core.Credential{}, fmt.Errorf("refresh: no provider for %s", current.Provider)
	}
	fresh, err := provider.Refresh(ctx, current)
	if err != nil {
		uc.log.Warn("refresh failed", "service", current.Service, "account", current.Account, "err", err.Error())
		return core.Credential{}, err
	}
	if err := uc.vault.Store(ctx, fresh.Ref(), fresh); err != nil {
		uc.log.Warn("refresh store failed", "service", fresh.Service, "err", err.Error())
		return core.Credential{}, core.ErrReAuthRequired{Service: fresh.Service, Account: fresh.Account}
	}
	return fresh, nil
}

func (uc *RefreshIfNeeded) shouldRefresh(cred core.Credential) bool {
	if cred.ExpiresAt == nil {
		return false
	}
	return uc.clock.Now().Add(uc.Margin).After(*cred.ExpiresAt)
}

func (uc *RefreshIfNeeded) providerFor(kind core.ProviderKind) ports.AuthProvider {
	for _, p := range uc.auth {
		if p.Supports(kind) {
			return p
		}
	}
	return nil
}

func (uc *RefreshIfNeeded) inProcMutex(key string) *sync.Mutex {
	uc.procLock.Lock()
	defer uc.procLock.Unlock()
	m, ok := uc.locks[key]
	if !ok {
		m = &sync.Mutex{}
		uc.locks[key] = m
	}
	return m
}

var _ = errors.New // keep package import stable
