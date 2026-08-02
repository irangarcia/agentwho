package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Marker = "# agentwho-managed-shim-v1"

func Content(agentwho, agent string) string {
	return "#!/bin/sh\n" + Marker + "\nexec " + shellQuote(agentwho) + " internal exec " + agent + " \"$@\"\n"
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func Install(binDir, agentwho string, agents []string) error {
	if !filepath.IsAbs(agentwho) {
		return fmt.Errorf("agentwho executable path must be absolute")
	}
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return fmt.Errorf("create shim directory: %w", err)
	}
	if err := os.Chmod(binDir, 0o700); err != nil {
		return fmt.Errorf("secure shim directory: %w", err)
	}
	// Preflight the complete set so a conflicting unmanaged file cannot leave a
	// partially updated installation.
	for _, name := range agents {
		path := filepath.Join(binDir, name)
		if existing, err := os.ReadFile(path); err == nil && !strings.Contains(string(existing), Marker) {
			return fmt.Errorf("AgentWho found an existing file it does not manage:\n\n  %s\n\nIt was not changed. Move or rename that file, then run:\n  agentwho install", path)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("inspect shim %s: %w", path, err)
		}
	}
	for _, name := range agents {
		path := filepath.Join(binDir, name)
		if err := atomicExecutable(path, []byte(Content(agentwho, name))); err != nil {
			return err
		}
	}
	return nil
}

func atomicExecutable(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".shim-*")
	if err != nil {
		return fmt.Errorf("create temporary shim: %w", err)
	}
	tmp := f.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o755); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("install shim %s: %w", path, err)
	}
	keep = true
	return nil
}

func IsManaged(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), Marker)
}

func Remove(binDir string, agents []string) error {
	for _, name := range agents {
		path := filepath.Join(binDir, name)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if !IsManaged(path) {
			return fmt.Errorf("AgentWho found an existing file it does not manage:\n\n  %s\n\nIt was not removed", path)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove shim %s: %w", path, err)
		}
	}
	return nil
}
