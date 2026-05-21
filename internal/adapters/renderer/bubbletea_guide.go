package renderer

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"elydelva/one/internal/core"
)

// InteractiveGuide renders an install guide as a navigable checklist when stdout
// is a TTY. Falls back to plain text otherwise.
func InteractiveGuide(out io.Writer, guide *core.InstallGuide) error {
	if guide == nil {
		return nil
	}
	if !isInteractive() {
		return plainGuide(out, guide)
	}
	steps := parseSteps(guide.Content)
	if len(steps) == 0 {
		return plainGuide(out, guide)
	}
	m := newGuideModel(guide, steps)
	p := tea.NewProgram(m, tea.WithOutput(out))
	_, err := p.Run()
	return err
}

func plainGuide(out io.Writer, g *core.InstallGuide) error {
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
	if err == nil {
		rendered, _ := r.Render("# " + g.Title + "\n\n" + g.Content)
		_, _ = fmt.Fprint(out, rendered)
	} else {
		_, _ = fmt.Fprintln(out, g.Title)
		_, _ = fmt.Fprintln(out, g.Content)
	}
	if g.Verify != nil {
		_, _ = fmt.Fprintf(out, "\nVerify: one %s %s\n", g.Service, g.Verify.Action)
	}
	return nil
}

func isInteractive() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// parseSteps extracts markdown checklist lines ("- [ ] foo") from content.
func parseSteps(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "- [ ]") || strings.HasPrefix(l, "- [x]") || strings.HasPrefix(l, "- [X]") {
			out = append(out, strings.TrimSpace(l[5:]))
		}
	}
	return out
}

type guideModel struct {
	guide  *core.InstallGuide
	steps  []string
	cursor int
	done   []bool
	quit   bool
}

func newGuideModel(g *core.InstallGuide, steps []string) guideModel {
	return guideModel{guide: g, steps: steps, done: make([]bool, len(steps))}
}

func (m guideModel) Init() tea.Cmd { return nil }

func (m guideModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.steps)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case " ", "x":
			m.done[m.cursor] = !m.done[m.cursor]
		}
	}
	return m, nil
}

var (
	stTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	stHint  = lipgloss.NewStyle().Faint(true)
	stDone  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func (m guideModel) View() string {
	var b strings.Builder
	b.WriteString(stTitle.Render(m.guide.Title) + "\n\n")
	for i, s := range m.steps {
		mark := "[ ]"
		if m.done[i] {
			mark = stDone.Render("[x]")
		}
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		b.WriteString(cursor + mark + " " + s + "\n")
	}
	b.WriteString("\n" + stHint.Render("j/k: move · space: toggle · enter: done · q: quit") + "\n")
	if m.guide.Verify != nil {
		b.WriteString(stHint.Render(fmt.Sprintf("verify: one %s %s", m.guide.Service, m.guide.Verify.Action)) + "\n")
	}
	return b.String()
}
