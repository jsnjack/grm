package cmd

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// LevelTrace is the logging level used for maximum-detail diagnostics
// written to the trace file.
const LevelTrace = slog.Level(-8)

// L is the package-wide logger, configured by initLogger from the
// --debug/--trace flags.
var L *slog.Logger

// initLogger configures the package logger from the --debug/--trace flags
// and returns a cleanup function to call before exit. debug logs to stderr,
// trace logs to tracePath (truncated on every start); the two are
// independent and may both be active at once.
func initLogger(debug, trace bool, tracePath string) func() {
	cleanup := func() {}
	var handlers []slog.Handler

	if debug {
		handlers = append(handlers, slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	if trace {
		f, err := os.OpenFile(tracePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err == nil {
			handlers = append(handlers, slog.NewTextHandler(f, &slog.HandlerOptions{Level: LevelTrace}))
			cleanup = func() { _ = f.Close() }
		}
	}

	var h slog.Handler
	switch len(handlers) {
	case 0:
		h = slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})
	case 1:
		h = handlers[0]
	default:
		h = multiHandler(handlers)
	}
	L = slog.New(h)
	slog.SetDefault(L)
	return cleanup
}

// multiHandler fans out log records to every handler in the slice, so
// --debug (stderr) and --trace (file) can be active independently.
type multiHandler []slog.Handler

func (m multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make(multiHandler, len(m))
	for i, h := range m {
		out[i] = h.WithAttrs(attrs)
	}
	return out
}

func (m multiHandler) WithGroup(name string) slog.Handler {
	out := make(multiHandler, len(m))
	for i, h := range m {
		out[i] = h.WithGroup(name)
	}
	return out
}
