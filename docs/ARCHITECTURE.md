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

## Timing & mechanism

The daemon is event-driven, not polling. It blocks in a Go `select` and
is woken up by fsnotify only when a tracked file actually changes.

### One sync, end-to-end

| Step | Typical | What |
|---|---|---|
| Zen flushes `zen-sessions.jsonlz4` | up to 15s after your action | Firefox session save interval; instant on tab close / app quit / some events |
| fsnotify event delivers | ~1ms after flush | Daemon wakes from `select` |
| Debounce window | 1000ms | Daemon waits in case more events arrive (timer resets per event). Avoids reading a half-flushed file and collapses bursts. |
| SHA-256 hash | ~10ms | If equal to the last pushed hash, skip — Zen sometimes rewrites without changing content. |
| Atomic copy to `sync_dir` | ~50ms | Write to `<file>.tmp`, `fsync`, rename. Stamp `.meta/last-push-host`. |
| Syncthing replicates | 1-3s | Outside zen-sync; depends on file size and network. |

Total host action → bytes on the other Mac: typically **3-20 seconds**.
The big variable is Zen's own flush interval, not anything we control.

### fsnotify vs polling

|  | fsnotify (ours) | Polling |
|---|---|---|
| Latency | ~1s (debounce dominates) | Half the interval on average |
| Idle CPU | ~0% (blocks in kernel) | Each wake-up costs |
| Events lost under heavy load | Possible (rare on real workloads with 3 small files) | Doesn't happen |
| Reacts to create / rename / delete | Yes, distinguishable events | Has to recompute state every tick |

We accept the rare-event-loss risk for 3 small files in exchange for
~0% idle CPU and lower latency.

### Auto-heal on upgrade

`launcher.Install` and `plist.Install` embed an `IOZenSyncCommit` marker
recording which binary commit wrote the bundle/plist. On each
`zen-sync open` or `zen-sync ensure-installation`, the binary compares
the embedded marker with its own commit and rewrites the bundle if they
differ. This is what makes `zen-sync upgrade` deliver new icons / plist
tweaks / launcher stub changes without a manual `zen-sync init` after
the binary swap.
