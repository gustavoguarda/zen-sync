package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/gustavoguarda/zen-sync/internal/sync"
)

func Status(w io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	last, _ := sync.ReadLastPushHost(cfg.SyncDir)

	fmt.Fprintf(w, "host:           %s\n", host)
	fmt.Fprintf(w, "sync_dir:       %s\n", cfg.SyncDir)
	fmt.Fprintf(w, "zen_profile:    %s\n", cfg.ZenProfile)
	fmt.Fprintf(w, "last-push-host: %s\n", last)
	if last == host {
		fmt.Fprintln(w, "  → this Mac was the last to push")
	}
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tLOCAL HASH\tSYNC HASH\tMATCH")
	for _, f := range cfg.Files {
		local, _ := sync.Hash(filepath.Join(cfg.ZenProfile, f))
		remote, _ := sync.Hash(filepath.Join(cfg.SyncDir, f))
		match := "no"
		if local != "" && local == remote {
			match = "yes"
		} else if local == "" && remote == "" {
			match = "n/a"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", f, short(local), short(remote), match)
	}
	return tw.Flush()
}

func short(h string) string {
	if len(h) < 12 {
		return h
	}
	return h[:12] + "…"
}
