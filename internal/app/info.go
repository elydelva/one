package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// InfoInput holds parameters for the ShowInfo use case.
type InfoInput struct {
	Service string
	Action  string
}

// InfoOutput holds the markdown documentation.
type InfoOutput struct {
	Markdown string
}

// ShowInfo returns documentation for a service or action.
type ShowInfo struct {
	catalog ports.Catalog
}

// NewShowInfo creates a ShowInfo use case.
func NewShowInfo(catalog ports.Catalog) *ShowInfo {
	return &ShowInfo{catalog: catalog}
}

// Run returns the info. Behavior:
//   - both empty: list every service.
//   - service only: service header + SKILL.md (if any) + actions list.
//   - both: action details (description, permission, inputs).
func (uc *ShowInfo) Run(ctx context.Context, in InfoInput) (InfoOutput, error) {
	switch {
	case in.Service == "" && in.Action == "":
		return uc.listServices(ctx)
	case in.Action == "":
		return uc.serviceInfo(ctx, core.ServiceID(in.Service))
	default:
		return uc.actionInfo(ctx, core.ServiceID(in.Service), core.ActionID(in.Action))
	}
}

func (uc *ShowInfo) listServices(ctx context.Context) (InfoOutput, error) {
	svcs, err := uc.catalog.ListServices(ctx)
	if err != nil {
		return InfoOutput{}, err
	}
	sort.Slice(svcs, func(i, j int) bool { return svcs[i].ID < svcs[j].ID })
	var b strings.Builder
	b.WriteString("# Available services\n\n")
	for _, s := range svcs {
		fmt.Fprintf(&b, "- **%s** — %s\n", s.ID, s.Name)
	}
	return InfoOutput{Markdown: b.String()}, nil
}

func (uc *ShowInfo) serviceInfo(ctx context.Context, id core.ServiceID) (InfoOutput, error) {
	svc, err := uc.catalog.GetService(ctx, id)
	if err != nil {
		return InfoOutput{}, err
	}
	var b strings.Builder
	// When a per-service SKILL.md exists it owns the H1 and intro, so we don't
	// emit our own `# <Name>` header (it duplicated the skill's heading).
	// Without a skill, generate a minimal header from the catalog metadata.
	if skill, err := uc.catalog.GetSkill(ctx, id); err == nil && skill != "" {
		b.WriteString(strings.TrimSpace(skill))
		b.WriteString("\n\n")
	} else {
		fmt.Fprintf(&b, "# %s\n\n%s\n\n", svc.Name, svc.Description)
	}
	b.WriteString("## Actions\n\n")
	actions := append([]core.Action(nil), svc.Actions...)
	sort.Slice(actions, func(i, j int) bool { return actions[i].ID < actions[j].ID })
	for _, a := range actions {
		fmt.Fprintf(&b, "- `%s` — %s (permission: `%s`)\n", a.ID, a.Description, a.Permission)
	}
	return InfoOutput{Markdown: b.String()}, nil
}

func (uc *ShowInfo) actionInfo(ctx context.Context, svc core.ServiceID, action core.ActionID) (InfoOutput, error) {
	act, err := uc.catalog.GetAction(ctx, svc, action)
	if err != nil {
		return InfoOutput{}, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n%s\n\n", svc, action, act.Description)
	fmt.Fprintf(&b, "**Permission:** `%s`\n\n", act.Permission)
	if act.Request != nil {
		fmt.Fprintf(&b, "**Request:** `%s %s`\n\n", act.Request.Method, act.Request.Path)
	}
	schema, err := core.ParseInputSchema(act.InputSchema)
	if err != nil {
		return InfoOutput{}, err
	}
	if len(schema.Defs) > 0 {
		b.WriteString("## Inputs\n\n")
		for _, d := range schema.Defs {
			req := ""
			if d.Required {
				req = " (required)"
			}
			loc := ""
			if d.Location != "" {
				loc = fmt.Sprintf(" [%s]", d.Location)
			}
			fmt.Fprintf(&b, "- `%s`: %s%s%s — %s\n", d.Name, d.Type, req, loc, d.Description)
		}
	}
	return InfoOutput{Markdown: b.String()}, nil
}

// ensure errors import not pruned by goimports if unused later.
var _ = errors.New
