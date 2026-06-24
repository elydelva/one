package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"elydelva/one/internal/adapters/catalog"
	adapterclock "elydelva/one/internal/adapters/clock"
	"elydelva/one/internal/adapters/transport"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fakeapi"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testing", "fixture", "catalog", "v1-minimal")
}

// staticResolver returns the same service for any ID (test helper).
type staticResolver struct{ svc *core.Service }

func (r *staticResolver) Resolve(_ context.Context, _ core.ServiceID) (*core.Service, error) {
	return r.svc, nil
}

func newDeclWithFakeAPI(t *testing.T, routes []fakeapi.Route) (*DeclarativeRuntime, *fakeapi.Server, *core.Service) {
	t.Helper()
	srv := fakeapi.New(t, routes)
	c := catalog.NewCatalogFS(fixtureRoot(t))
	svc, err := c.GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	svc.BaseURL = srv.URL // re-route to fakeapi

	tr := transport.NewNetHTTP(transport.WithAllowHTTP(true), transport.WithAllowedHosts("127.0.0.1"), transport.WithTimeout(5*time.Second))
	return NewDeclarativeRuntime(tr, adapterclock.NewSystemClock(), &staticResolver{svc: svc}), srv, svc
}

func TestDeclarative_GetHappyPath(t *testing.T) {
	rt, _, svc := newDeclWithFakeAPI(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues/1", Status: 200, Body: map[string]any{"number": 1, "title": "Hi"}},
	})
	action := findAction(t, svc, "issues.read")
	res, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action:     action,
		Inputs:     core.Inputs{"owner": "x", "repo": "y", "issue_number": 1},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(res.Output, &got); err != nil {
		t.Fatalf("output: %v", err)
	}
	if got["title"] != "Hi" {
		t.Errorf("got %+v", got)
	}
	if len(res.Calls) != 1 || res.Calls[0].Status != 200 {
		t.Errorf("calls = %+v", res.Calls)
	}
}

func TestDeclarative_AuthHeaderInjected(t *testing.T) {
	rt, srv, svc := newDeclWithFakeAPI(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues/1", Status: 200, Body: map[string]any{"ok": true}},
	})
	action := findAction(t, svc, "issues.read")
	_, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action:     action,
		Inputs:     core.Inputs{"owner": "x", "repo": "y", "issue_number": 1},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("ghp_xyz")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := srv.Received()[0].Header.Get("Authorization")
	if got != "Bearer ghp_xyz" {
		t.Errorf("got Authorization %q", got)
	}
}

func TestDeclarative_404MapsToErrNotFound(t *testing.T) {
	rt, _, svc := newDeclWithFakeAPI(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues/99", Status: 404, Body: map[string]any{"message": "Not Found"}},
	})
	action := findAction(t, svc, "issues.read")
	_, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action: action, Inputs: core.Inputs{"owner": "x", "repo": "y", "issue_number": 99},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
	})
	var nf core.ErrNotFound
	if !errors.As(err, &nf) {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestDeclarative_DryRunSkipsRequest(t *testing.T) {
	rt, srv, svc := newDeclWithFakeAPI(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues/1", Status: 500, Body: map[string]any{"never": "called"}},
	})
	action := findAction(t, svc, "issues.read")
	res, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action: action, Inputs: core.Inputs{"owner": "x", "repo": "y", "issue_number": 1},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("dry: %v", err)
	}
	if len(srv.Received()) != 0 {
		t.Errorf("dry-run hit fakeapi: %+v", srv.Received())
	}
	if !strings.Contains(string(res.Output), "issues/1") {
		t.Errorf("expected URL in preview, got %s", res.Output)
	}
}

func TestDeclarative_POSTSendsBody(t *testing.T) {
	rt, srv, svc := newDeclWithFakeAPI(t, []fakeapi.Route{
		{Method: "POST", Path: "/repos/x/y/issues", Status: 201, Body: map[string]any{"number": 42}},
	})
	action := findAction(t, svc, "issues.create")
	_, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action: action,
		Inputs: core.Inputs{
			"owner": "x", "repo": "y", "title": "Bug", "body": "describe",
		},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := srv.Received()[0]
	if got.Method != http.MethodPost {
		t.Errorf("method = %s", got.Method)
	}
	if !strings.Contains(string(got.Body), `"title":"Bug"`) {
		t.Errorf("body = %s", got.Body)
	}
	if strings.Contains(string(got.Body), `"owner"`) {
		t.Errorf("path input leaked into body: %s", got.Body)
	}
}

func TestDeclarative_QueryParamsApplied(t *testing.T) {
	rt, srv, svc := newDeclWithFakeAPI(t, []fakeapi.Route{
		{Method: "GET", Path: "/repos/x/y/issues", Status: 200, Body: []any{}},
	})
	action := findAction(t, svc, "issues.list")
	_, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action:     action,
		Inputs:     core.Inputs{"owner": "x", "repo": "y", "state": "closed"},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if srv.Received()[0].Query["state"] != "closed" {
		t.Errorf("query = %+v", srv.Received()[0].Query)
	}
}

func httpServerFromMux(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newDeclWithMux spins a httptest.Server with the given mux and builds a
// DeclarativeRuntime pointing to it. Returns a synthetic service with the test
// URL as base.
func newDeclWithMux(t *testing.T, mux *http.ServeMux) (*DeclarativeRuntime, string, *core.Service) {
	t.Helper()
	srv := httpServerFromMux(t, mux)
	svc := &core.Service{ID: "things", BaseURL: srv.URL, Injection: map[core.ProviderKind]core.AuthInjection{
		core.ProviderPAT: {Header: "Authorization", Format: "Bearer {access_token}"},
	}}
	tr := transport.NewNetHTTP(transport.WithAllowHTTP(true), transport.WithAllowedHosts("127.0.0.1"), transport.WithTimeout(5*time.Second))
	return NewDeclarativeRuntime(tr, adapterclock.NewSystemClock(), &staticResolver{svc: svc}), srv.URL, svc
}

func findAction(t *testing.T, svc *core.Service, id core.ActionID) core.Action {
	t.Helper()
	for _, a := range svc.Actions {
		if a.ID == id {
			return a
		}
	}
	t.Fatalf("action %s not found", id)
	return core.Action{}
}
