package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newTraceCommand(uc *app.ShowTrace) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "View the audit log of past invocations",
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
	cmd.Flags().BoolP("follow", "f", false, "follow the log in real time")
	cmd.Flags().IntP("limit", "n", 20, "number of entries to show")
	return cmd
}
