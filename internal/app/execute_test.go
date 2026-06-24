package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"elydelva/one/internal/adapters/crypto"
	"elydelva/one/internal/adapters/scopestore"
	"elydelva/one/internal/core"
	"elydelva/one/internal/testing/fake"
)

// fixtureService returns a minimal in-memory service for ExecuteAction tests.
func fixtureService() core.Service {
	return core.Service{
		ID: "github", Name: "GitHub",
		BaseURL: "https://api.github.com",
		Actions: []core.Action{
			{
				ID: "issues.read", Service: "github", Permission: "issues.read",
				Request: &core.RequestSpec{Method: "GET", Path: "/repos/{owner}/{repo}/issues/{n}"},
				InputSchema: json.RawMessage(`[
					{"name":"owner","type":"string","required":true,"location":"path"},
					{"name":"repo","type":"string","required":true,"location":"path"},
					{"name":"n","type":"integer","required":true,"location":"path"}
				]`),
			},
		},
	}
}

func newExec(t *testing.T) (*ExecuteAction, *fake.Vault, *fake.Runtime, string) {
	t.Helper()
	dir := t.TempDir()
	ss := scopestore.NewFileScopeStore()
	// Grant the scope.
	scope := core.Scope{Version: 1, Services: map[core.ServiceID]core.ServiceScope{
		"github": {Allow: []core.PermissionPattern{"issues.*"}},
	}}
	if err := ss.Save(dir, scope); err != nil {
		t.Fatalf("save scope: %v", err)
	}

	vlt := fake.NewVault()
	rt := fake.NewRuntime()
	cat := fake.NewCatalog([]core.Service{fixtureService()})

	uc := NewExecuteAction(cat, vlt, rt, ss, nil, fake.NewLogger(), fake.NewClock(), crypto.NewStdCrypto())
	return uc, vlt, rt, dir
}

func TestExecute_Happy(t *testing.T) {
	uc, vlt, rt, dir := newExec(t)
	_ = vlt.Store(context.Background(), core.AccountRef{Service: "github", Alias: "default"}, core.Credential{
		Provider: core.ProviderPAT, Service: "github", Account: "default", AccessToken: core.NewSecret("t"),
	})
	rt.SetResponse("github", "issues.read", json.RawMessage(`{"number":1}`))

	out, err := uc.Run(context.Background(), ExecuteInput{
		Service: "github", Action: "issues.read",
		Inputs:     map[string]any{"owner": "x", "repo": "y", "n": 1},
		ProjectDir: dir,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if string(out.Result) != `{"number":1}` {
		t.Errorf("output = %s", out.Result)
	}
	if out.TraceID == "" {
		t.Errorf("trace_id empty")
	}
}

func TestExecute_UnknownService(t *testing.T) {
	uc, _, _, dir := newExec(t)
	_, err := uc.Run(context.Background(), ExecuteInput{Service: "notion", Action: "x.y", ProjectDir: dir})
	var unk core.ErrUnknownService
	if !errors.As(err, &unk) {
		t.Errorf("got %T: %v", err, err)
	}
}

func TestExecute_UnknownAction(t *testing.T) {
	uc, _, _, dir := newExec(t)
	_, err := uc.Run(context.Background(), ExecuteInput{Service: "github", Action: "nope.xxx", ProjectDir: dir})
	var unk core.ErrUnknownAction
	if !errors.As(err, &unk) {
		t.Errorf("got %T: %v", err, err)
	}
}

func TestExecute_NotInScope(t *testing.T) {
	dir := t.TempDir()
	ss := scopestore.NewFileScopeStore()
	_ = ss.Save(dir, core.Scope{Version: 1, Services: map[core.ServiceID]core.ServiceScope{}})

	uc := NewExecuteAction(
		fake.NewCatalog([]core.Service{fixtureService()}),
		fake.NewVault(),
		fake.NewRuntime(),
		ss, nil, fake.NewLogger(), fake.NewClock(), crypto.NewStdCrypto(),
	)
	_, err := uc.Run(context.Background(), ExecuteInput{Service: "github", Action: "issues.read", ProjectDir: dir})
	var nis core.ErrNotInScope
	if !errors.As(err, &nis) {
		t.Errorf("got %T: %v", err, err)
	}
}

func TestExecute_NotAuthenticated(t *testing.T) {
	uc, _, _, dir := newExec(t)
	_, err := uc.Run(context.Background(), ExecuteInput{
		Service: "github", Action: "issues.read",
		Inputs:     map[string]any{"owner": "x", "repo": "y", "n": 1},
		ProjectDir: dir,
	})
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("got %T: %v", err, err)
	}
}

func TestExecute_NotInEnv(t *testing.T) {
	dir := t.TempDir() // no .onerc.yaml written
	uc := NewExecuteAction(
		fake.NewCatalog([]core.Service{fixtureService()}),
		fake.NewVault(),
		fake.NewRuntime(),
		scopestore.NewFileScopeStore(), nil, fake.NewLogger(), fake.NewClock(), crypto.NewStdCrypto(),
	)
	_, err := uc.Run(context.Background(), ExecuteInput{
		Service: "github", Action: "issues.read",
		Inputs:     map[string]any{"owner": "x", "repo": "y", "n": 1},
		ProjectDir: dir,
	})
	var nie core.ErrNotInEnv
	if !errors.As(err, &nie) {
		t.Fatalf("got %T: %v", err, err)
	}
	if nie.Dir != dir {
		t.Errorf("dir = %q want %q", nie.Dir, dir)
	}
}

func TestExecute_BypassPermissionsSkipsScopeCheck(t *testing.T) {
	dir := t.TempDir() // no .onerc.yaml, no scope grant
	t.Setenv("ONECLI_BYPASS_PERMISSION", "1")

	vlt := fake.NewVault()
	_ = vlt.Store(context.Background(), core.AccountRef{Service: "github", Alias: "default"}, core.Credential{
		Provider: core.ProviderPAT, Service: "github", Account: "default", AccessToken: core.NewSecret("t"),
	})
	rt := fake.NewRuntime()
	rt.SetResponse("github", "issues.read", json.RawMessage(`{"number":1}`))

	uc := NewExecuteAction(
		fake.NewCatalog([]core.Service{fixtureService()}),
		vlt, rt,
		scopestore.NewFileScopeStore(), nil, fake.NewLogger(), fake.NewClock(), crypto.NewStdCrypto(),
	)
	out, err := uc.Run(context.Background(), ExecuteInput{
		Service: "github", Action: "issues.read",
		Inputs:     map[string]any{"owner": "x", "repo": "y", "n": 1},
		ProjectDir: dir,
	})
	if err != nil {
		t.Fatalf("bypass should succeed: %v", err)
	}
	if string(out.Result) != `{"number":1}` {
		t.Errorf("output = %s", out.Result)
	}
}

func TestExecute_InvalidInput(t *testing.T) {
	uc, vlt, _, dir := newExec(t)
	_ = vlt.Store(context.Background(), core.AccountRef{Service: "github", Alias: "default"}, core.Credential{
		Provider: core.ProviderPAT, Service: "github", Account: "default", AccessToken: core.NewSecret("t"),
	})
	_, err := uc.Run(context.Background(), ExecuteInput{
		Service: "github", Action: "issues.read",
		Inputs:     map[string]any{"owner": "x"}, // missing repo, n
		ProjectDir: dir,
	})
	var iv core.ErrInputValidation
	if !errors.As(err, &iv) {
		t.Errorf("got %T: %v", err, err)
	}
}
