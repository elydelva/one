package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
)

func newExecCommand(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:                "exec <service> <action> [--flag value ...]",
		Short:              "Execute a service action (alias of 'one <service> <action>')",
		DisableFlagParsing: true,
		Args:               cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExecuteFromArgs(cmd.Context(), deps, args)
		},
	}
}

// runExecuteFromArgs is the dispatch core for both `one exec ...` and `one <svc> <act> ...`.
// args = [service, action, --flag, value, --flag, @file, ...].
func runExecuteFromArgs(ctx context.Context, deps Deps, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: one <service> <action> [--flag value ...]")
	}
	service := args[0]
	action := args[1]
	rest := args[2:]

	svc, err := deps.Catalog.GetService(ctx, core.ServiceID(service))
	if err != nil {
		return err
	}
	var actDef *core.Action
	for i := range svc.Actions {
		if svc.Actions[i].ID == core.ActionID(action) {
			actDef = &svc.Actions[i]
			break
		}
	}
	if actDef == nil {
		return core.ErrUnknownAction{Service: svc.ID, Action: core.ActionID(action)}
	}
	schema, err := core.ParseInputSchema(actDef.InputSchema)
	if err != nil {
		return err
	}
	inputs, account, dryRun, projectDir, confirmed, err := parseActionFlags(rest, schema)
	if err != nil {
		return err
	}

	if actDef.IsDestructive() && !dryRun && !confirmed {
		if isStdinTTY() {
			ok, perr := promptConfirm(fmt.Sprintf("Action %s/%s is destructive. Proceed?", service, action))
			if perr != nil {
				return perr
			}
			if !ok {
				return fmt.Errorf("aborted by user")
			}
		}
		// Non-TTY without --confirm: refuse silently to avoid surprises.
		if !isStdinTTY() {
			return fmt.Errorf("destructive action requires --confirm when stdin is not a TTY")
		}
	}

	out, err := deps.Execute.Run(ctx, app.ExecuteInput{
		Service:    service,
		Action:     action,
		Inputs:     inputs,
		Account:    account,
		DryRun:     dryRun,
		ProjectDir: projectDir,
	})
	if err != nil {
		return err
	}
	deps.Renderer.RenderResult(out.Result, out.TraceID)
	return nil
}

// parseActionFlags walks args looking for `--name value` pairs. Special keys:
//   - --as <alias>
//   - --dry-run
//   - --project <dir>
//
// Other --name values are coerced to the type declared in schema.Defs.
// A value of `@path` reads the file content as a string.
func parseActionFlags(args []string, schema core.InputSchema) (inputs map[string]any, account string, dryRun bool, projectDir string, confirmed bool, err error) {
	inputs = map[string]any{}
	defByName := map[string]core.InputDef{}
	for _, d := range schema.Defs {
		defByName[d.Name] = d
	}

	wd, _ := os.Getwd()
	projectDir = wd

	i := 0
	for i < len(args) {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			return nil, "", false, "", false, fmt.Errorf("unexpected positional %q (expected --name value pairs)", a)
		}
		name := strings.TrimPrefix(a, "--")
		// Support --flag=value too.
		var raw string
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			raw = name[idx+1:]
			name = name[:idx]
			i++
		} else {
			// Boolean flags (no value): --dry-run, --confirm, --json.
			if name == "dry-run" {
				dryRun = true
				i++
				continue
			}
			if name == "confirm" {
				confirmed = true
				i++
				continue
			}
			// --json is a global output flag, already handled when the renderer
			// is selected; accept and ignore it here so it never gets treated
			// as an action input or a value-taking flag.
			if name == "json" {
				i++
				continue
			}
			if i+1 >= len(args) {
				return nil, "", false, "", false, fmt.Errorf("flag --%s requires a value", name)
			}
			raw = args[i+1]
			i += 2
		}

		switch name {
		case "json":
			// Global output flag; renderer already chosen. Ignore value.
			continue
		case "as", "account":
			account = raw
			continue
		case "project":
			projectDir = raw
			continue
		case "dry-run":
			dryRun = (raw == "true" || raw == "1")
			continue
		case "confirm":
			confirmed = (raw == "true" || raw == "1")
			continue
		}

		def, ok := defByName[name]
		if !ok {
			return nil, "", false, "", false, fmt.Errorf("unknown input --%s for this action", name)
		}
		val, err := coerceValue(raw, def)
		if err != nil {
			return nil, "", false, "", false, err
		}
		inputs[name] = val
	}
	return inputs, account, dryRun, projectDir, confirmed, nil
}

func isStdinTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func promptConfirm(msg string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", msg)
	buf := make([]byte, 4)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	c := buf[0]
	return c == 'y' || c == 'Y', nil
}

func coerceValue(raw string, def core.InputDef) (any, error) {
	// @file syntax: read file content as string (for body/file_ref inputs).
	if strings.HasPrefix(raw, "@") {
		path := raw[1:]
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("--%s: read %s: %w", def.Name, path, err)
		}
		raw = string(b)
	}
	switch def.Type {
	case core.TypeInteger:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s: expected integer, got %q", def.Name, raw)
		}
		return int(n), nil
	case core.TypeNumber:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s: expected number, got %q", def.Name, raw)
		}
		return f, nil
	case core.TypeBoolean:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: expected bool, got %q", def.Name, raw)
		}
		return b, nil
	case core.TypeArray, core.TypeObject:
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, fmt.Errorf("--%s: expected JSON %s, got %q", def.Name, def.Type, raw)
		}
		return v, nil
	default:
		return raw, nil
	}
}
