package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newInstallCommand(uc *app.ShowGuide) *cobra.Command {
	var list bool
	cmd := &cobra.Command{
		Use:   "install <service> [guide]",
		Short: "Show a setup guide for a service",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := app.InstallInput{Service: args[0], List: list}
			if len(args) == 2 {
				in.Guide = args[1]
			}
			out, err := uc.Run(cmd.Context(), in)
			if err != nil {
				return err
			}
			if out.Guide != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "# "+out.Guide.Title)
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprint(cmd.OutOrStdout(), out.Guide.Content)
				if out.Guide.Verify != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "\nVerify: one %s %s\n", out.Service, out.Guide.Verify.Action)
				}
				return nil
			}
			for _, g := range out.All {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", g.ID, g.Title)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "list all guides available for the service")
	return cmd
}
