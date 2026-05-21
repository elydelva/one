package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"filippo.io/age"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// AgeVault stores credentials in an age-encrypted file (default ~/.one/vault.age).
//
// File format: age-encrypted JSON object keyed by "<service>:<alias>".
//
// Passphrase resolution:
//
//  1. ONE_AGE_PASSPHRASE env var
//  2. PassphraseFunc supplied at construction (for prompts)
//
// All operations are serialized by an in-process mutex; cross-process safety
// is handled by the lock layer in app/refresh.
type AgeVault struct {
	path           string
	passphraseFunc func() (string, error)

	mu sync.Mutex
}

// NewAgeVault creates a vault backed by the given age-encrypted file path.
// pass is the passphrase resolver; if nil, ONE_AGE_PASSPHRASE env var is required.
func NewAgeVault(path string, pass func() (string, error)) *AgeVault {
	return &AgeVault{path: path, passphraseFunc: pass}
}

func (v *AgeVault) passphrase() (string, error) {
	if p := os.Getenv("ONE_AGE_PASSPHRASE"); p != "" {
		return p, nil
	}
	if v.passphraseFunc != nil {
		return v.passphraseFunc()
	}
	return "", errors.New("age vault: no passphrase (set ONE_AGE_PASSPHRASE or supply a passphrase func)")
}

type ageStore struct {
	Creds map[string]json.RawMessage `json:"creds"`
}

func (v *AgeVault) loadLocked(pass string) (*ageStore, error) {
	raw, err := os.ReadFile(v.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ageStore{Creds: map[string]json.RawMessage{}}, nil
		}
		return nil, err
	}
	id, err := age.NewScryptIdentity(pass)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(raw), id)
	if err != nil {
		return nil, fmt.Errorf("age decrypt: %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var s ageStore
	if len(plain) == 0 {
		s.Creds = map[string]json.RawMessage{}
		return &s, nil
	}
	if err := json.Unmarshal(plain, &s); err != nil {
		return nil, fmt.Errorf("age vault: parse plaintext: %w", err)
	}
	if s.Creds == nil {
		s.Creds = map[string]json.RawMessage{}
	}
	return &s, nil
}

func (v *AgeVault) saveLocked(s *ageStore, pass string) error {
	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return err
	}
	plain, err := json.Marshal(s)
	if err != nil {
		return err
	}
	recipient, err := age.NewScryptRecipient(pass)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return err
	}
	if _, err := w.Write(plain); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, v.path)
}

func (v *AgeVault) Store(_ context.Context, ref core.AccountRef, cred core.Credential) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	pass, err := v.passphrase()
	if err != nil {
		return err
	}
	s, err := v.loadLocked(pass)
	if err != nil {
		return err
	}
	raw, err := cred.MarshalForStorage()
	if err != nil {
		return err
	}
	s.Creds[ref.String()] = raw
	return v.saveLocked(s, pass)
}

func (v *AgeVault) Fetch(_ context.Context, ref core.AccountRef) (core.Credential, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	pass, err := v.passphrase()
	if err != nil {
		return core.Credential{}, err
	}
	s, err := v.loadLocked(pass)
	if err != nil {
		return core.Credential{}, err
	}
	raw, ok := s.Creds[ref.String()]
	if !ok {
		return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	return core.UnmarshalCredentialFromStorage(raw)
}

func (v *AgeVault) Delete(_ context.Context, ref core.AccountRef) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	pass, err := v.passphrase()
	if err != nil {
		return err
	}
	s, err := v.loadLocked(pass)
	if err != nil {
		return err
	}
	if _, ok := s.Creds[ref.String()]; !ok {
		return core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	delete(s.Creds, ref.String())
	return v.saveLocked(s, pass)
}

func (v *AgeVault) List(_ context.Context, svc core.ServiceID) ([]core.AccountRef, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	pass, err := v.passphrase()
	if err != nil {
		return nil, err
	}
	s, err := v.loadLocked(pass)
	if err != nil {
		return nil, err
	}
	prefix := string(svc) + ":"
	var out []core.AccountRef
	for key := range s.Creds {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		alias := key[len(prefix):]
		out = append(out, core.AccountRef{Service: svc, Alias: core.AccountAlias(alias)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Alias < out[j].Alias })
	return out, nil
}

var _ ports.Vault = (*AgeVault)(nil)
