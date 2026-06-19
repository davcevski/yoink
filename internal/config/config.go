// Package config loads and persists yoink's TOML configuration, creating it
// with sane defaults on first run.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/davcevski/yoink/internal/paths"
)

// Config holds the user-tunable settings. Field tags map to the TOML keys
// documented in design.md.
type Config struct {
	PollIntervalMS int `toml:"poll_interval_ms"`
	RetentionDays  int `toml:"retention_days"`
	DisplayLimit   int `toml:"display_limit"`
	MaxClipBytes   int `toml:"max_clip_bytes"`
}

// Default returns the built-in configuration used on first run.
func Default() Config {
	return Config{
		PollIntervalMS: 500,
		RetentionDays:  14,
		DisplayLimit:   20,
		MaxClipBytes:   1 << 20, // 1 MiB
	}
}

// Validate reports the first invalid field, if any.
func (c Config) Validate() error {
	switch {
	case c.PollIntervalMS <= 0:
		return errors.New("config: poll_interval_ms must be > 0")
	case c.RetentionDays <= 0:
		return errors.New("config: retention_days must be > 0")
	case c.DisplayLimit <= 0:
		return errors.New("config: display_limit must be > 0")
	case c.MaxClipBytes <= 0:
		return errors.New("config: max_clip_bytes must be > 0")
	}
	return nil
}

// Load reads the config from path. Missing keys fall back to their defaults so
// older config files keep working when new fields are added.
func Load(path string) (Config, error) {
	cfg := Default()
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes cfg to path with owner-only permissions.
func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, paths.FilePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return err
	}
	return f.Close()
}

// LoadOrCreate returns the config at the default path, creating the directory
// and a defaults file on first run. It also returns the resolved path.
func LoadOrCreate() (Config, string, error) {
	if _, err := paths.EnsureConfigDir(); err != nil {
		return Config{}, "", err
	}
	path, err := paths.ConfigFile()
	if err != nil {
		return Config{}, "", err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if err := Save(path, cfg); err != nil {
			return Config{}, "", fmt.Errorf("config: writing defaults: %w", err)
		}
		return cfg, path, nil
	} else if err != nil {
		return Config{}, "", err
	}
	cfg, err := Load(path)
	if err != nil {
		return Config{}, "", err
	}
	return cfg, path, nil
}
