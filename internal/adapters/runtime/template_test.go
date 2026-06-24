package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elydelva/one/internal/core"
)

func TestInterpolate_Basic(t *testing.T) {
	out, err := Interpolate("/repos/{owner}/{repo}/issues", map[string]any{"owner": "x", "repo": "y"}, true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != "/repos/x/y/issues" {
		t.Errorf("got %q", out)
	}
}

func TestInterpolate_UnresolvedFails(t *testing.T) {
	_, err := Interpolate("/{nope}", map[string]any{}, true)
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestInterpolate_PathTraversalBlocked(t *testing.T) {
	_, err := Interpolate("/repos/{owner}/x", map[string]any{"owner": "../../etc/passwd"}, true)
	if err == nil {
		t.Errorf("expected error for path traversal")
	}
}

func TestInterpolate_EscapesSpecialChars(t *testing.T) {
	out, _ := Interpolate("/q/{q}", map[string]any{"q": "a b&c"}, true)
	if !strings.Contains(out, "%20") {
		t.Errorf("expected space escape, got %q", out)
	}
}

func TestInterpolate_NoEscape(t *testing.T) {
	out, err := Interpolate("Bearer {tok}", map[string]any{"tok": "ghp_xyz"}, false)
	if err != nil || out != "Bearer ghp_xyz" {
		t.Errorf("got %q err=%v", out, err)
	}
}

func TestInterpolate_JSONFilter_String(t *testing.T) {
	out, err := Interpolate(`{"content":{c|json}}`, map[string]any{"c": "Hello \"world\"\nNext"}, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := `{"content":"Hello \"world\"\nNext"}`
	if out != want {
		t.Errorf("got %q want %q", out, want)
	}
}

func TestInterpolate_JSONFilter_Bool(t *testing.T) {
	out, _ := Interpolate(`{"b":{flag|json}}`, map[string]any{"flag": true}, false)
	if out != `{"b":true}` {
		t.Errorf("got %q", out)
	}
}

func TestInterpolate_JSONFilter_Object(t *testing.T) {
	out, _ := Interpolate(`{"p":{pos|json}}`, map[string]any{"pos": map[string]any{"type": "end"}}, false)
	if out != `{"p":{"type":"end"}}` {
		t.Errorf("got %q", out)
	}
}

func TestInterpolate_UnknownFilter(t *testing.T) {
	_, err := Interpolate(`{x|nope}`, map[string]any{"x": "a"}, false)
	if err == nil {
		t.Errorf("expected error for unknown filter")
	}
}

func TestInjectAuth_Bearer(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com/", nil)
	cred := core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("ghp_abc")}
	inj := core.AuthInjection{Header: "Authorization", Format: "Bearer {access_token}"}
	if err := InjectAuth(req, cred, inj); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer ghp_abc" {
		t.Errorf("got %q", req.Header.Get("Authorization"))
	}
}

func TestInjectAuth_MissingRule(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://x/", nil)
	err := InjectAuth(req, core.Credential{}, core.AuthInjection{})
	if err == nil {
		t.Errorf("expected error")
	}
}
