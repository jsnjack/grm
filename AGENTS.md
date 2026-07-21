# AGENTS.md

> See [AGENTS.universal.md](./AGENTS.universal.md) and [AGENTS.go.md](./AGENTS.go.md) for universal conventions.
> Refresh: `make standards`

---

## Overview

`grm` is a CLI package manager that installs binaries distributed as GitHub
Releases (typically Go/Rust tools with no package-manager distribution of
their own). It inspects a release's assets, picks the one that matches the
current OS/architecture, downloads and installs it (as a plain binary, an
archive, or a `.deb`/`.rpm` system package), and tracks what's installed in a
local YAML config. Several other `jsnjack` tools (`monova`, `mech`,
`sslcheck`, ...) are distributed for installation via `grm`, and `grm` itself
self-updates via `grm install jsnjack/grm` — so its own release process and
install UX are load-bearing for other projects.

---

## Architecture

```
main.go                        Entry point; delegates to cmd.Execute()
cmd/
  root.go                      Root command, persistent flags, PersistentPreRunE
                                (wires the logger and resolves ConfigFile)
  logger.go                    slog setup for --debug (stderr) / --trace (file)
  main_config.go                GrmConfig (read/save), resolveConfigFile (XDG + legacy fallback)
  main_package.go               Package struct, CreatePackage (parses owner/repo[==v|@v|~=filter])
  main_utils.go                 Shared helpers: download, install/remove binary, GitHub client,
                                 interactive prompts, misc string/hash utils
  install.go                    `install` command; release/asset selection logic
  install_init.go                Install() — routes a downloaded asset to binary/archive/system-package
  install_archive.go             Archive unpacking + binary detection inside an archive
  install_system_package.go      .deb/.rpm install and removal via apt/dnf/dpkg/rpm
  info.go, list.go, lock.go,
  unlock.go, remove.go, set.go,
  settings.go, update.go,
  release.go, version.go,
  aliases.go                    One file per subcommand
  styles.go                     Terminal color/table helpers
```

---

## Key Flows

1. **Install** (`install.go`): `CreatePackage` parses the `owner/repo` spec →
   `selectRelease` resolves it to a GitHub release (latest, exact tag, or
   `~=` filter match) → `installRelease` calls `selectAsset` to pick the
   right asset for the current OS/arch/available package managers →
   `Install` (`install_init.go`) detects the asset's file type and routes to
   `installBinary`, `installArchive`, or `installSystemPackage` → the result
   is persisted via `GrmConfig.PutPackage`.
2. **Update** (`update.go`): checks all installed packages for newer
   releases concurrently (one goroutine per package), prints a combined
   status table, asks for one confirmation, then installs updates
   sequentially (installs need interactive prompts and serialize config
   writes).
3. **Config resolution** (`root.go` → `main_config.go`): on every invocation,
   `PersistentPreRunE` calls `resolveConfigFile`, which honors `--config` if
   given, otherwise uses `os.UserConfigDir()/grm/grm.yaml`, falling back to
   the legacy `~/.config/grm/grm.yaml` if only that exists.

---

## Build & Run

```bash
make build   # cross-compile bin/grm_{linux,darwin}_{amd64,arm64} + bin/grm symlink
make test    # go test ./...
make check   # fmt, vet, build, test, lint — run after every change
./grm --help
./grm install jsnjack/grm     # smoke test: self-install
./grm --debug list            # verify --debug logging to stderr
./grm --trace list; cat /tmp/grm.log   # verify --trace logging to file
```

`make release` builds, tars each platform binary, and pushes a GitHub release
via `grm release jsnjack/grm`. `bin/grm` (the raw linux/amd64 binary) is
listed first among the release's assets — the README's
`jq -r .assets[0].browser_download_url` manual-install one-liner depends on
that ordering.

---

## Configuration

- Config file: YAML at `~/.config/grm/grm.yaml` (or `$XDG_CONFIG_HOME/grm/grm.yaml`
  if set), overridable with `--config`/`-c`. Holds installed packages
  (`GrmConfig.Packages`) and settings (`GrmConfig.Settings`, currently just
  `token`).
- GitHub token resolution order: `--token` flag → `settings.token` in the
  config file → `GITHUB_TOKEN` env var → anonymous (rate-limited) client.
- `--debug`/`-d`: debug-level logs to stderr. `--trace`: trace-level logs to
  `/tmp/grm.log` (truncated every run). The two are independent and
  composable (see `cmd/logger.go`).

---

## Design Decisions

- **`--debug`/`-d` replaces the old `--verbose`/`-v`.** This is a breaking
  CLI change, made deliberately (not silently) when converting to the
  jsnjack/standards logging convention — confirmed with the maintainer
  rather than kept as a backward-compatible alias.
- **Other existing short flags were intentionally left alone.** The Go
  standard says only `--debug`/`-d` and `--config`/`-c` should have short
  aliases; `grm` has several older ones in real, documented use (`-y`, `-f`,
  `-r`, `-l`, `-n`, `-a`, `-t`). Stripping all of them would break scripts
  and muscle memory for a widely-installed tool with no way to
  auto-migrate. Only `list`'s `-d` (`--description`) was changed to
  long-form-only, because it directly collided with the new global `-d`.
- **Config directory keeps a legacy fallback instead of a hard XDG switch.**
  `resolveConfigFile` prefers `os.UserConfigDir()/grm/grm.yaml` but falls
  back to the pre-existing hardcoded `~/.config/grm/grm.yaml` if only that
  exists. A blind switch to `os.UserConfigDir()` would silently move (and
  appear to erase) existing macOS installs' package list, since macOS's
  standard config dir differs from the old hardcoded path.
- **The `grm` symlink at the repo root is intentionally git-tracked**
  (see its history — it was removed once and deliberately re-added).
  The standard Makefile pattern gitignores the binary name, but this
  project keeps it tracked on purpose; that part of the pattern was not
  applied here.
- **`--version` is a new root flag** (cobra's built-in support, wired via
  `rootCmd.Version`), registered without cobra's default `-v` shorthand so
  it doesn't reclaim the shorthand freed up from removing `--verbose`. The
  pre-existing `version` subcommand is unchanged.

---

## Gotchas

- `list -d` no longer means `--description` — it's the global `--debug`
  flag. Use `list --description` (long-form only).
- `Package` has no `yaml` struct tags, so `yaml.v3`'s default lower-cased
  field names apply when hand-editing `grm.yaml` (`owner`, `repo`,
  `version`, ... not `Owner`, `Repo`, `Version`).
- `resolveConfigFile`'s legacy fallback is checked once per run, at
  startup; there is no automatic migration from the legacy path to the
  XDG one.
