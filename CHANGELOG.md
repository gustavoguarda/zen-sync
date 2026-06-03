# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.3] - 2026-06-03

### Added
- `zen-sync upgrade` subcommand wraps `brew update && brew upgrade zen-sync`
  so updates are a single command. Streams brew's output directly so the
  user sees familiar progress + the auto-heal lines from `post_install`.
  Refuses gracefully when the running binary wasn't installed via Homebrew
  (source builds, manual cp, etc).

## [0.1.2] - 2026-06-02

### Added
- Auto-heal of `/Applications/Zen Sync.app` and the LaunchAgent plist:
  both are now regenerated automatically whenever the build commit
  embedded in them differs from the running binary's. This makes
  `brew upgrade zen-sync` deliver new icons, plist tweaks, and other
  bundle changes without the user re-running `zen-sync init`.
- `zen-sync ensure-installation` subcommand for explicit refresh.
- Homebrew formula `post_install` hook calls `ensure-installation` so
  brew upgrades land the refresh before the user even clicks the launcher.
- `IOZenSyncCommit` marker comment embedded in generated plists; surfaces
  in `Info.plist` alongside a real `CFBundleShortVersionString`, which
  also lets macOS invalidate the icon cache automatically across upgrades.

### Changed
- `launcher.Install`, `plist.Install`, and `plist.Render` now take
  `version` and `commit` arguments. Internal API only — no user impact.

## [0.1.1] - 2026-06-02

### Added
- Custom icon for `Zen Sync.app` (Zen icon + sync badge) embedded via
  `go:embed` and written to `Contents/Resources/icon.icns`.

## [0.1.0] - 2026-06-02

### Added
- Initial helper-only release for macOS.
- `init`, `open`, `push`, `pull`, `daemon`, `status`, `doctor`, `restore`,
  `uninstall`, `version` subcommands.
- LaunchAgent + `/Applications/Zen Sync.app` install via `init`.
- BYO transport (folder-based; Syncthing recommended).

### Known limitations (track for v0.1.x / v0.2)
- Daemon shutdown doesn't wait for in-flight flush goroutines to finish
  (`internal/daemon/daemon.go`). Edge case under SIGTERM with a push
  mid-flight.
- No heartbeat poll as secondary safety net for missed fsnotify events.
  Spec calls for 30s poll; deferred (fsnotify is reliable enough for the
  three small files this watches in practice).
- `plist.Render` interpolates label/binary path into XML via `fmt.Sprintf`
  without escaping. Safe today (label is a constant), but unsafe if either
  becomes user-configurable.
- `internal/cli` and `internal/plist` packages lack unit tests for
  subprocess wrappers (`launchctl`, `open`). Covered by T15 manual smoke;
  integration tests are a v0.1.x follow-up.
