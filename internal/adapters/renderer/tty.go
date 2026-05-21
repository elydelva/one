package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"elydelva/one/internal/ports"
)

// TTYRenderer writes human-readable output with color and formatting. Used in interactive terminals.
type TTYRenderer struct {
	out      io.Writer
	err      io.Writer
	noColor  bool
	mdRender *glamour.TermRenderer
}

// NewTTYRenderer creates a renderer that writes to the given writers.
func NewTTYRenderer(out, err io.Writer) *TTYRenderer {
	noColor := os.Getenv("NO_COLOR") != ""
	if noColor {
		lipgloss.SetColorProfile(0) // termenv.Ascii
	}
	md, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
	return &TTYRenderer{out: out, err: err, noColor: noColor, mdRender: md}
}

// NewTTYRendererStd creates a renderer writing to os.Stdout / os.Stderr.
func NewTTYRendererStd() *TTYRenderer { return NewTTYRenderer(os.Stdout, os.Stderr) }

var (
	styleKey     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleValue   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	styleErr     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	styleTraceID = lipgloss.NewStyle().Faint(true)
	styleService = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleAction  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

func (r *TTYRenderer) RenderResult(output json.RawMessage, traceID string) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, output, "", "  "); err != nil {
		_, _ = fmt.Fprintf(r.out, "%s\n", output)
	} else {
		_, _ = fmt.Fprintln(r.out, pretty.String())
	}
	if traceID != "" {
		_, _ = fmt.Fprintln(r.out, styleTraceID.Render("trace: "+traceID))
	}
}

func (r *TTYRenderer) RenderError(err error) {
	_, _ = fmt.Fprintln(r.err, styleErr.Render("error:")+" "+err.Error())
}

func (r *TTYRenderer) RenderInfo(markdown string) {
	if r.mdRender == nil {
		_, _ = fmt.Fprintln(r.out, markdown)
		return
	}
	rendered, err := r.mdRender.Render(markdown)
	if err != nil {
		_, _ = fmt.Fprintln(r.out, markdown)
		return
	}
	_, _ = fmt.Fprint(r.out, rendered)
}

func (r *TTYRenderer) RenderCapabilities(out ports.CapabilitiesOutput) {
	services := make([]ports.ServiceCapability, len(out.Services))
	copy(services, out.Services)
	sort.Slice(services, func(i, j int) bool { return services[i].ID < services[j].ID })
	for _, svc := range services {
		_, _ = fmt.Fprintln(r.out, styleService.Render(svc.ID))
		actions := append([]string(nil), svc.Actions...)
		sort.Strings(actions)
		for _, a := range actions {
			_, _ = fmt.Fprintln(r.out, "  "+styleAction.Render(a))
		}
	}
}

// RenderKV renders a key/value block. Used by ad-hoc commands (doctor, trace, accounts).
func (r *TTYRenderer) RenderKV(pairs ...[2]string) {
	for _, p := range pairs {
		_, _ = fmt.Fprintln(r.out, styleKey.Render(p[0]+":")+" "+styleValue.Render(p[1]))
	}
}

var _ ports.Renderer = (*TTYRenderer)(nil)
