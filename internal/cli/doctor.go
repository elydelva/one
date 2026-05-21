package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newDoctorCommand(uc *app.RunDoctor) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostic checks on your One CLI setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := uc.RunForDir(cmd.Context(), projectDir(cmd))
			if err != nil {
				return err
			}
			for _, c := range out.Checks {
				prefix := map[string]string{"ok": "✓", "warn": "!", "fail": "✗"}[c.Status]
				if prefix == "" {
					prefix = "?"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\t%s\n", prefix, c.Name, c.Message)
			}
			if !out.OK() {
				return fmt.Errorf("doctor: one or more checks failed")
			}
			return nil
		},
	}
}
