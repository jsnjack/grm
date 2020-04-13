grm
====

### What is it?
`grm` is an experimental package manager for GitHub Releases. It is probably only good in installing packages which are distributed as binaries (for example the ones written in Go).

`grm` inspects release assets for a binary file and when the file is found, downloads and installs it in `/usr/local/bin/` directory.

### How to use it?
```bash
$ ./grm
A package installer for github releases

Usage:
  grm [command]

Available Commands:
  help        Help about any command
  info        Show information about a package
  install     Install a package from github releases
  list        List installed packages
  lock        Lock a package
  remove      Remove a package
  unlock      Unlock a package
  update      Update installed packages
  version     Print version

Flags:
  -h, --help   help for grm
  -y, --yes    Confirm all
```

#### How to install specific version of hugo?
```grm
grm install gohugoio/hugo==v0.63.0 -f Linux-64
```

### How to install it?
```bash
curl -s https://api.github.com/repos/jsnjack/grm/releases/latest | jq -r .assets[0].browser_download_url | wget -qi - && chmod +x grm && sudo mv grm /usr/local/bin/
```
