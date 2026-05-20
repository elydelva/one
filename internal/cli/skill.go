package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSkillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Show or install the One skill in your IDE",
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
	cmd.Flags().Bool("install", false, "install the skill in the detected IDE")
	return cmd
}
