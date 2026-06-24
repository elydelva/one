package vault

import (
	"context"
	"os"
	"strings"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// envVarPrefix is the prefix for credential environment variables.
// Format: ONE_CREDS_<SERVICE>_<ACCOUNT>=<json>
const envVarPrefix = "ONE_CREDS_"

// EnvVarVault reads credentials from environment variables (ONE_CREDS_<SVC>_<ACCOUNT>=<json>).
// Read-only: Store/Delete return ErrReadOnly.
type EnvVarVault struct {
	getenv  func(string) string
	environ func() []string
}

// NewEnvVarVault creates a vault that reads from process environment variables.
func NewEnvVarVault() *EnvVarVault {
	return &EnvVarVault{getenv: os.Getenv, environ: os.Environ}
}

// envKey builds the env var name for a given account ref.
func envKey(ref core.AccountRef) string {
	return envVarPrefix + strings.ToUpper(string(ref.Service)) + "_" + strings.ToUpper(string(ref.Alias))
}

func (v *EnvVarVault) Store(_ context.Context, _ core.AccountRef, _ core.Credential) error {
	return core.ErrReadOnly{Source: "env"}
}

func (v *EnvVarVault) Fetch(_ context.Context, ref core.AccountRef) (core.Credential, error) {
	raw := v.getenv(envKey(ref))
	if raw == "" {
		return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service, Account: ref.Alias}
	}
	cred, err := core.UnmarshalCredentialFromStorage([]byte(raw))
	if err != nil {
		return core.Credential{}, core.ErrInvalidScopeFile{Path: envKey(ref), Reason: err.Error()}
	}
	// Ensure ref consistency.
	if cred.Service == "" {
		cred.Service = ref.Service
	}
	if cred.Account == "" {
		cred.Account = ref.Alias
	}
	return cred, nil
}

func (v *EnvVarVault) Delete(_ context.Context, _ core.AccountRef) error {
	return core.ErrReadOnly{Source: "env"}
}

func (v *EnvVarVault) List(_ context.Context, svc core.ServiceID) ([]core.AccountRef, error) {
	wantPrefix := envVarPrefix + strings.ToUpper(string(svc)) + "_"
	var refs []core.AccountRef
	for _, kv := range v.environ() {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		key := kv[:eq]
		if !strings.HasPrefix(key, wantPrefix) {
			continue
		}
		alias := strings.ToLower(strings.TrimPrefix(key, wantPrefix))
		if alias == "" {
			continue
		}
		refs = append(refs, core.AccountRef{Service: svc, Alias: core.AccountAlias(alias)})
	}
	return refs, nil
}

var _ ports.Vault = (*EnvVarVault)(nil)
