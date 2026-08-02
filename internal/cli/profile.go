package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/irangarcia/agentwho/internal/agent"
	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/execution"
	"github.com/spf13/cobra"
)

func (a *app) profileCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Create, view, and sign in to profiles"}
	cmd.AddCommand(a.profileAddCmd(), a.profileListCmd(), a.profileLoginCmd())
	return cmd
}

func (a *app) profileAddCmd() *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:     "add <name>",
		Short:   "Create a profile",
		Long:    "Create a separate personal or work profile for Claude and Codex sign-ins.",
		Example: "  agentwho profile add work --kind work\n  agentwho profile add client-acme --kind work",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.ValidateProfileName(name); err != nil {
				return err
			}
			if kind != "personal" && kind != "work" {
				return fmt.Errorf("invalid profile type %q\n\nChoose either:\n  --kind personal\n  --kind work", kind)
			}
			c, p, err := load()
			if err != nil {
				return err
			}
			if _, exists := c.Profiles[name]; exists {
				return fmt.Errorf("profile %q already exists\n\nRun `agentwho profile list` to see your profiles", name)
			}
			c.Profiles[name] = config.Profile{Kind: kind}
			if err := p.EnsureProfile(name); err != nil {
				return err
			}
			if err := config.Save(p.Config, c); err != nil {
				return err
			}
			fmt.Fprintln(a.out, a.success(fmt.Sprintf("✓ Created profile %q.", name)))
			fmt.Fprintf(a.out, "  Type: %s\n", kind)
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "profile type: personal or work")
	return cmd
}

type profileRow struct {
	Name   string           `json:"name"`
	Kind   string           `json:"kind"`
	Claude agent.AuthStatus `json:"claude"`
	Codex  agent.AuthStatus `json:"codex"`
}

func (a *app) profileListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "Show profiles and sign-in status",
		Long:    "Show every profile and check sign-in status using the official Claude and Codex CLIs.",
		Example: "  agentwho profile list\n  agentwho profile list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(c.Profiles))
			for name := range c.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			executables := map[string]string{}
			for _, ag := range agent.All() {
				executable, _ := execution.FindReal(ag.BinaryName(), p.BinDir, os.Getenv("PATH"))
				executables[ag.Name()] = executable
			}
			rows := make([]profileRow, 0, len(names))
			for _, name := range names {
				rows = append(rows, profileRow{Name: name, Kind: c.Profiles[name].Kind,
					Claude: agent.Status(context.Background(), mustAgent("claude"), name, p, executables["claude"]),
					Codex:  agent.Status(context.Background(), mustAgent("codex"), name, p, executables["codex"])})
			}
			if jsonOut {
				return writeJSON(a.out, rows)
			}
			if len(rows) == 0 {
				fmt.Fprintln(a.out, "No profiles found.\n\nRun `agentwho init` to create your first profile.")
				return nil
			}
			w := tabwriter.NewWriter(a.out, 0, 4, 3, ' ', 0)
			fmt.Fprintln(w, "PROFILE\tTYPE\tCLAUDE\tCODEX")
			for _, row := range rows {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row.Name, row.Kind, humanAuth(row.Claude), humanAuth(row.Codex))
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output stable JSON")
	return cmd
}

func mustAgent(name string) agent.Agent { ag, _ := agent.Get(name); return ag }

func humanAuth(status agent.AuthStatus) string {
	switch status {
	case agent.Authenticated:
		return "signed in"
	case agent.NotAuthenticated:
		return "not signed in"
	case agent.Unavailable:
		return "CLI not found"
	default:
		return "could not check"
	}
}

func (a *app) profileLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "login <profile> <claude|codex>",
		Short:   "Sign in to Claude or Codex for a profile",
		Long:    "Run an official agent sign-in flow and keep that sign-in isolated inside one AgentWho profile.",
		Example: "  agentwho profile login personal claude\n  agentwho profile login work codex",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, p, err := load()
			if err != nil {
				return err
			}
			if _, ok := c.Profiles[args[0]]; !ok {
				names := make([]string, 0, len(c.Profiles))
				for name := range c.Profiles {
					names = append(names, name)
				}
				sort.Strings(names)
				return fmt.Errorf("profile %q does not exist\n\nAvailable profiles:\n  %s\n\nCreate it with:\n  agentwho profile add %s --kind personal|work", args[0], strings.Join(names, "\n  "), args[0])
			}
			ag, ok := agent.Get(args[1])
			if !ok {
				return fmt.Errorf("agent %q is not supported\n\nChoose:\n  claude\n  codex", args[1])
			}
			if err := p.EnsureProfile(args[0]); err != nil {
				return err
			}
			executable, err := execution.FindReal(ag.BinaryName(), p.BinDir, os.Getenv("PATH"))
			if err != nil {
				return err
			}
			fmt.Fprintln(a.out, a.accent(fmt.Sprintf("Signing in to %s with profile %q...", ag.DisplayName(), args[0])))
			fmt.Fprintln(a.out)
			fmt.Fprintf(a.out, "The official %s sign-in flow will open.\n", ag.DisplayName())
			fmt.Fprintln(a.out, "AgentWho will keep this sign-in separate from your other profiles.")
			return execution.Replace(executable, ag.LoginArgs(), ag.Environment(args[0], p, os.Environ()))
		},
	}
}
