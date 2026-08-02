package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/irangarcia/agentwho/internal/shell"
	"github.com/irangarcia/agentwho/internal/shim"
	"github.com/spf13/cobra"
)

func (a *app) installCmd() *cobra.Command {
	var modify bool
	var shellName, shellConfig string
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Enable automatic profile selection",
		Long:    "Enable AgentWho for the normal `claude` and `codex` commands without replacing the official CLIs.",
		Example: "  agentwho install\n  agentwho install --shell zsh\n  agentwho install --modify-shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			if shellName == "" {
				shellName = shell.Detect(os.Getenv("SHELL"))
				if shellName != "zsh" && shellName != "bash" && shellName != "fish" {
					shellName = "zsh"
				}
			} else if shellName != "zsh" && shellName != "bash" && shellName != "fish" {
				return fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", shellName)
			}
			home, _ := os.UserHomeDir()
			configFile := shell.DefaultConfig(shellName, home)
			if shellConfig == "" {
				shellConfig = configFile
			}
			if modify && !interactive(a.stdinFile) {
				return fmt.Errorf("--modify-shell needs an interactive terminal so AgentWho can ask before changing %s", shellConfig)
			}
			if err := p.EnsureBase(); err != nil {
				return err
			}
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate agentwho executable: %w", err)
			}
			exe, err = filepath.Abs(exe)
			if err != nil {
				return err
			}
			if err := shim.Install(p.BinDir, exe, []string{"claude", "codex"}); err != nil {
				return err
			}
			fmt.Fprintln(a.out, a.success("✓ Automatic profile selection enabled."))
			fmt.Fprintln(a.out)
			fmt.Fprintln(a.out, "The `claude` and `codex` commands will now be routed through AgentWho.")
			fmt.Fprintln(a.out, "The official agent executables were not changed.")
			integrationFile := configFile
			if modify {
				integrationFile = shellConfig
			}
			configured, err := shell.IsConfigured(integrationFile, shellName)
			if err != nil {
				return fmt.Errorf("check shell integration in %s: %w", integrationFile, err)
			}
			if configured {
				fmt.Fprintf(a.out, "\n%s Shell integration is already configured in %s.\n", a.success("✓"), integrationFile)
				return nil
			}
			if modify {
				reader := bufio.NewReader(a.in)
				fmt.Fprintf(a.out, "\nAgentWho can update %s automatically.\nA backup will be created first.\n", shellConfig)
				if !askYes(reader, a.out, "Update "+shellConfig+"?") {
					fmt.Fprintln(a.out, "No shell files were changed.")
					printShellInstructions(a.out, shellConfig, shellName)
					return nil
				}
				backup, changed, err := shell.AddBlock(shellConfig, shellName)
				if err != nil {
					return err
				}
				if !changed {
					fmt.Fprintf(a.out, "AgentWho is already configured in %s.\nNo changes were needed.\n", shellConfig)
				} else if backup != "" {
					fmt.Fprintf(a.out, "%s Updated %s.\n  Backup: %s\n", a.success("✓"), shellConfig, backup)
				} else {
					fmt.Fprintf(a.out, "%s Updated %s.\n", a.success("✓"), shellConfig)
				}
			} else {
				printShellInstructions(a.out, configFile, shellName)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&modify, "modify-shell", false, "offer to update the shell configuration")
	cmd.Flags().StringVar(&shellName, "shell", "", "shell to configure: zsh, bash, or fish")
	cmd.Flags().StringVar(&shellConfig, "shell-config", "", "shell configuration path")
	return cmd
}

func printShellInstructions(w io.Writer, configFile, shellName string) {
	fmt.Fprintf(w, "\nAdd this line to %s:\n\n  %s\n", configFile, shell.EvalLine(shellName))
	fmt.Fprintf(w, "\nThen restart your terminal or run:\n\n  %s\n", shell.EvalLine(shellName))
}

func (a *app) uninstallCmd() *cobra.Command {
	var purge, removeShell bool
	var shellName, shellConfig string
	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Disable AgentWho integration",
		Long:    "Disable automatic profile selection without removing the official CLIs or profile sign-ins. Use --purge only for permanent deletion.",
		Example: "  agentwho uninstall\n  agentwho uninstall --remove-shell\n  agentwho uninstall --purge",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			if removeShell {
				if shellName == "" {
					shellName = shell.Detect(os.Getenv("SHELL"))
					if shellName != "zsh" && shellName != "bash" && shellName != "fish" {
						shellName = "zsh"
					}
				} else if shellName != "zsh" && shellName != "bash" && shellName != "fish" {
					return fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", shellName)
				}
				home, _ := os.UserHomeDir()
				if shellConfig == "" {
					shellConfig = shell.DefaultConfig(shellName, home)
				}
				if !interactive(a.stdinFile) {
					return fmt.Errorf("--remove-shell needs an interactive terminal so AgentWho can ask before changing %s", shellConfig)
				}
			}
			if err := shim.Remove(p.BinDir, []string{"claude", "codex"}); err != nil {
				return err
			}
			fmt.Fprintln(a.out, a.success("✓ Automatic profile selection disabled."))
			fmt.Fprintln(a.out)
			fmt.Fprintln(a.out, "The official Claude and Codex executables were not changed.")
			if removeShell {
				reader := bufio.NewReader(a.in)
				fmt.Fprintf(a.out, "\nA backup will be created first.\n")
				if askYes(reader, a.out, "Remove AgentWho setup from "+shellConfig+"?") {
					backup, changed, err := shell.RemoveBlocks(shellConfig)
					if err != nil {
						return err
					}
					if changed {
						fmt.Fprintf(a.out, "%s Removed AgentWho setup from %s.\n  Backup: %s\n", a.success("✓"), shellConfig, backup)
					} else {
						fmt.Fprintf(a.out, "No AgentWho setup was found in %s.\n", shellConfig)
					}
				}
			}
			if purge {
				fmt.Fprint(a.out, "\nPermanent deletion requested\n\n")
				fmt.Fprintln(a.out, "This will delete:")
				fmt.Fprintln(a.out, "  • AgentWho configuration")
				fmt.Fprintln(a.out, "  • All profiles")
				fmt.Fprintln(a.out, "  • Claude sign-ins stored in those profiles")
				fmt.Fprintln(a.out, "  • Codex sign-ins stored in those profiles")
				fmt.Fprintln(a.out, "\nThe official Claude and Codex applications will not be removed.")
				fmt.Fprint(a.out, "\nType \"purge\" to permanently delete this data: ")
				reader := bufio.NewReader(a.in)
				answer, _ := reader.ReadString('\n')
				if strings.TrimSpace(answer) != "purge" {
					fmt.Fprintln(a.out, "Purge cancelled. Your profiles and sign-ins were kept.")
					return nil
				}
				if err := os.RemoveAll(p.DataDir); err != nil {
					return fmt.Errorf("remove data directory: %w", err)
				}
				if err := os.RemoveAll(p.ConfigDir); err != nil {
					return fmt.Errorf("remove configuration directory: %w", err)
				}
				fmt.Fprintln(a.out, a.success("✓ AgentWho configuration, profiles, and profile sign-ins were deleted."))
			} else {
				fmt.Fprintln(a.out, "Your profiles and sign-ins were kept.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "permanently delete configuration, profiles, and profile sign-ins")
	cmd.Flags().BoolVar(&removeShell, "remove-shell", false, "offer to remove AgentWho from the shell configuration")
	cmd.Flags().StringVar(&shellName, "shell", "", "shell: zsh, bash, or fish")
	cmd.Flags().StringVar(&shellConfig, "shell-config", "", "shell configuration path")
	return cmd
}

func (a *app) shellCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "shell", Short: "Generate shell setup code"}
	cmd.AddCommand(&cobra.Command{
		Use:     "init <zsh|bash|fish>",
		Short:   "Print shell setup code",
		Long:    "Print code that enables automatic profile selection and defines an optional profile indicator for the chosen shell.",
		Example: "  eval \"$(agentwho shell init zsh)\"\n  eval \"$(agentwho shell init bash)\"\n  agentwho shell init fish | source",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			output, err := shell.Init(args[0], p.BinDir)
			if err == nil {
				fmt.Fprint(a.out, output)
			}
			return err
		},
	})
	return cmd
}
