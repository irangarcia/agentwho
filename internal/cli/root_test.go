package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/shell"
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
		"version", "help <command>",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("root help is missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "internal exec") {
		t.Fatal("internal command appeared in public help")
	}
}

func TestVersionCommandAndFlag(t *testing.T) {
	previous := Version
	Version = "0.1.0-test"
	t.Cleanup(func() { Version = previous })

	for _, args := range [][]string{{"version"}, {"--version"}} {
		cmd := New()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if got := out.String(); got != "agentwho 0.1.0-test\n" {
			t.Fatalf("%v output = %q", args, got)
		}
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
	if strings.Contains(text, "Shell setup") || strings.Contains(text, "Add this line") {
		t.Fatalf("declining terminal integration showed unnecessary shell setup:\n%s", text)
	}
}

func TestInitOffersBackedUpShellUpdateWhenProtectionEnabled(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(root, "config")
	dataHome := filepath.Join(root, "data")
	shellFile := filepath.Join(root, ".zshrc")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("HOME", root)
	t.Setenv("SHELL", "/bin/zsh")
	if err := os.WriteFile(shellFile, []byte("export KEEP=yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	a := &app{in: strings.NewReader("n\n1\ny\ny\nn\n"), out: &out, errout: &out}
	if err := a.initCmd().Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"Shell setup", "Update " + shellFile + " now? [Y/n]", "Updated " + shellFile,
		"Backup:", "Open a new terminal to activate AgentWho",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("onboarding output is missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "Add this line") {
		t.Fatalf("automatic shell update still showed manual instructions:\n%s", text)
	}
	b, err := os.ReadFile(shellFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "export KEEP=yes") || shell.CountBlocks(string(b)) != 1 {
		t.Fatalf("shell configuration was not preserved and updated:\n%s", b)
	}
	if _, err := os.Stat(filepath.Join(dataHome, "agentwho", "bin", "claude")); err != nil {
		t.Fatalf("Claude command was not installed: %v", err)
	}
}

func TestInitShowsManualShellSetupOnlyWhenAutomaticUpdateIsDeclined(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HOME", root)
	t.Setenv("SHELL", "/bin/zsh")

	var out bytes.Buffer
	a := &app{in: strings.NewReader("n\n1\ny\nn\nn\n"), out: &out, errout: &out}
	if err := a.initCmd().Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"No shell files were changed", "Add this line to " + filepath.Join(root, ".zshrc"),
		`eval "$(agentwho shell init zsh)"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("onboarding output is missing %q\n%s", want, text)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("declining the shell update changed .zshrc: %v", err)
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
	t.Cleanup(func() { _ = file.Close() })
	if interactive(file) {
		t.Fatal("/dev/null was incorrectly treated as an interactive terminal")
	}
}
