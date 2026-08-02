package cli

import (
	"fmt"
	"sort"

	"github.com/irangarcia/agentwho/internal/shell"
	"github.com/irangarcia/agentwho/internal/termstyle"
	"github.com/spf13/cobra"
)

func (a *app) useCmd() *cobra.Command {
	var automatic bool
	cmd := &cobra.Command{
		Use:   "use <profile>",
		Short: "Use a profile in the current shell",
		Long:  "Select an AgentWho profile for the current shell. Safety checks still apply. Use --auto to return to repository-based selection.",
		Example: "  agentwho use personal\n" +
			"  agentwho use work\n" +
			"  agentwho use --auto",
		Args: func(cmd *cobra.Command, args []string) error {
			if automatic {
				if len(args) != 0 {
					return fmt.Errorf("--auto does not accept a profile")
				}
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shellName := detectedShell()
			return fmt.Errorf("`agentwho use` changes the current shell and needs AgentWho shell integration\n\nRun:\n  %s\n\nThen retry the command", shell.EvalLine(shellName))
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			c, _, err := load()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names := make([]string, 0, len(c.Profiles))
			for name := range c.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			return names, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().BoolVar(&automatic, "auto", false, "return to automatic repository-based selection")
	return cmd
}

func (a *app) shellUseCmd() *cobra.Command {
	var automatic bool
	cmd := &cobra.Command{
		Use:    "shell-use <shell> [profile]",
		Hidden: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("internal shell selection requires a shell")
			}
			if automatic {
				if len(args) != 1 {
					return fmt.Errorf("--auto does not accept a profile")
				}
				return nil
			}
			if len(args) != 2 {
				return fmt.Errorf("choose a profile\n\nUsage:\n  agentwho use <profile>\n  agentwho use --auto")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			status, err := currentStatus(c, p.BinDir)
			if err != nil {
				return err
			}

			if automatic {
				code, err := shell.UseAutomatic(args[0])
				if err != nil {
					return err
				}
				fmt.Fprintln(a.out, code)
				fmt.Fprintln(a.errout, termstyle.Paint(a.errout, termstyle.Success, "✓ Automatic profile selection restored for this shell."))
				fmt.Fprintf(a.errout, "  Current profile here: %s\n", termstyle.Paint(a.errout, termstyle.Success, status.ExpectedProfile))
				return nil
			}

			profile := args[1]
			if _, ok := c.Profiles[profile]; !ok {
				return fmt.Errorf("profile %q does not exist\n\nRun `agentwho profile list` to see available profiles", profile)
			}
			code, err := shell.UseProfile(args[0], profile)
			if err != nil {
				return err
			}
			fmt.Fprintln(a.out, code)
			fmt.Fprintln(a.errout, termstyle.Paint(a.errout, termstyle.Success, fmt.Sprintf("✓ Using profile %q in this shell.", profile)))
			if profile != status.ExpectedProfile {
				fmt.Fprintln(a.errout, termstyle.Paint(a.errout, termstyle.Warning,
					fmt.Sprintf("⚠ This directory expects profile %q. Safety mode %q will apply.", status.ExpectedProfile, status.SafetyMode)))
			}
			fmt.Fprintln(a.errout, termstyle.Paint(a.errout, termstyle.Muted, "  Return to automatic selection with: agentwho use --auto"))
			return nil
		},
	}
	cmd.Flags().BoolVar(&automatic, "auto", false, "return to automatic repository-based selection")
	return cmd
}
