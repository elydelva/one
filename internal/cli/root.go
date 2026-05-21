package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// Deps groups all use cases injected into the CLI layer.
type Deps struct {
	Execute      *app.ExecuteAction
	Login        *app.Login
	Logout       *app.Logout
	Capabilities *app.ListCapabilities
	Info         *app.ShowInfo
	ShowScope    *app.ShowScope
	AddScope     *app.AddScope
	RemoveScope  *app.RemoveScope
	CheckScope   *app.CheckScope
	ShowGuide    *app.ShowGuide
	LockScope    *app.LockScope
	ShowTrace    *app.ShowTrace
	RunDoctor    *app.RunDoctor
	Init         *app.Init
	Accounts     *app.ListAccounts
	Rotate       *app.Rotate
	Refresh      *app.ForceRefresh
	VaultStatus  *app.VaultStatus
	VaultExport  *app.VaultExport
	VaultImport  *app.VaultImport
	VaultRotate  *app.VaultRotate
	Skill        *app.Skill
	CatalogOps   *app.CatalogOps
	Upgrade      *app.Upgrade
	Renderer     ports.Renderer
	Catalog      ports.Catalog
}

// BuildRoot assembles the root cobra command with all sub-commands plus a
// dispatcher: when the first positional arg matches a catalog service, the
// root runs ExecuteAction. Otherwise cobra routes to a regular sub-command.
func BuildRoot(deps Deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "one",
		Short: "Unified API access for AI agents",
		Long:  "One CLI provides governed, auditable access to third-party APIs via a local vault and scope file.",
		// Disable flag parsing so that action-specific flags like --owner / --repo
		// reach our dispatcher untouched. Sub-commands re-enable parsing.
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			// Root flags handled manually since DisableFlagParsing strips parsing.
			switch args[0] {
			case "--version", "-v":
				cmd.Printf("one version %s\n", cmd.Root().Version)
				return nil
			case "--help", "-h":
				return cmd.Help()
			}
			if deps.Catalog == nil {
				return cmd.Help()
			}
			if _, err := deps.Catalog.GetService(context.Background(), core.ServiceID(args[0])); err != nil {
				return cmd.Help()
			}
			if len(args) < 2 {
				return cmd.Help()
			}
			return runExecuteFromArgs(cmd.Context(), deps, args)
		},
	}

	root.PersistentFlags().Bool("json", false, "force JSON output (default: auto-detect TTY)")
	root.PersistentFlags().String("account", "", "account alias to use (default: from .onerc.yaml)")
	root.PersistentFlags().Bool("dry-run", false, "print what would happen without executing")
	root.PersistentFlags().String("project", "", "project directory (default: cwd)")

	root.AddCommand(
		newExecCommand(deps),
		newLoginCommand(deps.Login),
		newLogoutCommand(deps.Logout),
		newCapabilitiesCommand(deps.Capabilities, deps.Renderer),
		newInfoCommand(deps.Info, deps.Renderer),
		newScopeCommand(deps.ShowScope, deps.AddScope, deps.RemoveScope, deps.CheckScope, deps.Renderer),
		newInstallCommand(deps.ShowGuide),
		newLockCommand(deps.LockScope),
		newTraceCommand(deps.ShowTrace),
		newDoctorCommand(deps.RunDoctor),
		newSkillCommand(deps.Skill),
		newInitCommand(deps.Init),
		newCanCommand(deps.CheckScope),
		newAccountsCommand(deps.Accounts),
		newRotateCommand(deps.Rotate),
		newRefreshCommand(deps.Refresh),
		newVaultCommand(deps.VaultStatus, deps.VaultExport, deps.VaultImport, deps.VaultRotate),
		newCatalogCommand(deps.CatalogOps),
		newUpgradeCommand(deps.Upgrade),
	)

	return root
}

// projectDir returns the --project flag value, falling back to cwd.
func projectDir(cmd *cobra.Command) string {
	if p, _ := cmd.Flags().GetString("project"); p != "" {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
