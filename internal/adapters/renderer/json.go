package renderer

import (
	"encoding/json"
	"io"
	"os"

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
	payload := map[string]any{"data": output, "trace_id": traceID}
	_ = json.NewEncoder(r.out).Encode(payload)
}

func (r *JSONRenderer) RenderError(err error) {
	payload := map[string]any{"error": map[string]string{"message": err.Error()}}
	_ = json.NewEncoder(r.err).Encode(payload)
}

func (r *JSONRenderer) RenderInfo(markdown string) {
	payload := map[string]any{"info": markdown}
	_ = json.NewEncoder(r.out).Encode(payload)
}

func (r *JSONRenderer) RenderCapabilities(out ports.CapabilitiesOutput) {
	_ = json.NewEncoder(r.out).Encode(out)
}

var _ ports.Renderer = (*JSONRenderer)(nil)
