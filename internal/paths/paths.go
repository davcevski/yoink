// Package paths resolves the on-disk locations yoink uses. Centralizing them
// keeps the daemon, TUI, and installer in agreement about where state lives.
package paths

import (
	"os"
	"path/filepath"
)

// Filesystem permissions for yoink-owned state. The config directory is
// owner-only; the database and log may contain sensitive material (ciphertext
// and operational logs) and are owner read/write only.
const (
	DirPerm  os.FileMode = 0o700
	FilePerm os.FileMode = 0o600
)

const (
	appDir     = "yoink"
	configName = "config.toml"
	dbName     = "history.db"
	logName    = "yoinkd.log"
	plistName  = "com.yoink.daemon.plist"
)

// ConfigDir returns ~/.config/yoink, honoring XDG_CONFIG_HOME when set. The
// design pins this path explicitly, so we do not use os.UserConfigDir (which
// resolves to ~/Library/Application Support on macOS).
func ConfigDir() (string, error) {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, appDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", appDir), nil
}

// EnsureConfigDir creates the config directory (0700) if it does not exist and
// returns its path.
func EnsureConfigDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, DirPerm); err != nil {
		return "", err
	}
	return dir, nil
}

// ConfigFile returns ~/.config/yoink/config.toml.
func ConfigFile() (string, error) { return inConfigDir(configName) }

// DBFile returns ~/.config/yoink/history.db.
func DBFile() (string, error) { return inConfigDir(dbName) }

// LogFile returns ~/.config/yoink/yoinkd.log.
func LogFile() (string, error) { return inConfigDir(logName) }

func inConfigDir(name string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// LaunchAgentsDir returns ~/Library/LaunchAgents.
func LaunchAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

// PlistFile returns ~/Library/LaunchAgents/com.yoink.daemon.plist.
func PlistFile() (string, error) {
	dir, err := LaunchAgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, plistName), nil
}
