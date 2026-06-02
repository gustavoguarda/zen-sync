package sync

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helper: copy testdata/fake-profile into a fresh dir
func freshProfile(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "fake-profile")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			t.Fatalf("write %s: %v", e.Name(), err)
		}
	}
	return dst
}

func TestHash_DeterministicAndDifferent(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte("hello"), 0o644)
	os.WriteFile(b, []byte("world"), 0o644)

	h1, err := Hash(a)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := Hash(a)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("Hash not deterministic: %s vs %s", h1, h2)
	}
	h3, err := Hash(b)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h3 {
		t.Errorf("different files hashed equal: %s", h1)
	}
}

func TestPush_CopiesBytes(t *testing.T) {
	profile := freshProfile(t)
	sync := t.TempDir()
	if err := Push(profile, sync, "zen-sessions.jsonlz4"); err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(filepath.Join(profile, "zen-sessions.jsonlz4"))
	dst, _ := os.ReadFile(filepath.Join(sync, "zen-sessions.jsonlz4"))
	if !bytes.Equal(src, dst) {
		t.Errorf("push corrupted bytes")
	}
}

func TestPush_MissingFileIsNotAnError(t *testing.T) {
	profile := t.TempDir()
	sync := t.TempDir()
	if err := Push(profile, sync, "absent.jsonlz4"); err != nil {
		t.Errorf("missing file should be skipped, got error: %v", err)
	}
}

func TestPull_BackupsThenCopies(t *testing.T) {
	profile := freshProfile(t)
	sync := t.TempDir()
	backups := t.TempDir()

	// Pre-populate sync with different content
	os.WriteFile(filepath.Join(sync, "zen-sessions.jsonlz4"), []byte("REMOTE-NEW-BYTES"), 0o644)

	if err := Pull(profile, sync, backups, "zen-sessions.jsonlz4"); err != nil {
		t.Fatal(err)
	}

	// Local now matches sync
	got, _ := os.ReadFile(filepath.Join(profile, "zen-sessions.jsonlz4"))
	if string(got) != "REMOTE-NEW-BYTES" {
		t.Errorf("local not updated: %q", string(got))
	}
	// Backup exists with original content
	entries, _ := os.ReadDir(backups)
	if len(entries) == 0 {
		t.Fatal("no backup created")
	}
	backed, _ := os.ReadFile(filepath.Join(backups, entries[0].Name()))
	if !strings.Contains(string(backed), "SESSIONS_FAKE_A") {
		t.Errorf("backup content unexpected: %q", string(backed))
	}
}

func TestPull_NoSyncFileLeavesLocalAlone(t *testing.T) {
	profile := freshProfile(t)
	sync := t.TempDir()
	backups := t.TempDir()

	orig, _ := os.ReadFile(filepath.Join(profile, "zen-sessions.jsonlz4"))
	if err := Pull(profile, sync, backups, "zen-sessions.jsonlz4"); err != nil {
		t.Fatal(err)
	}
	now, _ := os.ReadFile(filepath.Join(profile, "zen-sessions.jsonlz4"))
	if !bytes.Equal(orig, now) {
		t.Errorf("local file changed despite empty sync")
	}
}

func TestRotateBackups_KeepsNewest(t *testing.T) {
	dir := t.TempDir()
	// Create 7 backups with increasing timestamps
	for i, name := range []string{
		"f.jsonlz4.20260101T000001",
		"f.jsonlz4.20260101T000002",
		"f.jsonlz4.20260101T000003",
		"f.jsonlz4.20260101T000004",
		"f.jsonlz4.20260101T000005",
		"f.jsonlz4.20260101T000006",
		"f.jsonlz4.20260101T000007",
	} {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte("x"), 0o644)
		// Stagger mtimes so sort by time is deterministic
		t := time.Now().Add(time.Duration(i) * time.Second)
		os.Chtimes(p, t, t)
	}
	if err := RotateBackups(dir, "f.jsonlz4", 5); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 5 {
		t.Errorf("got %d files after rotate, want 5", len(entries))
	}
	// The two oldest (000001 and 000002) should be gone
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "000001") || strings.HasSuffix(e.Name(), "000002") {
			t.Errorf("oldest backup not pruned: %s", e.Name())
		}
	}
}

func TestHostMarker_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := WriteLastPushHost(dir, "mac-mini"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLastPushHost(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "mac-mini" {
		t.Errorf("got %q, want %q", got, "mac-mini")
	}
}
