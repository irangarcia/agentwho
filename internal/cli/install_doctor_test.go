package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/shell"
)

func TestInstallRecognizesExistingShellIntegration(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("SHELL", "/bin/zsh")
	configFile := filepath.Join(root, ".zshrc")
	if _, changed, err := shell.AddBlock(configFile, "zsh"); err != nil || !changed {
		t.Fatalf("create shell integration: changed=%v err=%v", changed, err)
	}

	var output bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &output, errout: &output}
	cmd := a.installCmd()
	cmd.SetArgs([]string{"--shell", "zsh"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Shell integration is already configured") {
		t.Fatalf("missing configured message:\n%s", output.String())
	}
	if strings.Contains(output.String(), "Add this line") {
		t.Fatalf("install repeated shell instructions:\n%s", output.String())
	}
}

func TestDoctorDescribesMissingShims(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("SHELL", "/bin/zsh")

	var output bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &output, errout: &output}
	err := a.doctorCmd().Execute()
	if err == nil {
		t.Fatal("doctor unexpectedly succeeded without an installation")
	}
	for _, agentName := range []string{"Claude Code", "Codex"} {
		want := "AgentWho command for " + agentName + " is not installed correctly"
		if !strings.Contains(output.String(), want) {
			t.Errorf("doctor output missing %q:\n%s", want, output.String())
		}
	}
}
