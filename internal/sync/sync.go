// Package sync moves files between the Zen profile and a user-provided
// sync_dir. It does not interpret file contents — pure binary copy with
// integrity-by-hash and rotated backups.
package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Hash returns the hex-encoded SHA-256 of the file at path.
func Hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Push copies <profile>/<file> to <syncDir>/<file>. A missing source file
// is treated as a no-op (the user may not yet have that file — e.g. a
// fresh profile without live folders).
func Push(profile, syncDir, file string) error {
	src := filepath.Join(profile, file)
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.MkdirAll(syncDir, 0o755); err != nil {
		return err
	}
	return copyFile(src, filepath.Join(syncDir, file))
}

// Pull copies <syncDir>/<file> back into <profile>/<file>, backing up the
// current local copy into <backupDir> first. A missing sync file is a
// non-destructive no-op (preserves local).
func Pull(profile, syncDir, backupDir, file string) error {
	local := filepath.Join(profile, file)
	remote := filepath.Join(syncDir, file)

	// Backup local if it exists.
	if _, err := os.Stat(local); err == nil {
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return err
		}
		ts := time.Now().Format("20060102T150405")
		if err := copyFile(local, filepath.Join(backupDir, fmt.Sprintf("%s.%s", file, ts))); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Skip if sync doesn't have it.
	if _, err := os.Stat(remote); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	if err := os.MkdirAll(profile, 0o755); err != nil {
		return err
	}
	return copyFile(remote, local)
}

// RotateBackups keeps the `keep` newest entries whose basename starts with
// `pattern + "."` in dir, deleting the rest.
func RotateBackups(dir, pattern string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	type indexed struct {
		path  string
		mtime time.Time
	}
	var matched []indexed
	prefix := pattern + "."
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			return err
		}
		matched = append(matched, indexed{filepath.Join(dir, e.Name()), fi.ModTime()})
	}
	if len(matched) <= keep {
		return nil
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].mtime.After(matched[j].mtime) })
	for _, old := range matched[keep:] {
		if err := os.RemoveAll(old.path); err != nil {
			return err
		}
	}
	return nil
}

// ReadLastPushHost returns the contents of <syncDir>/.meta/last-push-host,
// or "" if the file doesn't exist.
func ReadLastPushHost(syncDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(syncDir, ".meta", "last-push-host"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// WriteLastPushHost stamps <syncDir>/.meta/last-push-host with host.
func WriteLastPushHost(syncDir, host string) error {
	dir := filepath.Join(syncDir, ".meta")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last-push-host"), []byte(host+"\n"), 0o644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// Write to temp then rename for atomicity within the same fs.
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
