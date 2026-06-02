package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/launcher"
	"github.com/gustavoguarda/zen-sync/internal/plist"
)

func Uninstall(w io.Writer) error {
	home, _ := os.UserHomeDir()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := plist.Uninstall(launchAgentLabel, plistDir); err != nil {
		fmt.Fprintf(w, "  ⚠ remove LaunchAgent: %v\n", err)
	} else {
		fmt.Fprintln(w, "✓ removed LaunchAgent")
	}
	if err := launcher.Uninstall("/Applications/Zen Sync.app"); err != nil {
		fmt.Fprintf(w, "  ⚠ remove launcher app: %v\n", err)
	} else {
		fmt.Fprintln(w, "✓ removed /Applications/Zen Sync.app")
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Kept (delete manually if you want them gone):")
	fmt.Fprintf(w, "  config:  %s\n", filepath.Join(home, ".config", "zen-sync"))
	fmt.Fprintf(w, "  logs:    %s\n", filepath.Join(home, "Library", "Logs", "zen-sync"))
	fmt.Fprintf(w, "  backups: %s\n", filepath.Join(home, ".local", "state", "zen-sync"))
	fmt.Fprintln(w, "  sync_dir (whatever you configured): kept")
	return nil
}
