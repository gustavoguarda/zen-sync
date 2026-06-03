package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender_ContainsLabelAndBinaryPath(t *testing.T) {
	out := Render(
		"io.github.gustavoguarda.zen-sync.daemon",
		"/opt/homebrew/bin/zen-sync",
		"/Users/x/Library/Logs/zen-sync/launchd.out",
		"/Users/x/Library/Logs/zen-sync/launchd.err",
		"0.1.2",
		"abc1234",
	)

	for _, want := range []string{
		"<string>io.github.gustavoguarda.zen-sync.daemon</string>",
		"<string>/opt/homebrew/bin/zen-sync</string>",
		"<string>daemon</string>",
		"<key>KeepAlive</key>",
		"<key>RunAtLoad</key>",
		"<string>/Users/x/Library/Logs/zen-sync/launchd.out</string>",
		"IOZenSyncCommit:abc1234",
		"IOZenSyncVersion:0.1.2",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestWriteFile_CreatesAtPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "io.test.plist")
	body := Render("io.test", "/bin/echo", "/tmp/out", "/tmp/err", "0.1.2", "abc")
	if err := WriteFile(path, body); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Errorf("body mismatch")
	}
}

// TestCommitRe_ExtractsMarker checks the regex used by Ensure to detect
// the build commit embedded in an installed plist. Splitting this out from
// Ensure itself (which needs launchctl) keeps it unit-testable.
func TestCommitRe_ExtractsMarker(t *testing.T) {
	body := Render("io.test", "/bin/echo", "/tmp/out", "/tmp/err", "0.1.2", "deadbeef")
	m := commitRe.FindStringSubmatch(body)
	if len(m) != 2 {
		t.Fatalf("expected 2 submatches, got %d (full=%v)", len(m), m)
	}
	if m[1] != "deadbeef" {
		t.Errorf("got commit %q, want deadbeef", m[1])
	}

	// A file with no marker returns no submatches.
	m = commitRe.FindStringSubmatch("<plist><dict/></plist>")
	if len(m) != 0 {
		t.Errorf("expected no match for marker-less plist; got %v", m)
	}
}
