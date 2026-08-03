package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/shell"
	"github.com/irangarcia/agentwho/internal/shim"
	"github.com/spf13/cobra"
)

func (a *app) initCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Short:   "Set up AgentWho",
		Long:    "Set up personal and work profiles, mismatch safety, automatic profile selection, and shell integration.",
		Example: "  agentwho init",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			reader := bufio.NewReader(a.in)
			fmt.Fprintln(a.out, a.accent("Welcome to AgentWho"))
			fmt.Fprintln(a.out, "\nAgentWho keeps your personal and work Claude and Codex accounts")
			fmt.Fprintln(a.out, "separate and selects the right profile for each project.")
			fmt.Fprintln(a.out, "\nYour credentials remain managed by the official Claude and Codex CLIs.")
			fmt.Fprint(a.out, "AgentWho never reads or copies them.\n\n")
			a.initSection("How account separation works")
			fmt.Fprintln(a.out, "AgentWho calls each isolated account setup a profile. Onboarding creates")
			fmt.Fprintln(a.out, "two profiles: personal and work.")
			fmt.Fprintln(a.out, "\nA profile groups the Claude and Codex accounts for one identity. It also")
			fmt.Fprintln(a.out, "has its own user-level settings and MCP setup, plugins or skills, and")
			fmt.Fprintln(a.out, "session history. Those do not automatically carry between profiles.")
			fmt.Fprintln(a.out, "\nYour existing Claude and Codex data stays untouched. AgentWho never copies,")
			fmt.Fprintln(a.out, "moves, or deletes it.")
			if _, err := os.Stat(p.Config); err == nil {
				if !interactive(a.stdinFile) {
					return fmt.Errorf("AgentWho is already configured at %s\n\nRun this command in an interactive terminal to replace it", p.Config)
				}
				fmt.Fprintln(a.out, "AgentWho is already configured at:")
				fmt.Fprintln(a.out, " ", p.Config)
				if !askYes(reader, a.out, "Replace the existing configuration?") {
					fmt.Fprintln(a.out, "No changes made.")
					return nil
				}
			} else if !os.IsNotExist(err) {
				return err
			}
			if err := p.EnsureBase(); err != nil {
				return err
			}
			c := config.Default()
			c.Profiles["work"] = config.Profile{Kind: "work"}
			a.initSection("Default profile")
			fmt.Fprintln(a.out, "AgentWho uses this profile in folders you have not assigned to personal")
			fmt.Fprintln(a.out, "or work.")
			value, err := a.selectOne(reader, "Which profile should AgentWho use by default?", []menuOption{
				{Label: "Personal", Description: "Use your personal Claude and Codex accounts.", Value: "personal"},
				{Label: "Work", Description: "Use your work Claude and Codex accounts.", Value: "work"},
			}, 0)
			if err != nil {
				return err
			}
			c.Defaults.Profile = value
			a.initSection("Mismatch protection")
			fmt.Fprintln(a.out, "A mismatch happens when a project expects one profile but your current")
			fmt.Fprintln(a.out, "shell is explicitly using another. For example, a work project expects")
			fmt.Fprintln(a.out, "\"work\", but the shell is using \"personal\".")
			fmt.Fprintln(a.out, "\nThat could send work code through a personal account—or personal code")
			fmt.Fprintln(a.out, "through a company-managed account.")
			safetyMode, err := a.selectOne(reader, "What should AgentWho do when profiles do not match?", safetyModeOptions(), 1)
			if err != nil {
				return err
			}
			c.Defaults.Enforcement = safetyMode
			for name := range c.Profiles {
				if err := p.EnsureProfile(name); err != nil {
					return err
				}
			}
			if err := config.Save(p.Config, c); err != nil {
				return err
			}
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			if err := shim.Install(p.BinDir, exe, []string{"claude", "codex"}); err != nil {
				return err
			}
			fmt.Fprintln(a.out, "\n"+a.success("✓ Automatic profile selection enabled for Claude and Codex."))
			shellName := shell.Detect(os.Getenv("SHELL"))
			if shellName != "zsh" && shellName != "bash" && shellName != "fish" {
				shellName = "zsh"
			}
			home, _ := os.UserHomeDir()
			configFile := shell.DefaultConfig(shellName, home)
			shellConfigured, err := shell.IsConfigured(configFile, shellName)
			if err != nil {
				return fmt.Errorf("check shell integration in %s: %w", configFile, err)
			}
			if shellConfigured {
				fmt.Fprintf(a.out, "\n%s Shell integration is already configured in %s.\n", a.success("✓"), configFile)
			} else {
				a.initSection("Shell setup")
				fmt.Fprintf(a.out, "AgentWho can update %s so new terminals use the protected commands.\n", configFile)
				fmt.Fprintln(a.out, "A backup will be created first if the file already exists.")
				if askYesDefault(reader, a.out, "Update "+configFile+" now?") {
					backup, changed, err := shell.AddBlock(configFile, shellName)
					if err != nil {
						return fmt.Errorf("update shell configuration %s: %w", configFile, err)
					}
					if changed {
						fmt.Fprintf(a.out, "%s Updated %s.\n", a.success("✓"), configFile)
						if backup != "" {
							fmt.Fprintf(a.out, "  Backup: %s\n", backup)
						}
						fmt.Fprintln(a.out, "  Open a new terminal to activate AgentWho.")
					}
				} else {
					fmt.Fprintln(a.out, "No shell files were changed.")
					printShellInstructions(a.out, configFile, shellName)
				}
			}
			a.initSection("Prompt indicator")
			if askYes(reader, a.out, "Show the active profile beside your command prompt (for example, [agent:work])?") {
				setup, err := shell.PromptSetup(shellName)
				if err != nil {
					return err
				}
				fmt.Fprintf(a.out, "\nAdd this after your existing prompt or theme setup in %s:\n\n%s\n", configFile, setup)
			}
			a.initSection("Setup complete")
			fmt.Fprintln(a.out, a.success("AgentWho is ready."))
			fmt.Fprintln(a.out)
			fmt.Fprintln(a.out, "Profiles:")
			fmt.Fprintln(a.out, "  personal")
			fmt.Fprintln(a.out, "  work")
			fmt.Fprintf(a.out, "\nDefault profile: %s\nDefault safety mode: %s\n", c.Defaults.Profile, c.Defaults.Enforcement)
			fmt.Fprintln(a.out, "Automatic profile selection: enabled")
			fmt.Fprintln(a.out, "\nNext steps:")
			fmt.Fprintln(a.out, "  agentwho profile login personal claude")
			fmt.Fprintln(a.out, "  agentwho profile login personal codex")
			fmt.Fprintln(a.out, "  agentwho profile login work claude")
			fmt.Fprintln(a.out, "  agentwho profile login work codex")
			fmt.Fprintln(a.out, "\nRun `agentwho status` at any time to see which profile applies.")
			return nil
		},
	}
}

func (a *app) initSection(title string) {
	fmt.Fprintf(a.out, "\n%s\n%s\n", a.muted("────────────────────────────────────────"), a.accent(title))
}
