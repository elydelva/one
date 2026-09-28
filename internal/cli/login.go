package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newLoginCommand(uc *app.Login) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <service>",
		Short: "Authenticate with a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, _ := cmd.Flags().GetString("as")
			provider, _ := cmd.Flags().GetString("provider")
			if err := uc.Run(cmd.Context(), app.LoginInput{
				Service:  args[0],
				Account:  alias,
				Provider: parseProviderKind(provider),
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Signed in to %s as %s.\n", args[0], alias)
			return nil
		},
	}
	cmd.Flags().StringP("as", "a", "default", "account alias")
	cmd.Flags().String("provider", "pat", "auth provider (pat, oauth2_device, api_key)")
	return cmd
}

func newLogoutCommand(uc *app.Logout) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout <service>",
		Short: "Remove stored credentials for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, _ := cmd.Flags().GetString("as")
			return uc.Run(cmd.Context(), app.LogoutInput{
				Service: args[0],
				Account: alias,
			})
		},
	}
	cmd.Flags().StringP("as", "a", "default", "account alias")
	return cmd
}
