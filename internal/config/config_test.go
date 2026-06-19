package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestValidateRejectsNonPositive(t *testing.T) {
	base := Default()
	mutate := []func(*Config){
		func(c *Config) { c.PollIntervalMS = 0 },
		func(c *Config) { c.RetentionDays = -1 },
		func(c *Config) { c.DisplayLimit = 0 },
		func(c *Config) { c.MaxClipBytes = -10 },
	}
	for i, m := range mutate {
		cfg := base
		m(&cfg)
		if err := cfg.Validate(); err == nil {
			t.Errorf("mutation %d: expected validation error", i)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := Config{PollIntervalMS: 250, RetentionDays: 30, DisplayLimit: 10, MaxClipBytes: 2048}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestLoadMissingKeysUseDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.toml")
	if err := os.WriteFile(path, []byte("display_limit = 5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DisplayLimit != 5 {
		t.Errorf("DisplayLimit = %d, want 5", got.DisplayLimit)
	}
	def := Default()
	if got.PollIntervalMS != def.PollIntervalMS || got.MaxClipBytes != def.MaxClipBytes {
		t.Errorf("missing keys not defaulted: %+v", got)
	}
}

func TestLoadRejectsInvalidFileContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("poll_interval_ms = -1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load accepted invalid config")
	}
}
