# zen-sync v0.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **CRITICAL — user preference override:** This user reviews and commits manually. **SKIP every `git commit` step in this plan.** Leave changes uncommitted in the working tree for the user to review and stage on their own schedule. `git add` is also discouraged unless the user asks. This overrides the skill's default of frequent commits.

**Goal:** Ship zen-sync v0.1 — single Go binary for macOS that gives Zen Browser Arc-like continuity between Macs (push state continuously while Zen runs, pull-then-launch via a Dock .app wrapper).

**Architecture:** Single Go binary with subcommands. A LaunchAgent runs `zen-sync daemon` which watches `zen-sessions.jsonlz4` (and friends) via fsnotify and pushes to a user-configured `sync_dir`. A `/Applications/Zen Sync.app` launcher calls `zen-sync open`, which pulls then launches Zen. User brings their own transport (Syncthing, iCloud Drive, Dropbox, etc.) — the helper operates on filesystem only.

**Tech Stack:**
- Go 1.23+
- `github.com/BurntSushi/toml` (config)
- `github.com/fsnotify/fsnotify` (file watching)
- `gopkg.in/natefinch/lumberjack.v2` (rotating log)
- stdlib `flag`, `os/exec`, `crypto/sha256`, `testing`
- Build: `GoReleaser` for release artifacts
- CI: GitHub Actions (`go test`, `go vet`, `golangci-lint`)
- Distribution: Homebrew tap + GitHub Releases

**Spec:** `docs/specs/2026-06-01-zen-sync-v01-design.md`

---

## File Structure

```
zen-sync/
├── README.md                              # Task 13
├── LICENSE                                # Task 1 (MIT)
├── CHANGELOG.md                           # Task 13
├── go.mod                                 # Task 1
├── go.sum                                 # generated
├── Makefile                               # Task 1
├── .gitignore                             # Task 1
├── .goreleaser.yaml                       # Task 14
├── .golangci.yaml                         # Task 12
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                         # Task 12
│   │   └── release.yml                    # Task 14
│   └── ISSUE_TEMPLATE/
│       ├── bug.yaml                       # Task 14
│       └── feature.yaml                   # Task 14
├── cmd/zen-sync/
│   └── main.go                            # Task 2 (skeleton), grows in CLI tasks
├── internal/
│   ├── logger/
│   │   ├── logger.go                      # Task 3
│   │   └── logger_test.go                 # Task 3
│   ├── config/
│   │   ├── config.go                      # Task 4
│   │   └── config_test.go                 # Task 4
│   ├── profile/
│   │   ├── profile.go                     # Task 5
│   │   └── profile_test.go                # Task 5
│   ├── sync/
│   │   ├── sync.go                        # Task 6
│   │   └── sync_test.go                   # Task 6
│   ├── daemon/
│   │   ├── daemon.go                      # Task 7
│   │   └── daemon_test.go                 # Task 7
│   ├── plist/
│   │   ├── plist.go                       # Task 8
│   │   └── plist_test.go                  # Task 8
│   ├── launcher/
│   │   ├── launcher.go                    # Task 9
│   │   └── launcher_test.go               # Task 9
│   └── cli/
│       ├── version.go                     # Task 2
│       ├── init.go                        # Task 10
│       ├── open.go                        # Task 10
│       ├── push_pull.go                   # Task 11
│       ├── status.go                      # Task 11
│       ├── doctor.go                      # Task 11
│       ├── restore.go                     # Task 11
│       ├── daemon_cmd.go                  # Task 7
│       └── uninstall.go                   # Task 11
├── docs/
│   ├── specs/2026-06-01-zen-sync-v01-design.md   # already exists
│   ├── plans/2026-06-01-zen-sync-v01-implementation.md  # this file
│   ├── INSTALL.md                         # Task 13
│   ├── ARCHITECTURE.md                    # Task 13
│   └── TROUBLESHOOTING.md                 # Task 13
└── testdata/
    └── fake-profile/                      # Task 6
        ├── zen-sessions.jsonlz4           # sintético
        ├── zen-live-folders.jsonlz4
        └── containers.json
```

**Responsibility per file:**

| File / Package | Responsibility |
|---|---|
| `cmd/zen-sync/main.go` | Subcommand dispatcher. Reads `os.Args[1]`, routes to `internal/cli`. No business logic. |
| `internal/logger` | Rotating file logger factory (`logger.New(path string) *log.Logger`). |
| `internal/config` | TOML parse, defaults, `~/` expansion, validation. `Load`, `Default`, `Save`. |
| `internal/profile` | Detect Zen profile path on disk. Detect whether Zen is running. |
| `internal/sync` | Hash files. Push file → sync_dir. Pull file ← sync_dir with backup. Rotate backups. Read/write `.meta/last-push-host`. |
| `internal/daemon` | fsnotify watcher + debounce + hash-check loop. Calls `sync.Push` on real changes. |
| `internal/plist` | Generate, install, load, unload LaunchAgent plist. Wraps `launchctl`. |
| `internal/launcher` | Generate, install, uninstall `/Applications/Zen Sync.app` bundle. |
| `internal/cli/*` | One subcommand per file (`init`, `open`, `push`, `pull`, `status`, `doctor`, `restore`, `daemon_cmd`, `uninstall`, `version`). Each is `func Run(args []string) error`. |
| `docs/*.md` | User-facing docs (pitch in README, install guide, architecture handoff, troubleshooting from `doctor` outputs). |
| `testdata/fake-profile/` | Static binary blobs used by integration tests. |

---

## Task 1: Project bootstrap

**Files:**
- Create: `~/Projects/zen-sync/go.mod`
- Create: `~/Projects/zen-sync/.gitignore`
- Create: `~/Projects/zen-sync/LICENSE`
- Create: `~/Projects/zen-sync/Makefile`

- [ ] **Step 1: Initialize Go module**

Working directory throughout this plan: `~/Projects/zen-sync`.

```bash
cd ~/Projects/zen-sync
go mod init github.com/gustavoguarda/zen-sync
```

Expected: creates `go.mod` with module path `github.com/gustavoguarda/zen-sync` and a Go version line.

- [ ] **Step 2: Write `.gitignore`**

Create `~/Projects/zen-sync/.gitignore` with this exact content:

```gitignore
# Build artifacts
/zen-sync
/dist/
/bin/

# Test / coverage
*.out
coverage.html

# Editor / OS
.DS_Store
.idea/
.vscode/
*.swp

# Local scratch
/tmp/
/.local/
```

- [ ] **Step 3: Write `LICENSE` (MIT)**

Create `~/Projects/zen-sync/LICENSE` with this content (replace year if needed — current is 2026):

```
MIT License

Copyright (c) 2026 Gustavo Guarda

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 4: Write `Makefile`**

Create `~/Projects/zen-sync/Makefile`:

```makefile
.PHONY: build test vet lint install-local clean

BINARY := zen-sync
PKG    := github.com/gustavoguarda/zen-sync

build:
	go build -o $(BINARY) ./cmd/zen-sync

test:
	go test -race -cover ./...

vet:
	go vet ./...

lint:
	golangci-lint run

install-local: build
	mv $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf dist/
```

- [ ] **Step 5: Verify `go build` works on an empty module**

```bash
go build ./...
```

Expected: no output and exit 0. (Empty module; `./...` resolves to nothing yet.)

- [ ] **Step 6: SKIP commit per user preference**

Do NOT run `git commit`. Leave files untracked.

---

## Task 2: CLI skeleton + `version` subcommand

Establishes the dispatcher pattern. `version` is the simplest subcommand, so it's first.

**Files:**
- Create: `cmd/zen-sync/main.go`
- Create: `internal/cli/version.go`
- Create: `internal/cli/version_test.go`

- [ ] **Step 1: Write the failing test (TDD)**

Create `internal/cli/version_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion_PrintsExpectedFields(t *testing.T) {
	var buf bytes.Buffer
	err := Version(&buf, "v0.1.0-test", "abc1234", "2026-06-01")
	if err != nil {
		t.Fatalf("Version returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"v0.1.0-test", "abc1234", "2026-06-01"} {
		if !strings.Contains(out, want) {
			t.Errorf("Version output missing %q; got: %q", want, out)
		}
	}
}
```

- [ ] **Step 2: Run test — expect compile failure**

```bash
go test ./internal/cli/...
```

Expected: build error "undefined: Version".

- [ ] **Step 3: Implement `Version`**

Create `internal/cli/version.go`:

```go
// Package cli implements zen-sync subcommand handlers.
//
// Each subcommand exposes a top-level function called by the dispatcher in
// cmd/zen-sync. Subcommands accept their own args slice (sans the program
// name and subcommand name) and return an error.
package cli

import (
	"fmt"
	"io"
)

// Version writes the build metadata to w.
// The values are passed in (rather than read from globals) so the test can
// pin them; production callers will use the values from main.go's ldflags.
func Version(w io.Writer, version, commit, buildDate string) error {
	_, err := fmt.Fprintf(w, "zen-sync %s\ncommit:  %s\nbuilt:   %s\n", version, commit, buildDate)
	return err
}
```

- [ ] **Step 4: Run test — expect pass**

```bash
go test ./internal/cli/...
```

Expected: `ok  github.com/gustavoguarda/zen-sync/internal/cli`.

- [ ] **Step 5: Write the dispatcher in main.go**

Create `cmd/zen-sync/main.go`:

```go
// Command zen-sync is the entrypoint for the zen-sync helper.
//
// It is a subcommand dispatcher. Each subcommand lives in internal/cli.
// Build metadata is injected via -ldflags by the release pipeline; the
// defaults below are used by `go build` and local dev.
package main

import (
	"fmt"
	"os"

	"github.com/gustavoguarda/zen-sync/internal/cli"
)

// Injected at build time by goreleaser (-ldflags "-X main.version=...").
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
	case "version", "-v", "--version":
		err = cli.Version(os.Stdout, version, commit, buildDate)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "zen-sync: unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "zen-sync %s: %v\n", cmd, err)
		os.Exit(1)
	}
	_ = args // unused for now; future subcommands consume it
}

func usage() {
	fmt.Fprint(os.Stderr, `zen-sync — Arc-like continuity for Zen Browser on macOS

Usage:
  zen-sync <command> [args]

Commands:
  init        Setup wizard: config + LaunchAgent + Zen Sync.app + first push
  open        Pull state then launch Zen
  push        Manual push (debug / source-of-truth)
  pull        Manual pull (debug / recovery)
  daemon      Long-running file watcher (called by LaunchAgent; not by user)
  status      Show last sync, hashes, last-push-host
  restore     List or restore backups
  doctor      Diagnose common problems
  uninstall   Remove LaunchAgent and Zen Sync.app
  version     Show build metadata

Run 'zen-sync help' for this list.
`)
}
```

- [ ] **Step 6: Build and run the binary**

```bash
go build -o zen-sync ./cmd/zen-sync && ./zen-sync version
```

Expected output:
```
zen-sync dev
commit:  none
built:   unknown
```

- [ ] **Step 7: Try an unknown command**

```bash
./zen-sync nope ; echo "exit=$?"
```

Expected: usage banner on stderr + `exit=2`.

- [ ] **Step 8: SKIP commit per user preference.**

Clean up the local binary so it doesn't end up tracked:

```bash
rm -f zen-sync
```

---

## Task 3: `internal/logger` — rotating file logger

**Files:**
- Create: `internal/logger/logger.go`
- Create: `internal/logger/logger_test.go`

- [ ] **Step 1: Add lumberjack dependency**

```bash
go get gopkg.in/natefinch/lumberjack.v2
```

Expected: `go.mod` and `go.sum` updated; no compile yet.

- [ ] **Step 2: Write the failing test**

Create `internal/logger/logger_test.go`:

```go
package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_WritesToPathAndRotates(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "sub", "test.log")

	lg, closeFn, err := New(logPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closeFn()

	lg.Println("hello-from-test")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "hello-from-test") {
		t.Errorf("log file missing message; got: %q", string(data))
	}
}

func TestNew_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "a", "b", "c", "test.log")

	_, closeFn, err := New(logPath)
	if err != nil {
		t.Fatalf("New should mkdir -p parents, got: %v", err)
	}
	defer closeFn()

	if _, err := os.Stat(filepath.Dir(logPath)); err != nil {
		t.Errorf("parent dir not created: %v", err)
	}
}
```

- [ ] **Step 3: Run test — expect compile failure**

```bash
go test ./internal/logger/...
```

Expected: "undefined: New".

- [ ] **Step 4: Implement**

Create `internal/logger/logger.go`:

```go
// Package logger creates rotating file loggers for the daemon and CLI.
//
// Rotation: 1 MiB per file, keep 3 backups. We pin those defaults here
// rather than parameterizing them because they are products of "what feels
// right for a small helper" and there is no use case yet for tuning them.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// New returns a *log.Logger that writes to path with size-based rotation.
// The returned closeFn flushes and closes the underlying sink and must be
// called before the process exits.
func New(path string) (*log.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	lj := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    1, // MiB
		MaxBackups: 3,
		LocalTime:  true,
	}
	lg := log.New(io.Writer(lj), "", log.LstdFlags|log.Lmicroseconds)
	return lg, lj.Close, nil
}
```

- [ ] **Step 5: Run test — expect pass**

```bash
go test ./internal/logger/... -race -v
```

Expected: both `TestNew_WritesToPathAndRotates` and `TestNew_CreatesParentDir` PASS.

- [ ] **Step 6: SKIP commit per user preference.**

---

## Task 4: `internal/config` — TOML parse + defaults + `~/` expansion

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Add TOML dependency**

```bash
go get github.com/BurntSushi/toml
```

- [ ] **Step 2: Write the failing test**

Create `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 3: Run test — expect compile failure**

```bash
go test ./internal/config/...
```

Expected: "undefined: Load" / "undefined: Save" / "undefined: Config".

- [ ] **Step 4: Implement**

Create `internal/config/config.go`:

```go
// Package config defines the on-disk configuration format and applies
// defaults + path expansion.
//
// The on-disk format is TOML at ~/.config/zen-sync/config.toml. Paths that
// begin with "~/" are expanded relative to $HOME at load time.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the parsed and validated configuration.
type Config struct {
	SyncDir           string   `toml:"sync_dir"`
	ZenProfile        string   `toml:"zen_profile"`
	ZenRunningPattern string   `toml:"zen_running_pattern"`
	Files             []string `toml:"files"`

	Daemon struct {
		DebounceMs int    `toml:"debounce_ms"`
		LogLevel   string `toml:"log_level"`
	} `toml:"daemon"`

	Backup struct {
		Keep int `toml:"keep"`
	} `toml:"backup"`
}

// Default returns a Config populated with default values for all optional
// fields. SyncDir is left empty — it must be provided by the user.
func Default() *Config {
	c := &Config{
		ZenRunningPattern: "/Zen.app/Contents/MacOS/",
		Files: []string{
			"zen-sessions.jsonlz4",
			"zen-live-folders.jsonlz4",
			"containers.json",
		},
	}
	c.Daemon.DebounceMs = 1000
	c.Daemon.LogLevel = "info"
	c.Backup.Keep = 5
	return c
}

// Load reads the TOML file at path, applies defaults to missing fields,
// expands "~/" paths against $HOME, and validates required fields.
func Load(path string) (*Config, error) {
	c := Default()
	if _, err := toml.DecodeFile(path, c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := expand(c); err != nil {
		return nil, err
	}
	if err := validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Save writes c to path as TOML. Parent dirs are created if missing.
func Save(c *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

func expand(c *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	c.SyncDir = expandTilde(c.SyncDir, home)
	c.ZenProfile = expandTilde(c.ZenProfile, home)
	return nil
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

func validate(c *Config) error {
	if strings.TrimSpace(c.SyncDir) == "" {
		return errors.New("config: sync_dir is required")
	}
	if c.Daemon.DebounceMs <= 0 {
		return fmt.Errorf("config: daemon.debounce_ms must be > 0 (got %d)", c.Daemon.DebounceMs)
	}
	if c.Backup.Keep <= 0 {
		return fmt.Errorf("config: backup.keep must be > 0 (got %d)", c.Backup.Keep)
	}
	if len(c.Files) == 0 {
		return errors.New("config: files must not be empty")
	}
	return nil
}
```

- [ ] **Step 5: Run tests — expect pass**

```bash
go test ./internal/config/... -race -v
```

Expected: 3 tests PASS.

- [ ] **Step 6: SKIP commit per user preference.**

---

## Task 5: `internal/profile` — Zen profile detection + process check

**Files:**
- Create: `internal/profile/profile.go`
- Create: `internal/profile/profile_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/profile/profile_test.go`:

```go
package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetect_FindsReleaseProfile(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "abc.Default (release)")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Detect(base)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDetect_FallsBackToDefaultWithoutSuffix(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "xyz.Default")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Detect(base)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDetect_PrefersReleaseOverPlainDefault(t *testing.T) {
	base := t.TempDir()
	release := filepath.Join(base, "rrr.Default (release)")
	plain := filepath.Join(base, "ppp.Default")
	for _, d := range []string{release, plain} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Detect(base)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got != release {
		t.Errorf("got %q, want %q (release should win)", got, release)
	}
}

func TestDetect_ErrorsOnMissingBase(t *testing.T) {
	_, err := Detect(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDetect_ErrorsOnMultiple(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"a.Default (release)", "b.Default (release)"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Detect(base)
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "multiple") {
		t.Errorf("error should mention 'multiple'; got: %v", err)
	}
}

func TestRunning_FalseWhenNoMatch(t *testing.T) {
	if Running("/this/path/will/never/match/anything") {
		t.Error("Running returned true for an impossible pattern")
	}
}
```

- [ ] **Step 2: Run test — expect compile failure**

```bash
go test ./internal/profile/...
```

Expected: "undefined: Detect" / "undefined: Running".

- [ ] **Step 3: Implement**

Create `internal/profile/profile.go`:

```go
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
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./internal/profile/... -race -v
```

Expected: 6 tests PASS.

- [ ] **Step 5: SKIP commit per user preference.**

---

## Task 6: `internal/sync` — Hash, Push, Pull, RotateBackups

**Files:**
- Create: `internal/sync/sync.go`
- Create: `internal/sync/sync_test.go`
- Create: `testdata/fake-profile/zen-sessions.jsonlz4`
- Create: `testdata/fake-profile/zen-live-folders.jsonlz4`
- Create: `testdata/fake-profile/containers.json`

- [ ] **Step 1: Write fake profile fixtures**

```bash
mkdir -p ~/Projects/zen-sync/testdata/fake-profile
printf 'mozLz40\0SESSIONS_FAKE_A'      > ~/Projects/zen-sync/testdata/fake-profile/zen-sessions.jsonlz4
printf 'mozLz40\0LIVE_FOLDERS_FAKE_B'  > ~/Projects/zen-sync/testdata/fake-profile/zen-live-folders.jsonlz4
printf '{"version":4,"identities":[{"name":"Work"}]}\n' > ~/Projects/zen-sync/testdata/fake-profile/containers.json
```

- [ ] **Step 2: Write the failing test**

Create `internal/sync/sync_test.go`:

```go
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
```

- [ ] **Step 3: Run test — expect compile failure**

```bash
go test ./internal/sync/...
```

Expected: undefined symbols for `Hash`, `Push`, `Pull`, `RotateBackups`, `ReadLastPushHost`, `WriteLastPushHost`.

- [ ] **Step 4: Implement**

Create `internal/sync/sync.go`:

```go
// Package sync moves files between the Zen profile and a user-provided
// sync_dir. It does not interpret file contents — pure binary copy with
// integrity-by-hash and rotated backups.
package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Hash returns the hex-encoded SHA-256 of the file at path.
func Hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Push copies <profile>/<file> to <syncDir>/<file>. A missing source file
// is treated as a no-op (the user may not yet have that file — e.g. a
// fresh profile without live folders).
func Push(profile, syncDir, file string) error {
	src := filepath.Join(profile, file)
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.MkdirAll(syncDir, 0o755); err != nil {
		return err
	}
	return copyFile(src, filepath.Join(syncDir, file))
}

// Pull copies <syncDir>/<file> back into <profile>/<file>, backing up the
// current local copy into <backupDir> first. A missing sync file is a
// non-destructive no-op (preserves local).
func Pull(profile, syncDir, backupDir, file string) error {
	local := filepath.Join(profile, file)
	remote := filepath.Join(syncDir, file)

	// Backup local if it exists.
	if _, err := os.Stat(local); err == nil {
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return err
		}
		ts := time.Now().Format("20060102T150405")
		if err := copyFile(local, filepath.Join(backupDir, fmt.Sprintf("%s.%s", file, ts))); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Skip if sync doesn't have it.
	if _, err := os.Stat(remote); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	if err := os.MkdirAll(profile, 0o755); err != nil {
		return err
	}
	return copyFile(remote, local)
}

// RotateBackups keeps the `keep` newest entries whose basename starts with
// `pattern + "."` in dir, deleting the rest.
func RotateBackups(dir, pattern string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	type indexed struct {
		path  string
		mtime time.Time
	}
	var matched []indexed
	prefix := pattern + "."
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			return err
		}
		matched = append(matched, indexed{filepath.Join(dir, e.Name()), fi.ModTime()})
	}
	if len(matched) <= keep {
		return nil
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].mtime.After(matched[j].mtime) })
	for _, old := range matched[keep:] {
		if err := os.RemoveAll(old.path); err != nil {
			return err
		}
	}
	return nil
}

// ReadLastPushHost returns the contents of <syncDir>/.meta/last-push-host,
// or "" if the file doesn't exist.
func ReadLastPushHost(syncDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(syncDir, ".meta", "last-push-host"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// WriteLastPushHost stamps <syncDir>/.meta/last-push-host with host.
func WriteLastPushHost(syncDir, host string) error {
	dir := filepath.Join(syncDir, ".meta")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last-push-host"), []byte(host+"\n"), 0o644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// Write to temp then rename for atomicity within the same fs.
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
```

- [ ] **Step 5: Run tests — expect pass**

```bash
go test ./internal/sync/... -race -v
```

Expected: 7 tests PASS.

- [ ] **Step 6: SKIP commit per user preference.**

---

## Task 7: `internal/daemon` — fsnotify watcher + debounce + `daemon` subcommand

**Files:**
- Create: `internal/daemon/daemon.go`
- Create: `internal/daemon/daemon_test.go`
- Create: `internal/cli/daemon_cmd.go`
- Modify: `cmd/zen-sync/main.go` (add `daemon` case to switch)

- [ ] **Step 1: Add fsnotify**

```bash
go get github.com/fsnotify/fsnotify
```

- [ ] **Step 2: Write the failing test**

Create `internal/daemon/daemon_test.go`:

```go
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
```

- [ ] **Step 3: Run test — expect compile failure**

```bash
go test ./internal/daemon/...
```

Expected: undefined `Run`, `Config`.

- [ ] **Step 4: Implement**

Create `internal/daemon/daemon.go`:

```go
// Package daemon watches the configured profile files via fsnotify and
// pushes changes to sync_dir after a debounce window.
//
// Two safeguards beyond raw fsnotify: (1) debounce so we don't catch a
// partially-flushed file; (2) SHA-256 hash check so we don't push when
// Zen rewrote the file without changing content.
package daemon

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/gustavoguarda/zen-sync/internal/sync"
)

// Config carries everything Run needs. Decoupled from internal/config to
// keep daemon package testable without a real TOML file.
type Config struct {
	Profile    string
	SyncDir    string
	Files      []string
	Hostname   string
	DebounceMs int
}

// LogFn is a printf-style logger. Caller provides one.
type LogFn func(format string, args ...any)

// Run blocks until ctx is cancelled. It watches the files listed in
// cfg.Files (within cfg.Profile) and pushes them to cfg.SyncDir on real
// changes (debounced, hash-checked).
func Run(ctx context.Context, cfg Config, logf LogFn) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("daemon: new watcher: %w", err)
	}
	defer w.Close()

	// Watch the profile directory itself (fsnotify on macOS doesn't always
	// pick up renames of individual files; watching the parent dir is more
	// reliable).
	if err := w.Add(cfg.Profile); err != nil {
		return fmt.Errorf("daemon: watch %s: %w", cfg.Profile, err)
	}
	tracked := map[string]bool{}
	for _, f := range cfg.Files {
		tracked[f] = true
	}

	debounce := time.Duration(cfg.DebounceMs) * time.Millisecond
	lastHash := map[string]string{}
	pending := map[string]*time.Timer{}

	flush := func(file string) {
		fullPath := filepath.Join(cfg.Profile, file)
		newHash, err := sync.Hash(fullPath)
		if err != nil {
			logf("daemon: hash %s: %v", file, err)
			return
		}
		if lastHash[file] == newHash {
			return // hash-check: redundant write
		}
		if err := sync.Push(cfg.Profile, cfg.SyncDir, file); err != nil {
			logf("daemon: push %s: %v", file, err)
			return
		}
		if err := sync.WriteLastPushHost(cfg.SyncDir, cfg.Hostname); err != nil {
			logf("daemon: stamp host: %v", err)
		}
		lastHash[file] = newHash
		logf("daemon: pushed %s (hash=%s…)", file, newHash[:8])
	}

	schedule := func(file string) {
		if t, ok := pending[file]; ok {
			t.Stop()
		}
		pending[file] = time.AfterFunc(debounce, func() { flush(file) })
	}

	for {
		select {
		case <-ctx.Done():
			for _, t := range pending {
				t.Stop()
			}
			return ctx.Err()
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logf("daemon: watcher error: %v", err)
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			name := filepath.Base(ev.Name)
			if !tracked[name] {
				continue
			}
			// Treat any event on a tracked file as a candidate.
			schedule(name)
		}
	}
}
```

Create `internal/cli/daemon_cmd.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gustavoguarda/zen-sync/internal/config"
	"github.com/gustavoguarda/zen-sync/internal/daemon"
	"github.com/gustavoguarda/zen-sync/internal/logger"
)

// Daemon runs the watcher loop until SIGTERM/SIGINT.
//
// The LaunchAgent invokes `zen-sync daemon`; the user does not normally
// run it directly.
func Daemon(w io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library", "Logs", "zen-sync", "daemon.log")
	lg, closeFn, err := logger.New(logPath)
	if err != nil {
		return fmt.Errorf("daemon: open log: %w", err)
	}
	defer closeFn()

	host, _ := os.Hostname()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dcfg := daemon.Config{
		Profile:    cfg.ZenProfile,
		SyncDir:    cfg.SyncDir,
		Files:      cfg.Files,
		Hostname:   host,
		DebounceMs: cfg.Daemon.DebounceMs,
	}
	lg.Printf("daemon: starting (profile=%s sync=%s files=%d)", dcfg.Profile, dcfg.SyncDir, len(dcfg.Files))
	err = daemon.Run(ctx, dcfg, func(f string, a ...any) { lg.Printf(f, a...) })
	if err != nil && err != context.Canceled {
		lg.Printf("daemon: exit error: %v", err)
		return err
	}
	lg.Printf("daemon: shutdown clean")
	return nil
}

// loadConfig is shared by subcommands that need the parsed Config.
// It reads ~/.config/zen-sync/config.toml. If the file is missing the
// returned error is a hint to run `zen-sync init`.
func loadConfig() (*config.Config, error) {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".config", "zen-sync", "config.toml")
	c, err := config.Load(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config at %s — run `zen-sync init`", p)
		}
		return nil, err
	}
	return c, nil
}
```

- [ ] **Step 5: Wire `daemon` into the main dispatcher**

Edit `cmd/zen-sync/main.go` — in the `switch cmd` block, add:

```go
	case "daemon":
		err = cli.Daemon(os.Stdout)
```

(Insert immediately after the `version` case.)

- [ ] **Step 6: Build and run tests**

```bash
go build ./... && go test ./internal/daemon/... -race -v
```

Expected: build succeeds; 2 daemon tests PASS. The redundant-write test depends on filesystem timing; if flaky on slower machines, run it 3 times to confirm stability.

- [ ] **Step 7: SKIP commit per user preference.**

---

## Task 8: `internal/plist` — generate, install, uninstall LaunchAgent

**Files:**
- Create: `internal/plist/plist.go`
- Create: `internal/plist/plist_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/plist/plist_test.go`:

```go
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
```

- [ ] **Step 2: Run test — expect compile failure**

```bash
go test ./internal/plist/...
```

Expected: undefined `Render`, `WriteFile`.

- [ ] **Step 3: Implement**

Create `internal/plist/plist.go`:

```go
// Package plist generates and installs the LaunchAgent plist that runs
// `zen-sync daemon` at user login.
//
// Install/Uninstall shell out to launchctl. The Render function is pure
// string generation and is unit-testable; Install/Uninstall require a
// real launchd and are exercised via manual verification.
package plist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Render returns the XML body of the LaunchAgent plist.
func Render(label, binaryPath, stdoutPath, stderrPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>

    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>ProcessType</key>
    <string>Background</string>

    <key>StandardOutPath</key>
    <string>%s</string>

    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, label, binaryPath, stdoutPath, stderrPath)
}

// WriteFile creates parent dirs and writes body to path with 0644.
func WriteFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// Install writes the plist to ~/Library/LaunchAgents/<label>.plist and
// loads it via `launchctl load`.
func Install(label, binaryPath, plistDir, logDir string) (string, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	stdout := filepath.Join(logDir, "launchd.out")
	stderr := filepath.Join(logDir, "launchd.err")
	body := Render(label, binaryPath, stdout, stderr)
	plistPath := filepath.Join(plistDir, label+".plist")
	if err := WriteFile(plistPath, body); err != nil {
		return "", err
	}
	// Try to unload first (idempotent), then load.
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return plistPath, fmt.Errorf("launchctl load: %w", err)
	}
	return plistPath, nil
}

// Uninstall unloads the agent and removes the plist file.
func Uninstall(label, plistDir string) error {
	plistPath := filepath.Join(plistDir, label+".plist")
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./internal/plist/... -race -v
```

Expected: 2 tests PASS.

- [ ] **Step 5: SKIP commit per user preference.**

---

## Task 9: `internal/launcher` — install `/Applications/Zen Sync.app`

**Files:**
- Create: `internal/launcher/launcher.go`
- Create: `internal/launcher/launcher_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/launcher/launcher_test.go`:

```go
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

	// Info.plist exists and contains bundle id
	info, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(info), "io.github.gustavoguarda.zen-sync.launcher") {
		t.Errorf("Info.plist missing bundle id")
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
```

- [ ] **Step 2: Run test — expect compile failure**

```bash
go test ./internal/launcher/...
```

Expected: undefined `Install`, `Uninstall`.

- [ ] **Step 3: Implement**

Create `internal/launcher/launcher.go`:

```go
// Package launcher generates and installs the Zen Sync.app bundle, a
// minimal .app whose only job is to exec `zen-sync open` so the user can
// put it in the Dock.
package launcher

import (
	"fmt"
	"os"
	"path/filepath"
)

// Install creates a minimal .app bundle at appPath. The bundle's executable
// is a short bash stub that exec's `<binaryPath> open` (forwarding any
// args the .app receives, like a URL).
func Install(appPath, binaryPath, bundleID, displayName string) error {
	macOSDir := filepath.Join(appPath, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		return err
	}

	// Info.plist
	info := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key><string>en</string>
    <key>CFBundleExecutable</key><string>launcher</string>
    <key>CFBundleIdentifier</key><string>%s</string>
    <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
    <key>CFBundleName</key><string>%s</string>
    <key>CFBundleDisplayName</key><string>%s</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleShortVersionString</key><string>0.1.0</string>
    <key>CFBundleVersion</key><string>1</string>
    <key>LSMinimumSystemVersion</key><string>11.0</string>
    <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
`, bundleID, displayName, displayName)

	if err := os.WriteFile(filepath.Join(appPath, "Contents", "Info.plist"), []byte(info), 0o644); err != nil {
		return err
	}

	// Launcher shell stub. Forwards extra args (Zen treats them as URLs/files).
	stub := fmt.Sprintf(`#!/bin/bash
# Auto-generated by zen-sync. Edit only if you know what you are doing.
exec %q open "$@"
`, binaryPath)
	stubPath := filepath.Join(macOSDir, "launcher")
	if err := os.WriteFile(stubPath, []byte(stub), 0o755); err != nil {
		return err
	}
	return nil
}

// Uninstall removes the .app bundle. Missing is treated as success.
func Uninstall(appPath string) error {
	if err := os.RemoveAll(appPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./internal/launcher/... -race -v
```

Expected: 2 tests PASS.

- [ ] **Step 5: SKIP commit per user preference.**

---

## Task 10: `init` and `open` subcommands

**Files:**
- Create: `internal/cli/init.go`
- Create: `internal/cli/open.go`
- Modify: `cmd/zen-sync/main.go` (add cases)

These two are the user's primary touchpoints. `init` runs once; `open` runs every Dock click.

- [ ] **Step 1: Implement `init`**

Create `internal/cli/init.go`:

```go
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// Init runs the setup wizard. It is interactive — uses stdin for prompts.
func Init(in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	home, _ := os.UserHomeDir()

	fmt.Fprintln(out, "zen-sync init — let's set this up.")
	fmt.Fprintln(out, "")

	// 1. sync_dir
	fmt.Fprintf(out, "Sync folder (where the helper writes state for your transport to replicate)\n")
	fmt.Fprintf(out, "  Example: ~/BrowserSync/Zen  (point this at your Syncthing/iCloud/Dropbox folder)\n")
	syncDir := prompt(r, out, "Sync folder", filepath.Join(home, "BrowserSync", "Zen"))
	syncDir = expandTilde(syncDir, home)
	if err := os.MkdirAll(syncDir, 0o700); err != nil {
		return fmt.Errorf("create sync_dir: %w", err)
	}

	// 2. zen profile (auto-detect, allow override)
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

	// 3. Build + save config
	c := config.Default()
	c.SyncDir = syncDir
	c.ZenProfile = zenProfile
	cfgPath := filepath.Join(home, ".config", "zen-sync", "config.toml")
	if err := config.Save(c, cfgPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(out, "✓ Config written: %s\n", cfgPath)

	// 4. Install LaunchAgent
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate own binary: %w", err)
	}
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	logDir := filepath.Join(home, "Library", "Logs", "zen-sync")
	if _, err := plist.Install(launchAgentLabel, binPath, plistDir, logDir); err != nil {
		return fmt.Errorf("install LaunchAgent: %w", err)
	}
	fmt.Fprintf(out, "✓ LaunchAgent installed and loaded (%s)\n", launchAgentLabel)

	// 5. Install /Applications/Zen Sync.app
	appPath := "/Applications/Zen Sync.app"
	if err := launcher.Install(appPath, binPath, launcherBundleID, "Zen Sync"); err != nil {
		return fmt.Errorf("install launcher app: %w", err)
	}
	fmt.Fprintf(out, "✓ Launcher installed: %s\n", appPath)
	fmt.Fprintln(out, "  → Drag it to your Dock in place of Zen.")

	// 6. First push (with confirmation)
	fmt.Fprintln(out, "")
	yesno := prompt(r, out, "Use THIS Mac as source-of-truth and do an initial push? [y/N]", "N")
	if strings.EqualFold(strings.TrimSpace(yesno), "y") {
		host, _ := os.Hostname()
		for _, f := range c.Files {
			if err := sync.Push(c.ZenProfile, c.SyncDir, f); err != nil {
				fmt.Fprintf(out, "  ⚠ push %s: %v\n", f, err)
			}
		}
		_ = sync.WriteLastPushHost(c.SyncDir, host)
		fmt.Fprintln(out, "✓ Initial push complete.")
	}

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Done. Open Zen Sync.app (or click it in the Dock) to launch Zen with sync.")
	return nil
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
```

- [ ] **Step 2: Implement `open`**

Create `internal/cli/open.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/profile"
	"github.com/gustavoguarda/zen-sync/internal/sync"
)

// Open is the launch path: pull state from sync_dir, then exec `open -W -na Zen`.
//
// Skips the pull (1) if Zen is already running — no point swapping files
// that Zen has cached in memory — or (2) if this host was the last to push.
// Extra args are forwarded to Zen (URLs, file paths).
func Open(w io.Writer, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if profile.Running(cfg.ZenRunningPattern) {
		fmt.Fprintln(w, "Zen is already running — bringing to front.")
		return exec.Command("open", "-a", "Zen").Run()
	}

	host, _ := os.Hostname()
	last, _ := sync.ReadLastPushHost(cfg.SyncDir)
	if last == host {
		fmt.Fprintln(w, "Local state is already current (last push from this host). Skipping pull.")
	} else {
		home, _ := os.UserHomeDir()
		backupDir := filepath.Join(home, ".local", "state", "zen-sync", "backups")
		for _, f := range cfg.Files {
			if err := sync.Pull(cfg.ZenProfile, cfg.SyncDir, backupDir, f); err != nil {
				fmt.Fprintf(w, "  ⚠ pull %s: %v\n", f, err)
			}
			if err := sync.RotateBackups(backupDir, f, cfg.Backup.Keep); err != nil {
				fmt.Fprintf(w, "  ⚠ rotate %s: %v\n", f, err)
			}
		}
		fmt.Fprintln(w, "Pulled state from sync_dir.")
	}

	// open -W blocks until app quits — but we don't wait, because the
	// daemon handles continuous push. We just launch.
	openArgs := append([]string{"-na", "Zen"}, args...)
	return exec.Command("open", openArgs...).Run()
}
```

- [ ] **Step 3: Wire into main.go**

Edit `cmd/zen-sync/main.go` — add cases to the switch:

```go
	case "init":
		err = cli.Init(os.Stdin, os.Stdout)
	case "open":
		err = cli.Open(os.Stdout, args)
```

- [ ] **Step 4: Build and run a sanity check**

```bash
go build ./... && ./zen-sync help
```

Expected: build succeeds; help banner prints.

```bash
rm -f zen-sync
```

- [ ] **Step 5: SKIP commit per user preference.**

---

## Task 11: `push`, `pull`, `status`, `doctor`, `restore`, `uninstall` subcommands

These are smaller wrappers around `internal/sync` plus diagnostics.

**Files:**
- Create: `internal/cli/push_pull.go`
- Create: `internal/cli/status.go`
- Create: `internal/cli/doctor.go`
- Create: `internal/cli/restore.go`
- Create: `internal/cli/uninstall.go`
- Modify: `cmd/zen-sync/main.go`

- [ ] **Step 1: Implement push/pull**

Create `internal/cli/push_pull.go`:

```go
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/profile"
	"github.com/gustavoguarda/zen-sync/internal/sync"
)

func Push(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print actions without performing them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if profile.Running(cfg.ZenRunningPattern) {
		return fmt.Errorf("Zen is running — close it before manual push (daemon handles live writes)")
	}
	host, _ := os.Hostname()
	for _, f := range cfg.Files {
		if *dryRun {
			fmt.Fprintf(w, "+ would push %s\n", f)
			continue
		}
		if err := sync.Push(cfg.ZenProfile, cfg.SyncDir, f); err != nil {
			fmt.Fprintf(w, "  ⚠ push %s: %v\n", f, err)
		}
	}
	if !*dryRun {
		_ = sync.WriteLastPushHost(cfg.SyncDir, host)
		fmt.Fprintf(w, "✓ pushed (host=%s)\n", host)
	}
	return nil
}

func Pull(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print actions without performing them")
	force := fs.Bool("force", false, "pull even if this host was the last to push")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if profile.Running(cfg.ZenRunningPattern) {
		return fmt.Errorf("Zen is running — close it before manual pull")
	}
	host, _ := os.Hostname()
	last, _ := sync.ReadLastPushHost(cfg.SyncDir)
	if !*force && last == host {
		fmt.Fprintln(w, "Last push was from this host — skip (use --force to override).")
		return nil
	}
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".local", "state", "zen-sync", "backups")
	for _, f := range cfg.Files {
		if *dryRun {
			fmt.Fprintf(w, "+ would pull %s\n", f)
			continue
		}
		if err := sync.Pull(cfg.ZenProfile, cfg.SyncDir, backupDir, f); err != nil {
			fmt.Fprintf(w, "  ⚠ pull %s: %v\n", f, err)
		}
		if err := sync.RotateBackups(backupDir, f, cfg.Backup.Keep); err != nil {
			fmt.Fprintf(w, "  ⚠ rotate %s: %v\n", f, err)
		}
	}
	if !*dryRun {
		fmt.Fprintln(w, "✓ pulled")
	}
	return nil
}
```

- [ ] **Step 2: Implement `status`**

Create `internal/cli/status.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/gustavoguarda/zen-sync/internal/sync"
)

func Status(w io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	last, _ := sync.ReadLastPushHost(cfg.SyncDir)

	fmt.Fprintf(w, "host:           %s\n", host)
	fmt.Fprintf(w, "sync_dir:       %s\n", cfg.SyncDir)
	fmt.Fprintf(w, "zen_profile:    %s\n", cfg.ZenProfile)
	fmt.Fprintf(w, "last-push-host: %s\n", last)
	if last == host {
		fmt.Fprintln(w, "  → this Mac was the last to push")
	}
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tLOCAL HASH\tSYNC HASH\tMATCH")
	for _, f := range cfg.Files {
		local, _ := sync.Hash(filepath.Join(cfg.ZenProfile, f))
		remote, _ := sync.Hash(filepath.Join(cfg.SyncDir, f))
		match := "no"
		if local != "" && local == remote {
			match = "yes"
		} else if local == "" && remote == "" {
			match = "n/a"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", f, short(local), short(remote), match)
	}
	return tw.Flush()
}

func short(h string) string {
	if len(h) < 12 {
		return h
	}
	return h[:12] + "…"
}
```

- [ ] **Step 3: Implement `doctor`**

Create `internal/cli/doctor.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gustavoguarda/zen-sync/internal/profile"
)

type check struct {
	name string
	fn   func() (string, error)
}

func Doctor(w io.Writer) error {
	cfg, cfgErr := loadConfig()

	checks := []check{
		{"config readable", func() (string, error) {
			if cfgErr != nil {
				return "", cfgErr
			}
			return "ok", nil
		}},
	}

	if cfg != nil {
		checks = append(checks,
			check{"sync_dir exists + writable", func() (string, error) {
				if err := os.MkdirAll(cfg.SyncDir, 0o700); err != nil {
					return "", err
				}
				probe := filepath.Join(cfg.SyncDir, ".zen-sync-probe")
				if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
					return "", err
				}
				_ = os.Remove(probe)
				return cfg.SyncDir, nil
			}},
			check{"zen_profile exists", func() (string, error) {
				if _, err := os.Stat(cfg.ZenProfile); err != nil {
					return "", err
				}
				return cfg.ZenProfile, nil
			}},
			check{"daemon running (LaunchAgent loaded)", func() (string, error) {
				out, err := exec.Command("launchctl", "list").Output()
				if err != nil {
					return "", err
				}
				if !strings.Contains(string(out), launchAgentLabel) {
					return "", fmt.Errorf("LaunchAgent %s not loaded", launchAgentLabel)
				}
				return "loaded", nil
			}},
			check{"Zen running?", func() (string, error) {
				if profile.Running(cfg.ZenRunningPattern) {
					return "yes (manual push/pull will be blocked)", nil
				}
				return "no", nil
			}},
			check{"launcher app installed", func() (string, error) {
				p := "/Applications/Zen Sync.app/Contents/MacOS/launcher"
				if _, err := os.Stat(p); err != nil {
					return "", err
				}
				return "/Applications/Zen Sync.app", nil
			}},
		)
	}

	failed := 0
	for _, c := range checks {
		val, err := c.fn()
		if err != nil {
			fmt.Fprintf(w, "✗ %s: %v\n", c.name, err)
			failed++
		} else {
			fmt.Fprintf(w, "✓ %s: %s\n", c.name, val)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d check(s) failed", failed)
	}
	return nil
}
```

- [ ] **Step 4: Implement `restore`**

Create `internal/cli/restore.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gustavoguarda/zen-sync/internal/profile"
)

func Restore(w io.Writer, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".local", "state", "zen-sync", "backups")

	if len(args) == 0 {
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			return fmt.Errorf("no backups dir at %s", backupDir)
		}
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		fmt.Fprintln(w, "Backups (most recent first):")
		for _, n := range names {
			fmt.Fprintf(w, "  %s\n", n)
		}
		fmt.Fprintln(w, "\nUsage: zen-sync restore <name>")
		return nil
	}

	if profile.Running(cfg.ZenRunningPattern) {
		return fmt.Errorf("Zen is running — close it before restore")
	}

	name := args[0]
	src := filepath.Join(backupDir, name)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup not found: %s", name)
	}

	// Derive original filename: strip ".<timestamp>"
	original := name
	if i := strings.LastIndex(name, "."); i > 0 {
		original = name[:i]
	}
	allowed := false
	for _, f := range cfg.Files {
		if f == original {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("backup name does not map to a tracked file (got %q)", original)
	}

	dst := filepath.Join(cfg.ZenProfile, original)
	// safety snapshot of current local
	if _, err := os.Stat(dst); err == nil {
		ts := name[strings.LastIndex(name, ".")+1:]
		safety := filepath.Join(backupDir, fmt.Sprintf("%s.before-restore.%s", original, ts))
		_ = copyTo(dst, safety)
	}
	if err := copyTo(src, dst); err != nil {
		return err
	}
	fmt.Fprintf(w, "✓ restored %s from %s\n", original, name)
	return nil
}

func copyTo(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
```

- [ ] **Step 5: Implement `uninstall`**

Create `internal/cli/uninstall.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gustavoguarda/zen-sync/internal/launcher"
	"github.com/gustavoguarda/zen-sync/internal/plist"
)

func Uninstall(w io.Writer) error {
	home, _ := os.UserHomeDir()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := plist.Uninstall(launchAgentLabel, plistDir); err != nil {
		fmt.Fprintf(w, "  ⚠ remove LaunchAgent: %v\n", err)
	} else {
		fmt.Fprintln(w, "✓ removed LaunchAgent")
	}
	if err := launcher.Uninstall("/Applications/Zen Sync.app"); err != nil {
		fmt.Fprintf(w, "  ⚠ remove launcher app: %v\n", err)
	} else {
		fmt.Fprintln(w, "✓ removed /Applications/Zen Sync.app")
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Kept (delete manually if you want them gone):")
	fmt.Fprintf(w, "  config:  %s\n", filepath.Join(home, ".config", "zen-sync"))
	fmt.Fprintf(w, "  logs:    %s\n", filepath.Join(home, "Library", "Logs", "zen-sync"))
	fmt.Fprintf(w, "  backups: %s\n", filepath.Join(home, ".local", "state", "zen-sync"))
	fmt.Fprintln(w, "  sync_dir (whatever you configured): kept")
	return nil
}
```

- [ ] **Step 6: Wire all into main.go**

The `cmd/zen-sync/main.go` switch should now look like (paste into the existing switch):

```go
	case "version", "-v", "--version":
		err = cli.Version(os.Stdout, version, commit, buildDate)
	case "init":
		err = cli.Init(os.Stdin, os.Stdout)
	case "open":
		err = cli.Open(os.Stdout, args)
	case "push":
		err = cli.Push(os.Stdout, args)
	case "pull":
		err = cli.Pull(os.Stdout, args)
	case "daemon":
		err = cli.Daemon(os.Stdout)
	case "status":
		err = cli.Status(os.Stdout)
	case "doctor":
		err = cli.Doctor(os.Stdout)
	case "restore":
		err = cli.Restore(os.Stdout, args)
	case "uninstall":
		err = cli.Uninstall(os.Stdout)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "zen-sync: unknown command %q\n", cmd)
		usage()
		os.Exit(2)
```

- [ ] **Step 7: Build**

```bash
go build ./...
```

Expected: build succeeds.

- [ ] **Step 8: SKIP commit per user preference.**

---

## Task 12: CI workflow + linter config

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.golangci.yaml`

- [ ] **Step 1: Write `.golangci.yaml`**

Create `~/Projects/zen-sync/.golangci.yaml`:

```yaml
run:
  timeout: 5m
  go: "1.23"

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell

linters-settings:
  goimports:
    local-prefixes: github.com/gustavoguarda/zen-sync

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

- [ ] **Step 2: Write the CI workflow**

Create `~/Projects/zen-sync/.github/workflows/ci.yml`:

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: macos-14
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true
      - name: go vet
        run: go vet ./...
      - name: go test
        run: go test -race -cover ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

(Tests run on macos-14 because `internal/profile` shells out to `pgrep`; lint can run anywhere.)

- [ ] **Step 3: SKIP commit per user preference.**

---

## Task 13: Docs (README + INSTALL + ARCHITECTURE + TROUBLESHOOTING + CHANGELOG)

**Files:**
- Create: `README.md`
- Create: `docs/INSTALL.md`
- Create: `docs/ARCHITECTURE.md`
- Create: `docs/TROUBLESHOOTING.md`
- Create: `CHANGELOG.md`

- [ ] **Step 1: Write `README.md`**

Create `~/Projects/zen-sync/README.md`:

```markdown
# zen-sync

Arc-like continuity for [Zen Browser](https://zen-browser.app) on macOS.
Work on Mac A with Zen open. Walk over to Mac B. Click Zen Sync in the
Dock. Your workspaces, pinned tabs, and essentials are exactly where you
left them.

## Why

Firefox Sync (which Zen inherits) covers bookmarks, history, extensions,
and passwords. It does **not** cover Zen's workspaces and tab state —
the things that make Zen feel like Zen. That gap is the #1 complaint from
Arc refugees migrating to Zen.

zen-sync fills it without touching the browser. It watches the on-disk
state files (`zen-sessions.jsonlz4`, `zen-live-folders.jsonlz4`,
`containers.json`), pushes them to a folder of your choice, and pulls
them back before Zen launches on the other Mac. You bring the transport
(Syncthing, iCloud Drive, Dropbox — anything that syncs a folder).

## Install

```sh
brew install gustavoguarda/zen-sync/zen-sync
zen-sync init
```

`init` walks you through:
- where the sync folder lives
- detecting your Zen profile
- installing a LaunchAgent (background daemon)
- installing `/Applications/Zen Sync.app` (drag to Dock)
- a first push so your current state becomes the source of truth

See [docs/INSTALL.md](docs/INSTALL.md) for transport setup details
(Syncthing recommended).

## How it works

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the high-level. TL;DR:
a small Go binary runs in the background watching three files, copies them
to your sync folder on change, and a Dock launcher swaps them back in
before launching Zen on the other Mac.

## Status

Early. v0.1 is macOS-only and helper-only (no browser extension yet).
Linux and Windows are wanted contributions.

## License

MIT
```

- [ ] **Step 2: Write `docs/INSTALL.md`**

Create `~/Projects/zen-sync/docs/INSTALL.md`:

```markdown
# Install guide

## 1. Install zen-sync

```sh
brew install gustavoguarda/zen-sync/zen-sync
zen-sync init
```

## 2. Pick a transport

zen-sync writes to a folder. Anything that syncs that folder works.
Recommended:

### Syncthing (recommended — P2P, E2E, free)

```sh
brew install syncthing
brew services start syncthing
# Open http://localhost:8384, add ~/BrowserSync/Zen as a folder,
# share with your other Mac.
```

### iCloud Drive

Point `sync_dir` at `~/Library/Mobile Documents/com~apple~CloudDocs/zen-sync`.
Apple replicates. Encryption is iCloud's.

### Dropbox

Point `sync_dir` at `~/Dropbox/zen-sync`.

### USB drive / Time Capsule

Whatever syncs the folder is fine. zen-sync only cares about filesystem.

## 3. Repeat on the second Mac

Install zen-sync there too. `zen-sync init`, point it at the same shared
folder. When the transport replicates, click `Zen Sync.app` and you're up.

## 4. Verify

```sh
zen-sync status      # hashes should match between sync_dir and profile
zen-sync doctor      # all checks should be ✓
```
```

- [ ] **Step 3: Write `docs/ARCHITECTURE.md`**

Create `~/Projects/zen-sync/docs/ARCHITECTURE.md`:

```markdown
# Architecture

High-level summary. For decisions, see [the design spec](specs/2026-06-01-zen-sync-v01-design.md).

## Components

Single Go binary with three modes:

1. **Daemon** (`zen-sync daemon`) — launched by a LaunchAgent at login.
   Watches `zen-sessions.jsonlz4`, `zen-live-folders.jsonlz4`, and
   `containers.json` via fsnotify. On change, debounces 1s, hashes (skip
   if identical), then copies to `sync_dir`. Stamps `last-push-host`.

2. **Launcher** (`Zen Sync.app` calling `zen-sync open`) — installed at
   `/Applications/Zen Sync.app`. On click: pulls fresh state (unless this
   host was the last to push), then `open -na Zen`.

3. **CLI** (`zen-sync init|status|doctor|restore|uninstall|...`) — config,
   diagnostics, recovery.

## Files synced

| File | Contents |
|---|---|
| `zen-sessions.jsonlz4` | workspaces + tabs + pinned + essentials |
| `zen-live-folders.jsonlz4` | live folders |
| `containers.json` | contextual identities |

Nothing else. Firefox Sync handles bookmarks/history/extensions/passwords.

## Conflict model

Last-write-wins per file. Documented limitation: don't have Zen open
simultaneously on two Macs. `zen-sync doctor` will warn if it detects
both sides pushing recently.

## Transport

User's choice. zen-sync writes to a directory; whatever syncs that
directory (Syncthing/iCloud/Dropbox) handles transport-level concerns
(encryption, conflict at the byte level, replication).
```

- [ ] **Step 4: Write `docs/TROUBLESHOOTING.md`**

Create `~/Projects/zen-sync/docs/TROUBLESHOOTING.md`:

```markdown
# Troubleshooting

Start with `zen-sync doctor`. Each failing check below maps to a fix.

## ✗ config readable

`zen-sync init` hasn't been run, or `~/.config/zen-sync/config.toml` was
deleted. Run `zen-sync init`.

## ✗ sync_dir exists + writable

`sync_dir` points at a path you don't have write access to. Edit
`~/.config/zen-sync/config.toml` and fix the path, then `launchctl unload`
+ `launchctl load` the LaunchAgent (or just reboot).

## ✗ zen_profile exists

Your Zen profile path moved (Zen reinstalled with a new UUID prefix, or
you renamed the profile). Run `zen-sync init` again — it re-detects.
Or set `zen_profile` manually in the config.

## ✗ daemon running (LaunchAgent loaded)

```sh
launchctl load ~/Library/LaunchAgents/io.github.gustavoguarda.zen-sync.daemon.plist
```

If that fails, the plist is malformed — `zen-sync uninstall` then
`zen-sync init` to reinstall it clean.

## ✗ launcher app installed

```sh
zen-sync init
```

Re-runs the launcher install step.

## Zen opens but my state is stale

- `zen-sync status` — does the sync_dir have newer hashes than the local
  profile?
- Was Zen already running when you launched `Zen Sync.app`? It won't
  pull while Zen runs. Quit Zen, then click Zen Sync.app.

## Both Macs were open simultaneously

Last-write-wins. The Mac that pushed most recently is the source of
truth. Recover the other Mac's state from `~/.local/state/zen-sync/backups/`
via `zen-sync restore`.

## Daemon eats CPU

Should be ~0% idle. If not, check `~/Library/Logs/zen-sync/daemon.log`
for an error loop. File a bug with the log excerpt.

## Gatekeeper blocks Zen Sync.app

Right-click → Open the first time. macOS will then trust it. v0.1 is
unsigned; notarization is on the roadmap.
```

- [ ] **Step 5: Write `CHANGELOG.md`**

Create `~/Projects/zen-sync/CHANGELOG.md`:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial helper-only release for macOS.
- `init`, `open`, `push`, `pull`, `daemon`, `status`, `doctor`, `restore`,
  `uninstall`, `version` subcommands.
- LaunchAgent + `/Applications/Zen Sync.app` install via `init`.
- BYO transport (folder-based; Syncthing recommended).
```

- [ ] **Step 6: SKIP commit per user preference.**

---

## Task 14: Release pipeline (GoReleaser + Homebrew formula stub) + issue templates

**Files:**
- Create: `.goreleaser.yaml`
- Create: `.github/workflows/release.yml`
- Create: `.github/ISSUE_TEMPLATE/bug.yaml`
- Create: `.github/ISSUE_TEMPLATE/feature.yaml`

- [ ] **Step 1: Write `.goreleaser.yaml`**

Create `~/Projects/zen-sync/.goreleaser.yaml`:

```yaml
version: 2

project_name: zen-sync

before:
  hooks:
    - go mod tidy

builds:
  - id: zen-sync
    main: ./cmd/zen-sync
    binary: zen-sync
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.buildDate={{.Date}}

archives:
  - id: default
    formats: [tar.gz]
    name_template: "{{ .ProjectName }}_v{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

snapshot:
  version_template: "{{ incpatch .Version }}-next"

changelog:
  use: github
  sort: asc

brews:
  - name: zen-sync
    repository:
      owner: gustavoguarda
      name: homebrew-zen-sync
    homepage: "https://github.com/gustavoguarda/zen-sync"
    description: "Arc-like continuity for Zen Browser on macOS"
    license: "MIT"
    install: |
      bin.install "zen-sync"
    test: |
      system "#{bin}/zen-sync", "version"
```

- [ ] **Step 2: Write the release workflow**

Create `~/Projects/zen-sync/.github/workflows/release.yml`:

```yaml
name: release

on:
  push:
    tags:
      - "v*.*.*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: macos-14
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

(`HOMEBREW_TAP_GITHUB_TOKEN` is a PAT with write access to `gustavoguarda/homebrew-zen-sync`. Set it as a repo secret before tagging the first release.)

- [ ] **Step 3: Write issue templates**

Create `~/Projects/zen-sync/.github/ISSUE_TEMPLATE/bug.yaml`:

```yaml
name: Bug report
description: Something is broken
labels: [bug]
body:
  - type: textarea
    id: doctor
    attributes:
      label: Output of `zen-sync doctor`
      render: shell
    validations:
      required: true
  - type: textarea
    id: what
    attributes:
      label: What happened?
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: What did you expect?
  - type: input
    id: zen-version
    attributes:
      label: Zen Browser version
  - type: input
    id: macos-version
    attributes:
      label: macOS version
```

Create `~/Projects/zen-sync/.github/ISSUE_TEMPLATE/feature.yaml`:

```yaml
name: Feature request
description: Suggest an improvement
labels: [enhancement]
body:
  - type: textarea
    id: what
    attributes:
      label: What would you like to see?
    validations:
      required: true
  - type: textarea
    id: why
    attributes:
      label: Why does this matter for your workflow?
```

- [ ] **Step 4: SKIP commit per user preference.**

---

## Task 15: Final end-to-end smoke verification (manual, user-driven)

**Files:** none modified — final verification before tagging v0.1.0.

This task is the user's hands-on validation. It mirrors what the shell
smoke test exercised, against the real Go binary, against the real Zen
profile.

- [ ] **Step 1: Full test pass**

```bash
go test -race -cover ./...
```

Expected: ALL tests PASS. No race warnings.

- [ ] **Step 2: Build the binary**

```bash
go build -o ./zen-sync ./cmd/zen-sync
./zen-sync version
```

Expected: prints `dev`, `none`, `unknown` (no ldflags from local build).

- [ ] **Step 3: Move binary somewhere on PATH and run `init` against a TEST sync_dir**

Important: use a throwaway sync_dir like `/tmp/zen-sync-validate/` so the
real `~/BrowserSync/Zen` isn't touched.

```bash
mkdir -p ~/.local/bin && cp ./zen-sync ~/.local/bin/
~/.local/bin/zen-sync init
# When prompted for sync_dir, enter: /tmp/zen-sync-validate
# Answer "y" to initial push
```

Expected: config saved, LaunchAgent installed, Zen Sync.app installed,
first push completes.

- [ ] **Step 4: Verify the sync_dir got populated**

```bash
ls -la /tmp/zen-sync-validate/
cat /tmp/zen-sync-validate/.meta/last-push-host
```

Expected: the three tracked files (or whichever exist in the profile) +
.meta directory + this host's name.

- [ ] **Step 5: Verify the daemon is running**

```bash
launchctl list | grep zen-sync
~/.local/bin/zen-sync doctor
```

Expected: LaunchAgent listed; all doctor checks ✓.

- [ ] **Step 6: Provoke a change and watch the daemon push**

Open Zen, open a new tab, pin it, close that tab. Watch the log:

```bash
tail -f ~/Library/Logs/zen-sync/daemon.log
```

Expected: within ~2s of the tab pin/close, the log shows `daemon: pushed zen-sessions.jsonlz4 (hash=...)`.

- [ ] **Step 7: Clean up the test install**

```bash
~/.local/bin/zen-sync uninstall
rm -rf /tmp/zen-sync-validate
rm -rf ~/.config/zen-sync
rm -rf ~/Library/Logs/zen-sync
rm -rf ~/.local/state/zen-sync
rm -f ~/.local/bin/zen-sync
```

Expected: LaunchAgent unloaded, Zen Sync.app removed, all helper state gone.

- [ ] **Step 8: Tag and release (when satisfied)**

```bash
git tag v0.1.0
git push origin v0.1.0
```

(Only after user has committed everything manually. The release workflow runs and publishes.)

- [ ] **Step 9: Verify install from brew**

```bash
brew tap gustavoguarda/zen-sync
brew install zen-sync
zen-sync version
```

Expected: prints `v0.1.0`, real commit, real build date.

No commit for this task (no source files changed).

---

## Self-Review Checklist (verify before executor handoff)

- ✅ **Spec coverage:** every section of the spec maps to a task.
  - Architecture (3 components, single binary) → Tasks 2, 7, 8, 9, 10, 11
  - Daemon push flow → Task 7
  - Pull+launch flow → Task 10 (`open`)
  - Config + paths → Task 4 (config) + Task 10/11 (use)
  - Conflict resolution (last-push-host) → Tasks 6, 7, 10, 11
  - Files synced (3 specific) → Task 4 (defaults) + Task 6 (fixtures)
  - Profile detection → Task 5
  - Process detection (`zen_running`) → Task 5
  - Repo layout → all tasks combined
  - Distribution (GoReleaser + brew tap) → Task 14
  - Acceptance criteria items → covered across Tasks 4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 15
  - Migration (zen-sync2 left alone, dotfiles deprecated later) → out of plan; documented decision in spec

- ✅ **No placeholders:** every step has complete code or exact commands.

- ✅ **Type consistency:**
  - `config.Config` struct fields used identically in `cli/daemon_cmd.go`, `cli/open.go`, `cli/push_pull.go`, `cli/status.go`, `cli/doctor.go`, `cli/restore.go`, `cli/uninstall.go`.
  - `sync.Hash`, `sync.Push`, `sync.Pull`, `sync.RotateBackups`, `sync.ReadLastPushHost`, `sync.WriteLastPushHost` declared in Task 6 and called unchanged in Tasks 7, 10, 11.
  - `daemon.Config{Profile, SyncDir, Files, Hostname, DebounceMs}` declared in Task 7 and constructed identically in `cli/daemon_cmd.go`.
  - `profile.Detect`, `profile.Running` declared in Task 5 and called unchanged in Tasks 10, 11.
  - `plist.Install`, `plist.Uninstall`, `launcher.Install`, `launcher.Uninstall` declared in Tasks 8, 9 and called unchanged in Tasks 10, 11.
  - Constants `launchAgentLabel` and `launcherBundleID` declared in `cli/init.go` (Task 10) and reused in `cli/doctor.go` and `cli/uninstall.go` (Task 11) via the same `cli` package.

- ✅ **Commit step handling:** every task ends with "SKIP commit per user preference" so the executor doesn't auto-commit. The user retains full control of git history.
