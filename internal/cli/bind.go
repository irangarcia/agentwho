package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/gitctx"
	"github.com/irangarcia/agentwho/internal/resolve"
	"github.com/spf13/cobra"
)

func (a *app) bindCmd() *cobra.Command {
	var repo, organization bool
	var pathValue, safetyMode, legacyEnforcement string
	cmd := &cobra.Command{
		Use:     "bind <profile>",
		Short:   "Use a profile for this repository or directory",
		Long:    "Bind a profile to the current repository, its Git organization, or a directory and its subdirectories.",
		Example: "  agentwho bind work\n  agentwho bind work --repo --safety-mode block\n  agentwho bind work --organization\n  agentwho bind personal --path ~/projects/personal",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			profile := args[0]
			if _, ok := c.Profiles[profile]; !ok {
				return fmt.Errorf("profile %q does not exist\n\nRun `agentwho profile list` to see available profiles", profile)
			}
			count := 0
			if repo {
				count++
			}
			if organization {
				count++
			}
			if pathValue != "" {
				count++
			}
			if count > 1 {
				return fmt.Errorf("choose only one binding type:\n\n  --repo\n  --organization\n  --path <directory>")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			ctx := gitctx.Detect(context.Background(), cwd, gitctx.ExecRunner{})
			if safetyMode != "" && legacyEnforcement != "" {
				return fmt.Errorf("use only one of --safety-mode or --enforcement")
			}
			if safetyMode == "" {
				safetyMode = legacyEnforcement
			}
			safetyModeProvided := safetyMode != ""
			if safetyMode == "" {
				safetyMode = c.Defaults.Enforcement
			}
			if safetyMode != "block" && safetyMode != "confirm" {
				return fmt.Errorf("invalid safety mode %q\n\nChoose:\n  block\n  confirm", safetyMode)
			}
			if count == 0 {
				if !interactive(a.stdinFile) {
					return fmt.Errorf("a non-interactive binding needs one of:\n\n  --repo\n  --organization\n  --path <directory>")
				}
				choice, err := a.chooseBinding(ctx, profile)
				if err != nil {
					return err
				}
				if choice == 0 {
					return nil
				}
				repo, organization = choice == 1, choice == 2
				if choice == 3 {
					pathValue = cwd
				}
				if !safetyModeProvided {
					safetyMode, err = a.chooseSafetyMode(c.Defaults.Enforcement)
					if err != nil {
						return err
					}
				}
			}
			var match config.Match
			switch {
			case repo:
				if ctx.Remote == "" {
					return fmt.Errorf("this repository does not have a usable `origin` remote\n\nUse a directory binding instead:\n  agentwho bind %s --path .", profile)
				}
				match.GitRemote = ctx.Remote
			case organization:
				if ctx.Organization == "" {
					return fmt.Errorf("the `origin` remote does not contain an organization or namespace\n\nUse `--repo` or `--path` instead")
				}
				match.GitOrganization = ctx.Organization
			default:
				home, _ := os.UserHomeDir()
				normalized, err := gitctx.ExpandPath(pathValue, home, cwd)
				if err != nil {
					return err
				}
				match.Path = normalized
			}
			rule := config.Rule{Match: match, Profile: profile, Enforcement: safetyMode}
			for _, existing := range c.Rules {
				if resolve.Equivalent(existing, rule) {
					kind, value := existing.Match.TypeValue()
					return fmt.Errorf("this %s already has a binding\n\n  Scope:       %s\n  Profile:     %s\n  Safety mode: %s\n\nRemove it first with:\n  agentwho unbind", friendlyMatcher(kind), value, existing.Profile, existing.Enforcement)
				}
			}
			typeName, value := rule.Match.TypeValue()
			newRuleIndex := len(c.Rules)
			c.Rules = append(c.Rules, rule)
			if err := config.Save(p.Config, c); err != nil {
				return err
			}
			fmt.Fprintln(a.out, a.success(fmt.Sprintf("✓ %s bound to profile %q.", matcherHeading(typeName), profile)))
			fmt.Fprintln(a.out)
			fmt.Fprintf(a.out, "  %-13s %s\n  Safety mode:  %s\n", matcherHeading(typeName)+":", value, safetyMode)
			resolved := resolve.Resolve(c, ctx)
			override := os.Getenv("AGENTWHO_PROFILE")
			switch {
			case resolved.RuleIndex != newRuleIndex:
				if bindingAppliesHere(rule, ctx) {
					fmt.Fprintln(a.out, "\n"+a.warning("⚠ Binding saved, but a more specific binding still controls this directory:"))
				} else {
					fmt.Fprintln(a.out, "\nNote: this binding does not apply to the current directory.")
				}
				if resolved.Matched != nil {
					activeType, activeValue := resolved.Matched.Match.TypeValue()
					fmt.Fprintf(a.out, "\n  %-17s %s\n", matcherHeading(activeType)+":", activeValue)
				} else {
					fmt.Fprintln(a.out, "\n  Binding:          default profile")
				}
				fmt.Fprintf(a.out, "  Current profile:  %s\n  Safety mode:      %s\n", resolved.Expected, resolved.Enforcement)
				if bindingAppliesHere(rule, ctx) {
					fmt.Fprintln(a.out, "\nTo let the new broader binding control this directory, remove the more specific binding:")
					fmt.Fprintln(a.out, "  agentwho unbind")
				}
			case override != "" && override != profile:
				fmt.Fprintln(a.out, "\n"+a.warning(fmt.Sprintf("⚠ Binding saved, but the current profile is still %q because it was selected for this shell.", override)))
				fmt.Fprintln(a.out, "  Return to automatic selection with:")
				fmt.Fprintln(a.out, "    agentwho use --auto")
			default:
				fmt.Fprintf(a.out, "\nCurrent profile here: %s\n", a.success(profile))
				fmt.Fprintln(a.out, "Claude and Codex will use this profile in this context.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&repo, "repo", false, "bind this repository")
	cmd.Flags().BoolVar(&organization, "organization", false, "bind this Git organization")
	cmd.Flags().StringVar(&pathValue, "path", "", "bind a directory and its subdirectories")
	cmd.Flags().StringVar(&safetyMode, "safety-mode", "", "safety mode: block or confirm")
	cmd.Flags().StringVar(&legacyEnforcement, "enforcement", "", "deprecated alias for --safety-mode")
	_ = cmd.Flags().MarkDeprecated("enforcement", "use --safety-mode")
	return cmd
}

func bindingAppliesHere(rule config.Rule, ctx gitctx.Context) bool {
	switch {
	case rule.Match.GitRemote != "":
		return rule.Match.GitRemote == ctx.Remote
	case rule.Match.GitOrganization != "":
		return rule.Match.GitOrganization == ctx.Organization
	case rule.Match.Path != "":
		return gitctx.PathContains(rule.Match.Path, ctx.Directory)
	default:
		return false
	}
}

func (a *app) chooseBinding(ctx gitctx.Context, profile string) (int, error) {
	options := []menuOption{}
	if ctx.Remote != "" {
		options = append(options, menuOption{Label: "This repository only", Description: ctx.Remote, Value: "repo"})
	}
	if ctx.Organization != "" {
		options = append(options, menuOption{Label: "Every repository in this organization", Description: ctx.Organization, Value: "organization"})
	}
	options = append(options, menuOption{Label: "This directory and everything below it", Description: ctx.Directory, Value: "path"})
	if len(options) == 1 {
		fmt.Fprintln(a.out)
		if ctx.GitRoot == "" {
			fmt.Fprintln(a.out, "No Git repository was found at this directory.")
		} else {
			fmt.Fprintln(a.out, "This Git repository does not have a usable `origin` remote.")
		}
		fmt.Fprintf(a.out, "\nAgentWho can bind profile %q to this directory tree instead:\n\n", profile)
		fmt.Fprintf(a.out, "  Directory: %s\n", ctx.Directory)
		fmt.Fprintln(a.out, "  Applies to: this directory and everything below it")
		if !askYes(bufio.NewReader(a.in), a.out, "\nCreate this directory binding?") {
			fmt.Fprintln(a.out, "No changes made.")
			return 0, nil
		}
		fmt.Fprintln(a.out, "\n"+a.success("✓ Selected: This directory and everything below it"))
		return 3, nil
	}
	value, err := a.selectOne(bufio.NewReader(a.in), fmt.Sprintf("Use profile %q for:", profile), options, 0)
	if err != nil {
		return 0, err
	}
	switch value {
	case "repo":
		return 1, nil
	case "organization":
		return 2, nil
	default:
		return 3, nil
	}
}

func (a *app) chooseSafetyMode(fallback string) (string, error) {
	defaultChoice := 1
	if fallback == "block" {
		defaultChoice = 0
	}
	return a.selectOne(bufio.NewReader(a.in), "Safety mode for this binding:", safetyModeOptions(), defaultChoice)
}

func safetyModeOptions() []menuOption {
	return []menuOption{
		{Label: "Block", Description: "Never start the agent with the wrong profile.", Value: "block"},
		{Label: "Confirm", Description: "Show a warning and ask before continuing.", Value: "confirm"},
	}
}

func friendlyMatcher(kind string) string {
	switch kind {
	case "git_remote":
		return "repository"
	case "git_organization":
		return "organization"
	default:
		return "directory"
	}
}

func matcherHeading(kind string) string {
	switch kind {
	case "git_remote":
		return "Repository"
	case "git_organization":
		return "Organization"
	default:
		return "Directory"
	}
}

func (a *app) unbindCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "unbind",
		Short:   "Remove the binding that applies here",
		Long:    "Remove only the most specific profile binding that applies to the current directory.",
		Example: "  agentwho unbind\n  agentwho unbind --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			ctx := gitctx.Detect(context.Background(), cwd, gitctx.ExecRunner{})
			result := resolve.Resolve(c, ctx)
			if result.RuleIndex < 0 {
				return fmt.Errorf("no binding applies to the current directory\n\nRun `agentwho rules` to see all bindings")
			}
			typeName, value := result.Matched.Match.TypeValue()
			fmt.Fprint(a.out, "Binding to remove:\n\n")
			fmt.Fprintf(a.out, "  %-13s %s\n  Profile:      %s\n  Safety mode:  %s\n", matcherHeading(typeName)+":", value, result.Matched.Profile, result.Matched.Enforcement)
			if !yes {
				if !interactive(a.stdinFile) {
					return fmt.Errorf("confirmation needs an interactive terminal\n\nTo remove this binding without asking, run:\n  agentwho unbind --yes")
				}
				if !askYes(bufio.NewReader(a.in), a.out, "\nRemove this binding?") {
					fmt.Fprintln(a.out, "No changes made.")
					return nil
				}
			}
			c.Rules = append(c.Rules[:result.RuleIndex], c.Rules[result.RuleIndex+1:]...)
			if err := config.Save(p.Config, c); err != nil {
				return err
			}
			fmt.Fprintln(a.out, a.success("✓ Binding removed."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "remove without asking")
	return cmd
}

type ruleJSON struct {
	MatcherType  string `json:"matcher_type"`
	MatcherValue string `json:"matcher_value"`
	Profile      string `json:"profile"`
	SafetyMode   string `json:"safety_mode"`
	// Enforcement is retained for JSON compatibility.
	Enforcement string `json:"enforcement"`
	Specificity string `json:"specificity"`
}

func (a *app) rulesCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "rules",
		Short:   "Show all profile bindings",
		Long:    "Show repository, organization, and directory bindings in the order AgentWho resolves them.",
		Example: "  agentwho rules\n  agentwho rules --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, err := load()
			if err != nil {
				return err
			}
			ordered := resolve.Ordered(c.Rules)
			rows := make([]ruleJSON, 0, len(ordered))
			for _, item := range ordered {
				kind, value := item.Rule.Match.TypeValue()
				rows = append(rows, ruleJSON{
					MatcherType: kind, MatcherValue: value, Profile: item.Rule.Profile,
					SafetyMode: item.Rule.Enforcement, Enforcement: item.Rule.Enforcement, Specificity: item.Specificity,
				})
			}
			if jsonOut {
				return writeJSON(a.out, rows)
			}
			if len(rows) == 0 {
				fmt.Fprintln(a.out, "No bindings configured.\n\nCreate one from a project directory:\n  agentwho bind <profile>")
				return nil
			}
			w := tabwriter.NewWriter(a.out, 0, 4, 3, ' ', 0)
			fmt.Fprintln(w, "TYPE\tSCOPE\tPROFILE\tSAFETY MODE\tPRIORITY")
			for _, row := range rows {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", friendlyMatcher(row.MatcherType), row.MatcherValue, row.Profile, row.SafetyMode, row.Specificity)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output stable JSON")
	return cmd
}
