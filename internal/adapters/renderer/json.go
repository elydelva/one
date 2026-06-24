package renderer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// JSONRenderer writes machine-readable JSON output. Used when stdout is piped (agent mode).
type JSONRenderer struct {
	out io.Writer
	err io.Writer
}

// NewJSONRenderer creates a renderer that writes to the given writers.
func NewJSONRenderer(out, err io.Writer) *JSONRenderer {
	return &JSONRenderer{out: out, err: err}
}

// NewJSONRendererStd creates a renderer writing to os.Stdout / os.Stderr.
func NewJSONRendererStd() *JSONRenderer { return NewJSONRenderer(os.Stdout, os.Stderr) }

func (r *JSONRenderer) RenderResult(output json.RawMessage, traceID string) {
	payload := map[string]any{"ok": true, "data": output, "trace_id": traceID}
	_ = json.NewEncoder(r.out).Encode(payload)
}

func (r *JSONRenderer) RenderError(err error) {
	errObj := map[string]any{"code": errorCode(err), "message": err.Error()}
	if inst := installHint(err); inst != nil {
		errObj["install"] = inst
	}
	payload := map[string]any{"ok": false, "error": errObj}
	_ = json.NewEncoder(r.err).Encode(payload)
}

// errorCode maps a typed core error to a stable string code agents can branch on.
func errorCode(err error) string {
	var notAuth core.ErrNotAuthenticated
	var notInScope core.ErrNotInScope
	var setup core.ErrSetupRequired
	var notInEnv core.ErrNotInEnv
	var unknownSvc core.ErrUnknownService
	var unknownAct core.ErrUnknownAction
	var inputVal core.ErrInputValidation
	switch {
	case errors.As(err, &setup):
		return "setup_required"
	case errors.As(err, &notInEnv):
		return "setup_required"
	case errors.As(err, &notInScope):
		return "not_in_scope"
	case errors.As(err, &notAuth):
		return "not_authenticated"
	case errors.As(err, &unknownSvc), errors.As(err, &unknownAct):
		return "unknown_service"
	case errors.As(err, &inputVal):
		return "invalid_input"
	default:
		return "error"
	}
}

// installHint returns the structured install directive for setup_required
// errors so an agent can surface `one install <service> <guide>` to the user.
func installHint(err error) map[string]any {
	var setup core.ErrSetupRequired
	if !errors.As(err, &setup) {
		return nil
	}
	hint := map[string]any{
		"service":        string(setup.Service),
		"requires_human": setup.Human,
	}
	if setup.Guide != "" {
		hint["guide"] = setup.Guide
		hint["command"] = fmt.Sprintf("one install %s %s", setup.Service, setup.Guide)
	} else {
		hint["command"] = fmt.Sprintf("one install %s", setup.Service)
	}
	return hint
}

func (r *JSONRenderer) RenderInfo(markdown string) {
	payload := map[string]any{"info": markdown}
	_ = json.NewEncoder(r.out).Encode(payload)
}

func (r *JSONRenderer) RenderCapabilities(out ports.CapabilitiesOutput) {
	_ = json.NewEncoder(r.out).Encode(out)
}

var _ ports.Renderer = (*JSONRenderer)(nil)
