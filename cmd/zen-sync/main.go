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
	// Plumb build metadata into the cli package so its subcommands can
	// embed it in generated bundles (used by auto-heal in Ensure).
	cli.BuildVersion = version
	cli.BuildCommit = commit
	cli.BuildDate = buildDate

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
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
	case "ensure-installation":
		err = cli.EnsureInstallation(os.Stdout)
	case "upgrade":
		err = cli.Upgrade(os.Stdout)
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
}

func usage() {
	fmt.Fprint(os.Stderr, `zen-sync — Arc-like continuity for Zen Browser on macOS

Usage:
  zen-sync <command> [args]

Commands:
  init                  Setup wizard: config + LaunchAgent + Zen Sync.app + first push
  open                  Pull state then launch Zen
  push                  Manual push (debug / source-of-truth)
  pull                  Manual pull (debug / recovery)
  daemon                Long-running file watcher (called by LaunchAgent; not by user)
  status                Show last sync, hashes, last-push-host
  restore               List or restore backups
  doctor                Diagnose common problems
  upgrade               Refresh tap + upgrade to latest version via Homebrew
  ensure-installation   Refresh .app + LaunchAgent if a binary upgrade changed them
  uninstall             Remove LaunchAgent and Zen Sync.app
  version               Show build metadata

Run 'zen-sync help' for this list.
`)
}
