package execution

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFindRealSkipsManagedRoutingShims(t *testing.T) {
	root := t.TempDir()
	shimDir := filepath.Join(root, "agentwho")
	legacyDir := filepath.Join(root, "agentctx")
	realDir := filepath.Join(root, "real")
	executable(t, filepath.Join(shimDir, "codex"))
	if err := os.WriteFile(filepath.Join(shimDir, "codex"), []byte("#!/bin/sh\n# agentwho-managed-shim-v1\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	executable(t, filepath.Join(legacyDir, "codex"))
	if err := os.WriteFile(filepath.Join(legacyDir, "codex"), []byte("#!/bin/sh\n# agentctx-managed-shim-v1\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	executable(t, filepath.Join(realDir, "codex"))

	pathValue := strings.Join([]string{shimDir, legacyDir, realDir}, string(os.PathListSeparator))
	got, err := FindReal("codex", shimDir, pathValue)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(realDir, "codex"); got != want {
		t.Fatalf("FindReal() = %q, want %q", got, want)
	}

	if _, err := FindReal("codex", shimDir, strings.Join([]string{shimDir, legacyDir}, string(os.PathListSeparator))); err == nil {
		t.Fatal("FindReal() accepted a managed routing shim as the real Codex binary")
	}
}

func TestIsActive(t *testing.T) {
	root := t.TempDir()
	shimDir := filepath.Join(root, "shims")
	otherDir := filepath.Join(root, "other")
	executable(t, filepath.Join(shimDir, "claude"))
	executable(t, filepath.Join(otherDir, "claude"))

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "unrelated directories may come first", path: strings.Join([]string{root, shimDir, otherDir}, string(os.PathListSeparator)), want: true},
		{name: "official command comes first", path: strings.Join([]string{otherDir, shimDir}, string(os.PathListSeparator)), want: false},
		{name: "command is missing", path: root, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsActive("claude", shimDir, tt.path); got != tt.want {
				t.Fatalf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
