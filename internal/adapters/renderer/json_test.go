package renderer

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"elydelva/one/internal/core"
)

func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", b, err)
	}
	return m
}

func TestRenderResultOK(t *testing.T) {
	var out, errBuf bytes.Buffer
	r := NewJSONRenderer(&out, &errBuf)
	r.RenderResult(json.RawMessage(`{"x":1}`), "trace-123")
	m := decode(t, out.Bytes())
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %v", m["ok"])
	}
	if m["trace_id"] != "trace-123" {
		t.Errorf("expected trace_id, got %v", m["trace_id"])
	}
}

func TestRenderErrorCodes(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"setup", core.ErrSetupRequired{Service: "github", Guide: "pat", Reason: "no token", Human: true}, "setup_required"},
		{"notInEnv", core.ErrNotInEnv{Dir: "/tmp"}, "setup_required"},
		{"notInScope", core.ErrNotInScope{Permission: core.Permission{Service: "github", Path: "issues.create"}}, "not_in_scope"},
		{"notAuth", core.ErrNotAuthenticated{Service: "github"}, "not_authenticated"},
		{"unknownSvc", core.ErrUnknownService{Service: "nope"}, "unknown_service"},
		{"unknownAct", core.ErrUnknownAction{Service: "github", Action: "nope"}, "unknown_service"},
		{"input", core.ErrInputValidation{Field: "x", Reason: "required"}, "invalid_input"},
		{"generic", errors.New("boom"), "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			r := NewJSONRenderer(&out, &errBuf)
			r.RenderError(tc.err)
			m := decode(t, errBuf.Bytes())
			if m["ok"] != false {
				t.Errorf("expected ok=false, got %v", m["ok"])
			}
			errObj, _ := m["error"].(map[string]any)
			if errObj == nil {
				t.Fatalf("missing error object in %v", m)
			}
			if errObj["code"] != tc.code {
				t.Errorf("expected code %q, got %v", tc.code, errObj["code"])
			}
		})
	}
}

func TestRenderErrorInstallCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	r := NewJSONRenderer(&out, &errBuf)
	r.RenderError(core.ErrSetupRequired{Service: "github", Guide: "pat", Reason: "no token", Human: true})
	m := decode(t, errBuf.Bytes())
	errObj := m["error"].(map[string]any)
	inst, ok := errObj["install"].(map[string]any)
	if !ok {
		t.Fatalf("expected install object, got %v", errObj)
	}
	if inst["command"] != "one install github pat" {
		t.Errorf("expected install.command 'one install github pat', got %v", inst["command"])
	}
	if inst["requires_human"] != true {
		t.Errorf("expected requires_human=true, got %v", inst["requires_human"])
	}
}

func TestRenderErrorNoInstallForNonSetup(t *testing.T) {
	var out, errBuf bytes.Buffer
	r := NewJSONRenderer(&out, &errBuf)
	r.RenderError(core.ErrNotAuthenticated{Service: "github"})
	m := decode(t, errBuf.Bytes())
	errObj := m["error"].(map[string]any)
	if _, ok := errObj["install"]; ok {
		t.Errorf("did not expect install hint for non-setup error: %v", errObj)
	}
}
