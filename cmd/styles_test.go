package cmd

import "testing"

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
