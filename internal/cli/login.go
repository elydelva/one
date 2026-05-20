package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newLoginCommand(uc *app.Login) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <service>",
		Short: "Authenticate with a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	cmd.Flags().StringP("as", "a", "default", "account alias")
	return cmd
}

func newLogoutCommand(uc *app.Logout) *cobra.Command {
	return &cobra.Command{
		Use:   "logout <service>",
		Short: "Remove stored credentials for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
}
