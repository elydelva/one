package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gofrs/flock"
	"github.com/zalando/go-keyring"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// KeyringVault stores credentials in the OS native keychain (macOS Keychain, Linux Secret Service, Windows CredMgr).
// Keychain layout: service = constructor arg (e.g. "one"), account = "<service>:<alias>" (e.g. "github:work").
type KeyringVault struct {
	service  string
	lockPath string
	// backend overrides for testing; nil = real keyring.
	set    func(service, user, password string) error
	get    func(service, user string) (string, error)
	delete func(service, user string) error
}

// NewKeyringVault creates a vault using the given keychain service name.
func NewKeyringVault(service string) *KeyringVault {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.TempDir()
	}
	return &KeyringVault{
		service:  service,
		lockPath: filepath.Join(home, ".one", "locks", "vault-index.lock"),
		set:      keyring.Set,
		get:      keyring.Get,
		delete:   keyring.Delete,
	}
}

// WithIndexLockPath overrides the cross-process lock path. It is primarily
// useful for tests that must not touch the user's One data directory.
func (v *KeyringVault) WithIndexLockPath(path string) *KeyringVault {
	v.lockPath = path
	return v
}

func (v *KeyringVault) keyringUser(ref core.AccountRef) string {
	return string(ref.Service) + ":" + string(ref.Alias)
}

func (v *KeyringVault) Store(ctx context.Context, ref core.AccountRef, cred core.Credential) error {
	data, err := cred.MarshalForStorage()
	if err != nil {
		return err
	}
	if err := v.set(v.service, v.keyringUser(ref), string(data)); err != nil {
		return err
	}
	if err := v.updateIndex(ctx, ref.Service, ref.Alias, true); err != nil {
		return fmt.Errorf("credential stored but account index update failed: %w", err)
	}
	return nil
}

func (v *KeyringVault) Fetch(_ context.Context, ref core.AccountRef) (core.Credential, error) {
	raw, err := v.get(v.service, v.keyringUser(ref))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
		}
		return core.Credential{}, err
	}
	return core.UnmarshalCredentialFromStorage([]byte(raw))
}

func (v *KeyringVault) Delete(ctx context.Context, ref core.AccountRef) error {
	err := v.delete(v.service, v.keyringUser(ref))
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	indexErr := v.updateIndex(ctx, ref.Service, ref.Alias, false)
	if indexErr != nil {
		if err != nil {
			return errors.Join(core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}, indexErr)
		}
		return fmt.Errorf("credential deleted but account index update failed: %w", indexErr)
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	return nil
}

// List reads the alias index stored alongside credentials in the OS keyring.
// Old installations without an index are migrated automatically for the
// conventional "default" account; other old aliases become discoverable when
// they are next stored (for example, by logging in again).
func (v *KeyringVault) List(ctx context.Context, svc core.ServiceID) ([]core.AccountRef, error) {
	unlock, err := v.lockIndex(ctx)
	if err != nil {
		return nil, err
	}
	aliases, indexed, err := v.readIndex(svc)
	unlock()
	if err != nil {
		return nil, err
	}
	if !indexed {
		if _, err := v.get(v.service, v.keyringUser(core.AccountRef{Service: svc, Alias: "default"})); err == nil {
			if err := v.updateIndex(ctx, svc, "default", true); err != nil {
				return nil, fmt.Errorf("migrate default account index: %w", err)
			}
			aliases = []string{"default"}
		} else if !errors.Is(err, keyring.ErrNotFound) {
			return nil, err
		}
	}

	refs := make([]core.AccountRef, 0, len(aliases))
	for _, alias := range aliases {
		ref := core.AccountRef{Service: svc, Alias: core.AccountAlias(alias)}
		if _, err := v.get(v.service, v.keyringUser(ref)); err == nil {
			refs = append(refs, ref)
		} else if !errors.Is(err, keyring.ErrNotFound) {
			return nil, err
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Alias < refs[j].Alias })
	return refs, nil
}

func indexKey(svc core.ServiceID) string { return "__one_accounts_index__:" + string(svc) }

func (v *KeyringVault) readIndex(svc core.ServiceID) ([]string, bool, error) {
	raw, err := v.get(v.service, indexKey(svc))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var aliases []string
	if err := json.Unmarshal([]byte(raw), &aliases); err != nil {
		return nil, true, fmt.Errorf("decode account index for %s: %w", svc, err)
	}
	return aliases, true, nil
}

func (v *KeyringVault) updateIndex(ctx context.Context, svc core.ServiceID, alias core.AccountAlias, add bool) error {
	unlock, err := v.lockIndex(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	aliases, _, err := v.readIndex(svc)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(aliases)+1)
	for _, item := range aliases {
		seen[item] = true
	}
	if add && alias != "" {
		seen[string(alias)] = true
	} else {
		delete(seen, string(alias))
	}
	aliases = aliases[:0]
	for item := range seen {
		aliases = append(aliases, item)
	}
	sort.Strings(aliases)
	if len(aliases) == 0 {
		err := v.delete(v.service, indexKey(svc))
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	data, err := json.Marshal(aliases)
	if err != nil {
		return err
	}
	return v.set(v.service, indexKey(svc), string(data))
}

func (v *KeyringVault) lockIndex(ctx context.Context) (func(), error) {
	if v.lockPath == "" {
		return nil, errors.New("keyring account index lock path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(v.lockPath), 0o700); err != nil {
		return nil, fmt.Errorf("create account index lock directory: %w", err)
	}
	lock := flock.New(v.lockPath)
	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	locked, err := lock.TryLockContext(lockCtx, 25*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("lock keyring account index: %w", err)
	}
	if !locked {
		return nil, errors.New("timed out locking keyring account index")
	}
	return func() { _ = lock.Unlock() }, nil
}

var _ ports.Vault = (*KeyringVault)(nil)
