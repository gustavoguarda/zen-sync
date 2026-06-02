// Package plist generates and installs the LaunchAgent plist that runs
// `zen-sync daemon` at user login.
//
// Install/Uninstall shell out to launchctl. The Render function is pure
// string generation and is unit-testable; Install/Uninstall require a
// real launchd and are exercised via manual verification.
package plist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Render returns the XML body of the LaunchAgent plist.
func Render(label, binaryPath, stdoutPath, stderrPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
`, label, binaryPath, stdoutPath, stderrPath)
}

// WriteFile creates parent dirs and writes body to path with 0644.
func WriteFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// Install writes the plist to ~/Library/LaunchAgents/<label>.plist and
// loads it via `launchctl load`.
func Install(label, binaryPath, plistDir, logDir string) (string, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	stdout := filepath.Join(logDir, "launchd.out")
	stderr := filepath.Join(logDir, "launchd.err")
	body := Render(label, binaryPath, stdout, stderr)
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

// Uninstall unloads the agent and removes the plist file.
func Uninstall(label, plistDir string) error {
	plistPath := filepath.Join(plistDir, label+".plist")
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
