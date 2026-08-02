package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentForwardsArguments(t *testing.T) {
	got := Content("/path with space/agentwho", "claude")
	if !strings.Contains(got, Marker) || !strings.Contains(got, `internal exec claude "$@"`) || !strings.Contains(got, `'/path with space/agentwho'`) {
		t.Fatal(got)
	}
}

func TestInstallAndRemove(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir, "/usr/local/bin/agentwho", []string{"claude", "codex"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"claude", "codex"} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o755 || !IsManaged(path) {
			t.Fatalf("bad shim %s", path)
		}
	}
	if err := Install(dir, "/new/agentwho", []string{"claude"}); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, []string{"claude", "codex"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "claude")); !os.IsNotExist(err) {
		t.Fatal("shim remains")
	}
}

func TestProtectUnmanagedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte("real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, "/agentwho", []string{"claude"}); err == nil {
		t.Fatal("overwrote unmanaged file")
	}
	if err := Remove(dir, []string{"claude"}); err == nil {
		t.Fatal("removed unmanaged file")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "real" {
		t.Fatal("unmanaged file changed")
	}
}

func TestInstallPreflightAvoidsPartialUpdate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte("unmanaged"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, "/agentwho", []string{"claude", "codex"}); err == nil {
		t.Fatal("expected unmanaged conflict")
	}
	if _, err := os.Stat(filepath.Join(dir, "claude")); !os.IsNotExist(err) {
		t.Fatal("preflight failure left a partial Claude shim")
	}
}
