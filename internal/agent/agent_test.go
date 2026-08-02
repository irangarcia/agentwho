package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/irangarcia/agentwho/internal/paths"
)

func TestProfileEnvironment(t *testing.T) {
	p := paths.FromHomes("/config", "/data")
	tests := []struct{ name, key, want string }{{"claude", "CLAUDE_CONFIG_DIR", filepath.Join("/data", "agentwho", "profiles", "work", "claude")}, {"codex", "CODEX_HOME", filepath.Join("/data", "agentwho", "profiles", "work", "codex")}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := Get(tt.name)
			env := a.Environment("work", p, []string{"PATH=/bin", tt.key + "=/wrong", "KEEP=yes"})
			found := 0
			for _, e := range env {
				if strings.HasPrefix(e, tt.key+"=") {
					found++
					if e != tt.key+"="+tt.want {
						t.Fatalf("got %s", e)
					}
				}
			}
			if found != 1 {
				t.Fatalf("found %d entries: %v", found, env)
			}
			if !contains(env, "KEEP=yes") {
				t.Fatal("base environment not preserved")
			}
		})
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
