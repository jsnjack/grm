# AGENTS.md

> See [AGENTS.universal.md](./AGENTS.universal.md) and [AGENTS.go.md](./AGENTS.go.md) for universal conventions.
> Refresh: `make standards`

---

## Overview

`grm` is a CLI package manager that installs binaries distributed as GitHub
Releases (typically Go/Rust tools with no package-manager distribution of
their own). It inspects a release's assets, selects files matching the
current OS/architecture, downloads and installs their selected binaries or
`.deb`/`.rpm` system packages, and tracks every component in a local YAML
config. Several other `jsnjack` tools (`monova`, `mech`,
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
  install_init.go                Prepares downloaded binaries, archives, and system packages
  install_selection.go           Multi-selection prompts, strict asset/binary selectors, remembered patterns
  install_plan.go                Plans, validates, and applies component installs; persists partial progress
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
   `~=` filter match) → `installRelease` resolves saved selections or prompts
   for multiple assets (compatible RPMs first) → `prepareAsset` inspects the
   downloaded contents and selects archive binaries → the complete plan
   is checked for conflicting destinations and printed; it asks for
   confirmation only when it removes old components → each successful
   binary/system-package operation is persisted. Cleanup belongs to the whole
   batch. The repository stays incomplete until every selected component is
   installed and obsolete components are removed.
2. **Update** (`update.go`): checks all installed packages for newer
   releases concurrently (one goroutine per package), prints a combined
   status table, asks for confirmation, then prepares and applies each
   installation plan sequentially (plans that remove components ask again).
   Installs serialize config writes; failed updates remain incomplete and
   return a nonzero exit status. Release lookup failures also return a
   nonzero exit status.
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
  (`GrmConfig.Packages`) and settings (`GrmConfig.Settings`: `token` and
  `bin_dir`).
- Binaries install to `GrmConfig.BinDir()`: the `bin_dir` setting when set,
  otherwise `DefaultBinDir` (`/usr/local/bin/`).
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
- **A repository owns multiple components at one release version.** `Selections`
  remembers asset patterns and archive member paths/renames; `Components`
  records each binary's path/hash or each system package's name/type/version.
  Legacy singular fields remain readable and migrate when installing. New
  records require the new grm version. Locks apply to the entire repository.
- **Legacy migration keeps the installed selection.** Without explicit asset,
  binary, or filter changes, legacy binary installs exclude DEB/RPM candidates
  and keep one binary at its existing installed name. Renamed legacy archives
  keep the old first-binary behavior because their source member was not saved.
  Legacy system packages keep their type and package name. Missing binaries or
  changed system package names fail before applying the plan. Explicit selectors
  allow users to expand an old installation.
- **Selection is remembered rather than recomputed on update.** `--asset` and
  `--binary` are strict, repeatable selectors. Changing them works at the same
  release version and replaces the affected selection. Unselected old outputs
  are listed in the plan and removed only after the new installs succeed.
  `--filter` counts as a change only when it differs from the saved filter,
  because scripts repeat the same install command and expect nothing to happen
  when the package is current.
- **`--yes` installs all archive binaries on initial selection.** Asset selection
  still defaults to the first compatible candidate, with RPMs ranked first.
  Remembered archive subsets stay unchanged on updates, including with `--yes`.
- **Batch failures persist progress.** An `incomplete` flag prevents same-version
  partial installs from being reported current. Binary replacements use staged
  files; config writes use atomic replacement, preserving existing permissions
  and following config symlinks. Native RPM fallback uses upgrade
  with `--replacepkgs` so failed batches can be retried. System packages cannot
  promise whole-batch rollback.
- **The binary directory is a setting, and sudo is used only when needed.**
  `bin_dir` changes where new binary components are installed. Components keep
  their recorded paths, so existing binaries move only when they are next
  updated or reinstalled, and the plan lists the old files for removal.
  Binary copies and removals run through sudo only when the current user cannot
  create files in the target directory, so a user-owned directory such as
  `~/.local/bin` needs no sudo and its binaries stay owned by the user. A
  missing target directory is created with `mkdir -p` before the first copy,
  and the check uses its nearest existing parent, so a new `~/.local/bin` needs
  no sudo either. A missing `/usr/local/bin` is created through sudo. System
  package operations always use `runPrivileged`.
- **Archive extraction writes only regular files and directories.**
  `unpackArchive` (mholt/archives) skips symlinks and hardlinks and rejects
  absolute entries or entries that escape the extraction directory. It keeps
  the archive layout without stripping a top-level folder, so remembered
  `--binary` member paths stay valid.
- **`--version` is a new root flag** (cobra's built-in support, wired via
  `rootCmd.Version`), registered without cobra's default `-v` shorthand so
  it doesn't reclaim the shorthand freed up from removing `--verbose`. The
  pre-existing `version` subcommand is unchanged.

---

## Gotchas

- `install` must work without a terminal (Ansible runs `grm install` with
  stdin closed). Only prompt when grm cannot decide: several candidate
  assets or archive binaries, or a plan that removes components. An
  unconditional plan confirmation broke this in v2.0.0.
- `list -d` no longer means `--description` — it's the global `--debug`
  flag. Use `list --description` (long-form only).
- Config keys are lower-case (`owner`, `repo`, `version`, `selections`,
  `components`). `list --long` shows every tracked component and its version.
- A binary selector without `asset-glob::` applies to every selected binary
  asset. Scope selectors when installing archives with different contents.
- `resolveConfigFile`'s legacy fallback is checked once per run, at
  startup; there is no automatic migration from the legacy path to the
  XDG one.
