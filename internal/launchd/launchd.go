// Package launchd manages the macOS LaunchAgent that keeps yoinkd running at
// login. It renders the plist, (un)loads it via launchctl, and reports whether
// FileVault — yoink's at-rest "Layer 1" — is enabled.
package launchd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/davcevski/yoink/internal/paths"
)

// Label is the LaunchAgent's reverse-DNS identifier.
const Label = "com.yoink.daemon"

// plistOptions are the values substituted into the plist template.
type plistOptions struct {
	Label      string
	BinaryPath string
	LogPath    string
}

var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.BinaryPath}}</string>
		<string>daemon</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ProcessType</key>
	<string>Background</string>
	<key>StandardOutPath</key>
	<string>{{.LogPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{.LogPath}}</string>
</dict>
</plist>
`))

// RenderPlist returns the LaunchAgent plist for the given binary and log path.
func RenderPlist(binaryPath, logPath string) (string, error) {
	var buf bytes.Buffer
	err := plistTemplate.Execute(&buf, plistOptions{
		Label:      Label,
		BinaryPath: binaryPath,
		LogPath:    logPath,
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Install writes the plist and loads it so the daemon starts now and at every
// login. It is idempotent: an already-loaded agent is reloaded.
func Install(binaryPath, logPath string) error {
	plistPath, err := paths.PlistFile()
	if err != nil {
		return err
	}
	dir, err := paths.LaunchAgentsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	content, err := RenderPlist(binaryPath, logPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(plistPath, []byte(content), 0o600); err != nil {
		return err
	}

	domain := guiDomain()
	// Clear any previous registration so bootstrap does not fail with EEXIST.
	_ = run("launchctl", "bootout", domain, plistPath)
	if out, err := output("launchctl", "bootstrap", domain, plistPath); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w: %s", err, out)
	}
	return nil
}

// Uninstall unloads the agent and removes the plist.
func Uninstall() error {
	plistPath, err := paths.PlistFile()
	if err != nil {
		return err
	}
	_ = run("launchctl", "bootout", guiDomain(), plistPath)
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// FileVaultEnabled reports whether macOS FileVault is on. A false result is
// yoink's cue to warn the user about the missing at-rest baseline.
func FileVaultEnabled() (bool, error) {
	out, err := output("fdesetup", "status")
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "FileVault is On"), nil
}

func guiDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
