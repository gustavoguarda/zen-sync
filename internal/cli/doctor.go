package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/profile"
)

type check struct {
	name string
	fn   func() (string, error)
}

func Doctor(w io.Writer) error {
	cfg, cfgErr := loadConfig()

	checks := []check{
		{"config readable", func() (string, error) {
			if cfgErr != nil {
				return "", cfgErr
			}
			return "ok", nil
		}},
	}

	if cfg != nil {
		checks = append(checks,
			check{"sync_dir exists + writable", func() (string, error) {
				if err := os.MkdirAll(cfg.SyncDir, 0o700); err != nil {
					return "", err
				}
				probe := filepath.Join(cfg.SyncDir, ".zen-sync-probe")
				if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
					return "", err
				}
				_ = os.Remove(probe)
				return cfg.SyncDir, nil
			}},
			check{"zen_profile exists", func() (string, error) {
				if _, err := os.Stat(cfg.ZenProfile); err != nil {
					return "", err
				}
				return cfg.ZenProfile, nil
			}},
			check{"daemon running (LaunchAgent loaded)", func() (string, error) {
				if err := exec.Command("launchctl", "list", launchAgentLabel).Run(); err != nil {
					return "", fmt.Errorf("LaunchAgent %s not loaded", launchAgentLabel)
				}
				return "loaded", nil
			}},
			check{"Zen running?", func() (string, error) {
				if profile.Running(cfg.ZenRunningPattern) {
					return "yes (manual push/pull will be blocked)", nil
				}
				return "no", nil
			}},
			check{"launcher app installed", func() (string, error) {
				p := "/Applications/Zen Sync.app/Contents/MacOS/launcher"
				if _, err := os.Stat(p); err != nil {
					return "", err
				}
				return "/Applications/Zen Sync.app", nil
			}},
		)
	}

	failed := 0
	for _, c := range checks {
		val, err := c.fn()
		if err != nil {
			fmt.Fprintf(w, "✗ %s: %v\n", c.name, err)
			failed++
		} else {
			fmt.Fprintf(w, "✓ %s: %s\n", c.name, val)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d check(s) failed", failed)
	}
	return nil
}
