package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newInitCommand(uc *app.Init) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Bootstrap .onerc.yaml and update .gitignore",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.Run(app.InitInput{ProjectDir: projectDir(cmd)})
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if out.ScopeCreated {
				fmt.Fprintf(w, "created %s\n", out.ScopePath)
			} else {
				fmt.Fprintf(w, "%s already exists\n", out.ScopePath)
			}
			if out.GitignoreAppended {
				fmt.Fprintf(w, "appended .onerc.local.yaml to %s\n", out.GitignorePath)
			}
			return nil
		},
	}
}
