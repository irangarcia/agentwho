package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/paths"
	"github.com/irangarcia/agentwho/internal/termstyle"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type app struct {
	in          io.Reader
	out, errout io.Writer
	stdinFile   *os.File
}

func New() *cobra.Command {
	a := &app{in: os.Stdin, out: os.Stdout, errout: os.Stderr, stdinFile: os.Stdin}
	root := &cobra.Command{
		Use:          "agentwho",
		Short:        "Use the right Claude or Codex account for every project",
		Long:         "AgentWho keeps Claude and Codex on the right account for each project.",
		SilenceUsage: true, SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	root.SetIn(a.in)
	root.SetOut(a.out)
	root.SetErr(a.errout)
	root.AddCommand(
		a.initCmd(), a.installCmd(), a.uninstallCmd(), a.doctorCmd(), a.profileCmd(),
		a.bindCmd(), a.unbindCmd(), a.rulesCmd(), a.useCmd(), a.currentCmd(), a.statusCmd(), a.promptCmd(), a.shellCmd(), a.completionCmd(), a.internalCmd(),
	)
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == root {
			fmt.Fprint(cmd.OutOrStdout(), styledRootHelp(cmd.OutOrStdout()))
			return
		}
		defaultHelp(cmd, args)
	})
	return root
}

func styledRootHelp(w io.Writer) string {
	text := rootHelp
	first, rest, _ := strings.Cut(text, "\n")
	text = termstyle.Paint(w, termstyle.Accent, first) + "\n" + rest
	for _, heading := range []string{"Usage:", "Get started:", "Profiles:", "Bindings:", "Current directory:", "Setup and maintenance:", "Shell:", "Help:"} {
		text = strings.Replace(text, heading, termstyle.Paint(w, termstyle.Bold, heading), 1)
	}
	return text
}

const rootHelp = `AgentWho keeps Claude and Codex on the right account for each project.

Usage:
  agentwho <command> [options]

Get started:
  init                              Set up AgentWho

Profiles:
  profile add <name>                Create a profile
  profile list                      Show profiles and sign-in status
  profile login <profile> <agent>   Sign in to Claude or Codex for a profile

Bindings:
  bind <profile>                    Use a profile for this repository or directory
  unbind                            Remove the binding that applies here
  rules                             Show all profile bindings

Current directory:
  use <profile>                     Use a profile in this shell
  current                           Print the current profile name for scripts
  status                            Show which profile applies here and why
  prompt                            Print the active profile for a shell prompt

Setup and maintenance:
  install                           Enable automatic profile selection
  uninstall                         Disable AgentWho integration
  doctor                            Check setup and show how to fix problems

Shell:
  shell init <shell>                Print shell setup code
  completion <shell>                Print shell completion code

Help:
  help <command>                    Show help for a command

Run ` + "`agentwho <command> --help`" + ` for examples and options.
`

type silentError struct{ err error }

func (e silentError) Error() string { return e.err.Error() }
func (e silentError) Unwrap() error { return e.err }
func silent(err error) error        { return silentError{err: err} }

// IsSilent reports errors whose details were already presented to the user.
func IsSilent(err error) bool {
	var target silentError
	return errors.As(err, &target)
}

func getPaths() (paths.Paths, error) { return paths.Discover() }

func load() (config.Config, paths.Paths, error) {
	p, err := getPaths()
	if err != nil {
		return config.Config{}, p, err
	}
	c, err := config.Load(p.Config)
	if err != nil {
		if os.IsNotExist(err) {
			return c, p, fmt.Errorf("AgentWho is not set up yet.\n\nRun:\n  agentwho init")
		}
		return c, p, fmt.Errorf("AgentWho could not read its configuration.\n\nFile:\n  %s\n\nProblem:\n  %w", p.Config, err)
	}
	return c, p, nil
}

func (a *app) completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "completion <bash|zsh|fish>",
		Short:   "Print shell completion code",
		Long:    "Print a completion script for a supported shell.",
		Example: "  agentwho completion zsh > ~/.zfunc/_agentwho\n  agentwho completion bash > ~/.local/share/bash-completion/completions/agentwho\n  agentwho completion fish > ~/.config/fish/completions/agentwho.fish",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(a.out)
			case "zsh":
				return cmd.Root().GenZshCompletion(a.out)
			case "fish":
				return cmd.Root().GenFishCompletion(a.out, true)
			default:
				return fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", args[0])
			}
		},
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func interactive(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}

func askYes(r *bufio.Reader, w io.Writer, prompt string) bool {
	fmt.Fprint(w, prompt+" [y/N] ")
	answer, _ := r.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func askYesDefault(r *bufio.Reader, w io.Writer, prompt string) bool {
	fmt.Fprint(w, prompt+" [Y/n] ")
	answer, _ := r.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

func ask(r *bufio.Reader, w io.Writer, prompt, fallback string) string {
	fmt.Fprintf(w, "%s [%s]: ", prompt, fallback)
	answer, _ := r.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return fallback
	}
	return answer
}
