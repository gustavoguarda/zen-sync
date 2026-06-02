package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/profile"
	"github.com/gustavoguarda/zen-sync/internal/sync"
)

// Open is the launch path: pull state from sync_dir, then exec `open -W -na Zen`.
//
// Skips the pull (1) if Zen is already running — no point swapping files
// that Zen has cached in memory — or (2) if this host was the last to push.
// Extra args are forwarded to Zen (URLs, file paths).
func Open(w io.Writer, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
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
