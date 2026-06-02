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
