package gitctx

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Context struct {
	Directory    string `json:"directory"`
	GitRoot      string `json:"git_root,omitempty"`
	Remote       string `json:"git_remote,omitempty"`
	Host         string `json:"git_host,omitempty"`
	Organization string `json:"git_organization,omitempty"`
}

type Runner interface {
	Output(ctx context.Context, dir string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Output(ctx context.Context, dir string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	return c.Output()
}

func Detect(ctx context.Context, dir string, runner Runner) Context {
	dir = canonical(dir)
	result := Context{Directory: dir}
	root, err := runner.Output(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return result
	}
	result.GitRoot = canonical(strings.TrimSpace(string(root)))
	remote, err := runner.Output(ctx, result.GitRoot, "remote", "get-url", "origin")
	if err != nil {
		return result
	}
	result.Remote, err = NormalizeRemote(strings.TrimSpace(string(remote)))
	if err != nil {
		result.Remote = ""
		return result
	}
	result.Host = Host(result.Remote)
	result.Organization = Organization(result.Remote)
	return result
}

func canonical(path string) string {
	p, err := filepath.Abs(path)
	if err == nil {
		path = p
	}
	path = filepath.Clean(path)
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return path
}

func NormalizeRemote(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty Git remote")
	}
	var host, path string
	// SCP-style SSH syntax: user@host:path.
	if !strings.Contains(s, "://") {
		colon := strings.IndexByte(s, ':')
		if colon > 0 && !strings.Contains(s[:colon], "/") {
			left := s[:colon]
			if at := strings.LastIndexByte(left, '@'); at >= 0 {
				left = left[at+1:]
			}
			host, path = left, s[colon+1:]
		}
	}
	if host == "" {
		u, err := url.Parse(s)
		if err != nil || u.Hostname() == "" {
			return "", fmt.Errorf("invalid Git remote %q", raw)
		}
		host, path = u.Hostname(), u.Path
		if port := u.Port(); port != "" {
			host += ":" + port
		}
	}
	host = strings.ToLower(strings.TrimSpace(host))
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || path == "" || strings.Contains(path, "..") {
		return "", fmt.Errorf("invalid Git remote %q", raw)
	}
	return host + "/" + path, nil
}

func Host(remote string) string {
	if i := strings.IndexByte(remote, '/'); i >= 0 {
		return remote[:i]
	}
	return remote
}

func Organization(remote string) string {
	parts := strings.Split(remote, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

func ExpandPath(value, home, base string) (string, error) {
	if value == "~" {
		value = home
	}
	if strings.HasPrefix(value, "~/") {
		value = filepath.Join(home, value[2:])
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(base, value)
	}
	p, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("normalize path %q: %w", value, err)
	}
	p = filepath.Clean(p)
	if evaluated, err := filepath.EvalSymlinks(p); err == nil {
		p = evaluated
	}
	return p, nil
}

func PathContains(parent, child string) bool {
	parent, child = canonical(parent), canonical(child)
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
