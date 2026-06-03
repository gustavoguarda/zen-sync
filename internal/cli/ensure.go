package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/launcher"
	"github.com/gustavoguarda/zen-sync/internal/plist"
)

// EnsureInstallation refreshes the .app bundle and LaunchAgent plist if
// the current binary's commit differs from the one that last wrote them.
// Quiet no-op if there's no installation to refresh (config missing).
//
// Wired into the Homebrew formula's post_install hook so `brew upgrade
// zen-sync` lands the fresh icon, plist tweaks, etc. without the user
// having to remember to run anything.
func EnsureInstallation(w io.Writer) error {
	home, _ := os.UserHomeDir()
	cfgPath := filepath.Join(home, ".config", "zen-sync", "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// Never been init'd — nothing to refresh. Stay quiet so brew's
		// post_install doesn't print noise on fresh installs.
		return nil
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate own binary: %w", err)
	}
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	logDir := filepath.Join(home, "Library", "Logs", "zen-sync")

	if changed, err := launcher.Ensure(
		"/Applications/Zen Sync.app", binPath,
		launcherBundleID, "Zen Sync",
		BuildVersion, BuildCommit,
	); err != nil {
		fmt.Fprintf(w, "  ⚠ launcher: %v\n", err)
	} else if changed {
		fmt.Fprintln(w, "✓ refreshed /Applications/Zen Sync.app")
	}

	if changed, err := plist.Ensure(
		launchAgentLabel, binPath, plistDir, logDir,
		BuildVersion, BuildCommit,
	); err != nil {
		fmt.Fprintf(w, "  ⚠ plist: %v\n", err)
	} else if changed {
		fmt.Fprintln(w, "✓ refreshed LaunchAgent")
	}

	return nil
}
