package execution

import (
	"os"
	"path/filepath"
	"testing"
)

func executable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFindReal(t *testing.T) {
	root := t.TempDir()
	shimDir := filepath.Join(root, "shims")
	real1 := filepath.Join(root, "real1")
	real2 := filepath.Join(root, "real2")
	executable(t, filepath.Join(shimDir, "claude"))
	executable(t, filepath.Join(real1, "claude"))
	executable(t, filepath.Join(real2, "claude"))
	got, err := FindReal("claude", shimDir, filepath.Join(shimDir)+string(os.PathListSeparator)+real1+string(os.PathListSeparator)+real2)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(real1, "claude") {
		t.Fatalf("got %s", got)
	}
}

func TestFindRealExcludesAliasesAndMissing(t *testing.T) {
	root := t.TempDir()
	shimDir := filepath.Join(root, "shims")
	executable(t, filepath.Join(shimDir, "codex"))
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(shimDir, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := FindReal("codex", shimDir, alias); err == nil {
		t.Fatal("symlinked shim directory caused recursion")
	}
	nonexec := filepath.Join(root, "notexec")
	if err := os.Mkdir(nonexec, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonexec, "codex"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := FindReal("codex", shimDir, nonexec); err == nil {
		t.Fatal("non-executable should not be discovered")
	}
}
