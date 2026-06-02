// Package profile resolves the Zen Browser profile directory and detects
// whether the Zen app is currently running.
//
// Resolution prefers a "*.Default (release)" directory; falls back to
// "*.Default" so non-release builds still work. Zero matches → error; more
// than one match → error (user must set zen_profile in config).
package profile

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
)

// Detect returns the absolute path of the Zen profile directory under base.
//
// base is typically "~/Library/Application Support/zen/Profiles" (already
// tilde-expanded by the caller). Pass it explicitly so tests can supply
// a temp dir.
func Detect(base string) (string, error) {
	if release, err := globOne(base, "*.Default (release)"); err == nil && release != "" {
		return release, nil
	} else if err != nil {
		return "", err
	}
	plain, err := globOne(base, "*.Default")
	if err != nil {
		return "", err
	}
	if plain == "" {
		return "", fmt.Errorf("profile: no Zen profile found under %s (set zen_profile in config to override)", base)
	}
	return plain, nil
}

// globOne returns the single match for the pattern under base, or "" if
// there were zero matches. More than one match returns an error.
func globOne(base, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(base, pattern))
	if err != nil {
		return "", fmt.Errorf("profile: glob %q under %s: %w", pattern, base, err)
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return "", fmt.Errorf("profile: multiple matches for %q under %s: %v (set zen_profile in config to disambiguate)", pattern, base, matches)
	}
	return matches[0], nil
}

// Running reports whether a process matching pattern is currently active.
// Uses pgrep -f under the hood; pattern is a path fragment, typically
// "/Zen.app/Contents/MacOS/".
func Running(pattern string) bool {
	// pgrep exits 0 when at least one match exists, 1 when none.
	cmd := exec.Command("pgrep", "-f", pattern)
	return cmd.Run() == nil
}
