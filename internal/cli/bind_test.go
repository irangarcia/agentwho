package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/gitctx"
)

func TestChooseBindingSingleDirectoryExplainsAndConfirms(t *testing.T) {
	var output bytes.Buffer
	a := &app{in: strings.NewReader("y\n"), out: &output}

	choice, err := a.chooseBinding(gitctx.Context{Directory: "/tmp/projects"}, "work")
	if err != nil {
		t.Fatal(err)
	}
	if choice != 3 {
		t.Fatalf("got choice %d, want directory choice 3", choice)
	}
	for _, want := range []string{
		"No Git repository was found", "Directory: /tmp/projects",
		"everything below it", "Create this directory binding? [y/N]",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output missing %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "Use ↑/↓") {
		t.Fatalf("single choice unexpectedly used an arrow menu:\n%s", output.String())
	}
}

func TestChooseBindingSingleDirectoryCanBeCancelled(t *testing.T) {
	var output bytes.Buffer
	a := &app{in: strings.NewReader("n\n"), out: &output}

	choice, err := a.chooseBinding(gitctx.Context{Directory: "/tmp/projects"}, "work")
	if err != nil || choice != 0 {
		t.Fatalf("got choice %d and error %v, want a clean cancellation", choice, err)
	}
	if !strings.Contains(output.String(), "No changes made.") {
		t.Fatalf("missing cancellation confirmation:\n%s", output.String())
	}
}

func TestChooseBindingRepositoryWithoutOriginExplainsLimitation(t *testing.T) {
	var output bytes.Buffer
	a := &app{in: strings.NewReader("n\n"), out: &output}

	_, _ = a.chooseBinding(gitctx.Context{Directory: "/tmp/project", GitRoot: "/tmp/project"}, "work")
	if !strings.Contains(output.String(), "does not have a usable `origin` remote") {
		t.Fatalf("missing origin explanation:\n%s", output.String())
	}
}

func TestBindingAppliesHere(t *testing.T) {
	ctx := gitctx.Context{
		Directory: "/code/acme/project", Remote: "github.com/acme/project",
		Organization: "github.com/acme",
	}
	tests := []struct {
		name string
		rule config.Rule
		want bool
	}{
		{"repository", config.Rule{Match: config.Match{GitRemote: "github.com/acme/project"}}, true},
		{"other repository", config.Rule{Match: config.Match{GitRemote: "github.com/acme/other"}}, false},
		{"organization", config.Rule{Match: config.Match{GitOrganization: "github.com/acme"}}, true},
		{"path", config.Rule{Match: config.Match{Path: "/code/acme"}}, true},
		{"path boundary", config.Rule{Match: config.Match{Path: "/code/ac"}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := bindingAppliesHere(test.rule, ctx); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
