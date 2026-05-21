package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
)

func newAccountsCommand(uc *app.ListAccounts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts <service>",
		Short: "List accounts registered for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			refs, err := uc.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if len(refs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no accounts")
				return nil
			}
			for _, r := range refs {
				fmt.Fprintln(cmd.OutOrStdout(), r.String())
			}
			return nil
		},
	}
	return cmd
}

func newRotateCommand(uc *app.Rotate) *cobra.Command {
	var provider string
	cmd := &cobra.Command{
		Use:   "rotate <service> <account>",
		Short: "Re-run login flow for an account (rotate credential)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uc.Run(cmd.Context(), app.LoginInput{
				Service:  args[0],
				Account:  args[1],
				Provider: core.ProviderKind(provider),
			})
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider kind (defaults to PAT)")
	return cmd
}

func newRefreshCommand(uc *app.ForceRefresh) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh <service> <account>",
		Short: "Force a credential refresh ignoring the expiry margin",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uc.Run(cmd.Context(), args[0], args[1])
		},
	}
}
