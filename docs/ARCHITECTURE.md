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
