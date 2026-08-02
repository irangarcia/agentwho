package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func forceColor(t *testing.T) {
	t.Helper()
	old, existed := os.LookupEnv("NO_COLOR")
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("NO_COLOR", old)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	})
	t.Setenv("CLICOLOR_FORCE", "1")
}

func TestStatusColorMatchesSemanticPalette(t *testing.T) {
	setupCurrentTest(t)
	forceColor(t)
	var output bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &output, errout: &output}
	if err := a.statusCmd().Execute(); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{
		"\x1b[1;38;2;210;168;255mAgentWho status",
		"\x1b[1;38;2;86;211;100mExpected profile:  personal",
		"Safety mode:       confirm",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("colored status is missing %q:\n%q", want, text)
		}
	}
}

func TestMachineOutputsNeverContainColor(t *testing.T) {
	setupCurrentTest(t)
	forceColor(t)

	var current bytes.Buffer
	a := &app{in: strings.NewReader(""), out: &current, errout: &current}
	if err := a.currentCmd().Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(current.String(), "\x1b[") || current.String() != "personal\n" {
		t.Fatalf("current output is not stable plain text: %q", current.String())
	}

	var status bytes.Buffer
	a = &app{in: strings.NewReader(""), out: &status, errout: &status}
	cmd := a.statusCmd()
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(status.String(), "\x1b[") {
		t.Fatalf("status JSON contains ANSI color: %q", status.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(status.Bytes(), &decoded); err != nil {
		t.Fatalf("status JSON is invalid: %v\n%s", err, status.String())
	}
}
