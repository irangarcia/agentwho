package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/irangarcia/agentwho/internal/agent"
	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/execution"
	"github.com/irangarcia/agentwho/internal/shell"
	"github.com/irangarcia/agentwho/internal/shim"
	"github.com/spf13/cobra"
)

type check struct {
	Group        string
	OK, Critical bool
	Message, Fix string
}

func (a *app) doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doctor",
		Short:   "Check setup and show how to fix problems",
		Long:    "Check configuration, profiles, automatic command routing, official CLIs, sign-ins, PATH, and shell setup.",
		Example: "  agentwho doctor",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			checks := []check{}
			c, configErr := config.Load(p.Config)
			checks = append(checks, check{"Configuration", configErr == nil, true, choose(configErr == nil, "Configuration is valid", "Configuration is missing or invalid: "+errorText(configErr)), "Run `agentwho init`."})
			if info, err := os.Stat(p.DataDir); err == nil && info.IsDir() && info.Mode().Perm()&0o077 == 0 {
				checks = append(checks, check{"Configuration", true, false, "Data directory permissions are secure", ""})
			} else {
				checks = append(checks, check{"Configuration", false, true, "Data directory is missing or accessible by other users", "Run `chmod 700 \"" + p.DataDir + "\"` or `agentwho init`."})
			}
			if configErr == nil {
				names := make([]string, 0, len(c.Profiles))
				for name := range c.Profiles {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					for _, ag := range []string{"claude", "codex"} {
						dir := p.Profile(name, ag)
						info, err := os.Stat(dir)
						checks = append(checks, check{"Configuration", err == nil && info.IsDir(), true, fmt.Sprintf("Profile %s / %s directory is available", name, agentLabel(ag)), "Run `agentwho init` or recreate the profile."})
					}
				}
			}
			claudeActive := execution.IsActive("claude", p.BinDir, os.Getenv("PATH"))
			codexActive := execution.IsActive("codex", p.BinDir, os.Getenv("PATH"))
			for _, ag := range agent.All() {
				shimPath := filepath.Join(p.BinDir, ag.Name())
				if target, err := os.Readlink(shimPath); err == nil {
					if !filepath.IsAbs(target) {
						target = filepath.Join(filepath.Dir(shimPath), target)
					}
					if _, err := os.Stat(target); err != nil {
						checks = append(checks, check{"Automatic profile selection", false, true, "Broken command link: " + shimPath, "Run `agentwho install`."})
					}
				}
				managed := shim.IsManaged(shimPath)
				executable := false
				if info, err := os.Stat(shimPath); err == nil {
					executable = info.Mode().Perm()&0o111 != 0
				}
				installed := managed && executable
				checks = append(checks, check{"Automatic profile selection", installed, true, choose(installed, "AgentWho command for "+ag.DisplayName()+" is installed", "AgentWho command for "+ag.DisplayName()+" is not installed correctly"), "Run `agentwho install`."})
				real, realErr := execution.FindReal(ag.BinaryName(), p.BinDir, os.Getenv("PATH"))
				if realErr == nil {
					checks = append(checks, check{"Official CLIs", true, false, ag.DisplayName() + ": " + real, ""})
				} else {
					checks = append(checks, check{"Official CLIs", false, true, ag.DisplayName() + ": not found", "Install the official CLI, then run `agentwho doctor` again."})
				}
				if managed && realErr == nil && sameFile(shimPath, real) {
					checks = append(checks, check{"Automatic profile selection", false, true, ag.DisplayName() + " command points back to AgentWho instead of the official CLI", "Remove duplicate command links from PATH, then run `agentwho install`."})
				}
				if configErr == nil && realErr == nil {
					names := make([]string, 0, len(c.Profiles))
					for name := range c.Profiles {
						names = append(names, name)
					}
					sort.Strings(names)
					for _, name := range names {
						status := agent.Status(context.Background(), ag, name, p, real)
						ok := status == agent.Authenticated
						fix := "Run `agentwho profile login " + name + " " + ag.Name() + "`."
						checks = append(checks, check{"Profiles", ok, false, fmt.Sprintf("%s / %s: %s", name, ag.DisplayName(), humanAuth(status)), fix})
					}
				}
			}
			commandsActive := claudeActive && codexActive
			checks = append(checks, check{"Automatic profile selection", commandsActive, true, choose(commandsActive, "AgentWho commands are active in PATH", "AgentWho commands are not active in PATH"), "Run `eval \"$(agentwho shell init " + detectedShell() + ")\"` and add it to your shell configuration."})
			shellName := detectedShell()
			checks = append(checks, check{"Shell", shellName == "zsh" || shellName == "bash" || shellName == "fish", false, "Shell: " + shellName, "Use zsh, bash, or fish."})
			home, _ := os.UserHomeDir()
			shellConfig := shell.DefaultConfig(shellName, home)
			if shellConfig != "" {
				if b, err := os.ReadFile(shellConfig); err == nil {
					switch count := shell.CountBlocks(string(b)); {
					case count == 1:
						checks = append(checks, check{"Shell", true, false, "Shell configuration contains one AgentWho setup block", ""})
					case count > 1:
						checks = append(checks, check{"Shell", false, false, "Multiple AgentWho setup blocks found in " + shellConfig, "Keep one AgentWho block and remove the duplicates."})
					default:
						checks = append(checks, check{"Shell", false, false, "AgentWho setup was not found in " + shellConfig, "Add `" + shell.EvalLine(shellName) + "` to the file."})
					}
				} else if os.IsNotExist(err) {
					checks = append(checks, check{"Shell", false, false, "Shell configuration does not exist: " + shellConfig, "Create the file and add `" + shell.EvalLine(shellName) + "`."})
				}
			}
			critical := false
			problems := 0
			order := map[string]int{"Configuration": 0, "Automatic profile selection": 1, "Official CLIs": 2, "Profiles": 3, "Shell": 4}
			sort.SliceStable(checks, func(i, j int) bool {
				left, right := order[checks[i].Group], order[checks[j].Group]
				if left != right {
					return left < right
				}
				if checks[i].Group == "Profiles" {
					return checks[i].Message < checks[j].Message
				}
				return false
			})
			fmt.Fprintln(a.out, a.accent("AgentWho doctor"))
			group := ""
			for _, item := range checks {
				if item.Group != group {
					group = item.Group
					fmt.Fprintf(a.out, "\n%s\n", a.bold(group))
				}
				line := "✓ " + item.Message
				if !item.OK {
					line = "✗ " + item.Message
					problems++
					if item.Critical {
						critical = true
					}
				}
				if item.OK {
					line = a.success(line)
				} else {
					line = a.danger(line)
				}
				fmt.Fprintf(a.out, "  %s\n", line)
				if !item.OK && item.Fix != "" {
					fmt.Fprintln(a.out, "   ", a.warning("Fix:"), item.Fix)
				}
			}
			switch problems {
			case 0:
				fmt.Fprintln(a.out, "\n"+a.success("Result: Everything looks good."))
			case 1:
				fmt.Fprintln(a.out, "\n"+a.warning("Result: 1 problem found"))
			default:
				fmt.Fprintln(a.out, "\n"+a.warning(fmt.Sprintf("Result: %d problems found", problems)))
			}
			if critical {
				return silent(errors.New("AgentWho setup problems found"))
			}
			return nil
		},
	}
}

func agentLabel(name string) string {
	if name == "claude" {
		return "Claude"
	}
	return "Codex"
}

func choose(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func detectedShell() string {
	value := shell.Detect(os.Getenv("SHELL"))
	if value == "" {
		return "zsh"
	}
	return value
}

func sameFile(a, b string) bool {
	ai, ae := os.Stat(a)
	bi, be := os.Stat(b)
	return ae == nil && be == nil && os.SameFile(ai, bi)
}
