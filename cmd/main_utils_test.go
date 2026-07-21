package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	for _, n := range []int{0, 1, 6, 20} {
		t.Run(fmt.Sprintf("length %d", n), func(t *testing.T) {
			got := generateRandomString(n)
			if len(got) != n {
				t.Errorf("expected length %d, got %d (%q)", n, len(got), got)
			}
			for _, r := range got {
				if !strings.ContainsRune(letterBytes, r) {
					t.Errorf("unexpected character %q in %q", r, got)
				}
			}
		})
	}
}

func TestPosString(t *testing.T) {
	cases := []struct {
		name    string
		slice   []string
		element string
		want    int
	}{
		{"found first", []string{"a", "b", "c"}, "a", 0},
		{"found middle", []string{"a", "b", "c"}, "b", 1},
		{"not found", []string{"a", "b", "c"}, "z", -1},
		{"empty slice", []string{}, "a", -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := posString(c.slice, c.element); got != c.want {
				t.Errorf("posString(%v, %q) = %d, want %d", c.slice, c.element, got, c.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	cases := []struct {
		name    string
		slice   []string
		element string
		want    bool
	}{
		{"present", []string{"y", "yes"}, "yes", true},
		{"absent", []string{"y", "yes"}, "no", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containsString(c.slice, c.element); got != c.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", c.slice, c.element, got, c.want)
			}
		})
	}
}

func TestToMD5(t *testing.T) {
	t.Run("known content", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
			t.Fatalf("setup: %s", err)
		}
		got, err := tomd5(path)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		const want = "5d41402abc4b2a76b9719d911017c592" // md5("hello")
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := tomd5(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Errorf("expected error for missing file, got nil")
		}
	})
}

func TestGetTmpDir(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"grm tmp path", DefaultTmpDirPattern + "abcdef/somefile", "/tmp/grm.abcdef/"},
		{"unrelated path", "/usr/local/bin/tool", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := getTmpDir(c.path); got != c.want {
				t.Errorf("getTmpDir(%q) = %q, want %q", c.path, got, c.want)
			}
		})
	}
}
