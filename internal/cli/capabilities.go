package cli

import (
	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/ports"
)

func newCapabilitiesCommand(uc *app.ListCapabilities, rnd ports.Renderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capabilities [<service>]",
		Short: "List available actions (JSON output for agents)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scopeOnly, _ := cmd.Flags().GetBool("scope-only")
			in := app.CapabilitiesInput{
				ScopeOnly:  scopeOnly,
				ProjectDir: projectDir(cmd),
			}
			if len(args) == 1 {
				in.Service = args[0]
			}
			out, err := uc.Run(cmd.Context(), in)
			if err != nil {
				return err
			}
			rnd.RenderCapabilities(ports.CapabilitiesOutput{Services: out.Services})
			return nil
		},
	}
	cmd.Flags().Bool("scope-only", false, "only show actions permitted by .onerc.yaml")
	return cmd
}
