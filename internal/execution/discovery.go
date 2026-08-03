package execution

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const shimHeaderLimit = 4096

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
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		resolved := canonical(candidate)
		managed := canonical(filepath.Join(shimDir, binary))
		if samePath(resolved, managed) {
			continue
		}
		// Skip routing commands left by AgentWho or its former AgentCtx name.
		// Otherwise two managed shims in PATH can repeatedly launch each other
		// instead of ever reaching the official agent CLI.
		if isManagedRoutingShim(candidate) {
			continue
		}
		return candidate, nil
	}
	display := "Codex"
	if binary == "claude" {
		display = "Claude Code"
	}
	return "", fmt.Errorf("the official %s CLI could not be found\n\nAgentWho searched PATH after excluding managed command shims, including:\n  %s\n\nInstall %s, then run:\n  agentwho doctor", display, shimDir, display)
}

// IsActive reports whether the first executable command with binary's name in
// PATH is the AgentWho-managed command in shimDir.
func IsActive(binary, shimDir, pathValue string) bool {
	managed := canonical(filepath.Join(shimDir, binary))
	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			dir = "."
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		candidate := filepath.Join(abs, binary)
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return samePath(canonical(candidate), managed)
	}
	return false
}

func isManagedRoutingShim(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	header, err := io.ReadAll(io.LimitReader(f, shimHeaderLimit))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(header), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# agentwho-managed-shim-") ||
			strings.HasPrefix(line, "# agentctx-managed-shim-") {
			return true
		}
	}
	return false
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
