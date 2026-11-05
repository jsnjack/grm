package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/go-github/v93/github"
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
	cMag   = "\033[35m"
	cCyan  = "\033[36m"
)

// Symbols. Each one has a single meaning across all commands.
const (
	sCheck   = "✓" // succeeded
	sCross   = "✗" // failed
	sUp      = "↑" // newer version or upgraded component
	sAdd     = "+" // component to add
	sRemove  = "−" // component to remove
	sLocked  = "●" // locked package
	sWarn    = "!" // warning that does not stop the command
	sAsk     = "?" // question
	sSelect  = "›" // item grm selected
	sDown    = "↓" // download
	sArrow   = "→"
	sDot     = "·"
	sTreeMid = "├─"
	sTreeEnd = "└─"
)

// ---------------------------------------------------------------------------
// Low-level: wrap text with ANSI codes
// ---------------------------------------------------------------------------

func style(codes, text string) string {
	if !colorEnabled || text == "" {
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
func magenta(t string) string   { return style(cMag, t) }
func boldGreen(t string) string { return style(cBold+cGreen, t) }
func boldCyan(t string) string  { return style(cBold+cCyan, t) }
func boldRed(t string) string   { return style(cBold+cRed, t) }

var ansiPattern = regexp.MustCompile("\033\\[[0-9;]*m")

// visibleWidth returns the number of terminal cells text occupies, ignoring
// ANSI escape codes.
func visibleWidth(text string) int {
	return utf8.RuneCountInString(ansiPattern.ReplaceAllString(text, ""))
}

// padVisible pads styled text with spaces to width visible cells.
func padVisible(text string, width int) string {
	if n := width - visibleWidth(text); n > 0 {
		return text + strings.Repeat(" ", n)
	}
	return text
}

// padCol pads text to width, then applies an optional style function.
// This keeps visible width correct despite invisible ANSI codes.
func padCol(text string, width int, styleFn func(string) string) string {
	padded := fmt.Sprintf("%-*s", width, text)
	if styleFn != nil {
		return styleFn(padded)
	}
	return padded
}

// ---------------------------------------------------------------------------
// Domain formatting
// ---------------------------------------------------------------------------

// pkgLabel renders owner/repo with a dim owner and a bold repository name.
func pkgLabel(fullName string) string {
	owner, repo, ok := strings.Cut(fullName, "/")
	if !ok {
		return bold(fullName)
	}
	return dim(owner+"/") + bold(repo)
}

// versionChange renders "old → new" and makes only the changed part of the
// new version bold.
func versionChange(previous, next string) string {
	if previous == "" || previous == next {
		return boldCyan(next)
	}
	return dim(previous) + " " + cyan(sArrow) + " " + highlightVersion(previous, next)
}

func highlightVersion(previous, next string) string {
	common := 0
	for common < len(previous) && common < len(next) && previous[common] == next[common] {
		common++
	}
	// Cut at the start of the version segment that differs.
	cut := strings.LastIndexAny(next[:common], ".-") + 1
	if cut == 0 && strings.HasPrefix(next, "v") && strings.HasPrefix(previous, "v") {
		cut = 1
	}
	return cyan(next[:cut]) + boldCyan(next[cut:])
}

func formatSize(bytes int) string {
	const unit = 1024
	switch {
	case bytes < unit:
		return fmt.Sprintf("%d B", bytes)
	case bytes < unit*unit:
		return fmt.Sprintf("%.1f KB", float64(bytes)/unit)
	case bytes < unit*unit*unit:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(unit*unit))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(unit*unit*unit))
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Round(time.Second).String()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// joinDetails joins non-empty parts with a dim " · " separator.
func joinDetails(parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, dim(" "+sDot+" "))
}

// ---------------------------------------------------------------------------
// High-level: semantic status messages
//
// All status output goes through these so the look is uniform everywhere.
// ---------------------------------------------------------------------------

func msgOK(format string, a ...any) {
	fmt.Printf("%s %s\n", green(sCheck), fmt.Sprintf(format, a...))
}

func msgWarn(format string, a ...any) {
	fmt.Printf("%s %s\n", yellow(sWarn), fmt.Sprintf(format, a...))
}

func msgLocked(format string, a ...any) {
	fmt.Printf("%s %s\n", yellow(sLocked), fmt.Sprintf(format, a...))
}

// msgSection starts an indented section with a dim title and optional summary.
func msgSection(title string, summary string) {
	fmt.Println()
	if summary != "" {
		fmt.Printf("  %s %s\n", dim(title), summary)
		return
	}
	fmt.Printf("  %s\n", dim(title))
}

// msgResult prints the last line of a command. It starts with ✓ or ✗ and
// should agree with the exit status.
func msgResult(ok bool, format string, a ...any) {
	symbol := boldGreen(sCheck)
	if !ok {
		symbol = boldRed(sCross)
	}
	fmt.Printf("%s %s\n", symbol, fmt.Sprintf(format, a...))
}

// printTable aligns pre-styled cells by their visible width. A nil header
// prints no header row. Trailing empty cells do not add padding.
func printTable(indent string, header []string, rows [][]string) {
	all := rows
	if header != nil {
		all = append([][]string{header}, rows...)
	}
	var widths []int
	for _, row := range all {
		for i, cell := range row {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], visibleWidth(cell))
		}
	}
	print := func(row []string, styleFn func(string) string) {
		var b strings.Builder
		for i, cell := range row {
			if styleFn != nil {
				cell = styleFn(cell)
			}
			if i == len(row)-1 {
				b.WriteString(cell)
			} else {
				b.WriteString(padVisible(cell, widths[i]+3))
			}
		}
		fmt.Println(indent + strings.TrimRight(b.String(), " "))
	}
	if header != nil {
		print(header, dim)
	}
	for _, row := range rows {
		print(row, nil)
	}
}

// dropEmptyColumns removes columns in which every cell is empty, so optional
// columns leave no gap.
func dropEmptyColumns(rows [][]string) [][]string {
	used := map[int]bool{}
	for _, row := range rows {
		for i, cell := range row {
			if cell != "" {
				used[i] = true
			}
		}
	}
	out := make([][]string, len(rows))
	for r, row := range rows {
		for i, cell := range row {
			if used[i] {
				out[r] = append(out[r], cell)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Prompts
// ---------------------------------------------------------------------------

var (
	stdinReader *bufio.Reader
	stdinFile   *os.File
)

// stdin returns a buffered reader for the current os.Stdin. It is created
// again when os.Stdin changes, which tests do.
func stdin() *bufio.Reader {
	if stdinReader == nil || stdinFile != os.Stdin {
		stdinReader, stdinFile = bufio.NewReader(os.Stdin), os.Stdin
	}
	return stdinReader
}

// readAnswer prints a question and reads one line. An empty line returns
// def. A closed stdin returns an error, so non-interactive runs fail instead
// of guessing.
func readAnswer(question, hint, def string) (string, error) {
	fmt.Printf("%s %s %s ", magenta(sAsk), question, dim(hint))
	line, err := stdin().ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		fmt.Println()
		if errors.Is(err, io.EOF) {
			return "", closedStdinError{}
		}
		return "", fmt.Errorf("read answer: %w", err)
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// closedStdinError explains why a question could not be asked. It matches
// io.EOF for callers that check for it.
type closedStdinError struct{}

func (closedStdinError) Error() string {
	return "grm needs an answer, but stdin is closed; pass --yes to accept"
}

func (closedStdinError) Unwrap() error { return io.EOF }

// hintedError carries the data grm compared and a command that fixes the
// problem, so the error can be printed as a block instead of one long line.
type hintedError struct {
	msg     string
	details [][2]string
	fix     string
	cause   error
}

func (e *hintedError) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

func (e *hintedError) Unwrap() error { return e.cause }

// silentError reports a failure that the command already printed.
type silentError struct{ error }

func printError(w io.Writer, err error) {
	if errors.As(err, new(silentError)) {
		return
	}
	var hinted *hintedError
	if errors.As(err, &hinted) {
		fmt.Fprintf(w, "%s %s\n", boldRed(sCross), hinted.msg)
		labelWidth := 0
		for _, d := range hinted.details {
			labelWidth = max(labelWidth, len(d[0]))
		}
		for _, d := range hinted.details {
			for i, line := range strings.Split(d[1], "\n") {
				label := ""
				if i == 0 {
					label = d[0]
				}
				fmt.Fprintf(w, "    %s  %s\n", dim(padCol(label, labelWidth, nil)), line)
			}
		}
		if hinted.cause != nil {
			fmt.Fprintf(w, "    %s\n", dim(hinted.cause.Error()))
		}
		if hinted.fix != "" {
			fmt.Fprintf(w, "  %s  %s\n", bold("Fix"), hinted.fix)
		}
		return
	}
	fmt.Fprintf(w, "%s %s\n", boldRed(sCross), err)
	if fix := errorFix(err); fix != "" {
		fmt.Fprintf(w, "  %s  %s\n", bold("Fix"), fix)
	}
}

// errorFix suggests a fix for common GitHub API failures.
func errorFix(err error) string {
	var rate *github.RateLimitError
	if errors.As(err, &rate) {
		return "grm set token <token>, or export GITHUB_TOKEN " + dim("(limit resets at "+rate.Rate.Reset.Local().Format("15:04")+")")
	}
	var response *github.ErrorResponse
	if errors.As(err, &response) && response.Response != nil {
		switch response.Response.StatusCode {
		case http.StatusUnauthorized:
			return "the token is invalid or expired; replace it with grm set token <token>"
		case http.StatusNotFound:
			return "check the repository name; a private repository needs a token that can read it"
		}
	}
	return ""
}
