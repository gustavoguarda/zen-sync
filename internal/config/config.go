// Package config defines the on-disk configuration format and applies
// defaults + path expansion.
//
// The on-disk format is TOML at ~/.config/zen-sync/config.toml. Paths that
// begin with "~/" are expanded relative to $HOME at load time.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the parsed and validated configuration.
type Config struct {
	SyncDir           string   `toml:"sync_dir"`
	ZenProfile        string   `toml:"zen_profile"`
	ZenRunningPattern string   `toml:"zen_running_pattern"`
	Files             []string `toml:"files"`

	Daemon struct {
		DebounceMs int    `toml:"debounce_ms"`
		LogLevel   string `toml:"log_level"`
	} `toml:"daemon"`

	Backup struct {
		Keep int `toml:"keep"`
	} `toml:"backup"`
}

// Default returns a Config populated with default values for all optional
// fields. SyncDir is left empty — it must be provided by the user.
func Default() *Config {
	c := &Config{
		ZenRunningPattern: "/Zen.app/Contents/MacOS/",
		Files: []string{
			"zen-sessions.jsonlz4",
			"zen-live-folders.jsonlz4",
			"containers.json",
		},
	}
	c.Daemon.DebounceMs = 1000
	c.Daemon.LogLevel = "info"
	c.Backup.Keep = 5
	return c
}

// Load reads the TOML file at path, applies defaults to missing fields,
// expands "~/" paths against $HOME, and validates required fields.
func Load(path string) (*Config, error) {
	c := Default()
	if _, err := toml.DecodeFile(path, c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := expand(c); err != nil {
		return nil, err
	}
	if err := validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Save writes c to path as TOML. Parent dirs are created if missing.
func Save(c *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

func expand(c *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	c.SyncDir = expandTilde(c.SyncDir, home)
	c.ZenProfile = expandTilde(c.ZenProfile, home)
	return nil
}

func expandTilde(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

func validate(c *Config) error {
	if strings.TrimSpace(c.SyncDir) == "" {
		return errors.New("config: sync_dir is required")
	}
	if c.Daemon.DebounceMs <= 0 {
		return fmt.Errorf("config: daemon.debounce_ms must be > 0 (got %d)", c.Daemon.DebounceMs)
	}
	if c.Backup.Keep <= 0 {
		return fmt.Errorf("config: backup.keep must be > 0 (got %d)", c.Backup.Keep)
	}
	if len(c.Files) == 0 {
		return errors.New("config: files must not be empty")
	}
	return nil
}
