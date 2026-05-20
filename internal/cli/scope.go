package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

func newScopeCommand(show *app.ShowScope, add *app.AddScope, remove *app.RemoveScope, check *app.CheckScope, rnd ports.Renderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scope",
		Short: "Manage .onerc.yaml permissions",
	}

	cmd.AddCommand(
		newScopeShowCmd(show, rnd),
		newScopeAddCmd(add),
		newScopeRemoveCmd(remove),
		newScopeCheckCmd(check),
		newScopeExplainCmd(check, rnd),
	)
	return cmd
}

func newScopeShowCmd(uc *app.ShowScope, rnd ports.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "show [<service>]",
		Short: "Display current scope",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := app.ShowScopeInput{ProjectDir: projectDir(cmd)}
			if len(args) == 1 {
				in.Service = args[0]
			}
			scope, err := uc.Run(cmd.Context(), in)
			if err != nil {
				return err
			}
			data, err := json.Marshal(scopeToJSON(scope))
			if err != nil {
				return err
			}
			rnd.RenderResult(data, "")
			return nil
		},
	}
}

func newScopeAddCmd(uc *app.AddScope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <service> <permission>",
		Short: "Add a permission to .onerc.yaml",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			deny, _ := cmd.Flags().GetBool("deny")
			return uc.Run(cmd.Context(), app.AddScopeInput{
				Service:    args[0],
				Permission: args[1],
				Deny:       deny,
				ProjectDir: projectDir(cmd),
			})
		},
	}
	cmd.Flags().Bool("deny", false, "add to deny list instead of allow")
	return cmd
}

func newScopeRemoveCmd(uc *app.RemoveScope) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <service> <permission>",
		Short: "Remove a permission from .onerc.yaml",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uc.Run(cmd.Context(), app.RemoveScopeInput{
				Service:    args[0],
				Permission: args[1],
				ProjectDir: projectDir(cmd),
			})
		},
	}
}

func newScopeCheckCmd(uc *app.CheckScope) *cobra.Command {
	return &cobra.Command{
		Use:   "check <service> <action>",
		Short: "Exit 0 if allowed, 3 if not in scope",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := uc.Run(cmd.Context(), app.CheckScopeInput{
				Service:    args[0],
				Action:     args[1],
				ProjectDir: projectDir(cmd),
			})
			if err != nil {
				return err
			}
			if !out.Allowed {
				return core.ErrNotInScope{Permission: core.Permission{Service: core.ServiceID(args[0]), Path: core.PermissionPath(args[1])}}
			}
			return nil
		},
	}
}

func newScopeExplainCmd(uc *app.CheckScope, rnd ports.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "explain <service> <action>",
		Short: "Explain why a permission is allowed or denied",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := uc.Run(cmd.Context(), app.CheckScopeInput{
				Service:    args[0],
				Action:     args[1],
				ProjectDir: projectDir(cmd),
			})
			if err != nil {
				return err
			}
			data, _ := json.Marshal(map[string]any{
				"allowed": out.Allowed,
				"reason":  out.Reason,
				"service": args[0],
				"action":  args[1],
			})
			rnd.RenderResult(data, "")
			if !out.Allowed {
				return fmt.Errorf("%s", out.Reason)
			}
			return nil
		},
	}
}

// scopeToJSON converts core.Scope to a stable JSON-friendly shape.
func scopeToJSON(s core.Scope) map[string]any {
	services := map[string]any{}
	for id, ss := range s.Services {
		allow := make([]string, 0, len(ss.Allow))
		for _, p := range ss.Allow {
			allow = append(allow, string(p))
		}
		deny := make([]string, 0, len(ss.Deny))
		for _, p := range ss.Deny {
			deny = append(deny, string(p))
		}
		services[string(id)] = map[string]any{
			"account": string(ss.Account),
			"allow":   allow,
			"deny":    deny,
		}
	}
	return map[string]any{
		"version":  s.Version,
		"services": services,
	}
}
