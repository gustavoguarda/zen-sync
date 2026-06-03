// Package plist generates and installs the LaunchAgent plist that runs
// `zen-sync daemon` at user login.
//
// Install rewrites + reloads via launchctl. Ensure does the same but only
// when the build-time commit recorded in the file's marker comment differs
// from the supplied one — letting `brew upgrade` silently refresh the
// LaunchAgent next time the user opens Zen Sync.app.
package plist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// commitRe extracts the build-time commit hash from the marker comment we
// embed at the top of generated plists. Same pattern as launcher.
var commitRe = regexp.MustCompile(`IOZenSyncCommit:(\S+)`)

// Render returns the XML body of the LaunchAgent plist, including a marker
// comment with the build version + commit so Ensure can detect drift.
func Render(label, binaryPath, stdoutPath, stderrPath, version, commit string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!-- IOZenSyncCommit:%s IOZenSyncVersion:%s -->
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>

    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>ProcessType</key>
    <string>Background</string>

    <key>StandardOutPath</key>
    <string>%s</string>

    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, commit, version, label, binaryPath, stdoutPath, stderrPath)
}

// WriteFile creates parent dirs and writes body to path with 0644.
func WriteFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// Install writes the plist to ~/Library/LaunchAgents/<label>.plist and
// loads it via `launchctl load`. The unload+load pair is idempotent so
// re-running this on an existing install refreshes the daemon's config
// without rebooting.
func Install(label, binaryPath, plistDir, logDir, version, commit string) (string, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	stdout := filepath.Join(logDir, "launchd.out")
	stderr := filepath.Join(logDir, "launchd.err")
	body := Render(label, binaryPath, stdout, stderr, version, commit)
	plistPath := filepath.Join(plistDir, label+".plist")
	if err := WriteFile(plistPath, body); err != nil {
		return "", err
	}
	// Try to unload first (idempotent), then load.
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return plistPath, fmt.Errorf("launchctl load: %w", err)
	}
	return plistPath, nil
}

// Ensure regenerates the plist if the embedded commit differs from the
// supplied one (or the file doesn't exist yet). Returns whether anything
// changed. When changed=true, Install was called, which also reloads
// launchctl so the daemon picks up the new plist immediately.
func Ensure(label, binaryPath, plistDir, logDir, version, commit string) (bool, error) {
	plistPath := filepath.Join(plistDir, label+".plist")
	if body, err := os.ReadFile(plistPath); err == nil {
		if m := commitRe.FindStringSubmatch(string(body)); len(m) == 2 && m[1] == commit {
			return false, nil
		}
	}
	if _, err := Install(label, binaryPath, plistDir, logDir, version, commit); err != nil {
		return false, err
	}
	return true, nil
}

// Uninstall unloads the agent and removes the plist file.
func Uninstall(label, plistDir string) error {
	plistPath := filepath.Join(plistDir, label+".plist")
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
