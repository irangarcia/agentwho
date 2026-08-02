package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func FindReal(binary, shimDir, pathValue string) (string, error) {
	canonicalShim := canonical(shimDir)
	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			dir = "."
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if samePath(canonical(abs), canonicalShim) {
			continue
		}
		candidate := filepath.Join(abs, binary)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		resolved := canonical(candidate)
		managed := canonical(filepath.Join(shimDir, binary))
		if samePath(resolved, managed) {
			continue
		}
		return candidate, nil
	}
	display := "Codex"
	if binary == "claude" {
		display = "Claude Code"
	}
	return "", fmt.Errorf("the official %s CLI could not be found\n\nAgentWho searched PATH after excluding its own command directory:\n  %s\n\nInstall %s, then run:\n  agentwho doctor", display, shimDir, display)
}

func canonical(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	path = filepath.Clean(path)
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		path = evaluated
	}
	return path
}

func samePath(a, b string) bool {
	if runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
