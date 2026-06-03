package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Upgrade refreshes the Homebrew tap and upgrades zen-sync to the latest
// published version. Equivalent to `brew update && brew upgrade zen-sync`,
// but wrapped so users don't have to remember the two-step ritual.
//
// Only works when zen-sync was installed via brew. For source builds or
// manual installs we print a friendly hint and exit 0.
func Upgrade(w io.Writer) error {
	brew, err := findBrew()
	if err != nil {
		return err
	}
	if !installedViaBrew() {
		fmt.Fprintln(w, "zen-sync was not installed via Homebrew — nothing to upgrade automatically.")
		fmt.Fprintln(w, "Rebuild from source or grab a release artifact at:")
		fmt.Fprintln(w, "  https://github.com/gustavoguarda/zen-sync/releases")
		return nil
	}

	fmt.Fprintln(w, "→ brew update")
	if err := streamCmd(w, brew, "update"); err != nil {
		return fmt.Errorf("brew update failed: %w", err)
	}

	fmt.Fprintln(w, "→ brew upgrade zen-sync")
	if err := streamCmd(w, brew, "upgrade", "zen-sync"); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}

	// Belt-and-suspenders: modern brew silently skips the formula's
	// post_install hook for untrusted taps. We can't count on it to
	// refresh the .app + LaunchAgent. Exec the freshly-upgraded binary
	// (it has the new commit hash baked in) so Ensure regenerates them
	// against the new marker.
	fmt.Fprintln(w, "→ zen-sync ensure-installation")
	zsyncPath, _ := exec.LookPath("zen-sync")
	if zsyncPath == "" {
		for _, p := range []string{"/opt/homebrew/bin/zen-sync", "/usr/local/bin/zen-sync"} {
			if _, statErr := os.Stat(p); statErr == nil {
				zsyncPath = p
				break
			}
		}
	}
	if zsyncPath == "" {
		return fmt.Errorf("could not locate the upgraded zen-sync binary to call ensure-installation")
	}
	if err := streamCmd(w, zsyncPath, "ensure-installation"); err != nil {
		return fmt.Errorf("ensure-installation failed: %w", err)
	}

	return nil
}

// findBrew locates the brew binary. Hardcoded fallbacks cover the case
// where the user invoked `zen-sync upgrade` from a shell without brew on
// PATH (less common but possible from LaunchAgents or .app shells).
func findBrew() (string, error) {
	if p, err := exec.LookPath("brew"); err == nil {
		return p, nil
	}
	for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("brew not found — install Homebrew first (https://brew.sh) or upgrade manually")
}

// installedViaBrew checks whether this binary lives under one of brew's
// prefix directories. If so, brew can manage upgrades; otherwise we won't
// know what to do (source build, manual cp, etc).
func installedViaBrew() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	for _, prefix := range []string{"/opt/homebrew/", "/usr/local/Cellar/", "/usr/local/opt/"} {
		if strings.HasPrefix(exe, prefix) {
			return true
		}
	}
	return false
}

// streamCmd runs name with args, piping stdout+stderr to w so the user
// sees brew's familiar progress output in real time.
func streamCmd(w io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
