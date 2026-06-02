package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gustavoguarda/zen-sync/internal/config"
	"github.com/gustavoguarda/zen-sync/internal/daemon"
	"github.com/gustavoguarda/zen-sync/internal/logger"
)

// Daemon runs the watcher loop until SIGTERM/SIGINT.
//
// The LaunchAgent invokes `zen-sync daemon`; the user does not normally
// run it directly.
func Daemon(w io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library", "Logs", "zen-sync", "daemon.log")
	lg, closeFn, err := logger.New(logPath)
	if err != nil {
		return fmt.Errorf("daemon: open log: %w", err)
	}
	defer func() { _ = closeFn() }()

	host, _ := os.Hostname()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dcfg := daemon.Config{
		Profile:    cfg.ZenProfile,
		SyncDir:    cfg.SyncDir,
		Files:      cfg.Files,
		Hostname:   host,
		DebounceMs: cfg.Daemon.DebounceMs,
	}
	lg.Printf("daemon: starting (profile=%s sync=%s files=%d)", dcfg.Profile, dcfg.SyncDir, len(dcfg.Files))
	err = daemon.Run(ctx, dcfg, func(f string, a ...any) { lg.Printf(f, a...) })
	if err != nil && err != context.Canceled {
		lg.Printf("daemon: exit error: %v", err)
		return err
	}
	lg.Printf("daemon: shutdown clean")
	return nil
}

// loadConfig is shared by subcommands that need the parsed Config.
// It reads ~/.config/zen-sync/config.toml. If the file is missing the
// returned error is a hint to run `zen-sync init`.
func loadConfig() (*config.Config, error) {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".config", "zen-sync", "config.toml")
	c, err := config.Load(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config at %s — run `zen-sync init`", p)
		}
		return nil, err
	}
	return c, nil
}
