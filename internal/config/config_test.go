package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_AppliesDefaultsAndExpandsTilde(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	body := `
sync_dir = "~/BrowserSync/Zen"
zen_profile = "~/Library/Application Support/zen/Profiles/abc.Default (release)"
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	c, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	home, _ := os.UserHomeDir()

	if c.SyncDir != filepath.Join(home, "BrowserSync", "Zen") {
		t.Errorf("SyncDir not expanded: %q", c.SyncDir)
	}
	if !strings.HasPrefix(c.ZenProfile, home) {
		t.Errorf("ZenProfile not expanded: %q", c.ZenProfile)
	}

	// Defaults applied
	want := []string{"zen-sessions.jsonlz4", "zen-live-folders.jsonlz4", "containers.json"}
	if len(c.Files) != len(want) {
		t.Fatalf("Files default not applied: got %v", c.Files)
	}
	for i, f := range want {
		if c.Files[i] != f {
			t.Errorf("Files[%d] = %q, want %q", i, c.Files[i], f)
		}
	}
	if c.Daemon.DebounceMs != 1000 {
		t.Errorf("Daemon.DebounceMs default = %d, want 1000", c.Daemon.DebounceMs)
	}
	if c.Backup.Keep != 5 {
		t.Errorf("Backup.Keep default = %d, want 5", c.Backup.Keep)
	}
}

func TestLoad_ErrorsOnMissingSyncDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing sync_dir")
	}
	if !strings.Contains(err.Error(), "sync_dir") {
		t.Errorf("error should mention sync_dir; got: %v", err)
	}
}

func TestSave_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "out.toml")
	c := &Config{
		SyncDir:    "/tmp/sync",
		ZenProfile: "/tmp/profile",
		Files:      []string{"a", "b"},
	}
	c.Daemon.DebounceMs = 500
	c.Backup.Keep = 3
	if err := Save(c, cfgPath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SyncDir != c.SyncDir || got.ZenProfile != c.ZenProfile {
		t.Errorf("round-trip mismatch: %+v vs %+v", got, c)
	}
}
