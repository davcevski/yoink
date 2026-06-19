```
█   █   ███   █████  █   █  █   █
 █ █   █   █    █    ██  █  █  █
  █    █   █    █    █ █ █  ███
  █    █   █    █    █  ██  █  █
  █     ███   █████  █   █  █   █
```

# yoink

An **encrypted** macOS clipboard history with a terminal UI.

A tiny background daemon captures everything you copy; a fast terminal UI lets
you browse, search, and "yoink" any past clip back onto your clipboard. The
entire history is encrypted at rest, so a stolen laptop or a copied database
leaks nothing.

- AES-256-GCM at rest, key in the macOS Keychain, password-manager secrets skipped
- Always-on capture via `NSPasteboard` polling (no Accessibility permissions)
- Instant in-memory incremental search (substring + fuzzy)
- Homebrew installable, single self-contained binary

---

## Install

yoink ships its Homebrew cask inside this repo (a custom-URL tap):

```sh
brew tap davcevski/yoink https://github.com/davcevski/yoink
brew install davcevski/yoink/yoink
```

Then start the background capture daemon **once** (it runs at login from then on):

```sh
yoink install
```

> yoink encrypts clip history at rest. Enabling **FileVault** is recommended for
> full at-rest protection — `yoink install` warns you if it's off.

### Build from source

Requires Go 1.25+ and the Xcode Command Line Tools (for the cgo bindings).

```sh
git clone https://github.com/davcevski/yoink
cd yoink
go build -o yoink .
./yoink install
```

## Usage

```sh
yoink                 # launch the clipboard browser (TUI)
yoink --limit 10      # show 10 clips instead of the configured default
yoink install         # install + start the background daemon (run once)
yoink uninstall       # stop + remove the background daemon
yoink config          # print the config file path and contents
yoink config --edit   # open the config in $EDITOR
yoink version         # version info
yoink daemon          # run the capture loop (what the LaunchAgent invokes)
```

### TUI keys

| Key | Action |
|---|---|
| `↑`/`↓` or `j`/`k` | move selection |
| `Enter` | yoink the selected clip back to the clipboard, then quit |
| `d` | delete the selected clip |
| `/` | start a search (or just **start typing** to search incrementally) |
| `<` / `>` (or `Tab`) | cycle the display limit (5 / 10 / 20) |
| `Esc` | clear the search, or quit |
| `q` / `Ctrl+C` | quit |

> Because typing filters the list live, the delete/quit/limit actions also have
> chord equivalents (`Ctrl+D` delete, `Ctrl+C` quit) that work while searching.

## Configuration

`~/.config/yoink/config.toml`, created with defaults on first run:

```toml
poll_interval_ms = 500
retention_days   = 14
display_limit    = 20
max_clip_bytes   = 1048576   # skip clips larger than 1 MiB
```

History lives at `~/.config/yoink/history.db` (SQLite, WAL mode, owner-only).

## How it works

- **Capture:** the daemon polls `NSPasteboard.changeCount` (a free integer read)
  and only reads clipboard contents when it changes — negligible CPU. It works
  for any copy (⌘C, right-click, app buttons) with no special permissions.
- **Storage:** pure-Go SQLite (`modernc.org/sqlite`) bundled into the binary.
- **Encryption:** each clip is sealed with AES-256-GCM under a 256-bit key kept
  in the login Keychain; dedup uses a keyed HMAC-SHA256 so plaintext is never
  compared or stored. See [SECURITY.md](SECURITY.md) for the full threat model.
- **Daemon:** a `launchctl` LaunchAgent (`com.yoink.daemon`) runs it at login
  with `KeepAlive`. Logs go to `~/.config/yoink/yoinkd.log`.

## Development

```sh
go test ./...                 # unit tests (fakes for clipboard/keyring)
go test -race ./...           # with the race detector
go vet ./...

# Opt-in live tests that touch real macOS subsystems:
YOINK_LIVE_CLIPBOARD=1 go test ./internal/clipboard/
YOINK_LIVE_KEYCHAIN=1  go test ./internal/keyring/
YOINK_LIVE_E2E=1       go test ./internal/daemon/
```

The native `NSPasteboard`/Keychain glue is the only code outside the normal
suite; the live tests above exercise it on demand.

## Releasing

Releases are cut by [GoReleaser](https://goreleaser.com) on a version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The `Release` workflow (macOS runner) builds a universal (arm64 + amd64),
ad-hoc-signed binary, publishes the GitHub release with checksums, and commits
`Casks/yoink.rb` back to this repo — all with the built-in `GITHUB_TOKEN`, no
extra secrets. The repo must be **public** for `brew` to fetch release assets.

## License

[MIT](LICENSE) © Mario Davcevski
