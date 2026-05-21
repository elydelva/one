package cli

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newVaultCommand(status *app.VaultStatus, export *app.VaultExport, imp *app.VaultImport) *cobra.Command {
	root := &cobra.Command{
		Use:   "vault",
		Short: "Inspect or move credentials in the local vault",
	}
	root.AddCommand(
		&cobra.Command{
			Use:   "status",
			Short: "Show vault backend status",
			RunE: func(cmd *cobra.Command, args []string) error {
				out, err := status.Run(cmd.Context(), projectDir(cmd))
				if err != nil {
					return err
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			},
		},
		&cobra.Command{
			Use:   "export",
			Short: "Dump in-scope credentials as JSON to stdout (pipe to age -p)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return export.Run(cmd.Context(), projectDir(cmd), cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:   "import",
			Short: "Restore credentials from a JSON bundle read on stdin",
			RunE: func(cmd *cobra.Command, args []string) error {
				return imp.Run(cmd.Context(), os.Stdin)
			},
		},
	)
	return root
}
