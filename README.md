grm
====

### What is it?
`grm` is an experimental package manager for GitHub Releases. It is probably only good for installing packages which are distributed as binaries (for example the ones written in Go, Rust). The following popular packages can be installed with `grm`:
 - mozilla/geckodriver
 - gohugoio/hugo
 - go-acme/lego
 - zyedidia/micro
 - ...

`grm` installs selected release binaries to `/usr/local/bin/` by default and installs RPM/DEB assets through the system package manager. One repository can provide several binaries or a mix of binaries and system packages.

### How to use it?
```bash
$ ./grm
A package manager for GitHub releases

Usage:
  grm [command]

Available Commands:
  aliases     Print table of known package aliases
  help        Help about any command
  info        Show information about a package
  install     Install a package from GitHub releases
  list        List installed packages
  lock        Lock a package
  release     Create a release in GitHub
  remove      Remove a package
  set         Modify settings
  settings    Print settings
  unlock      Unlock a package
  update      Update installed packages
  version     Print version

Flags:
  -c, --config string   Path to the config file (default ~/.config/grm/grm.yaml)
  -d, --debug           Debug-level logging on stderr.
  -h, --help            help for grm
      --no-progress     Disable progress bar
      --token string    GitHub API token
      --trace           Trace-level logs to /tmp/grm.log (truncated each run).
      --version         Print the version and exit
  -y, --yes             Confirm all

Use "grm [command] --help" for more information about a command.
```

#### How to install specific version of hugo?
```grm
grm install gohugoio/hugo==v0.63.0 -f Linux-64
```

#### How to install the most recent major version?
Use the `~=` operator to filter releases by a version prefix or glob pattern.
The most recent matching release is selected automatically.
```bash
# Install the most recent v0.63.x release
grm install gohugoio/hugo~=v0.63

# Install any v0.6x release (glob)
grm install gohugoio/hugo~=v0.6*

# Install any versioned stable release (glob with character class)
grm install gohugoio/hugo~=v[0-9]*-stable
```

#### Installing several binaries or assets

The asset prompt accepts comma-separated numbers, ranges, or `all`. Compatible
RPM assets appear first when `rpm` is available. You can select a CLI archive
and a GUI RPM from the same release together.

After unpacking an archive, grm asks which binaries to install. With `--yes`,
it installs every eligible binary in each selected archive. For asset selection,
`--yes` chooses the first suitable asset; use repeated `--asset` flags to select
several assets without a prompt.

```bash
# Install both binaries from Yazi's selected archive on a new install
grm install sxyazi/yazi --yes

# Expand an existing single-binary Yazi installation
grm install sxyazi/yazi --binary ya --binary yazi --yes

# Select a CLI archive and a GUI RPM explicitly
grm install jsnjack/bunk \
  --asset 'bunk_linux_amd64.tar.gz' \
  --asset 'bunk-gui-*.x86_64.rpm'

# Select individual archive binaries, optionally renaming them
grm install sxyazi/yazi --binary ya --binary 'yazi=my-yazi'

# Scope a binary selector to one of several archives
grm install owner/repo \
  --asset 'client_*linux_amd64.tar.gz' \
  --asset 'tools_*linux_amd64.tar.gz' \
  --binary 'client_*.tar.gz::client' \
  --binary 'tools_*.tar.gz::admin=repo-admin'

grm list --long
```

Each `--asset` name or glob must match exactly one release asset. A `--binary`
selector matches an archive-relative path or basename; quote globs so the shell
does not expand them. A global binary selector applies to every selected binary
asset; use `asset-glob::path-glob` to scope it when archives have different
contents. `--rename` remains available when installing exactly one binary.

Asset and binary choices, including renames, are saved for later installs and
updates. Version strings in asset and member paths are replaced with globs so
those choices can follow subsequent releases. Missing or ambiguous remembered
choices produce an error with instructions to select again. Newly appearing
archive binaries are not added to remembered selections automatically.

Passing `--asset` replaces the saved asset selection; passing `--binary`
replaces the binary selection in the assets it applies to. Changing `--filter`
starts asset selection again; passing the saved filter again keeps the
selection, so a repeated install command does not reinstall the package. These
flags work even when the release version is already installed. The installation
plan lists all outputs and any previously tracked components that will be
removed when the new selection succeeds. grm asks for confirmation only when the
plan removes components, so an install with one suitable asset and one binary
runs without prompts.

All components share a repository's release and lock. `update` updates them
together; `remove` removes every tracked binary and system package. Downloads
and conflict checks finish before installation. Completed operations are saved
individually, and failed batches stay marked incomplete so a later install or
update can retry them. System package operations are not rolled back.

Existing configs work without a manual conversion. Reading a config leaves it
unchanged. On the next install or update, old binary installs keep one binary
and its installed name, and old DEB/RPM installs keep their package type and
name. Renamed legacy archive installs retain the old first-binary selection.
Use `--asset` or `--binary` to expand an existing selection. Missing binaries
or changed system package names stop migration before installation.

Migration preserves settings, filters, and locks. New and migrated installs use
`selections` and `components` records; older grm versions cannot manage these
records if you downgrade. For new archive selections, `--yes` installs all
eligible binaries; existing selections stay unchanged.

#### How to install binaries to another directory?
Set `bin_dir` to an absolute path, for example a directory in your home folder:
```bash
grm set bin_dir ~/.local/bin
```

grm creates the directory if it does not exist. It uses sudo for binary files
only when the current user cannot write to the directory, or to its nearest
existing parent when the directory has not been created yet. Binaries that are
already installed move to the new directory when they are next updated or
reinstalled with `--refresh`. Set `bin_dir` to an empty value to use
`/usr/local/bin/` again.

### Keeping grm up to date
Once installed, `grm` can update itself:
```bash
grm install jsnjack/grm
```

### How to install it?
> Make sure you have `curl` and `jq` installed (`sudo dnf install curl jq`)
```bash
curl -s https://api.github.com/repos/jsnjack/grm/releases/latest | jq -r .assets[0].browser_download_url | xargs curl -LOs && chmod +x grm && sudo mv grm /usr/local/bin/
```
