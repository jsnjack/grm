package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerateSettingsHelp(t *testing.T) {
	got := generateSettingsHelp()
	for key, desc := range Settings {
		want := fmt.Sprintf(" - %s - %s", key, desc)
		if !strings.Contains(got, want) {
			t.Errorf("expected help to contain %q, got %q", want, got)
		}
	}
}
