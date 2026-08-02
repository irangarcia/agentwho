package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ok   bool
	}{
		{"personal", true}, {"client-acme2", true}, {"a", true},
		{"", false}, {"Work", false}, {"with space", false}, {"../work", false},
		{"work/client", false}, {"work_client", false}, {"work;rm", false}, {"-work", false}, {"work-", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileName(tt.name)
			if (err == nil) != tt.ok {
				t.Fatalf("ValidateProfileName(%q) error=%v", tt.name, err)
			}
		})
	}
}

func TestParseStrictAndValidate(t *testing.T) {
	t.Parallel()
	valid := `version: 1
defaults: {profile: personal, enforcement: confirm}
profiles: {personal: {kind: personal}}
rules: []
`
	if _, err := Parse(strings.NewReader(valid)); err != nil {
		t.Fatal(err)
	}
	tests := []struct{ name, replace, replacement, want string }{
		{"unknown field", "rules: []", "unknown: true", "field unknown not found"},
		{"version", "version: 1", "version: 2", "version"},
		{"kind", "kind: personal", "kind: client", "kind"},
		{"default", "profile: personal", "profile: missing", "defaults.profile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(strings.Replace(valid, tt.replace, tt.replacement, 1)))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestDuplicateAndInvalidRules(t *testing.T) {
	c := Default()
	c.Rules = []Rule{
		{Match: Match{GitRemote: "github.com/a/b"}, Profile: "personal", Enforcement: "block"},
		{Match: Match{GitRemote: "github.com/a/b"}, Profile: "personal", Enforcement: "confirm"},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "duplicates") {
		t.Fatalf("got %v", err)
	}
	c.Rules = []Rule{{Match: Match{Path: "relative"}, Profile: "personal", Enforcement: "block"}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("got %v", err)
	}
}

func TestAtomicSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")
	c := Default()
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode=%o", got)
	}
	c.Defaults.Enforcement = "block"
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Defaults.Enforcement != "block" {
		t.Fatal("atomic replacement did not persist")
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".config.yaml.tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
	b, _ := Marshal(c)
	if bytes.Contains(b, []byte("rules: null")) {
		t.Fatal("unstable null rules")
	}
}

func TestParseExpandsHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	yaml := `version: 1
defaults: {profile: personal, enforcement: confirm}
profiles: {personal: {kind: personal}}
rules:
  - match: {path: ~/agentwho-test-path}
    profile: personal
    enforcement: block
`
	c, err := Parse(strings.NewReader(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Rules[0].Match.Path, filepath.Join(home, "agentwho-test-path"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
