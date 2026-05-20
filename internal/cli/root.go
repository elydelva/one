package cli

import (
	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
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
}

// BuildRoot assembles the root cobra command with all sub-commands.
func BuildRoot(deps Deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "one",
		Short:         "Unified API access for AI agents",
		Long:          "One CLI provides governed, auditable access to third-party APIs via a local vault and scope file.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().Bool("json", false, "force JSON output (default: auto-detect TTY)")
	root.PersistentFlags().String("account", "", "account alias to use (default: from .onerc.yaml)")
	root.PersistentFlags().Bool("dry-run", false, "print what would happen without executing")

	root.AddCommand(
		newExecCommand(deps.Execute),
		newLoginCommand(deps.Login),
		newLogoutCommand(deps.Logout),
		newCapabilitiesCommand(deps.Capabilities),
		newInfoCommand(deps.Info),
		newScopeCommand(deps.ShowScope, deps.AddScope, deps.RemoveScope, deps.CheckScope),
		newInstallCommand(deps.ShowGuide),
		newLockCommand(deps.LockScope),
		newTraceCommand(deps.ShowTrace),
		newDoctorCommand(deps.RunDoctor),
		newSkillCommand(),
	)

	return root
}
