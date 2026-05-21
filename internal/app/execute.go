package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// BypassPermissionEnv, when set to a truthy value (1/true/yes), skips the
// project-env presence check AND the scope allowlist enforcement. Intended for
// agents running outside a checked-out project (e.g. one-shot scripts).
const BypassPermissionEnv = "ONECLI_BYPASS_PERMISSION"

func bypassPermissions() bool {
	switch os.Getenv(BypassPermissionEnv) {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	}
	return false
}

// ExecuteInput holds the parameters for the ExecuteAction use case.
type ExecuteInput struct {
	Service    string
	Action     string
	Inputs     map[string]any
	Account    string
	DryRun     bool
	ProjectDir string
}

// ExecuteOutput holds the result of a successful action execution.
type ExecuteOutput struct {
	Result  json.RawMessage
	TraceID string
	Calls   []ports.HTTPCall
}

// ExecuteAction orchestrates a full action execution: scope check, auth, run, audit.
type ExecuteAction struct {
	catalog ports.Catalog
	vault   ports.Vault
	runtime ports.Runtime
	scope   ports.ScopeStore
	auth    []ports.AuthProvider
	log     ports.Logger
	clock   ports.Clock
	crypto  ports.Crypto
	refresh *RefreshIfNeeded
	audit   ports.Audit
}

// WithAudit installs an audit recorder. EXEC events emitted with trace_id.
func (uc *ExecuteAction) WithAudit(a ports.Audit) *ExecuteAction { uc.audit = a; return uc }

// WithRefresh installs a refresh use case (lazy + file-locked).
// If not called, refresh falls back to the legacy in-process path.
func (uc *ExecuteAction) WithRefresh(r *RefreshIfNeeded) *ExecuteAction {
	uc.refresh = r
	return uc
}

// NewExecuteAction creates an ExecuteAction use case.
func NewExecuteAction(
	catalog ports.Catalog,
	vault ports.Vault,
	runtime ports.Runtime,
	scope ports.ScopeStore,
	auth []ports.AuthProvider,
	log ports.Logger,
	clock ports.Clock,
	crypto ports.Crypto,
) *ExecuteAction {
	return &ExecuteAction{
		catalog: catalog, vault: vault, runtime: runtime, scope: scope,
		auth: auth, log: log, clock: clock, crypto: crypto,
	}
}

// Run executes the action.
func (uc *ExecuteAction) Run(ctx context.Context, in ExecuteInput) (out ExecuteOutput, rerr error) {
	if in.Service == "" || in.Action == "" {
		return ExecuteOutput{}, core.ErrInputValidation{Field: "service+action", Reason: "required"}
	}
	defer func() {
		emit(ctx, uc.audit, core.AuditEvent{
			TraceID: out.TraceID, Kind: core.AuditExec,
			Service: in.Service, Account: in.Account, Action: in.Action,
			Outcome: outcomeOf(rerr), Err: errMsg(rerr),
		})
	}()
	svcID := core.ServiceID(in.Service)
	actionID := core.ActionID(in.Action)

	// 1. Resolve catalog.
	svc, err := uc.catalog.GetService(ctx, svcID)
	if err != nil {
		return ExecuteOutput{}, err
	}
	var action *core.Action
	for i := range svc.Actions {
		if svc.Actions[i].ID == actionID {
			action = &svc.Actions[i]
			break
		}
	}
	if action == nil {
		return ExecuteOutput{}, core.ErrUnknownAction{Service: svcID, Action: actionID}
	}

	// 2. Load scope (gated by env presence + bypass env var).
	bypass := bypassPermissions()
	if !bypass {
		dir := in.ProjectDir
		if dir == "" {
			if wd, werr := os.Getwd(); werr == nil {
				dir = wd
			}
		}
		if _, serr := os.Stat(filepath.Join(dir, ".onerc.yaml")); errors.Is(serr, os.ErrNotExist) {
			return ExecuteOutput{}, core.ErrNotInEnv{Dir: dir}
		}
	} else {
		uc.log.Warn("scope checks bypassed via "+BypassPermissionEnv, "service", svcID, "action", actionID)
	}
	scope, err := uc.scope.Load(in.ProjectDir)
	if err != nil {
		return ExecuteOutput{}, err
	}

	// 3. Check scope (skipped under bypass).
	if !bypass {
		perm := core.Permission{Service: svcID, Path: action.Permission}
		if !scope.Allows(perm) {
			return ExecuteOutput{}, core.ErrNotInScope{Permission: perm}
		}
	}

	// 4. Resolve account alias.
	alias := core.AccountAlias(in.Account)
	if alias == "" {
		alias = scope.AccountFor(svcID)
	}

	// 5. Fetch credential.
	cred, err := uc.vault.Fetch(ctx, core.AccountRef{Service: svcID, Alias: alias})
	if err != nil {
		return ExecuteOutput{}, err
	}

	// 6. Lazy refresh.
	if uc.refresh != nil {
		refreshed, rerr := uc.refresh.Run(ctx, cred)
		if rerr != nil {
			return ExecuteOutput{}, rerr
		}
		cred = refreshed
	} else if uc.clock != nil && cred.NeedsRefresh(uc.clock.Now()) {
		refreshed, rerr := uc.legacyRefresh(ctx, cred)
		if rerr != nil {
			return ExecuteOutput{}, rerr
		}
		cred = refreshed
		if err := uc.vault.Store(ctx, cred.Ref(), cred); err != nil {
			uc.log.Warn("refresh store failed", "service", svcID, "err", err.Error())
		}
	}

	// 7. Validate inputs.
	schema, err := core.ParseInputSchema(action.InputSchema)
	if err != nil {
		return ExecuteOutput{}, err
	}
	inputs := core.Inputs(in.Inputs)
	if inputs == nil {
		inputs = core.Inputs{}
	}
	inputs = schema.ApplyDefaults(inputs)
	if err := schema.Validate(inputs); err != nil {
		return ExecuteOutput{}, err
	}

	// 8. Generate trace ID.
	traceID := uc.newTraceID()

	// 9. Execute.
	req := ports.ExecuteRequest{
		Action:     *action,
		Inputs:     inputs,
		Credential: cred,
		DryRun:     in.DryRun,
		TraceID:    traceID,
	}
	result, err := uc.runtime.Execute(ctx, req)
	if err != nil {
		uc.log.Info("action failed",
			"service", svcID, "action", actionID, "trace_id", traceID, "err", err.Error())
		return ExecuteOutput{TraceID: traceID, Calls: result.Calls}, err
	}

	uc.log.Info("action succeeded",
		"service", svcID, "action", actionID, "trace_id", traceID, "calls", len(result.Calls))

	return ExecuteOutput{Result: result.Output, TraceID: traceID, Calls: result.Calls}, nil
}

// legacyRefresh finds a provider supporting the credential's kind and refreshes it.
// Used only when WithRefresh has not been called.
func (uc *ExecuteAction) legacyRefresh(ctx context.Context, cred core.Credential) (core.Credential, error) {
	for _, p := range uc.auth {
		if p.Supports(cred.Provider) {
			return p.Refresh(ctx, cred)
		}
	}
	return core.Credential{}, fmt.Errorf("no auth provider supports %s for refresh", cred.Provider)
}

// newTraceID returns a 16-byte hex identifier from the crypto port.
func (uc *ExecuteAction) newTraceID() string {
	if uc.crypto == nil {
		return ""
	}
	buf, err := uc.crypto.RandomBytes(8)
	if err != nil || len(buf) == 0 {
		return ""
	}
	return hex.EncodeToString(buf)
}

// ensure errors-as keeps typed errors visible above.
var _ = errors.As
