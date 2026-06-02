// Package daemon watches the configured profile files via fsnotify and
// pushes changes to sync_dir after a debounce window.
//
// Two safeguards beyond raw fsnotify: (1) debounce so we don't catch a
// partially-flushed file; (2) SHA-256 hash check so we don't push when
// Zen rewrote the file without changing content.
package daemon

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	isync "github.com/gustavoguarda/zen-sync/internal/sync"
)

// Config carries everything Run needs. Decoupled from internal/config to
// keep daemon package testable without a real TOML file.
type Config struct {
	Profile    string
	SyncDir    string
	Files      []string
	Hostname   string
	DebounceMs int
}

// LogFn is a printf-style logger. Caller provides one.
type LogFn func(format string, args ...any)

// Run blocks until ctx is cancelled. It watches the files listed in
// cfg.Files (within cfg.Profile) and pushes them to cfg.SyncDir on real
// changes (debounced, hash-checked).
func Run(ctx context.Context, cfg Config, logf LogFn) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("daemon: new watcher: %w", err)
	}
	defer w.Close()

	// Watch the profile directory itself (fsnotify on macOS doesn't always
	// pick up renames of individual files; watching the parent dir is more
	// reliable).
	if err := w.Add(cfg.Profile); err != nil {
		return fmt.Errorf("daemon: watch %s: %w", cfg.Profile, err)
	}
	tracked := map[string]bool{}
	for _, f := range cfg.Files {
		tracked[f] = true
	}

	debounce := time.Duration(cfg.DebounceMs) * time.Millisecond
	var mu sync.Mutex
	lastHash := map[string]string{}
	pending := map[string]*time.Timer{}

	flush := func(file string) {
		fullPath := filepath.Join(cfg.Profile, file)
		newHash, err := isync.Hash(fullPath)
		if err != nil {
			logf("daemon: hash %s: %v", file, err)
			return
		}
		// Check-and-claim: only one goroutine wins the "I'll push this hash" race.
		mu.Lock()
		if lastHash[file] == newHash {
			mu.Unlock()
			return // hash-check: redundant write
		}
		lastHash[file] = newHash // claim now, even before push, so concurrent flushes see it
		mu.Unlock()

		if err := isync.Push(cfg.Profile, cfg.SyncDir, file); err != nil {
			logf("daemon: push %s: %v", file, err)
			// roll back the claim so a retry can succeed
			mu.Lock()
			delete(lastHash, file)
			mu.Unlock()
			return
		}
		if err := isync.WriteLastPushHost(cfg.SyncDir, cfg.Hostname); err != nil {
			logf("daemon: stamp host: %v", err)
		}
		logf("daemon: pushed %s (hash=%s…)", file, newHash[:8])
	}

	schedule := func(file string) {
		mu.Lock()
		if t, ok := pending[file]; ok {
			t.Stop()
		}
		pending[file] = time.AfterFunc(debounce, func() { flush(file) })
		mu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			for _, t := range pending {
				t.Stop()
			}
			return ctx.Err()
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logf("daemon: watcher error: %v", err)
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			name := filepath.Base(ev.Name)
			if !tracked[name] {
				continue
			}
			// Treat any event on a tracked file as a candidate.
			schedule(name)
		}
	}
}
