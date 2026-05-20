package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newInfoCommand(uc *app.ShowInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "info [<service> [<action>]]",
		Short: "Show documentation for a service or action",
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
}
