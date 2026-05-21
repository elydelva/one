package renderer

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"elydelva/one/internal/ports"
)

func TestTTYRenderer_RenderResult(t *testing.T) {
	var out, errb bytes.Buffer
	r := NewTTYRenderer(&out, &errb)
	r.RenderResult(json.RawMessage(`{"k":1}`), "abc123")
	s := out.String()
	if !strings.Contains(s, `"k"`) || !strings.Contains(s, "abc123") {
		t.Fatalf("unexpected output: %q", s)
	}
}

func TestTTYRenderer_RenderError(t *testing.T) {
	var out, errb bytes.Buffer
	r := NewTTYRenderer(&out, &errb)
	r.RenderError(errors.New("boom"))
	if !strings.Contains(errb.String(), "boom") {
		t.Fatalf("expected error message, got %q", errb.String())
	}
}

func TestTTYRenderer_RenderCapabilities(t *testing.T) {
	var out, errb bytes.Buffer
	r := NewTTYRenderer(&out, &errb)
	r.RenderCapabilities(ports.CapabilitiesOutput{Services: []ports.ServiceCapability{
		{ID: "github", Actions: []string{"issues.list"}},
	}})
	s := out.String()
	if !strings.Contains(s, "github") || !strings.Contains(s, "issues.list") {
		t.Fatalf("expected service+action, got %q", s)
	}
}

func TestTTYRenderer_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var out, errb bytes.Buffer
	r := NewTTYRenderer(&out, &errb)
	r.RenderError(errors.New("x"))
	// Just verify no panic and output present
	if !strings.Contains(errb.String(), "x") {
		t.Fatalf("expected output")
	}
}
