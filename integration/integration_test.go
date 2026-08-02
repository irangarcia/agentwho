package integration

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/config"
	"github.com/irangarcia/agentwho/internal/paths"
)

func TestTransparentShimFlow(t *testing.T) {
	root := moduleRoot(t)
	temp := t.TempDir()
	bin := filepath.Join(temp, "agentwho")
	build := exec.Command("go", "build", "-o", bin, "./cmd/agentwho")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	configHome := filepath.Join(temp, "config")
	dataHome := filepath.Join(temp, "data")
	emptyPromptEnv := []string{"XDG_CONFIG_HOME=" + configHome, "XDG_DATA_HOME=" + dataHome}
	if output := run(t, temp, emptyPromptEnv, bin, "prompt"); len(output) != 0 {
		t.Fatalf("uninitialized prompt produced output: %q", output)
	}
	if got := outputJSON(t, temp, emptyPromptEnv, bin, "prompt", "--json"); got["initialized"] != false {
		t.Fatalf("uninitialized prompt JSON: %v", got)
	}
	p := paths.FromHomes(configHome, dataHome)
	if err := p.EnsureBase(); err != nil {
		t.Fatal(err)
	}
	c := config.Default()
	c.Profiles["work"] = config.Profile{Kind: "work"}
	for name := range c.Profiles {
		if err := p.EnsureProfile(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Save(p.Config, c); err != nil {
		t.Fatal(err)
	}

	repo := filepath.Join(temp, "repo")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	run(t, repo, nil, "git", "init", "-q")
	run(t, repo, nil, "git", "remote", "add", "origin", "git@github.com:acme/backend.git")

	realBin := filepath.Join(temp, "real-bin")
	if err := os.Mkdir(realBin, 0o700); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(realBin, "claude")
	script := "#!/bin/sh\nprintf '%s\\n' \"$CLAUDE_CONFIG_DIR\" > \"$CAPTURE_DIR/env\"\nprintf '%s\\n' \"$@\" > \"$CAPTURE_DIR/args\"\nexit 23\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	basePath := os.Getenv("PATH")
	env := []string{"XDG_CONFIG_HOME=" + configHome, "XDG_DATA_HOME=" + dataHome, "PATH=" + realBin + string(os.PathListSeparator) + basePath}
	bindOutput := run(t, repo, env, bin, "bind", "work", "--repo", "--safety-mode", "block")
	if !strings.Contains(string(bindOutput), "Current profile here: work") || !strings.Contains(string(bindOutput), "Safety mode:  block") {
		t.Fatalf("bind success did not explain the effective profile and safety mode:\n%s", bindOutput)
	}
	priorityRepo := filepath.Join(temp, "priority-repo")
	if err := os.Mkdir(priorityRepo, 0o700); err != nil {
		t.Fatal(err)
	}
	run(t, priorityRepo, nil, "git", "init", "-q")
	run(t, priorityRepo, nil, "git", "remote", "add", "origin", "git@github.com:acme/priority.git")
	run(t, priorityRepo, env, bin, "bind", "work", "--repo", "--safety-mode", "confirm")
	priorityOutput := run(t, priorityRepo, env, bin, "bind", "work", "--organization", "--safety-mode", "block")
	if !strings.Contains(string(priorityOutput), "more specific binding still controls") ||
		!strings.Contains(string(priorityOutput), "Safety mode:      confirm") ||
		!strings.Contains(string(priorityOutput), "agentwho unbind") {
		t.Fatalf("bind success did not explain rule precedence:\n%s", priorityOutput)
	}
	personalRepo := filepath.Join(temp, "personal-repo")
	if err := os.Mkdir(personalRepo, 0o700); err != nil {
		t.Fatal(err)
	}
	run(t, personalRepo, nil, "git", "init", "-q")
	run(t, personalRepo, nil, "git", "remote", "add", "origin", "git@github.com:example/personal.git")
	overrideEnv := append(append([]string(nil), env...), "AGENTWHO_PROFILE=work")
	overrideOutput := run(t, personalRepo, overrideEnv, bin, "bind", "personal", "--repo", "--safety-mode", "confirm")
	if !strings.Contains(string(overrideOutput), "current profile is still \"work\"") || !strings.Contains(string(overrideOutput), "agentwho use --auto") {
		t.Fatalf("bind success did not explain the active override:\n%s", overrideOutput)
	}
	run(t, repo, env, bin, "install", "--shell", "zsh")

	shellEnv := append(append([]string(nil), env...), "PATH="+temp+string(os.PathListSeparator)+realBin+string(os.PathListSeparator)+basePath)
	shellSelection := exec.Command("zsh", "-c", `eval "$(agentwho shell init zsh)"
agentwho use personal
agentwho current
agentwho use --auto
agentwho current`)
	shellSelection.Dir = repo
	shellSelection.Env = append(os.Environ(), shellEnv...)
	shellOutput, err := shellSelection.CombinedOutput()
	if err != nil {
		t.Fatalf("shell-local profile selection failed: %v\n%s", err, shellOutput)
	}
	for _, want := range []string{
		`Using profile "personal" in this shell`,
		"This directory expects profile \"work\"",
		"personal\n",
		"Automatic profile selection restored",
		"work\n",
	} {
		if !strings.Contains(string(shellOutput), want) {
			t.Fatalf("shell-local selection output is missing %q:\n%s", want, shellOutput)
		}
	}

	capture := filepath.Join(temp, "capture")
	if err := os.Mkdir(capture, 0o700); err != nil {
		t.Fatal(err)
	}
	shimPath := filepath.Join(p.BinDir, "claude")
	shimEnv := append(env, "CAPTURE_DIR="+capture, "PATH="+p.BinDir+string(os.PathListSeparator)+realBin+string(os.PathListSeparator)+basePath)
	t.Setenv("PATH", p.BinDir+string(os.PathListSeparator)+realBin+string(os.PathListSeparator)+basePath)
	cmd := exec.Command("claude", "--model", "test model", "$NOT_EXPANDED")
	cmd.Dir, cmd.Env = repo, append(os.Environ(), shimEnv...)
	err = cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("real child exit code not propagated: %v", err)
	}
	envValue, _ := os.ReadFile(filepath.Join(capture, "env"))
	if got, want := strings.TrimSpace(string(envValue)), p.Profile("work", "claude"); got != want {
		t.Fatalf("CLAUDE_CONFIG_DIR=%q want %q", got, want)
	}
	argsValue, _ := os.ReadFile(filepath.Join(capture, "args"))
	if got, want := string(argsValue), "--model\ntest model\n$NOT_EXPANDED\n"; got != want {
		t.Fatalf("arguments changed: %q", got)
	}

	if err := os.Remove(filepath.Join(capture, "args")); err != nil {
		t.Fatal(err)
	}
	mismatch := exec.Command("claude", "must-not-run")
	mismatch.Dir = repo
	mismatch.Env = append(os.Environ(), append(shimEnv, "AGENTWHO_PROFILE=personal")...)
	output, err := mismatch.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Execution blocked") {
		t.Fatalf("mismatch was not blocked: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "Error:") || strings.Contains(string(output), "profile mismatch: execution refused") {
		t.Fatalf("blocked mismatch printed a redundant trailing error:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(capture, "args")); !os.IsNotExist(err) {
		t.Fatal("real Claude ran during blocked mismatch")
	}

	status := outputJSON(t, repo, env, bin, "status", "--json")
	for _, key := range []string{"directory", "expected_profile", "current_profile", "safety_mode", "selected_profile", "enforcement", "claude_shim_installed", "automatic_selection_active", "status"} {
		if _, ok := status[key]; !ok {
			t.Fatalf("status JSON missing stable key %q: %v", key, status)
		}
	}
	prompt := outputJSON(t, repo, append(env, "AGENTWHO_PROFILE=personal"), bin, "prompt", "--json")
	if prompt["text"] != "[agent:personal!]" || prompt["mismatch"] != true {
		t.Fatalf("unexpected prompt JSON: %v", prompt)
	}

	run(t, repo, env, bin, "uninstall")
	if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
		t.Fatal("uninstall retained managed shim")
	}
	if _, err := os.Stat(p.Config); err != nil {
		t.Fatal("uninstall removed configuration")
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(file))
}

func run(t *testing.T, dir string, env []string, name string, args ...string) []byte {
	t.Helper()
	c := exec.Command(name, args...)
	c.Dir = dir
	if env != nil {
		c.Env = append(os.Environ(), env...)
	}
	output, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return output
}

func outputJSON(t *testing.T, dir string, env []string, name string, args ...string) map[string]any {
	t.Helper()
	output := run(t, dir, env, name, args...)
	var value map[string]any
	if err := json.Unmarshal(output, &value); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	return value
}
