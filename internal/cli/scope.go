package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newScopeCommand(show *app.ShowScope, add *app.AddScope, remove *app.RemoveScope, check *app.CheckScope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scope",
		Short: "Manage .onerc.yaml permissions",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "show [<service>]",
			Short: "Display current scope",
			RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
		},
		&cobra.Command{
			Use:   "add <service> <permission>",
			Short: "Add a permission to .onerc.yaml",
			Args:  cobra.ExactArgs(2),
			RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
		},
		&cobra.Command{
			Use:   "remove <service> <permission>",
			Short: "Remove a permission from .onerc.yaml",
			Args:  cobra.ExactArgs(2),
			RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
		},
		&cobra.Command{
			Use:   "check <service> <action>",
			Short: "Exit 0 if allowed, 3 if not in scope",
			Args:  cobra.ExactArgs(2),
			RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
		},
		&cobra.Command{
			Use:   "explain <service> <action>",
			Short: "Explain why a permission is allowed or denied",
			Args:  cobra.ExactArgs(2),
			RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
		},
	)

	return cmd
}
