package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const Version = 1

type Config struct {
	Version  int                `yaml:"version" json:"version"`
	Defaults Defaults           `yaml:"defaults" json:"defaults"`
	Profiles map[string]Profile `yaml:"profiles" json:"profiles"`
	Rules    []Rule             `yaml:"rules,omitempty" json:"rules"`
}

type Defaults struct {
	Profile     string `yaml:"profile" json:"profile"`
	Enforcement string `yaml:"enforcement" json:"enforcement"`
}

type Profile struct {
	Kind string `yaml:"kind" json:"kind"`
}

type Rule struct {
	Match       Match  `yaml:"match" json:"match"`
	Profile     string `yaml:"profile" json:"profile"`
	Enforcement string `yaml:"enforcement" json:"enforcement"`
}

type Match struct {
	GitRemote       string `yaml:"git_remote,omitempty" json:"git_remote,omitempty"`
	GitOrganization string `yaml:"git_organization,omitempty" json:"git_organization,omitempty"`
	Path            string `yaml:"path,omitempty" json:"path,omitempty"`
}

func Default() Config {
	return Config{
		Version:  Version,
		Defaults: Defaults{Profile: "personal", Enforcement: "confirm"},
		Profiles: map[string]Profile{"personal": {Kind: "personal"}},
		Rules:    []Rule{},
	}
}

var slug = regexp.MustCompile(`^[a-z0-9][a-z0-9]*(?:-[a-z0-9]+)*$`)

func ValidateProfileName(name string) error {
	if !slug.MatchString(name) || len(name) > 63 || strings.Contains(name, "..") {
		return fmt.Errorf("invalid profile name %q\n\nUse lowercase letters, numbers, and hyphens.\nExamples: personal, work, client-acme", name)
	}
	return nil
}

func validKind(k string) bool        { return k == "personal" || k == "work" }
func validEnforcement(e string) bool { return e == "block" || e == "confirm" }

func (c Config) Validate() error {
	if c.Version != Version {
		return fmt.Errorf("version: expected %d, got %d", Version, c.Version)
	}
	if len(c.Profiles) == 0 {
		return errors.New("profiles: at least one profile is required")
	}
	for name, p := range c.Profiles {
		if err := ValidateProfileName(name); err != nil {
			return fmt.Errorf("profiles.%s: %w", name, err)
		}
		if !validKind(p.Kind) {
			return fmt.Errorf("profiles.%s.kind: must be personal or work", name)
		}
	}
	if _, ok := c.Profiles[c.Defaults.Profile]; !ok {
		return fmt.Errorf("defaults.profile: unknown profile %q", c.Defaults.Profile)
	}
	if !validEnforcement(c.Defaults.Enforcement) {
		return errors.New("defaults.enforcement: must be block or confirm")
	}
	seen := map[string]int{}
	for i, r := range c.Rules {
		prefix := fmt.Sprintf("rules[%d]", i)
		count := 0
		kind, value := "", ""
		if r.Match.GitRemote != "" {
			count++
			kind, value = "git_remote", r.Match.GitRemote
		}
		if r.Match.GitOrganization != "" {
			count++
			kind, value = "git_organization", r.Match.GitOrganization
		}
		if r.Match.Path != "" {
			count++
			kind, value = "path", r.Match.Path
		}
		if count != 1 {
			return fmt.Errorf("%s.match: exactly one of git_remote, git_organization, or path is required", prefix)
		}
		if !filepath.IsAbs(value) && kind == "path" {
			return fmt.Errorf("%s.match.path: must be an absolute normalized path", prefix)
		}
		if _, ok := c.Profiles[r.Profile]; !ok {
			return fmt.Errorf("%s.profile: unknown profile %q", prefix, r.Profile)
		}
		if !validEnforcement(r.Enforcement) {
			return fmt.Errorf("%s.enforcement: must be block or confirm", prefix)
		}
		key := kind + "\x00" + value
		if previous, ok := seen[key]; ok {
			return fmt.Errorf("%s.match: duplicates rules[%d]", prefix, previous)
		}
		seen[key] = i
	}
	return nil
}

func Parse(r io.Reader) (Config, error) {
	var c Config
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("parse configuration: multiple YAML documents are not allowed")
		}
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	if err := normalizeLoadedPaths(&c); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return c, nil
}

func normalizeLoadedPaths(c *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve path rules: find home directory: %w", err)
	}
	for i := range c.Rules {
		value := c.Rules[i].Match.Path
		if value == "" {
			continue
		}
		if value == "~" {
			value = home
		} else if strings.HasPrefix(value, "~/") {
			value = filepath.Join(home, value[2:])
		}
		if !filepath.IsAbs(value) {
			return fmt.Errorf("rules[%d].match.path: must be absolute or start with ~/", i)
		}
		value = filepath.Clean(value)
		if evaluated, err := filepath.EvalSymlinks(value); err == nil {
			value = evaluated
		}
		c.Rules[i].Match.Path = value
	}
	return nil
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	c, parseErr := Parse(f)
	closeErr := f.Close()
	if parseErr != nil {
		return Config{}, parseErr
	}
	if closeErr != nil {
		return Config{}, fmt.Errorf("close configuration: %w", closeErr)
	}
	return c, nil
}

func Marshal(c Config) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return nil, fmt.Errorf("encode configuration: %w", err)
	}
	return b.Bytes(), nil
}

func Save(path string, c Config) error {
	b, err := Marshal(c)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure configuration directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".config.yaml.tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fmt.Errorf("secure temporary configuration: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync temporary configuration: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace configuration atomically: %w", err)
	}
	ok = true
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func (m Match) TypeValue() (string, string) {
	switch {
	case m.GitRemote != "":
		return "git_remote", m.GitRemote
	case m.GitOrganization != "":
		return "git_organization", m.GitOrganization
	default:
		return "path", m.Path
	}
}
