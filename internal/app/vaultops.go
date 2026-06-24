package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// VaultStatus reports the configured vault backends and total credential count.
type VaultStatus struct {
	vault   ports.Vault
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewVaultStatus creates the use case.
func NewVaultStatus(v ports.Vault, c ports.Catalog, s ports.ScopeStore) *VaultStatus {
	return &VaultStatus{vault: v, catalog: c, scope: s}
}

// VaultStatusOutput is the structured status response.
type VaultStatusOutput struct {
	Services map[string]int `json:"services"`
	Total    int            `json:"total"`
}

// Run summarizes the vault by counting accounts per in-scope service.
func (uc *VaultStatus) Run(ctx context.Context, dir string) (VaultStatusOutput, error) {
	sc, err := uc.scope.Load(dir)
	if err != nil {
		return VaultStatusOutput{}, err
	}
	out := VaultStatusOutput{Services: map[string]int{}}
	for id := range sc.Services {
		refs, err := uc.vault.List(ctx, id)
		if err != nil {
			continue
		}
		out.Services[string(id)] = len(refs)
		out.Total += len(refs)
	}
	return out, nil
}

// VaultExport serializes every credential reachable through the vault for the
// in-scope services. The output is plaintext JSON — callers MUST pipe to
// `age -p > backup.age` (or similar) before persisting.
type VaultExport struct {
	vault ports.Vault
	scope ports.ScopeStore
}

// NewVaultExport creates the use case.
func NewVaultExport(v ports.Vault, s ports.ScopeStore) *VaultExport {
	return &VaultExport{vault: v, scope: s}
}

// Run writes a JSON object {service:alias → credential storage blob} to out.
func (uc *VaultExport) Run(ctx context.Context, dir string, out io.Writer) error {
	sc, err := uc.scope.Load(dir)
	if err != nil {
		return err
	}
	bundle := map[string]json.RawMessage{}
	for id := range sc.Services {
		refs, err := uc.vault.List(ctx, id)
		if err != nil {
			continue
		}
		for _, ref := range refs {
			cred, err := uc.vault.Fetch(ctx, ref)
			if err != nil {
				return fmt.Errorf("export %s: %w", ref, err)
			}
			raw, err := cred.MarshalForStorage()
			if err != nil {
				return err
			}
			bundle[ref.String()] = raw
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(bundle)
}

// VaultImport reads a bundle previously produced by VaultExport and stores
// every credential into the vault. Existing entries are overwritten.
type VaultImport struct{ vault ports.Vault }

// NewVaultImport creates the use case.
func NewVaultImport(v ports.Vault) *VaultImport { return &VaultImport{vault: v} }

// Run decodes the bundle from in and stores each credential.
func (uc *VaultImport) Run(ctx context.Context, in io.Reader) error {
	raw, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	var bundle map[string]json.RawMessage
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return err
	}
	for key, blob := range bundle {
		cred, err := core.UnmarshalCredentialFromStorage(blob)
		if err != nil {
			return fmt.Errorf("import %s: %w", key, err)
		}
		if err := uc.vault.Store(ctx, cred.Ref(), cred); err != nil {
			return fmt.Errorf("import %s: %w", key, err)
		}
	}
	return nil
}

// VaultRotate re-encrypts every in-scope credential by fetching then re-storing
// it. With the age vault, this effectively re-encrypts under the currently
// configured passphrase (typically a new one set via ONE_AGE_PASSPHRASE).
type VaultRotate struct {
	vault ports.Vault
	scope ports.ScopeStore
	log   ports.Logger
	audit ports.Audit
}

// NewVaultRotate creates the use case.
func NewVaultRotate(v ports.Vault, s ports.ScopeStore, log ports.Logger) *VaultRotate {
	return &VaultRotate{vault: v, scope: s, log: log}
}

// WithAudit installs an audit recorder.
func (uc *VaultRotate) WithAudit(a ports.Audit) *VaultRotate { uc.audit = a; return uc }

// VaultRotateOutput summarizes how many credentials were rotated.
type VaultRotateOutput struct {
	Rotated int               `json:"rotated"`
	Failed  map[string]string `json:"failed,omitempty"`
}

// Run iterates every in-scope account and re-stores its credential.
func (uc *VaultRotate) Run(ctx context.Context, dir string) (VaultRotateOutput, error) {
	out := VaultRotateOutput{Failed: map[string]string{}}
	sc, err := uc.scope.Load(dir)
	if err != nil {
		return out, err
	}
	for id := range sc.Services {
		refs, err := uc.vault.List(ctx, id)
		if err != nil {
			out.Failed[string(id)] = err.Error()
			continue
		}
		for _, ref := range refs {
			cred, ferr := uc.vault.Fetch(ctx, ref)
			if ferr != nil {
				out.Failed[ref.String()] = ferr.Error()
				continue
			}
			if serr := uc.vault.Store(ctx, ref, cred); serr != nil {
				out.Failed[ref.String()] = serr.Error()
				emit(ctx, uc.audit, core.AuditEvent{
					Kind: core.AuditRotate, Service: string(ref.Service), Account: string(ref.Alias),
					Outcome: core.OutcomeError, Err: serr.Error(),
				})
				continue
			}
			out.Rotated++
			emit(ctx, uc.audit, core.AuditEvent{
				Kind: core.AuditRotate, Service: string(ref.Service), Account: string(ref.Alias),
				Outcome: core.OutcomeOK,
			})
		}
	}
	if uc.log != nil {
		uc.log.Info("vault rotate completed", "rotated", out.Rotated, "failed", len(out.Failed))
	}
	return out, nil
}

// Ensure os import stays in case of future helpers.
var _ = os.Stderr
