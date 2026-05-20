package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newLockCommand(uc *app.LockScope) *cobra.Command {
	return &cobra.Command{
		Use:   "lock",
		Short: "Freeze catalog service versions in .onerc.lock",
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
}
