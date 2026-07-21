package cmd

import (
	"fmt"
	"os"
	"strings"
)

// Terminal capability detection
var (
	isTTY        bool
	colorEnabled bool
)

func init() {
	if fi, err := os.Stdout.Stat(); err == nil {
		isTTY = (fi.Mode() & os.ModeCharDevice) != 0
	}
	colorEnabled = isTTY && os.Getenv("NO_COLOR") == ""
}

// ANSI escape codes
const (
	cReset = "\033[0m"
	cBold  = "\033[1m"
	cDim   = "\033[2m"
	cRed   = "\033[31m"
	cGreen = "\033[32m"
	cYell  = "\033[33m"
	cCyan  = "\033[36m"
)

// Unicode symbols
const (
	sCheck  = "✓"
	sCross  = "✗"
	sSync   = "⟳"
	sArrow  = "→"
	sTriang = "▸"
)

// ---------------------------------------------------------------------------
// Low-level: wrap text with ANSI codes
// ---------------------------------------------------------------------------

func style(codes, text string) string {
	if !colorEnabled {
		return text
	}
	return codes + text + cReset
}

func bold(t string) string      { return style(cBold, t) }
func dim(t string) string       { return style(cDim, t) }
func red(t string) string       { return style(cRed, t) }
func green(t string) string     { return style(cGreen, t) }
func yellow(t string) string    { return style(cYell, t) }
func cyan(t string) string      { return style(cCyan, t) }
func boldGreen(t string) string { return style(cBold+cGreen, t) }
func boldCyan(t string) string  { return style(cBold+cCyan, t) }

// ---------------------------------------------------------------------------
// High-level: semantic status messages
//
// All status output goes through these so the look is uniform everywhere.
// Pattern: "SYMBOL text\n" — no manual indentation at call sites.
// ---------------------------------------------------------------------------

func msgOK(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", green(sCheck), fmt.Sprintf(format, a...))
}

func msgFail(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", red(sCross), fmt.Sprintf(format, a...))
}

func msgWarn(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", yellow("!"), fmt.Sprintf(format, a...))
}

func msgLocked(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", yellow("●"), fmt.Sprintf(format, a...))
}

func msgInfo(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", cyan(sTriang), fmt.Sprintf(format, a...))
}

func msgUpdate(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", cyan("↑"), fmt.Sprintf(format, a...))
}

func msgSync(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", yellow(sSync), fmt.Sprintf(format, a...))
}

// msgStep prints an indented line (progress detail, no symbol).
func msgStep(format string, a ...interface{}) {
	fmt.Printf("  %s\n", fmt.Sprintf(format, a...))
}

// msgDone prints the final "Done" success line.
func msgDone() {
	fmt.Printf("%s Done\n", boldGreen(sCheck))
}

// ---------------------------------------------------------------------------
// Tables: header + separator + rows with per-column styling
// ---------------------------------------------------------------------------

// padCol pads text to width, then applies an optional style function.
// This keeps visible width correct despite invisible ANSI codes.
func padCol(text string, width int, styleFn func(string) string) string {
	padded := fmt.Sprintf("%-*s", width, text)
	if styleFn != nil {
		return styleFn(padded)
	}
	return padded
}

// tableHeader prints a bold header row and a dim ─ separator.
func tableHeader(widths []int, names []string) {
	var b strings.Builder
	total := 0
	for i, name := range names {
		if i < len(widths) {
			fmt.Fprintf(&b, "%-*s ", widths[i], name)
			total += widths[i] + 1
		} else {
			b.WriteString(name)
			total += len(name)
		}
	}
	fmt.Printf("  %s\n", bold(strings.TrimRight(b.String(), " ")))
	fmt.Printf("  %s\n", dim(strings.Repeat("─", total)))
}

// tableRow prints a row with pre-styled columns (use padCol to build them).
func tableRow(cols ...string) {
	fmt.Printf("  %s\n", strings.Join(cols, " "))
}

// ---------------------------------------------------------------------------
// Terminal control (for update animation)
// ---------------------------------------------------------------------------

func cursorUp(n int) {
	if isTTY {
		fmt.Printf("\033[%dA", n)
	}
}
