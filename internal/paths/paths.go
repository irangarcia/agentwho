package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigDir string
	Config    string
	DataDir   string
	BinDir    string
	Profiles  string
}

func Discover() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("find home directory: %w", err)
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	return FromHomes(configHome, dataHome), nil
}

func FromHomes(configHome, dataHome string) Paths {
	c := filepath.Join(configHome, "agentwho")
	d := filepath.Join(dataHome, "agentwho")
	return Paths{
		ConfigDir: c,
		Config:    filepath.Join(c, "config.yaml"),
		DataDir:   d,
		BinDir:    filepath.Join(d, "bin"),
		Profiles:  filepath.Join(d, "profiles"),
	}
}

func (p Paths) Profile(name, agent string) string {
	return filepath.Join(p.Profiles, name, agent)
}

func (p Paths) EnsureBase() error {
	for _, dir := range []string{p.ConfigDir, p.DataDir, p.BinDir, p.Profiles} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure %s: %w", dir, err)
		}
	}
	return nil
}

func (p Paths) EnsureProfile(name string) error {
	for _, agent := range []string{"claude", "codex"} {
		dir := p.Profile(name, agent)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create profile directory %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure profile directory %s: %w", dir, err)
		}
	}
	return nil
}
