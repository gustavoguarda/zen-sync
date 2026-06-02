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
