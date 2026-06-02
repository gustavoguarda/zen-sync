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
- App bundle's `CFBundleShortVersionString` is hardcoded to `0.1.0`. Wire
  to build-time version when bumping past v0.1.x.
