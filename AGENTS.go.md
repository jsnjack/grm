# Go Project Conventions

Applies to all Go projects in addition to `AGENTS.universal.md`.

---

**CLI framework:** Use cobra. Register commands and flags in `init()` within
`cmd/` files. Use `RunE` (not `Run`) so errors propagate rather than being
handled inside the command.

**Doc comments:** Every exported symbol's doc comment must start with the
symbol's name (godoc convention): `// Server handles incoming HTTP requests.`

**Versioning:** Every binary declares a version variable stamped at build time:
```go
// Version is set at build time via ldflags.
var Version = "dev"
```
The Makefile stamps it via `-ldflags="-X <pkg>.Version=$(VERSION)"` using
`monova` for the version value.

**Error wrapping:** Always wrap with context — never return a bare error:
```go
return fmt.Errorf("load config: %w", err)
```

**Never ignore errors:** No `_ = fn()` or bare discards. If an error genuinely
can't be acted on, log it at trace level so `--trace` surfaces it:
```go
if err := f.Close(); err != nil {
    slog.Log(ctx, LevelTrace, "close file", "err", err)
}
```

**Logging:** Use `log/slog` with `slog.NewTextHandler`. Wire it from the
`--debug` / `--trace` flags as described in `AGENTS.universal.md`. Do not use
stdlib `log`.

**Server startup:** When the application starts a server on any port, log the
listening address unconditionally — to stderr and the trace log — regardless
of debug flags. Use a consistent format: `Listening on <addr>`. This ensures
operators and agents always know what address was bound.

**Testing:** Table-driven tests using `t.Run()` — one named sub-test per case.
Use only the standard `testing` package; no third-party assertion libraries.

**Commands & flags:**
- `--version` flag on root prints the version and exits. No short alias, no
  subcommand.
- `--debug` / `-d` and `--trace` are persistent flags on root so all
  subcommands inherit them without re-declaring.
- `--config` / `-c` sets the config file path. Default location:
  `~/.config/<app>/config.<ext>`.
- Only `--debug` (`-d`) and `--config` (`-c`) get short aliases. All other
  flags are long-form only.

**Directory layout (XDG):** Use `os.UserConfigDir`, `os.UserCacheDir`, and
`os.UserHomeDir` — never hardcode `~`. Respect the XDG env vars automatically
(`$XDG_CONFIG_HOME`, `$XDG_CACHE_HOME`, `$XDG_DATA_HOME`).

| Purpose | Default path |
|---------|-------------|
| Config file | `~/.config/<app>/config.<ext>` |
| Cache / downloaded data | `~/.cache/<app>/` |
| Persistent app state | `~/.local/share/<app>/` |
| Trace log | `/tmp/<app>.log` (truncated on start) |
