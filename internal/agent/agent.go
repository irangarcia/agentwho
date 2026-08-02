package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/irangarcia/agentwho/internal/paths"
)

type AuthStatus string

const (
	Authenticated    AuthStatus = "authenticated"
	NotAuthenticated AuthStatus = "not authenticated"
	Unavailable      AuthStatus = "unavailable"
	Unknown          AuthStatus = "unknown"
)

type Agent interface {
	Name() string
	DisplayName() string
	BinaryName() string
	Environment(profile string, paths paths.Paths, base []string) []string
	LoginArgs() []string
	StatusArgs() []string
}

type adapter struct {
	name, display, env string
	login, status      []string
}

func (a adapter) Name() string         { return a.name }
func (a adapter) DisplayName() string  { return a.display }
func (a adapter) BinaryName() string   { return a.name }
func (a adapter) LoginArgs() []string  { return append([]string(nil), a.login...) }
func (a adapter) StatusArgs() []string { return append([]string(nil), a.status...) }
func (a adapter) Environment(profile string, p paths.Paths, base []string) []string {
	return replaceEnv(base, a.env, p.Profile(profile, a.name))
}

var agents = map[string]Agent{
	"claude": adapter{name: "claude", display: "Claude Code", env: "CLAUDE_CONFIG_DIR", login: []string{"auth", "login"}, status: []string{"auth", "status"}},
	"codex":  adapter{name: "codex", display: "Codex", env: "CODEX_HOME", login: []string{"login"}, status: []string{"login", "status"}},
}

func Get(name string) (Agent, bool) { a, ok := agents[name]; return a, ok }
func All() []Agent                  { return []Agent{agents["claude"], agents["codex"]} }

func replaceEnv(base []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(base)+1)
	for _, item := range base {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}

func Status(parent context.Context, a Agent, profile string, p paths.Paths, executable string) AuthStatus {
	if executable == "" {
		return Unavailable
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, executable, a.StatusArgs()...)
	c.Env = a.Environment(profile, p, os.Environ())
	c.Stdin = nil
	c.Stdout = nil
	c.Stderr = nil
	err := c.Run()
	if err == nil {
		return Authenticated
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Unknown
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return NotAuthenticated
	}
	return Unknown
}
