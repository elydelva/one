package cli

import (
	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newExecCommand(uc *app.ExecuteAction) *cobra.Command {
	return &cobra.Command{
		Use:                "exec",
		Hidden:             true,
		DisableFlagParsing: true,
		Short:              "Execute a service action (internal, use 'one <service> <action>' instead)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}
}
