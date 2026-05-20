package cli

import (
	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
)

func newCanCommand(uc *app.CheckScope) *cobra.Command {
	return &cobra.Command{
		Use:   "can <service> <action>",
		Short: "Pre-check whether an action is allowed (exit 0 / 3)",
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
