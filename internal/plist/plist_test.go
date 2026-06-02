package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender_ContainsLabelAndBinaryPath(t *testing.T) {
	out := Render("io.github.gustavoguarda.zen-sync.daemon", "/opt/homebrew/bin/zen-sync", "/Users/x/Library/Logs/zen-sync/launchd.out", "/Users/x/Library/Logs/zen-sync/launchd.err")

	for _, want := range []string{
		"<string>io.github.gustavoguarda.zen-sync.daemon</string>",
		"<string>/opt/homebrew/bin/zen-sync</string>",
		"<string>daemon</string>",
		"<key>KeepAlive</key>",
		"<key>RunAtLoad</key>",
		"<string>/Users/x/Library/Logs/zen-sync/launchd.out</string>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestWriteFile_CreatesAtPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "io.test.plist")
	body := Render("io.test", "/bin/echo", "/tmp/out", "/tmp/err")
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
