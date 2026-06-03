package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/launcher"
	"github.com/gustavoguarda/zen-sync/internal/plist"
	"github.com/gustavoguarda/zen-sync/internal/profile"
	"github.com/gustavoguarda/zen-sync/internal/sync"
)

// Open is the launch path: pull state from sync_dir, then exec `open -W -na Zen`.
//
// Skips the pull (1) if Zen is already running — no point swapping files
// that Zen has cached in memory — or (2) if this host was the last to push.
// Extra args are forwarded to Zen (URLs, file paths).
//
// Also auto-heals the .app bundle and LaunchAgent plist if the current
// binary's commit differs from the one that wrote them. This is what makes
// `brew upgrade zen-sync` Just Work: the next click on Zen Sync.app
// silently refreshes everything before launching Zen.
func Open(w io.Writer, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Auto-heal first — a stale .app/plist shouldn't block Zen from
	// opening, so we log errors but don't return them.
	if err := autoHeal(w); err != nil {
		fmt.Fprintf(w, "  ⚠ refresh: %v\n", err)
	}

	if profile.Running(cfg.ZenRunningPattern) {
		fmt.Fprintln(w, "Zen is already running — bringing to front.")
		return exec.Command("open", "-a", "Zen").Run()
	}

	host, _ := os.Hostname()
	last, _ := sync.ReadLastPushHost(cfg.SyncDir)
	if last == host {
		fmt.Fprintln(w, "Local state is already current (last push from this host). Skipping pull.")
	} else {
		home, _ := os.UserHomeDir()
		backupDir := filepath.Join(home, ".local", "state", "zen-sync", "backups")
		for _, f := range cfg.Files {
			if err := sync.Pull(cfg.ZenProfile, cfg.SyncDir, backupDir, f); err != nil {
				fmt.Fprintf(w, "  ⚠ pull %s: %v\n", f, err)
			}
			if err := sync.RotateBackups(backupDir, f, cfg.Backup.Keep); err != nil {
				fmt.Fprintf(w, "  ⚠ rotate %s: %v\n", f, err)
			}
		}
		fmt.Fprintln(w, "Pulled state from sync_dir.")
	}

	// `-W` deliberately omitted: the daemon handles continuous push while
	// Zen runs, so we don't need to block here waiting for Zen to quit.
	// `-n` opens a new instance (harmless if one already exists).
	openArgs := append([]string{"-na", "Zen"}, args...)
	return exec.Command("open", openArgs...).Run()
}

// autoHeal regenerates the .app bundle and LaunchAgent plist if the
// installed commit markers don't match this binary's. No-op when both are
// current. Prints a brief confirmation for each thing that changed.
func autoHeal(w io.Writer) error {
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	logDir := filepath.Join(home, "Library", "Logs", "zen-sync")

	if changed, err := launcher.Ensure(
		"/Applications/Zen Sync.app", binPath,
		launcherBundleID, "Zen Sync",
		BuildVersion, BuildCommit,
	); err != nil {
		fmt.Fprintf(w, "  ⚠ launcher refresh: %v\n", err)
	} else if changed {
		fmt.Fprintln(w, "✓ refreshed /Applications/Zen Sync.app")
	}

	if changed, err := plist.Ensure(
		launchAgentLabel, binPath, plistDir, logDir,
		BuildVersion, BuildCommit,
	); err != nil {
		fmt.Fprintf(w, "  ⚠ plist refresh: %v\n", err)
	} else if changed {
		fmt.Fprintln(w, "✓ refreshed LaunchAgent")
	}

	return nil
}
