package cli

import (
	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newLockCommand(uc *app.LockScope) *cobra.Command {
	var (
		update    string
		updateAll bool
		check     bool
	)
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Freeze catalog service versions in .onerc.lock",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := app.LockGenerate
			svc := ""
			switch {
			case check:
				mode = app.LockCheck
			case updateAll:
				mode = app.LockUpdateAll
			case update != "":
				mode = app.LockUpdate
				svc = update
			}
			return uc.Run(cmd.Context(), app.LockInput{
				ProjectDir: projectDir(cmd),
				Mode:       mode,
				Service:    svc,
			})
		},
	}
	cmd.Flags().StringVar(&update, "update", "", "refresh a single service")
	cmd.Flags().BoolVar(&updateAll, "update-all", false, "refresh every service in scope")
	cmd.Flags().BoolVar(&check, "check", false, "verify the lock matches the catalog (exit 1 on drift)")
	return cmd
}
