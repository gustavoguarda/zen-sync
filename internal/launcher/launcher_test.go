package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_CreatesAppBundleStructure(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "Zen Sync.app")
	if err := Install(appPath, "/opt/homebrew/bin/zen-sync", "io.github.gustavoguarda.zen-sync.launcher", "Zen Sync"); err != nil {
		t.Fatal(err)
	}

	// MacOS exec exists and is executable
	exe := filepath.Join(appPath, "Contents", "MacOS", "launcher")
	fi, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("launcher exe missing: %v", err)
	}
	if fi.Mode()&0o111 == 0 {
		t.Errorf("launcher exe not executable: mode=%v", fi.Mode())
	}

	body, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "/opt/homebrew/bin/zen-sync") {
		t.Errorf("launcher script missing binary path: %q", string(body))
	}
	if !strings.Contains(string(body), "open") {
		t.Errorf("launcher script should invoke open subcommand")
	}

	// Info.plist exists, contains bundle id, and references the icon
	info, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(info), "io.github.gustavoguarda.zen-sync.launcher") {
		t.Errorf("Info.plist missing bundle id")
	}
	if !strings.Contains(string(info), "<key>CFBundleIconFile</key><string>icon</string>") {
		t.Errorf("Info.plist missing CFBundleIconFile reference")
	}

	// Icon file written to Resources/, non-empty, has the icns magic
	icon, err := os.ReadFile(filepath.Join(appPath, "Contents", "Resources", "icon.icns"))
	if err != nil {
		t.Fatalf("icon.icns missing: %v", err)
	}
	if len(icon) < 1024 {
		t.Errorf("icon.icns suspiciously small: %d bytes", len(icon))
	}
	if string(icon[:4]) != "icns" {
		t.Errorf("icon.icns does not start with icns magic: %q", string(icon[:8]))
	}
}

func TestUninstall_RemovesBundle(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "Zen Sync.app")
	if err := Install(appPath, "/bin/echo", "io.test", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(appPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		t.Errorf("bundle still exists after Uninstall: err=%v", err)
	}
}
