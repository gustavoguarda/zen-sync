package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gustavoguarda/zen-sync/internal/profile"
)

func Restore(w io.Writer, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".local", "state", "zen-sync", "backups")

	if len(args) == 0 {
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			return fmt.Errorf("no backups dir at %s", backupDir)
		}
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		fmt.Fprintln(w, "Backups (most recent first):")
		for _, n := range names {
			fmt.Fprintf(w, "  %s\n", n)
		}
		fmt.Fprintln(w, "\nUsage: zen-sync restore <name>")
		return nil
	}

	if profile.Running(cfg.ZenRunningPattern) {
		return fmt.Errorf("zen is running — close it before restore")
	}

	name := args[0]
	src := filepath.Join(backupDir, name)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup not found: %s", name)
	}

	// Derive original filename: strip ".<timestamp>"
	original := name
	if i := strings.LastIndex(name, "."); i > 0 {
		original = name[:i]
	}
	allowed := false
	for _, f := range cfg.Files {
		if f == original {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("backup name does not map to a tracked file (got %q)", original)
	}

	dst := filepath.Join(cfg.ZenProfile, original)
	// safety snapshot of current local
	if _, err := os.Stat(dst); err == nil {
		ts := name[strings.LastIndex(name, ".")+1:]
		safety := filepath.Join(backupDir, fmt.Sprintf("%s.before-restore.%s", original, ts))
		_ = copyTo(dst, safety)
	}
	if err := copyTo(src, dst); err != nil {
		return err
	}
	fmt.Fprintf(w, "✓ restored %s from %s\n", original, name)
	return nil
}

func copyTo(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
