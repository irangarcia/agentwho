package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestShellUseSelectsProfileAndRestoresAutomatic(t *testing.T) {
	setupCurrentTest(t)
	t.Setenv("AGENTWHO_PROFILE", "")

	var code, messages bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &code, errout: &messages}
	cmd := a.shellUseCmd()
	cmd.SetArgs([]string{"zsh", "work"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := code.String(); got != "export AGENTWHO_PROFILE='work'\n" {
		t.Fatalf("unexpected shell code: %q", got)
	}
	if !strings.Contains(messages.String(), `Using profile "work" in this shell`) || !strings.Contains(messages.String(), "Safety mode") {
		t.Fatalf("selection message is incomplete: %s", messages.String())
	}

	code.Reset()
	messages.Reset()
	a = &app{in: strings.NewReader(""), out: &code, errout: &messages}
	cmd = a.shellUseCmd()
	cmd.SetArgs([]string{"zsh", "--auto"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := code.String(); got != "unset AGENTWHO_PROFILE\n" {
		t.Fatalf("unexpected automatic shell code: %q", got)
	}
	if !strings.Contains(messages.String(), "Automatic profile selection restored") || !strings.Contains(messages.String(), "personal") {
		t.Fatalf("automatic message is incomplete: %s", messages.String())
	}
}

func TestShellUseRejectsUnknownProfile(t *testing.T) {
	setupCurrentTest(t)
	a := &app{in: strings.NewReader(""), out: &bytes.Buffer{}, errout: &bytes.Buffer{}}
	cmd := a.shellUseCmd()
	cmd.SetArgs([]string{"zsh", "missing"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `profile "missing" does not exist`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublicUseExplainsShellIntegration(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	a := &app{in: strings.NewReader(""), out: &bytes.Buffer{}, errout: &bytes.Buffer{}}
	cmd := a.useCmd()
	cmd.SetArgs([]string{"work"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "needs AgentWho shell integration") || !strings.Contains(err.Error(), "agentwho shell init zsh") {
		t.Fatalf("unexpected error: %v", err)
	}
}
