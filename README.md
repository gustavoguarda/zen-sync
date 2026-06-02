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

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the high-level. TL;DR:
a small Go binary runs in the background watching three files, copies them
to your sync folder on change, and a Dock launcher swaps them back in
before launching Zen on the other Mac.

## Status

Early. v0.1 is macOS-only and helper-only (no browser extension yet).
Linux and Windows are wanted contributions.

## License

MIT
