package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newUpgradeCommand(uc *app.Upgrade) *cobra.Command {
	var (
		target string
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Self-update the binary to a target release version",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := uc.Run(cmd.Context(), app.UpgradeInput{
				TargetVersion: target,
				DryRun:        dryRun,
			})
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if dryRun {
				fmt.Fprintf(w, "would upgrade %s → %s\nurl: %s\nsha256: %s\n", out.From, out.To, out.URL, out.SHA256)
				return nil
			}
			fmt.Fprintf(w, "upgraded %s → %s\n", out.From, out.To)
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "to", "", "target version tag (e.g. v1.0.0) — required")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned download + sum without applying")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
