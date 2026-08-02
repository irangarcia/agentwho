package gitctx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRemote(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"git@github.com:acme/backend.git":               "github.com/acme/backend",
		"https://github.com/acme/backend.git":           "github.com/acme/backend",
		"ssh://git@github.com/acme/backend.git":         "github.com/acme/backend",
		"ssh://git@git.example.test:2222/team/repo.git": "git.example.test:2222/team/repo",
		"git@gitlab.com:group/subgroup/repo.git":        "gitlab.com/group/subgroup/repo",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := NormalizeRemote(input)
			if err != nil || got != want {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		})
	}
	for _, bad := range []string{"", "not-a-remote", "https://github.com"} {
		if _, err := NormalizeRemote(bad); err == nil {
			t.Errorf("expected %q to fail", bad)
		}
	}
}

func TestHostAndOrganization(t *testing.T) {
	remote := "git.example.test:2222/team/sub/repo"
	if got := Host(remote); got != "git.example.test:2222" {
		t.Fatal(got)
	}
	if got := Organization(remote); got != "git.example.test:2222/team" {
		t.Fatal(got)
	}
}

func TestExpandSymlinkAndBoundary(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := ExpandPath("~/link", root, "/")
	if err != nil {
		t.Fatal(err)
	}
	real, _ = filepath.EvalSymlinks(real)
	if got != real {
		t.Fatalf("got %s want %s", got, real)
	}
	child := filepath.Join(real, "child")
	if !PathContains(real, child) {
		t.Fatal("child should match")
	}
	if PathContains(real, real+"-other") {
		t.Fatal("boundary prefix must not match")
	}
	if !PathContains(link, child) {
		t.Fatal("symlink should resolve consistently")
	}
}
