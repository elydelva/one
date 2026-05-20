package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newCapabilitiesCommand(uc *app.ListCapabilities) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capabilities [<service>]",
		Short: "List available actions (JSON output for agents)",
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
	cmd.Flags().Bool("scope-only", false, "only show actions permitted by .onerc.yaml")
	return cmd
}
