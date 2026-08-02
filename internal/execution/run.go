package execution

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/irangarcia/agentwho/internal/agent"
	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/enforce"
	"github.com/irangarcia/agentwho/internal/gitctx"
	"github.com/irangarcia/agentwho/internal/paths"
	"github.com/irangarcia/agentwho/internal/resolve"
	"golang.org/x/term"
)

type Request struct {
	AgentName string
	Args      []string
	Paths     paths.Paths
	Stdin     *os.File
	Stderr    *os.File
	Getenv    func(string) string
	Environ   func() []string
	Getwd     func() (string, error)
	Runner    gitctx.Runner
	Replace   func(string, []string, []string) error
}

func Run(ctx context.Context, r Request) error {
	a, ok := agent.Get(r.AgentName)
	if !ok {
		return fmt.Errorf("agent %q is not supported\n\nChoose:\n  claude\n  codex", r.AgentName)
	}
	cfg, err := config.Load(r.Paths.Config)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("AgentWho is not set up yet\n\nRun:\n  agentwho init")
		}
		return fmt.Errorf("AgentWho could not read its configuration at %s:\n  %w", r.Paths.Config, err)
	}
	dir, err := r.Getwd()
	if err != nil {
		return fmt.Errorf("current directory: %w", err)
	}
	gctx := gitctx.Detect(ctx, dir, r.Runner)
	resolution := resolve.Resolve(cfg, gctx)
	selected, explicit := resolution.Expected, false
	if value := r.Getenv("AGENTWHO_PROFILE"); value != "" {
		selected, explicit = value, true
	}
	if err := config.ValidateProfileName(selected); err != nil {
		return fmt.Errorf("AGENTWHO_PROFILE: %w", err)
	}
	if _, ok := cfg.Profiles[selected]; !ok {
		return fmt.Errorf("current profile %q does not exist\n\nCreate it with:\n  agentwho profile add %s --kind personal|work", selected, selected)
	}
	interactive := interactiveInput(r.Stdin)
	selected, err = enforce.Check(enforce.Input{
		Expected: resolution.Expected, Selected: selected, Enforcement: resolution.Enforcement,
		AgentDisplayName: a.DisplayName(), Context: gctx, Profiles: cfg.Profiles,
		Interactive: interactive, Force: r.Getenv("AGENTWHO_FORCE") == "1", ExplicitProfile: explicit,
	}, r.Stdin, r.Stderr)
	if err != nil {
		return err
	}
	if err := r.Paths.EnsureProfile(selected); err != nil {
		return err
	}
	executable, err := FindReal(a.BinaryName(), r.Paths.BinDir, r.Getenv("PATH"))
	if err != nil {
		return err
	}
	env := a.Environment(selected, r.Paths, executionEnvironment(r.Environ(), selected))
	return r.Replace(executable, r.Args, env)
}

func interactiveInput(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}

func executionEnvironment(base []string, selected string) []string {
	out := make([]string, 0, len(base)+1)
	for _, item := range base {
		if strings.HasPrefix(item, "AGENTWHO_PROFILE=") || strings.HasPrefix(item, "AGENTWHO_FORCE=") {
			continue
		}
		out = append(out, item)
	}
	return append(out, "AGENTWHO_PROFILE="+selected)
}

func DefaultRequest(name string, args []string, p paths.Paths) Request {
	return Request{AgentName: name, Args: args, Paths: p, Stdin: os.Stdin, Stderr: os.Stderr,
		Getenv: os.Getenv, Environ: os.Environ, Getwd: os.Getwd, Runner: gitctx.ExecRunner{}, Replace: Replace}
}
