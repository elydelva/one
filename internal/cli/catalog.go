package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

func newCatalogCommand(uc *app.CatalogOps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Inspect and manage the service catalog",
	}
	cmd.AddCommand(
		catalogSearchCmd(uc),
		catalogUpdateCmd(uc),
		catalogLintCmd(uc),
		catalogScaffoldCmd(uc),
		catalogTestCmd(uc),
	)
	return cmd
}

func catalogSearchCmd(uc *app.CatalogOps) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search services in the catalog",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := ""
			if len(args) == 1 {
				q = args[0]
			}
			res, err := uc.Search(cmd.Context(), q)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if asJSON {
				return json.NewEncoder(w).Encode(res)
			}
			if len(res) == 0 {
				fmt.Fprintln(w, "no services match")
				return nil
			}
			for _, s := range res {
				fmt.Fprintf(w, "%-20s %3d actions  %s\n", s.ID, s.Actions, s.Description)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func catalogUpdateCmd(uc *app.CatalogOps) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Invalidate the local catalog cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uc.Update(cmd.Context()); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "catalog cache invalidated")
			return nil
		},
	}
}

func catalogLintCmd(uc *app.CatalogOps) *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "Lint every service in the local catalog root",
		RunE: func(cmd *cobra.Command, args []string) error {
			reps, err := uc.Lint(cmd.Context())
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fail := false
			for _, r := range reps {
				if len(r.Errors) == 0 {
					fmt.Fprintf(w, "OK   %s\n", r.Service)
					continue
				}
				fail = true
				for _, e := range r.Errors {
					fmt.Fprintf(w, "FAIL %s: %s\n", r.Service, e)
				}
			}
			if fail {
				return fmt.Errorf("lint failed")
			}
			return nil
		},
	}
}

func catalogScaffoldCmd(uc *app.CatalogOps) *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold <id>",
		Short: "Generate a service skeleton at <catalog-root>/services/<id>/",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := uc.Scaffold(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "scaffolded %s\n", dir)
			return nil
		},
	}
}

func catalogTestCmd(uc *app.CatalogOps) *cobra.Command {
	return &cobra.Command{
		Use:   "test <service>",
		Short: "Smoke-test a service definition (parse input schemas, etc.)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := uc.Test(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fail := false
			for _, r := range results {
				status := "OK"
				if !r.Pass {
					status = "FAIL"
					fail = true
				}
				line := fmt.Sprintf("%-5s %s/%s", status, r.Service, r.Action)
				if r.Err != "" {
					line += " — " + r.Err
				}
				fmt.Fprintln(w, line)
			}
			if fail {
				return fmt.Errorf("test failed")
			}
			return nil
		},
	}
}
