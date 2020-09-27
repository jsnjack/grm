grm
====

### What is it?
`grm` is an experimental package manager for GitHub Releases. It is probably only good in installing packages which are distributed as binaries (for example the ones written in Go, Rust). The following popular packages can be installed with `grm`:
 - mozilla/geckodriver
 - gohugoio/hugo
 - go-acme/lego
 - zyedidia/micro
 - ...

`grm` inspects release assets for a binary file and when the file is found, downloads and installs it in `/usr/local/bin/` directory.

### How to use it?
```bash
$ ./grm
A package manager for GitHub releases

Usage:
  grm [command]

Available Commands:
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
  -h, --help           help for grm
      --token string   GitHub API token
  -y, --yes            Confirm all

```

#### How to install specific version of hugo?
```grm
grm install gohugoio/hugo==v0.63.0 -f Linux-64
```

### How to install it?
> Make sure you have `curl` and `jq` installed (`sudo dnf install curl jq`)
```bash
curl -s https://api.github.com/repos/jsnjack/grm/releases/latest | jq -r .assets[0].browser_download_url | xargs curl -LOs && chmod +x grm && sudo mv grm /usr/local/bin/
```
