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

### "But Zen has a Workspaces checkbox in the sync settings, doesn't it?"

It does. It doesn't work. As of 2026, multiple open issues confirm that
toggling **Workspaces** in the Mozilla account sync preferences has no
effect — names and icons never appear on the second device:

- [#11407 — Workspaces are not synced across different devices on a single firefox account](https://github.com/zen-browser/desktop/issues/11407)
- [#12339 — Workspaces aren't syncing to a new device](https://github.com/zen-browser/desktop/issues/12339)
- [#12749 — Workspaces not syncing across devices via Mozilla account](https://github.com/zen-browser/desktop/issues/12749)

The fix requires Mozilla to extend the Firefox Sync schema, which is an
upstream dependency outside Zen's control. zen-sync sidesteps the problem
entirely by syncing the on-disk state files instead of going through the
account.

## Install

```sh
brew install gustavoguarda/zen-sync/zen-sync
zen-sync init
```

`init` walks you through:
- picking a sync folder (auto-detects Syncthing, iCloud Drive, Dropbox)
- detecting your Zen profile
- fixing your hostname if it looks IP-derived (so the two Macs can be told apart)
- installing a LaunchAgent (background daemon)
- installing `/Applications/Zen Sync.app` (drag to Dock)
- doing an initial push if this is the first device, or a clean join if the sync folder already has state
- running `doctor` at the end so any setup gap is visible right away

See [docs/INSTALL.md](docs/INSTALL.md) for transport setup details
(Syncthing recommended).

## How it works

The daemon doesn't poll on an interval — it sleeps in a kernel `select`
and is woken up by fsnotify only when one of the tracked files actually
changes.

End-to-end timeline of one sync (host change → bytes arrive on the other Mac):

| Step | When | What happens |
|---|---|---|
| 0 | T+0ms | You pin a tab, create a workspace, etc. in Zen |
| 1 | T+up to 15s | Zen flushes `zen-sessions.jsonlz4` to disk (Firefox's session save interval; instant on tab close, app quit, or some events) |
| 2 | T+~1ms after flush | fsnotify event delivers; daemon wakes from `select` |
| 3 | T+1000ms after flush | Debounce timer expires (1s so rapid writes collapse into one push and we don't read a half-flushed file) |
| 4 | T+~10ms | SHA-256 of the file. If unchanged from last push, skip (Zen sometimes rewrites with same content). |
| 5 | T+~50ms | Atomic copy to `sync_dir` (write to temp + `fsync` + rename). Stamp `.meta/last-push-host` with our hostname. |
| 6 | T+1-3s | Syncthing on the host detects the change and ships it across. |
| 7 | T+<1s | Syncthing on the other Mac writes the file. Bytes match the host now. |

**Total host action → other Mac has the bytes**: typically 3-20 seconds.
The wide range is mostly Zen's own flush interval, not anything zen-sync
controls.

**Idle CPU**: effectively 0%. The daemon blocks in `select`; no CPU is
used until fsnotify wakes it.

To watch the daemon in real time:

```sh
tail -f ~/Library/Logs/zen-sync/daemon.log
# You'll see: daemon: pushed zen-sessions.jsonlz4 (hash=a5823f19…)
```

To check the current sync state:

```sh
zen-sync status
# Local hash vs sync hash per file + last-push-host
```

For mechanism details (fsnotify vs polling trade-offs, debounce rationale,
hash-check semantics), see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Keeping up to date

```sh
zen-sync upgrade
```

That wraps `brew update && brew upgrade zen-sync` and explicitly refreshes
the `.app` bundle + LaunchAgent in one step. Safe to re-run anytime — it's
a no-op when nothing changed.

Use it instead of the manual `brew upgrade zen-sync` so you never have to
remember to `brew trust` the tap, re-run `init`, or kill Finder for icon
cache.

## Status

Early. v0.1 is macOS-only and helper-only (no browser extension yet).
Linux and Windows are wanted contributions.

## License

MIT
