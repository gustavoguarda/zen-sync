package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gustavoguarda/zen-sync/internal/config"
	"github.com/gustavoguarda/zen-sync/internal/launcher"
	"github.com/gustavoguarda/zen-sync/internal/plist"
	"github.com/gustavoguarda/zen-sync/internal/profile"
	"github.com/gustavoguarda/zen-sync/internal/sync"
)

const (
	launchAgentLabel = "io.github.gustavoguarda.zen-sync.daemon"
	launcherBundleID = "io.github.gustavoguarda.zen-sync.launcher"
)

// Init runs the setup wizard. Interactive — uses stdin for prompts.
func Init(in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	home, _ := os.UserHomeDir()

	fmt.Fprintln(out, "zen-sync init — let's set this up.")
	fmt.Fprintln(out)

	// 1. sync_dir — menu of detected transports.
	syncDir := pickSyncDir(r, out, home)
	if err := os.MkdirAll(syncDir, 0o700); err != nil {
		return fmt.Errorf("create sync_dir: %w", err)
	}

	// 2. Zen profile — auto-detect, manual override on failure.
	profileBase := filepath.Join(home, "Library", "Application Support", "zen", "Profiles")
	zenProfile, derr := profile.Detect(profileBase)
	if derr != nil {
		fmt.Fprintf(out, "\n⚠ Could not auto-detect Zen profile: %v\n", derr)
		zenProfile = prompt(r, out, "Zen profile (absolute path)", "")
		if zenProfile == "" {
			return fmt.Errorf("zen profile required")
		}
	} else {
		fmt.Fprintf(out, "\n✓ Found Zen profile: %s\n", zenProfile)
	}

	// 3. Hostname sanity — IP-derived hostnames break last-write-wins identity.
	if err := checkHostname(r, out); err != nil {
		fmt.Fprintf(out, "  ⚠ hostname check: %v (continuing)\n", err)
	}

	// 4. Build + save config.
	c := config.Default()
	c.SyncDir = syncDir
	c.ZenProfile = zenProfile
	cfgPath := filepath.Join(home, ".config", "zen-sync", "config.toml")
	if err := config.Save(c, cfgPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(out, "\n✓ Config written: %s\n", cfgPath)

	// 5. Install LaunchAgent.
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate own binary: %w", err)
	}
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	logDir := filepath.Join(home, "Library", "Logs", "zen-sync")
	if _, err := plist.Install(launchAgentLabel, binPath, plistDir, logDir, BuildVersion, BuildCommit); err != nil {
		return fmt.Errorf("install LaunchAgent: %w", err)
	}
	fmt.Fprintf(out, "✓ LaunchAgent installed (%s)\n", launchAgentLabel)

	// 6. Install /Applications/Zen Sync.app.
	appPath := "/Applications/Zen Sync.app"
	if err := launcher.Install(appPath, binPath, launcherBundleID, "Zen Sync", BuildVersion, BuildCommit); err != nil {
		return fmt.Errorf("install launcher app: %w", err)
	}
	fmt.Fprintf(out, "✓ Launcher installed: %s\n", appPath)

	// 7. Role auto-detection: empty sync = first device pushes; otherwise this Mac is joining.
	fmt.Fprintln(out)
	if isFirstDevice(syncDir, c.Files) {
		host, _ := os.Hostname()
		for _, f := range c.Files {
			if err := sync.Push(c.ZenProfile, c.SyncDir, f); err != nil {
				fmt.Fprintf(out, "  ⚠ push %s: %v\n", f, err)
			}
		}
		_ = sync.WriteLastPushHost(c.SyncDir, host)
		fmt.Fprintf(out, "✓ Sync folder was empty — pushed this Mac's state as initial source of truth (host=%s).\n", host)
	} else {
		fmt.Fprintln(out, "ℹ Sync folder already has state from another device — this Mac is joining.")
		fmt.Fprintln(out, "  Open Zen Sync.app (NOT Zen.app directly) to pull and launch with the synced state.")
		fmt.Fprintln(out, "  Opening Zen.app first would let the daemon push your stale local state and clobber the shared sync.")
	}

	// 8. Doctor.
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Running doctor checks…")
	fmt.Fprintln(out)
	if err := Doctor(out); err != nil {
		fmt.Fprintln(out)
		return fmt.Errorf("init finished but doctor reports problems: %w", err)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "All set. Drag /Applications/Zen Sync.app to your Dock and use it instead of Zen.app.")
	return nil
}

// pickSyncDir presents a menu of detected sync transports plus a custom-path option.
// "Detected" means the parent dir of the candidate exists on disk — a soft signal
// the user already has that service set up.
func pickSyncDir(r *bufio.Reader, w io.Writer, home string) string {
	type candidate struct {
		label    string
		path     string
		detected bool
	}
	cands := []candidate{
		{
			"Syncthing",
			filepath.Join(home, "BrowserSync", "Zen"),
			dirExists(filepath.Join(home, "BrowserSync")),
		},
		{
			"iCloud Drive",
			filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs", "zen-sync"),
			dirExists(filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs")),
		},
		{
			"Dropbox",
			filepath.Join(home, "Dropbox", "zen-sync"),
			dirExists(filepath.Join(home, "Dropbox")),
		},
	}

	fmt.Fprintln(w, "Where should zen-sync write state for your transport to replicate?")
	fmt.Fprintln(w)
	for i, c := range cands {
		mark := "[not detected]"
		if c.detected {
			mark = "[detected]"
		}
		fmt.Fprintf(w, "  %d) %s %s\n", i+1, c.label, mark)
		fmt.Fprintf(w, "     %s\n", c.path)
	}
	custom := len(cands) + 1
	fmt.Fprintf(w, "  %d) Custom path\n\n", custom)

	// Default: first detected candidate, falling back to option 1.
	def := "1"
	for i, c := range cands {
		if c.detected {
			def = strconv.Itoa(i + 1)
			break
		}
	}

	for {
		choice := strings.TrimSpace(prompt(r, w, "Pick", def))
		n, err := strconv.Atoi(choice)
		if err != nil {
			fmt.Fprintln(w, "Invalid choice; please enter a number.")
			continue
		}
		if n >= 1 && n <= len(cands) {
			return cands[n-1].path
		}
		if n == custom {
			p := prompt(r, w, "Path", filepath.Join(home, "BrowserSync", "Zen"))
			return expandTilde(p, home)
		}
		fmt.Fprintf(w, "Out of range. Pick 1-%d.\n", custom)
	}
}

// checkHostname warns and offers to set a proper HostName if the current value
// looks IP-derived. Same-IP-prefix hostnames on two Macs collapse zen-sync's
// last-write-wins identity into a single bucket, defeating skip-self pulls.
func checkHostname(r *bufio.Reader, w io.Writer) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	short := host
	if i := strings.IndexByte(host, '.'); i > 0 {
		short = host[:i]
	}
	if !looksIPLike(short) {
		return nil
	}

	suggested := getLocalHostName()
	if suggested == "" {
		suggested = "Mac"
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "⚠ Your hostname is %q — looks IP-derived.\n", short)
	fmt.Fprintln(w, "  zen-sync uses hostname to identify which Mac last pushed; distinct")
	fmt.Fprintln(w, "  hostnames per Mac are recommended so skip-self detection works.")
	fmt.Fprintln(w)
	answer := prompt(r, w, fmt.Sprintf("Set HostName to %q now? (sudo required) [Y/n]", suggested), "Y")
	if !strings.EqualFold(strings.TrimSpace(answer), "y") {
		fmt.Fprintln(w, "  Skipped — run `sudo scutil --set HostName \"<name>\"` later if you change your mind.")
		return nil
	}
	fmt.Fprintln(w, "  (macOS may prompt for your sudo password)")
	cmd := exec.Command("sudo", "scutil", "--set", "HostName", suggested)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = w, w, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scutil failed: %w", err)
	}
	fmt.Fprintf(w, "✓ HostName set to %q\n", suggested)
	return nil
}

// isFirstDevice reports whether the sync_dir has any signal of prior pushes.
// Both the tracked files AND .meta/last-push-host count as "someone was here".
func isFirstDevice(syncDir string, files []string) bool {
	if _, err := os.Stat(filepath.Join(syncDir, ".meta", "last-push-host")); err == nil {
		return false
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(syncDir, f)); err == nil {
			return false
		}
	}
	return true
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// looksIPLike returns true for an empty string or one starting with a digit
// (catches "192", "10", "172.x", etc — all common IP-derived macOS hostnames).
func looksIPLike(s string) bool {
	if s == "" {
		return true
	}
	if s[0] >= '0' && s[0] <= '9' {
		return true
	}
	return false
}

func getLocalHostName() string {
	out, err := exec.Command("scutil", "--get", "LocalHostName").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// prompt asks the user with a default. Returns the default on empty input.
func prompt(r *bufio.Reader, w io.Writer, label, def string) string {
	if def != "" {
		fmt.Fprintf(w, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(w, "%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func expandTilde(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
