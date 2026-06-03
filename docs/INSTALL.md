# Install guide

## 1. Install zen-sync

```sh
brew install gustavoguarda/zen-sync/zen-sync
zen-sync init
```

`init` is an interactive wizard. It will:

1. **Ask where the sync folder should live.** It scans for known transport
   locations and lets you pick from a menu. Detected entries are marked
   `[detected]`; you can also pick `Custom path` for anywhere else.

   ```
   Where should zen-sync write state for your transport to replicate?

     1) Syncthing [detected]
        /Users/you/BrowserSync/Zen
     2) iCloud Drive [detected]
        /Users/you/Library/Mobile Documents/com~apple~CloudDocs/zen-sync
     3) Dropbox [not detected]
        /Users/you/Dropbox/zen-sync
     4) Custom path

   Pick [1]:
   ```

2. **Detect your Zen profile** under
   `~/Library/Application Support/zen/Profiles/`. Falls back through
   `*.Default (release)` and `*.Default` so nightly builds work too.

3. **Offer to fix your hostname** if it looks IP-derived (the macOS
   default on fresh installs is sometimes `192` from your local IP).
   Two Macs need distinct hostnames so zen-sync's `last-push-host`
   actually distinguishes them.

4. **Install the background pieces**: the LaunchAgent plist that runs the
   daemon, and `/Applications/Zen Sync.app` (drag to your Dock — use it
   instead of `Zen.app`).

5. **Pick a role automatically**:
   - Sync folder empty → this Mac is the source of truth, initial push happens.
   - Sync folder has state from another device → "joining" message, no push
     (so we don't clobber what's already there). You'll open Zen Sync.app
     to pull and launch with the synced state.

6. **Run `doctor`** to verify the install. All `✓` means you're done. Any
   `✗` and the doctor tells you exactly what's wrong.

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

## 5. Keeping up to date

```sh
zen-sync upgrade
```

Equivalent to `brew update && brew upgrade zen-sync`, plus a guaranteed
refresh of `/Applications/Zen Sync.app` and the LaunchAgent. Safe to
re-run anytime — no-op when nothing changed.

Why the wrapper exists: brew sometimes silently skips a tap's
`post_install` hook when the tap isn't "trusted" (a default in modern
brew). `zen-sync upgrade` calls `ensure-installation` explicitly after
the brew step, so the `.app` and LaunchAgent always pick up the new
binary's commit — regardless of trust state.
