package enforce

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/gitctx"
	"github.com/irangarcia/agentwho/internal/termstyle"
	"github.com/irangarcia/agentwho/internal/tui"
)

type Input struct {
	Expected         string
	Selected         string
	Enforcement      string
	AgentDisplayName string
	Context          gitctx.Context
	Profiles         map[string]config.Profile
	Interactive      bool
	Force            bool
	ExplicitProfile  bool
}

var ErrRefused = fmt.Errorf("profile mismatch: execution refused")

func Check(in Input, input io.Reader, errout io.Writer) (string, error) {
	if in.Expected == in.Selected {
		return in.Selected, nil
	}
	reader := bufio.NewReader(input)
	if in.Force {
		printMismatch(in, errout, "Safety override requested")
		if in.Interactive {
			fmt.Fprintf(errout, "\nType %q to continue: ", in.Selected)
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(answer) != in.Selected {
				fmt.Fprintln(errout, termstyle.Paint(errout, termstyle.Danger, fmt.Sprintf("Override cancelled. %s was not started.", in.AgentDisplayName)))
				return "", ErrRefused
			}
			fmt.Fprintln(errout, termstyle.Paint(errout, termstyle.Warning, fmt.Sprintf("⚠ Safety override accepted. Continuing with profile %q.", in.Selected)))
			return in.Selected, nil
		}
		if in.ExplicitProfile {
			fmt.Fprintln(errout, "\n"+termstyle.Paint(errout, termstyle.Warning, fmt.Sprintf("⚠ Safety override accepted in a non-interactive session. Continuing with profile %q.", in.Selected)))
			return in.Selected, nil
		}
		fmt.Fprintln(errout, "\nA non-interactive override requires both:")
		fmt.Fprintln(errout, "  AGENTWHO_PROFILE=<profile>")
		fmt.Fprintln(errout, "  AGENTWHO_FORCE=1")
		return "", ErrRefused
	}
	if in.Enforcement == "block" {
		printMismatch(in, errout, in.AgentDisplayName+" was not started")
		fmt.Fprintln(errout, "\n"+termstyle.Paint(errout, termstyle.Warning, "Safety mode: block")+"\n\n"+termstyle.Paint(errout, termstyle.Danger, "Execution blocked."))
		return "", ErrRefused
	}
	if !in.Interactive {
		printMismatch(in, errout, in.AgentDisplayName+" was not started")
		fmt.Fprintln(errout, "\nConfirmation is required, but no interactive terminal is available.")
		fmt.Fprintln(errout, "\nTo override explicitly, set both:")
		fmt.Fprintln(errout, "  AGENTWHO_PROFILE=<profile>")
		fmt.Fprintln(errout, "  AGENTWHO_FORCE=1")
		return "", ErrRefused
	}
	printMismatch(in, errout, in.AgentDisplayName+" profile mismatch")
	inputFile, _ := input.(*os.File)
	decision, err := tui.SelectOne(reader, inputFile, errout, "What would you like to do?", []tui.Option{
		{
			Label:       fmt.Sprintf("Switch to profile %q (recommended)", in.Expected),
			Description: "Use the profile expected for this directory.",
			Value:       "use_expected",
		},
		{
			Label:       fmt.Sprintf("Continue with profile %q", in.Selected),
			Description: "Ignore this binding for this command.",
			Value:       "use_selected",
		},
		{
			Label:       "Cancel",
			Description: fmt.Sprintf("Do not start %s.", in.AgentDisplayName),
			Value:       "cancel",
		},
	}, 0)
	if errors.Is(err, tui.ErrCancelled) {
		fmt.Fprintln(errout, termstyle.Paint(errout, termstyle.Warning, fmt.Sprintf("Cancelled. %s was not started.", in.AgentDisplayName)))
		return "", ErrRefused
	}
	if err != nil {
		return "", err
	}
	switch decision {
	case "use_expected":
		fmt.Fprintln(errout, "\n"+termstyle.Paint(errout, termstyle.Success, fmt.Sprintf("Using profile %q for this command.", in.Expected)))
		return in.Expected, nil
	case "use_selected":
		fmt.Fprintln(errout, "\n"+termstyle.Paint(errout, termstyle.Warning, fmt.Sprintf("Continuing with profile %q for this command.", in.Selected)))
		return in.Selected, nil
	default:
		fmt.Fprintln(errout, "\n"+termstyle.Paint(errout, termstyle.Warning, fmt.Sprintf("Cancelled. %s was not started.", in.AgentDisplayName)))
		return "", ErrRefused
	}
}

func printMismatch(in Input, w io.Writer, heading string) {
	fmt.Fprintln(w, termstyle.Paint(w, termstyle.Warning, heading))
	fmt.Fprintln(w)
	if in.Context.Remote != "" {
		fmt.Fprintf(w, "Repository:        %s\n", in.Context.Remote)
	} else {
		fmt.Fprintf(w, "Directory:         %s\n", in.Context.Directory)
	}
	fmt.Fprintln(w, termstyle.Paint(w, termstyle.Success, "Expected profile:  "+in.Expected))
	fmt.Fprintln(w, termstyle.Paint(w, termstyle.Danger, "Current profile:   "+in.Selected))
	fmt.Fprintln(w, "\n"+termstyle.Paint(w, termstyle.Warning, "Risk:"))
	expected, eok := in.Profiles[in.Expected]
	selected, sok := in.Profiles[in.Selected]
	switch {
	case eok && sok && expected.Kind == "work" && selected.Kind == "personal":
		fmt.Fprintln(w, termstyle.Paint(w, termstyle.Danger, "Company source code could be sent through your personal account."))
	case eok && sok && expected.Kind == "personal" && selected.Kind == "work":
		fmt.Fprintln(w, termstyle.Paint(w, termstyle.Danger, "Personal source code could be exposed to a company-managed account."))
	default:
		fmt.Fprintln(w, termstyle.Paint(w, termstyle.Danger, "This directory is bound to a different profile."))
	}
}
