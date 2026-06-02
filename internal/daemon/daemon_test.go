package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gustavoguarda/zen-sync/internal/sync"
)

// TestRun_PushesOnChange spins up a Run goroutine watching a fake profile,
// modifies a tracked file, and verifies the sync_dir picks up the change
// within debounce + slack.
func TestRun_PushesOnChange(t *testing.T) {
	profile := t.TempDir()
	syncDir := t.TempDir()
	file := "zen-sessions.jsonlz4"

	// Seed the file so fsnotify has something to watch.
	target := filepath.Join(profile, file)
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		Profile:    profile,
		SyncDir:    syncDir,
		Files:      []string{file},
		Hostname:   "test-host",
		DebounceMs: 100,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- Run(ctx, cfg, t.Logf) }()

	// Wait for the watcher to register, then change the file.
	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(target, []byte("v2-CHANGED"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Poll up to 2s for the sync copy to appear with the new bytes.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(filepath.Join(syncDir, file))
		if err == nil && string(b) == "v2-CHANGED" {
			goto check_host
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("sync copy not updated within 2s")

check_host:
	host, err := sync.ReadLastPushHost(syncDir)
	if err != nil {
		t.Fatal(err)
	}
	if host != "test-host" {
		t.Errorf("last-push-host = %q, want %q", host, "test-host")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("Run returned unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Run did not exit after cancel")
	}
}

// TestRun_SkipsRedundantWrite verifies hash-check: a write that doesn't
// change content should NOT bump last-push-host's mtime.
func TestRun_SkipsRedundantWrite(t *testing.T) {
	profile := t.TempDir()
	syncDir := t.TempDir()
	file := "zen-sessions.jsonlz4"
	target := filepath.Join(profile, file)
	os.WriteFile(target, []byte("same"), 0o644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := Config{
		Profile: profile, SyncDir: syncDir,
		Files: []string{file}, Hostname: "h",
		DebounceMs: 50,
	}
	go Run(ctx, cfg, t.Logf)
	time.Sleep(150 * time.Millisecond)

	// First write -> push
	os.WriteFile(target, []byte("v1"), 0o644)
	time.Sleep(300 * time.Millisecond)
	host1Info, _ := os.Stat(filepath.Join(syncDir, ".meta", "last-push-host"))
	if host1Info == nil {
		t.Fatal("expected last-push-host after first change")
	}
	firstMtime := host1Info.ModTime()

	// Touch with same content -> hash same -> NO push
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(target, []byte("v1"), 0o644)
	time.Sleep(300 * time.Millisecond)
	host2Info, _ := os.Stat(filepath.Join(syncDir, ".meta", "last-push-host"))
	if !host2Info.ModTime().Equal(firstMtime) {
		t.Errorf("last-push-host mtime changed despite identical content")
	}
}
