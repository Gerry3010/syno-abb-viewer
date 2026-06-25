# syno-abb-viewer

A small Go desktop GUI for browsing and inspecting Synology DiskStation backups
over SSH/SFTP. Built with [Fyne](https://fyne.io) — pure Go, single binary, with
a dark terminal aesthetic.

> The name nods to Synology *Active Backup for Business*. Milestone 1 is a generic
> remote-filesystem browser; ABB-specific views come later.

## Status

- [x] Connect over SSH with **key or password** auth (host-key verified via `known_hosts`)
- [x] Browse the backup directory tree, lazily loaded over SFTP
- [x] Dark terminal theme (monospace)
- [x] View backup runs (dated folders, with size / file count)
- [x] Inspect DB dump files (stream `.sql.gz`, list tables)
- [ ] Compare two backup runs
- [ ] Download / extract individual files

## Build & run

```sh
make run        # or: go run ./cmd/syno-abb-viewer
make build      # builds ./syno-abb-viewer
make test       # runs unit tests (no DiskStation needed)
```

Requires Go 1.26+ and the system libraries Fyne needs (OpenGL / X11 or Wayland
dev headers on Linux).

## Configuration

Settings are entered in the in-app **Connect** dialog and persisted to
`~/.config/syno-abb-viewer/config.json`. Defaults: port `22`, key
`~/.ssh/id_ed25519`, root path `/volume1`.

SSH passwords and key passphrases are **never written to disk** — they are
entered each time you connect.

## Security notes

- Host keys are checked against `~/.ssh/known_hosts`. An unknown host prompts a
  trust-on-first-use confirmation showing the SHA256 fingerprint; a changed key
  is refused outright.

## License

MIT — see [LICENSE](LICENSE).
