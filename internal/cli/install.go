package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newInstallCommand(uc *app.ShowGuide) *cobra.Command {
	return &cobra.Command{
		Use:   "install <service> <guide>",
		Short: "Show a setup guide for a service",
		Args:  cobra.ExactArgs(2),
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
}
