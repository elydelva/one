package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newSkillCommand(uc *app.Skill) *cobra.Command {
	var (
		install bool
		ide     string
	)
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Show or install the One skill in your IDE",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := uc.Run(cmd.Context(), app.SkillInput{
				IDE:     app.IDE(ide),
				Install: install,
			})
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if install {
				fmt.Fprintf(w, "installed skill for %s at %s\n", out.IDE, out.Destination)
				return nil
			}
			fmt.Fprintf(w, "ide: %s\ndestination: %s\n\n", out.IDE, out.Destination)
			fmt.Fprint(w, out.Content)
			return nil
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install the skill in the detected (or specified) IDE")
	cmd.Flags().StringVar(&ide, "ide", "", "target IDE: claude-code, cursor, or aider (default: auto-detect)")
	return cmd
}
