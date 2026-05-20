package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newDoctorCommand(uc *app.RunDoctor) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostic checks on your One CLI setup",
		RunE:  func(cmd *cobra.Command, args []string) error { return errors.New("not implemented") },
	}
}
