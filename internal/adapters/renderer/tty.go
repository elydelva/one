package renderer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"elydelva/one/internal/ports"
)

// TTYRenderer writes human-readable output with color and formatting. Used in interactive terminals.
type TTYRenderer struct {
	out io.Writer
	err io.Writer
}

// NewTTYRenderer creates a renderer that writes to the given writers.
func NewTTYRenderer(out, err io.Writer) *TTYRenderer {
	return &TTYRenderer{out: out, err: err}
}

// NewTTYRendererStd creates a renderer writing to os.Stdout / os.Stderr.
func NewTTYRendererStd() *TTYRenderer { return NewTTYRenderer(os.Stdout, os.Stderr) }

func (r *TTYRenderer) RenderResult(output json.RawMessage, _ string) {
	// TODO: pretty-print with lipgloss
	_, _ = fmt.Fprintf(r.out, "%s\n", output)
}

func (r *TTYRenderer) RenderError(err error) {
	_, _ = fmt.Fprintf(r.err, "error: %s\n", err)
}

func (r *TTYRenderer) RenderInfo(markdown string) {
	// TODO: render markdown with glamour
	_, _ = fmt.Fprintln(r.out, markdown)
}

func (r *TTYRenderer) RenderCapabilities(out ports.CapabilitiesOutput) {
	for _, svc := range out.Services {
		_, _ = fmt.Fprintf(r.out, "%s: %v\n", svc.ID, svc.Actions)
	}
}

var _ ports.Renderer = (*TTYRenderer)(nil)
