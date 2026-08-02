package paths

import (
	"os"
	"testing"
)

func TestEnsurePermissions(t *testing.T) {
	p := FromHomes(t.TempDir(), t.TempDir())
	if err := p.EnsureBase(); err != nil {
		t.Fatal(err)
	}
	if err := p.EnsureProfile("work"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{p.ConfigDir, p.DataDir, p.BinDir, p.Profiles, p.Profile("work", "claude"), p.Profile("work", "codex")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("%s mode %o", path, info.Mode().Perm())
		}
	}
}
