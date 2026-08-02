package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
)

func TestRootHelpListsEveryPublicCommand(t *testing.T) {
	cmd := New()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"profile add <name>", "profile list", "profile login <profile> <agent>",
		"bind <profile>", "unbind", "rules", "use <profile>", "current", "status", "prompt", "install",
		"uninstall", "doctor", "shell init <shell>", "completion <shell>",
		"help <command>",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("root help is missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "internal exec") {
		t.Fatal("internal command appeared in public help")
	}
}

func TestInitUsesUserFacingCopyAndChoices(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(root, "config")
	dataHome := filepath.Join(root, "data")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("HOME", root)
	t.Setenv("SHELL", "/bin/zsh")

	var out bytes.Buffer
	a := &app{in: strings.NewReader("y\n2\n1\nn\nn\n"), out: &out, errout: &out}
	cmd := a.initCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"Welcome to AgentWho", "Create a separate work profile", "Which profile should be used",
		"How should AgentWho handle a profile mismatch", "Route the `claude` and `codex` terminal commands",
		"VS Code extension panels are", "Enable terminal integration now? [Y/n]",
		"Show the active profile beside your command prompt", "AgentWho is ready",
		"Default profile: work", "Default safety mode: block",
		"Safety mode\n", "Terminal integration\n", "Prompt indicator\n", "Setup complete\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("onboarding output is missing %q\n%s", want, text)
		}
	}
	c, err := config.Load(filepath.Join(configHome, "agentwho", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Defaults.Profile != "work" || c.Defaults.Enforcement != "block" {
		t.Fatalf("choices were not persisted: %+v", c.Defaults)
	}
	if _, ok := c.Profiles["work"]; !ok {
		t.Fatal("work profile was not created")
	}
	if _, err := os.Stat(filepath.Join(dataHome, "agentwho", "bin", "claude")); !os.IsNotExist(err) {
		t.Fatal("declined automatic selection unexpectedly installed a Claude command")
	}
}

func TestCompletionRejectsUnsupportedShell(t *testing.T) {
	cmd := New()
	cmd.SetArgs([]string{"completion", "powershell"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Supported shells") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDevNullIsNotInteractive(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if interactive(file) {
		t.Fatal("/dev/null was incorrectly treated as an interactive terminal")
	}
}
