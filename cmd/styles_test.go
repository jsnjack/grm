package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestStyle(t *testing.T) {
	original := colorEnabled
	defer func() { colorEnabled = original }()

	t.Run("color enabled", func(t *testing.T) {
		colorEnabled = true
		got := style(cBold, "hi")
		want := cBold + "hi" + cReset
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("color disabled", func(t *testing.T) {
		colorEnabled = false
		if got := style(cBold, "hi"); got != "hi" {
			t.Errorf("got %q, want %q", got, "hi")
		}
	})
}

func TestStyleHelpers_passThroughWhenColorDisabled(t *testing.T) {
	original := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = original }()

	helpers := map[string]func(string) string{
		"bold": bold, "dim": dim, "red": red, "green": green,
		"yellow": yellow, "cyan": cyan, "boldGreen": boldGreen, "boldCyan": boldCyan,
	}
	for name, fn := range helpers {
		t.Run(name, func(t *testing.T) {
			if got := fn("text"); got != "text" {
				t.Errorf("%s(%q) = %q, want unchanged", name, "text", got)
			}
		})
	}
}

func TestPadCol(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		width   int
		styleFn func(string) string
		want    string
	}{
		{"pads to width", "a", 4, nil, "a   "},
		{"no truncation beyond width", "abcdef", 4, nil, "abcdef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := padCol(c.text, c.width, c.styleFn); got != c.want {
				t.Errorf("padCol(%q, %d) = %q, want %q", c.text, c.width, got, c.want)
			}
		})
	}
}

func TestHighlightVersionBoldsOnlyTheChangedSegment(t *testing.T) {
	original := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = original }()
	cases := []struct{ previous, next, plain, changed string }{
		{"v0.176.0", "v0.179.0", "v0.", "179.0"},
		{"v2.1.0", "v3.0.0", "v", "3.0.0"},
		{"1.4", "1.5", "1.", "5"},
		{"selenium-4.41.0", "selenium-4.52.0", "selenium-4.", "52.0"},
	}
	for _, c := range cases {
		want := cyan(c.plain) + boldCyan(c.changed)
		if got := highlightVersion(c.previous, c.next); got != want {
			t.Errorf("highlightVersion(%q, %q) = %q, want %q", c.previous, c.next, got, want)
		}
	}
}

func TestPrintTableAlignsStyledCells(t *testing.T) {
	original := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = original }()
	out := captureStdout(t, func() {
		printTable("", []string{"A", "B"}, [][]string{{bold("long-name"), "x"}, {"s", cyan("y")}})
	})
	lines := strings.Split(strings.TrimRight(ansiPattern.ReplaceAllString(out, ""), "\n"), "\n")
	want := []string{"A           B", "long-name   x", "s           y"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("got %q, want %q", lines, want)
	}
}

func TestMaskTokenKeepsPrefixAndTail(t *testing.T) {
	cases := map[string]string{
		"ghp_abcdefghijklmnop1234": "ghp_••••••••1234",
		"abcdefghijkl":             "••••••••ijkl",
		"short":                    "•••••",
	}
	for token, want := range cases {
		if got := maskToken(token); got != want {
			t.Errorf("maskToken(%q) = %q, want %q", token, got, want)
		}
	}
}

func TestConfirmationUsesDefaultOnEnter(t *testing.T) {
	withYes(t, false)
	for _, c := range []struct {
		input      string
		defaultYes bool
		want       bool
	}{{"\n", true, true}, {"\n", false, false}, {"maybe\ny\n", false, true}, {"N\n", true, false}} {
		withStdin(t, c.input)
		var got bool
		var err error
		captureStdout(t, func() { got, err = askForConfirmation("Continue?", c.defaultYes) })
		if err != nil || got != c.want {
			t.Errorf("input %q default %v: got %v, %v; want %v", c.input, c.defaultYes, got, err, c.want)
		}
	}
}

func TestPrintErrorShowsDetailsAndFix(t *testing.T) {
	original := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = original }()
	var b strings.Builder
	printError(&b, fmt.Errorf("update: %w", &hintedError{msg: "nothing matches", details: [][2]string{{"selection", "a*"}, {"assets", "b\nc"}}, fix: "grm install x --asset 'b'"}))
	want := "✗ nothing matches\n    selection  a*\n    assets     b\n               c\n  Fix  grm install x --asset 'b'\n"
	if b.String() != want {
		t.Fatalf("got\n%s\nwant\n%s", b.String(), want)
	}
	b.Reset()
	printError(&b, silentError{errors.New("already printed")})
	if b.Len() != 0 {
		t.Fatalf("silent error printed %q", b.String())
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = writer
	fn()
	os.Stdout = old
	logClose(writer)
	data, err := io.ReadAll(reader)
	logClose(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func withStdin(t *testing.T, input string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString(input); err != nil {
		t.Fatal(err)
	}
	logClose(writer)
	old := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = old; logClose(reader) })
}
