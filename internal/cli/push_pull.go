package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/profile"
	"github.com/gustavoguarda/zen-sync/internal/sync"
)

func Push(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print actions without performing them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if profile.Running(cfg.ZenRunningPattern) {
		return fmt.Errorf("Zen is running — close it before manual push (daemon handles live writes)")
	}
	host, _ := os.Hostname()
	for _, f := range cfg.Files {
		if *dryRun {
			fmt.Fprintf(w, "+ would push %s\n", f)
			continue
		}
		if err := sync.Push(cfg.ZenProfile, cfg.SyncDir, f); err != nil {
			fmt.Fprintf(w, "  ⚠ push %s: %v\n", f, err)
		}
	}
	if !*dryRun {
		_ = sync.WriteLastPushHost(cfg.SyncDir, host)
		fmt.Fprintf(w, "✓ pushed (host=%s)\n", host)
	}
	return nil
}

func Pull(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print actions without performing them")
	force := fs.Bool("force", false, "pull even if this host was the last to push")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if profile.Running(cfg.ZenRunningPattern) {
		return fmt.Errorf("Zen is running — close it before manual pull")
	}
	host, _ := os.Hostname()
	last, _ := sync.ReadLastPushHost(cfg.SyncDir)
	if !*force && last == host {
		fmt.Fprintln(w, "Last push was from this host — skip (use --force to override).")
		return nil
	}
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".local", "state", "zen-sync", "backups")
	for _, f := range cfg.Files {
		if *dryRun {
			fmt.Fprintf(w, "+ would pull %s\n", f)
			continue
		}
		if err := sync.Pull(cfg.ZenProfile, cfg.SyncDir, backupDir, f); err != nil {
			fmt.Fprintf(w, "  ⚠ pull %s: %v\n", f, err)
		}
		if err := sync.RotateBackups(backupDir, f, cfg.Backup.Keep); err != nil {
			fmt.Fprintf(w, "  ⚠ rotate %s: %v\n", f, err)
		}
	}
	if !*dryRun {
		fmt.Fprintln(w, "✓ pulled")
	}
	return nil
}
