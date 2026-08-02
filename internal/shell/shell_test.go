package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	for _, name := range []string{"zsh", "bash", "fish"} {
		t.Run(name, func(t *testing.T) {
			got, err := Init(name, "/data/agentwho/bin")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, "/data/agentwho/bin") || !strings.Contains(got, "agentwho_prompt") || !strings.Contains(got, "internal shell-use "+name) {
				t.Fatal(got)
			}
			if name != "fish" && !strings.Contains(got, `case ":$PATH:"`) {
				t.Fatal("missing duplicate-safe PATH check")
			}
		})
	}
	if _, err := Init("powershell", "/bin"); err == nil {
		t.Fatal("unsupported shell")
	}
}

func TestInitShellSyntax(t *testing.T) {
	for _, name := range []string{"zsh", "bash"} {
		t.Run(name, func(t *testing.T) {
			executable, err := exec.LookPath(name)
			if err != nil {
				t.Skipf("%s is not installed", name)
			}
			got, err := Init(name, "/data/agentwho/bin")
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(executable, "-n")
			cmd.Stdin = strings.NewReader(got)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generated %s syntax is invalid: %v\n%s\n%s", name, err, output, got)
			}
		})
	}
}

func TestUseProfileCommands(t *testing.T) {
	tests := []struct {
		shell, profile, wantUse, wantAuto string
	}{
		{"zsh", "work", "export AGENTWHO_PROFILE='work'", "unset AGENTWHO_PROFILE"},
		{"bash", "personal", "export AGENTWHO_PROFILE='personal'", "unset AGENTWHO_PROFILE"},
		{"fish", "client-acme", "set -gx AGENTWHO_PROFILE 'client-acme'", "set -e AGENTWHO_PROFILE"},
	}
	for _, test := range tests {
		t.Run(test.shell, func(t *testing.T) {
			got, err := UseProfile(test.shell, test.profile)
			if err != nil || got != test.wantUse {
				t.Fatalf("UseProfile() = %q, %v; want %q", got, err, test.wantUse)
			}
			got, err = UseAutomatic(test.shell)
			if err != nil || got != test.wantAuto {
				t.Fatalf("UseAutomatic() = %q, %v; want %q", got, err, test.wantAuto)
			}
		})
	}
	if _, err := UseProfile("powershell", "work"); err == nil {
		t.Fatal("unsupported shell profile command succeeded")
	}
	if _, err := UseAutomatic("powershell"); err == nil {
		t.Fatal("unsupported shell automatic command succeeded")
	}
}

func TestPromptSetup(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{"zsh", "setopt PROMPT_SUBST"},
		{"bash", "PS1="},
		{"fish", "fish_prompt"},
	}
	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got, err := PromptSetup(tt.shell)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tt.want) || !strings.Contains(got, "agentwho prompt --plain") {
				t.Fatalf("unexpected setup: %s", got)
			}
		})
	}
	if _, err := PromptSetup("powershell"); err == nil {
		t.Fatal("unsupported shell")
	}
}

func TestShellBlocksReversibleAndUnique(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zshrc")
	original := "export KEEP=yes\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	backup, changed, err := AddBlock(path, "zsh")
	if err != nil || !changed || backup == "" {
		t.Fatalf("%q %v %v", backup, changed, err)
	}
	if _, changed, err := AddBlock(path, "zsh"); err != nil || changed {
		t.Fatalf("duplicate block added: %v %v", changed, err)
	}
	b, _ := os.ReadFile(path)
	if CountBlocks(string(b)) != 1 {
		t.Fatal(string(b))
	}
	backup, changed, err = RemoveBlocks(path)
	if err != nil || !changed || backup == "" {
		t.Fatalf("remove: %q %v %v", backup, changed, err)
	}
	b, _ = os.ReadFile(path)
	if string(b) != original {
		t.Fatalf("got %q want %q", b, original)
	}
}

func TestIsConfigured(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"managed block", Block("zsh"), true},
		{"manual line", "export KEEP=yes\n" + EvalLine("zsh") + "\n", true},
		{"unrelated config", "export KEEP=yes\n", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(test.name, " ", "-"))
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := IsConfigured(path, "zsh")
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
	got, err := IsConfigured(filepath.Join(dir, "missing"), "zsh")
	if err != nil || got {
		t.Fatalf("missing file: got %v, error %v", got, err)
	}
}
