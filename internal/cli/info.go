package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/ports"
)

func newInfoCommand(uc *app.ShowInfo, rnd ports.Renderer) *cobra.Command {
	return &cobra.Command{
		Use:   "info [<service> [<action>]]",
		Short: "Show documentation for a service or action",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := app.InfoInput{}
			if len(args) > 0 {
				in.Service = args[0]
			}
			if len(args) > 1 {
				in.Action = args[1]
			}
			out, err := uc.Run(cmd.Context(), in)
			if err != nil {
				return err
			}
			// info uses the markdown payload directly; render as a JSON envelope.
			data, _ := json.Marshal(map[string]string{"markdown": out.Markdown})
			rnd.RenderResult(data, "")
			return nil
		},
	}
}
