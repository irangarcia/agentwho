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
		Long:    "Set up profiles, mismatch safety, automatic profile selection, and shell integration.",
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
			a.initSection("Profiles")
			fmt.Fprintln(a.out, a.success("✓ Created profile \"personal\"."))
			fmt.Fprintln(a.out)
			work := askYes(reader, a.out, "Create a separate work profile now?")
			if work {
				c.Profiles["work"] = config.Profile{Kind: "work"}
				fmt.Fprintln(a.out, a.success("✓ Created profile \"work\"."))
			}
			if work {
				value, err := a.selectOne(reader, "Which profile should be used when no binding matches?", []menuOption{
					{Label: "personal", Value: "personal"},
					{Label: "work", Value: "work"},
				}, 0)
				if err != nil {
					return err
				}
				c.Defaults.Profile = value
			}
			a.initSection("Safety mode")
			safetyMode, err := a.selectOne(reader, "How should AgentWho handle a profile mismatch?", safetyModeOptions(), 1)
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
			shimsEnabled := false
			a.initSection("Terminal integration")
			fmt.Fprint(a.out, "Route the `claude` and `codex` terminal commands through AgentWho?\n\n")
			fmt.Fprintln(a.out, "When enabled, you keep using `claude` and `codex` normally.")
			fmt.Fprintln(a.out, "AgentWho selects the expected profile for the current directory and")
			fmt.Fprint(a.out, "blocks or confirms profile mismatches.\n\n")
			fmt.Fprintln(a.out, "This protects terminal commands only. VS Code extension panels are")
			fmt.Fprint(a.out, "not currently protected.\n\n")
			if askYesDefault(reader, a.out, "Enable terminal integration now?") {
				exe, err := os.Executable()
				if err != nil {
					return err
				}
				if err := shim.Install(p.BinDir, exe, []string{"claude", "codex"}); err != nil {
					return err
				}
				shimsEnabled = true
				fmt.Fprintln(a.out, a.success("✓ Automatic profile selection enabled for Claude and Codex."))
			}
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
				fmt.Fprintf(a.out, "\nAdd this line to %s:\n\n  %s\n", configFile, shell.EvalLine(shellName))
				fmt.Fprintf(a.out, "\nThen restart your terminal or run it now:\n\n  %s\n", shell.EvalLine(shellName))
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
			if work {
				fmt.Fprintln(a.out, "  work")
			}
			fmt.Fprintf(a.out, "\nDefault profile: %s\nDefault safety mode: %s\n", c.Defaults.Profile, c.Defaults.Enforcement)
			if shimsEnabled {
				fmt.Fprintln(a.out, "Automatic profile selection: enabled")
			} else {
				fmt.Fprintln(a.out, "Automatic profile selection: not enabled")
				fmt.Fprintln(a.out, "Enable it later with: agentwho install")
			}
			fmt.Fprintln(a.out, "\nNext steps:")
			fmt.Fprintln(a.out, "  agentwho profile login personal claude")
			fmt.Fprintln(a.out, "  agentwho profile login personal codex")
			if work {
				fmt.Fprintln(a.out, "  agentwho profile login work claude")
				fmt.Fprintln(a.out, "  agentwho profile login work codex")
			}
			fmt.Fprintln(a.out, "\nRun `agentwho status` at any time to see which profile applies.")
			return nil
		},
	}
}

func (a *app) initSection(title string) {
	fmt.Fprintf(a.out, "\n%s\n%s\n", a.muted("────────────────────────────────────────"), a.accent(title))
}
