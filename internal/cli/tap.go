package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"elydelva/one/internal/app"
)

// errConsentDeclined is returned by the interactive consent prompt when the
// user does not type "y" / "yes". TapOps wraps this and bubbles it up so the
// CLI can show a clean message.
var errConsentDeclined = errors.New("declined")

func newTapCommand(uc *app.TapOps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tap",
		Short: "Manage third-party catalog repositories (taps)",
		Long: "A tap is a GitHub repository whose root is a service catalog. " +
			"Taps extend the built-in official catalog. The official catalog " +
			"wins on service-name conflicts.",
	}
	cmd.AddCommand(tapAddCmd(uc), tapRemoveCmd(uc), tapListCmd(uc), tapUpdateCmd(uc))
	return cmd
}

func tapAddCmd(uc *app.TapOps) *cobra.Command {
	var (
		yes        bool
		verifyKey  string
		verifyFile string
	)
	cmd := &cobra.Command{
		Use:   "add <user>/<repo>|<https-url>",
		Short: "Add a tap, optionally verified by a minisign public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			consent := consentPrompt(cmd, yes)
			key, err := resolveVerifyKey(verifyKey, verifyFile)
			if err != nil {
				return err
			}
			entry, err := uc.AddWith(cmd.Context(), args[0], app.AddOptions{VerifyKey: key}, consent)
			if errors.Is(err, errConsentDeclined) {
				return fmt.Errorf("tap add: declined")
			}
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if entry.PublicKey != "" {
				fmt.Fprintf(w, "tap %s pinned at %s (signed)\n", entry.Name, shortSHA(entry.SHA))
			} else {
				fmt.Fprintf(w, "tap %s pinned at %s\n", entry.Name, shortSHA(entry.SHA))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "accept TOFU consent without prompting")
	cmd.Flags().StringVar(&verifyKey, "verify-key", "", "minisign public key (literal, one line)")
	cmd.Flags().StringVar(&verifyFile, "verify-key-file", "", "path to minisign public key file")
	cmd.MarkFlagsMutuallyExclusive("verify-key", "verify-key-file")
	return cmd
}

// resolveVerifyKey returns the public key text (or empty when no flag set).
func resolveVerifyKey(literal, file string) (string, error) {
	if literal != "" {
		return strings.TrimSpace(literal), nil
	}
	if file == "" {
		return "", nil
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read verify-key-file: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func tapRemoveCmd(uc *app.TapOps) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <user>/<repo>",
		Short: "Remove a tap and delete its local clone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uc.Remove(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "tap %s removed\n", args[0])
			return nil
		},
	}
}

func tapListCmd(uc *app.TapOps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed taps",
		RunE: func(cmd *cobra.Command, _ []string) error {
			taps, err := uc.List(cmd.Context())
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if len(taps) == 0 {
				fmt.Fprintln(w, "no taps installed")
				return nil
			}
			for _, t := range taps {
				fmt.Fprintf(w, "%-40s %s\n", t.Name, shortSHA(t.SHA))
			}
			return nil
		},
	}
}

func tapUpdateCmd(uc *app.TapOps) *cobra.Command {
	var (
		yes bool
		all bool
	)
	cmd := &cobra.Command{
		Use:   "update [<user>/<repo>]",
		Short: "Fetch upstream and re-pin SHA (re-consent required on change)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			consent := consentPrompt(cmd, yes)
			w := cmd.OutOrStdout()

			targets, err := updateTargets(cmd.Context(), uc, args, all)
			if err != nil {
				return err
			}
			var lastErr error
			for _, name := range targets {
				res, err := uc.Update(cmd.Context(), name, consent)
				if errors.Is(err, errConsentDeclined) {
					fmt.Fprintf(w, "tap %s: declined (pinned SHA unchanged)\n", name)
					continue
				}
				if err != nil {
					fmt.Fprintf(w, "tap %s: error: %v\n", name, err)
					lastErr = err
					continue
				}
				if !res.Changed {
					fmt.Fprintf(w, "tap %s: up to date (%s)\n", res.Name, shortSHA(res.NewSHA))
					continue
				}
				fmt.Fprintf(w, "tap %s: %s → %s\n", res.Name, shortSHA(res.OldSHA), shortSHA(res.NewSHA))
			}
			return lastErr
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "accept TOFU consent without prompting")
	cmd.Flags().BoolVar(&all, "all", false, "update every installed tap")
	return cmd
}

// updateTargets resolves which taps to update based on args + --all.
func updateTargets(ctx context.Context, uc *app.TapOps, args []string, all bool) ([]string, error) {
	if all {
		if len(args) != 0 {
			return nil, fmt.Errorf("tap update: --all and a tap name are mutually exclusive")
		}
		list, err := uc.List(ctx)
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(list))
		for _, t := range list {
			names = append(names, t.Name)
		}
		return names, nil
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("tap update: name required (or pass --all)")
	}
	return args, nil
}

// consentPrompt returns a ConsentFunc that auto-accepts when yes is set or
// when stdin is not a TTY (scripted use). Otherwise reads "y"/"yes" from stdin.
func consentPrompt(cmd *cobra.Command, yes bool) app.ConsentFunc {
	return func(prompt string) error {
		if yes {
			return nil
		}
		if !stdinIsTTY() {
			return fmt.Errorf("%w: non-interactive shell, pass --yes to accept", errConsentDeclined)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", prompt)
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil {
			return fmt.Errorf("%w: %v", errConsentDeclined, err)
		}
		s := strings.ToLower(strings.TrimSpace(line))
		if s == "y" || s == "yes" {
			return nil
		}
		return errConsentDeclined
	}
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func shortSHA(sha string) string {
	if len(sha) < 12 {
		return sha
	}
	return sha[:12]
}
