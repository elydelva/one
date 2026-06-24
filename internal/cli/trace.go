package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
)

func newTraceCommand(uc *app.ShowTrace) *cobra.Command {
	var (
		since   string
		service string
		kind    string
		traceID string
		limit   int
		asJSON  bool
	)
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "View the audit log of past invocations",
		RunE: func(cmd *cobra.Command, args []string) error {
			in := app.TraceInput{
				Service: service,
				Kind:    core.AuditKind(kind),
				TraceID: traceID,
				Limit:   limit,
			}
			if since != "" {
				d, err := time.ParseDuration(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				in.Since = d
			}
			events, err := uc.Run(cmd.Context(), in)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				return json.NewEncoder(out).Encode(events)
			}
			if len(events) == 0 {
				fmt.Fprintln(out, "no trace entries")
				return nil
			}
			for _, ev := range events {
				ts := ev.At.Format(time.RFC3339)
				line := fmt.Sprintf("%s %-12s %-12s %-20s %s",
					ts, ev.Kind, ev.Service, ev.Action, ev.Outcome)
				if ev.TraceID != "" {
					line += " trace=" + ev.TraceID
				}
				if ev.Err != "" {
					line += " err=" + ev.Err
				}
				if n := len(ev.HTTPCalls); n > 0 {
					line += fmt.Sprintf(" calls=%d", n)
				}
				fmt.Fprintln(out, line)
				for _, c := range ev.HTTPCalls {
					fmt.Fprintf(out, "    %-6s %d %s (%dms)\n", c.Method, c.Status, c.URL, c.DurationMS)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "duration window (e.g. 1h, 24h)")
	cmd.Flags().StringVar(&service, "service", "", "filter by service ID")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by event kind (LOGIN/LOGOUT/REFRESH/EXEC/...)")
	cmd.Flags().StringVar(&traceID, "trace-id", "", "show a single trace by ID")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "max entries to return")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
