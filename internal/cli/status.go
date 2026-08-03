package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/execution"
	"github.com/irangarcia/agentwho/internal/gitctx"
	"github.com/irangarcia/agentwho/internal/resolve"
	"github.com/irangarcia/agentwho/internal/shim"
	"github.com/spf13/cobra"
)

type statusOutput struct {
	Directory       string            `json:"directory"`
	GitRoot         string            `json:"git_root,omitempty"`
	GitRemote       string            `json:"git_remote,omitempty"`
	MatchedRule     *statusRuleOutput `json:"matched_rule,omitempty"`
	Specificity     string            `json:"specificity"`
	ExpectedProfile string            `json:"expected_profile"`
	CurrentProfile  string            `json:"current_profile"`
	SafetyMode      string            `json:"safety_mode"`
	// SelectedProfile and Enforcement are retained for JSON compatibility.
	SelectedProfile string `json:"selected_profile"`
	Enforcement     string `json:"enforcement"`
	ClaudeShim      bool   `json:"claude_shim_installed"`
	CodexShim       bool   `json:"codex_shim_installed"`
	AutomaticActive bool   `json:"automatic_selection_active"`
	ClaudeActive    bool   `json:"-"`
	CodexActive     bool   `json:"-"`
	Mismatch        bool   `json:"mismatch"`
	Status          string `json:"status"`
}

type statusRuleOutput struct {
	Match      config.Match `json:"match"`
	Profile    string       `json:"profile"`
	SafetyMode string       `json:"safety_mode"`
	// Enforcement is retained for JSON compatibility.
	Enforcement string `json:"enforcement"`
}

func currentStatus(c config.Config, pBin string) (statusOutput, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return statusOutput{}, err
	}
	ctx := gitctx.Detect(context.Background(), cwd, gitctx.ExecRunner{})
	r := resolve.Resolve(c, ctx)
	selected := r.Expected
	if value := os.Getenv("AGENTWHO_PROFILE"); value != "" {
		selected = value
	}
	mismatch := selected != r.Expected
	label := "OK"
	if mismatch {
		label = "MISMATCH"
	}
	var matched *statusRuleOutput
	if r.Matched != nil {
		matched = &statusRuleOutput{
			Match: r.Matched.Match, Profile: r.Matched.Profile,
			SafetyMode: r.Matched.Enforcement, Enforcement: r.Matched.Enforcement,
		}
	}
	pathValue := os.Getenv("PATH")
	claudeActive := execution.IsActive("claude", pBin, pathValue)
	codexActive := execution.IsActive("codex", pBin, pathValue)
	return statusOutput{Directory: ctx.Directory, GitRoot: ctx.GitRoot, GitRemote: ctx.Remote, MatchedRule: matched,
		Specificity: r.Specificity, ExpectedProfile: r.Expected, CurrentProfile: selected, SafetyMode: r.Enforcement,
		SelectedProfile: selected, Enforcement: r.Enforcement,
		ClaudeShim: shim.IsManaged(filepath.Join(pBin, "claude")), CodexShim: shim.IsManaged(filepath.Join(pBin, "codex")),
		AutomaticActive: claudeActive && codexActive, ClaudeActive: claudeActive, CodexActive: codexActive,
		Mismatch: mismatch, Status: label}, nil
}

func (a *app) statusCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Show which profile applies here and why",
		Long:    "Show the current directory, matching binding, expected and current profiles, safety mode, and command integration.",
		Example: "  agentwho status\n  agentwho use personal\n  agentwho status --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			s, err := currentStatus(c, p.BinDir)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(a.out, s)
			}
			fmt.Fprint(a.out, a.accent("AgentWho status")+"\n\n")
			fmt.Fprintf(a.out, "Directory:         %s\n", s.Directory)
			if s.GitRoot != "" {
				fmt.Fprintf(a.out, "Git root:          %s\n", s.GitRoot)
			}
			if s.GitRemote != "" {
				fmt.Fprintf(a.out, "Repository:        %s\n", s.GitRemote)
			}
			if s.MatchedRule != nil {
				kind, value := s.MatchedRule.Match.TypeValue()
				fmt.Fprintf(a.out, "Matched by:        %s %s\n", friendlyMatcher(kind), value)
			} else {
				fmt.Fprintln(a.out, "Matched by:        default profile")
			}
			currentLine := a.success("Current profile:   " + s.CurrentProfile)
			if s.Mismatch {
				currentLine = a.danger("Current profile:   " + s.CurrentProfile)
			}
			fmt.Fprintf(a.out, "\n%s\n%s\nSafety mode:       %s\n", a.success("Expected profile:  "+s.ExpectedProfile), currentLine, s.SafetyMode)
			fmt.Fprintf(a.out, "\nClaude command:    %s\nCodex command:     %s\n", integrationState(s.ClaudeShim, s.ClaudeActive), integrationState(s.CodexShim, s.CodexActive))
			if s.Mismatch {
				fmt.Fprintln(a.out, "\n"+a.warning("⚠ Profile mismatch"))
			} else if s.ClaudeShim && s.CodexShim && s.AutomaticActive {
				fmt.Fprintln(a.out, "\n"+a.success("✓ Ready"))
			} else if s.ClaudeShim && s.CodexShim {
				fmt.Fprintln(a.out, "\n"+a.warning("⚠ Automatic profile selection is installed but not active in this shell."))
				fmt.Fprintf(a.out, "  Run: eval \"$(agentwho shell init %s)\"\n", detectedShell())
			} else {
				fmt.Fprintln(a.out, "\n"+a.warning("⚠ Automatic profile selection is not installed."))
				fmt.Fprintln(a.out, "  Run: agentwho install")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output stable JSON")
	return cmd
}

type currentOutput struct {
	Version         int    `json:"version"`
	Profile         string `json:"profile"`
	ExpectedProfile string `json:"expected_profile"`
	Source          string `json:"source"`
	Mismatch        bool   `json:"mismatch"`
}

func (a *app) currentCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "current",
		Short:   "Print the current profile name",
		Long:    "Print a stable, script-friendly current profile. The plain form writes only the profile name and a newline.",
		Example: "  agentwho current\n  profile=$(agentwho current)\n  agentwho current --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			s, err := currentStatus(c, p.BinDir)
			if err != nil {
				return err
			}
			if _, ok := c.Profiles[s.CurrentProfile]; !ok {
				return fmt.Errorf("current profile %q does not exist\n\nReturn to automatic selection with:\n  agentwho use --auto\n\nOr list available profiles with:\n  agentwho profile list", s.CurrentProfile)
			}
			source := "automatic"
			if os.Getenv("AGENTWHO_PROFILE") != "" {
				source = "explicit"
			}
			if jsonOut {
				return writeJSON(a.out, currentOutput{
					Version: 1, Profile: s.CurrentProfile, ExpectedProfile: s.ExpectedProfile,
					Source: source, Mismatch: s.Mismatch,
				})
			}
			fmt.Fprintln(a.out, s.CurrentProfile)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output stable versioned JSON")
	return cmd
}

func integrationState(installed, active bool) string {
	if installed && active {
		return "AgentWho active"
	}
	if installed {
		return "installed, not active in PATH"
	}
	return "AgentWho not installed"
}

type promptOutput struct {
	Initialized bool   `json:"initialized"`
	Profile     string `json:"profile,omitempty"`
	Mismatch    bool   `json:"mismatch"`
	Text        string `json:"text,omitempty"`
}

func (a *app) promptCmd() *cobra.Command {
	var plain, jsonOut bool
	cmd := &cobra.Command{
		Use:     "prompt",
		Short:   "Print the active profile for a shell prompt",
		Long:    "Print a fast local indicator such as [agent:work]. This command performs no network or authentication checks.",
		Example: "  agentwho prompt\n  agentwho prompt --plain\n  agentwho prompt --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			c, err := config.Load(p.Config)
			if os.IsNotExist(err) {
				if jsonOut {
					return writeJSON(a.out, promptOutput{Initialized: false})
				}
				return nil
			}
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			ctx := gitctx.Detect(context.Background(), cwd, gitctx.ExecRunner{})
			resolution := resolve.Resolve(c, ctx)
			selected := resolution.Expected
			if value := os.Getenv("AGENTWHO_PROFILE"); value != "" {
				selected = value
			}
			mismatch := selected != resolution.Expected
			mark := ""
			if mismatch {
				mark = "!"
			}
			text := "[agent:" + selected + mark + "]"
			if jsonOut {
				return writeJSON(a.out, promptOutput{Initialized: true, Profile: selected, Mismatch: mismatch, Text: text})
			}
			fmt.Fprintln(a.out, text)
			_ = plain
			return nil
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "print plain text without styling")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output stable JSON")
	return cmd
}
