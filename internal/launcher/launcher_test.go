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
	if err := Install(appPath, "/opt/homebrew/bin/zen-sync", "io.github.gustavoguarda.zen-sync.launcher", "Zen Sync", "0.1.2", "abc1234"); err != nil {
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

	// Info.plist exists, contains bundle id, icon key, and commit marker
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
	if !strings.Contains(string(info), "IOZenSyncCommit:abc1234") {
		t.Errorf("Info.plist missing IOZenSyncCommit marker")
	}
	if !strings.Contains(string(info), "<key>CFBundleShortVersionString</key><string>0.1.2</string>") {
		t.Errorf("Info.plist missing version string")
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
	if err := Install(appPath, "/bin/echo", "io.test", "Test", "0.1.2", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(appPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		t.Errorf("bundle still exists after Uninstall: err=%v", err)
	}
}

// TestEnsure_NoOpWhenCommitMatches checks that Ensure does NOT touch any
// file when the installed bundle's commit marker matches the current build.
// We tamper with the stub to detect any rewrite.
func TestEnsure_NoOpWhenCommitMatches(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "Zen Sync.app")
	if err := Install(appPath, "/bin/echo", "io.test", "Test", "0.1.2", "samecommit"); err != nil {
		t.Fatal(err)
	}

	stubPath := filepath.Join(appPath, "Contents", "MacOS", "launcher")
	const tampered = "# tampered by test\n"
	if err := os.WriteFile(stubPath, []byte(tampered), 0o755); err != nil {
		t.Fatal(err)
	}

	changed, err := Ensure(appPath, "/bin/echo", "io.test", "Test", "0.1.2", "samecommit")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("Ensure reported changed=true when commits match")
	}

	got, _ := os.ReadFile(stubPath)
	if string(got) != tampered {
		t.Errorf("Ensure regenerated despite matching commit; stub overwritten")
	}
}

// TestEnsure_RegeneratesWhenCommitDiffers checks that Ensure rewrites the
// bundle when the installed commit doesn't match the current build.
func TestEnsure_RegeneratesWhenCommitDiffers(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "Zen Sync.app")
	if err := Install(appPath, "/bin/echo", "io.test", "Test", "0.1.2", "oldcommit"); err != nil {
		t.Fatal(err)
	}

	changed, err := Ensure(appPath, "/bin/echo", "io.test", "Test", "0.1.3", "newcommit")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("Ensure reported changed=false when commits differ")
	}

	info, _ := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if !strings.Contains(string(info), "IOZenSyncCommit:newcommit") {
		t.Error("Info.plist not updated with the new commit")
	}
}

// TestEnsure_RegeneratesWhenMissing covers the case where the bundle was
// removed entirely (e.g. user did `rm -rf /Applications/Zen Sync.app`).
func TestEnsure_RegeneratesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "Zen Sync.app")

	changed, err := Ensure(appPath, "/bin/echo", "io.test", "Test", "0.1.2", "anycommit")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("Ensure reported changed=false for a missing bundle")
	}
	if _, err := os.Stat(filepath.Join(appPath, "Contents", "Info.plist")); err != nil {
		t.Errorf("Ensure should have created the bundle: %v", err)
	}
}
