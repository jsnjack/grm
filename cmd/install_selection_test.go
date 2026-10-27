package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-github/v32/github"
)

func withYes(t *testing.T, yes bool) {
	t.Helper()
	previous := rootYes
	rootYes = yes
	t.Cleanup(func() { rootYes = previous })
}

func fakeProgram(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestParseNumbers(t *testing.T) {
	cases := []struct {
		name, input string
		count       int
		want        []int
		bad         bool
	}{
		{"several", "1,3", 3, []int{1, 3}, false},
		{"range", "2-4,1,2", 4, []int{1, 2, 3, 4}, false},
		{"all", "all", 3, []int{1, 2, 3}, false},
		{"zero", "0", 3, nil, true},
		{"too large", "4", 3, nil, true},
		{"backward", "3-1", 3, nil, true},
		{"empty", "", 3, nil, true},
		{"text", "one", 3, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseNumbers(tc.input, tc.count)
			if (err != nil) != tc.bad || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, %v; want %v, error=%v", got, err, tc.want, tc.bad)
			}
		})
	}
}

func TestSelectAssetsRPMPreference(t *testing.T) {
	for _, rpm := range []bool{true, false} {
		t.Run(map[bool]string{true: "rpm available", false: "rpm unavailable"}[rpm], func(t *testing.T) {
			withYes(t, true)
			dir := t.TempDir()
			if rpm {
				fakeProgram(t, dir, "rpm", "exit 0")
			}
			t.Setenv("PATH", dir)
			assets := []*github.ReleaseAsset{
				{Name: github.String("bunk_linux_amd64.tar.gz")},
				{Name: github.String("bunk-gui-0.74.0-1.fc45.x86_64.rpm")},
				{Name: github.String("bunk-gui-0.74.0-1.fc45.src.rpm")},
				{Name: github.String("bunk_linux_arm64.tar.gz")},
			}
			got, err := selectAssets(assets, nil)
			if err != nil || len(got) != 1 {
				t.Fatalf("got %v, %v", got, err)
			}
			want := assets[0].GetName()
			if rpm {
				want = assets[1].GetName()
			}
			if got[0].GetName() != want {
				t.Fatalf("got %s, want %s", got[0].GetName(), want)
			}
		})
	}
}

func TestResolveAssetSelections(t *testing.T) {
	assets := []*github.ReleaseAsset{
		{Name: github.String("bunk_1.2.0_linux_amd64.tar.gz")},
		{Name: github.String("bunk-gui-1.2.0-1.fc45.x86_64.rpm")},
	}
	cases := []struct {
		name      string
		saved     []AssetSelection
		requested []string
		bad       bool
	}{
		{"remembered mixed formats", []AssetSelection{{Pattern: "bunk_*_linux_amd64.tar.gz", Binaries: []BinarySelection{{Path: "bunk"}}}, {Pattern: "bunk-gui-*-1.fc45.x86_64.rpm"}}, nil, false},
		{"explicit mixed formats", nil, []string{"bunk_*_linux_amd64.tar.gz", "bunk-gui-*.rpm"}, false},
		{"missing choice", []AssetSelection{{Pattern: "missing*.zip"}}, nil, true},
		{"ambiguous choice", nil, []string{"bunk*"}, true},
		{"duplicate", nil, []string{"bunk-gui-*.rpm", "bunk-gui-*.rpm"}, true},
		{"invalid glob", nil, []string{"["}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selected, saved, err := resolveAssetSelections(assets, tc.saved, tc.requested, nil, "v1.2.0")
			if (err != nil) != tc.bad {
				t.Fatalf("error=%v, want error=%v", err, tc.bad)
			}
			if tc.bad {
				return
			}
			if len(selected) != 2 || len(saved) != 2 {
				t.Fatalf("got %v, %v", selected, saved)
			}
			if saved[0].Pattern != "bunk_*_linux_amd64.tar.gz" {
				t.Fatalf("version was not generalized: %+v", saved[0])
			}
			next := []*github.ReleaseAsset{{Name: github.String("bunk_1.3.0_linux_amd64.tar.gz")}, {Name: github.String("bunk-gui-1.3.0-1.fc45.x86_64.rpm")}}
			if _, _, err := resolveAssetSelections(next, saved, nil, nil, "v1.3.0"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSelectBinaryFiles(t *testing.T) {
	files := []fileEntry{{relPath: "yazi-1.2.0/ya"}, {relPath: "yazi-1.2.0/yazi"}}
	cases := []struct {
		name  string
		saved []BinarySelection
		flags []string
		want  int
		bad   bool
	}{
		{"yes installs all", nil, nil, 2, false},
		{"remembered subset stays selected", []BinarySelection{{Path: "yazi-*/yazi", Rename: "my-yazi"}}, nil, 1, false},
		{"explicit basename", nil, []string{"yazi"}, 1, false},
		{"scoped rename", nil, []string{"yazi*.zip::yazi=my-yazi"}, 1, false},
		{"missing remembered member", []BinarySelection{{Path: "yazi-*/gone"}}, nil, 0, true},
		{"missing flag", nil, []string{"gone"}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withYes(t, true)
			requests, err := parseBinaryRequests(tc.flags)
			if err != nil {
				t.Fatal(err)
			}
			got, saved, err := selectBinaryFiles(files, "yazi-linux.zip", tc.saved, requests, "v1.2.0")
			if (err != nil) != tc.bad {
				t.Fatalf("error=%v, want error=%v", err, tc.bad)
			}
			if tc.bad {
				return
			}
			if len(got) != tc.want || len(saved) != tc.want {
				t.Fatalf("got %v, %v", got, saved)
			}
			if saved[0].Path != "yazi-*/ya" && saved[0].Path != "yazi-*/yazi" {
				t.Fatalf("path not reusable: %+v", saved)
			}
			if len(tc.flags) > 0 && !requests[0].matched {
				t.Fatal("request not marked as matched")
			}
		})
	}
}

func TestParseBinaryRequests(t *testing.T) {
	for _, value := range []string{"", "[", "foo=../bar", "foo=", "[::foo", "foo=..", "foo=a/b"} {
		t.Run(value, func(t *testing.T) {
			if _, err := parseBinaryRequests([]string{value}); err == nil {
				t.Fatalf("accepted %q", value)
			}
		})
	}
}

func TestInteractiveSelections(t *testing.T) {
	withYes(t, false)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = old; logClose(reader) })
	if _, err := writer.WriteString("1,3\n2-3\n"); err != nil {
		t.Fatal(err)
	}
	logClose(writer)
	for i, want := range [][]int{{1, 3}, {2, 3}} {
		got, err := askForNumbers("Choose:", 3, false)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("prompt %d got %v,%v; want %v", i, got, err, want)
		}
	}
}

func TestPlatformFilteringRetainsMixedArchitectureAliases(t *testing.T) {
	dir := t.TempDir()
	fakeProgram(t, dir, "rpm", "exit 0")
	t.Setenv("PATH", dir)
	input := []string{"client-linux-x64.tar.gz", "admin_linux_amd64.zip", "bunk-gui-1.2.0.x86_64.rpm", "bunk-gui-1.2.0.aarch64.rpm"}
	got := filterSuitableAssetsForPlatform(input, nil, "linux", "amd64")
	if !reflect.DeepEqual(got, input[:3]) {
		t.Fatalf("got %v, want %v", got, input[:3])
	}
}

func TestExplicitAssetGlobSurvivesRPMBuildChanges(t *testing.T) {
	assets := []*github.ReleaseAsset{{Name: github.String("bunk-gui-1.2.0-1.gitabcdef0.fc45.x86_64.rpm")}}
	_, saved, err := resolveAssetSelections(assets, nil, []string{"bunk-gui-*.x86_64.rpm"}, nil, "v1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	next := []*github.ReleaseAsset{{Name: github.String("bunk-gui-1.3.0-2.git0123456.fc45.x86_64.rpm")}}
	if _, _, err := resolveAssetSelections(next, saved, nil, nil, "v1.3.0"); err != nil {
		t.Fatalf("explicit glob was narrowed: %v", err)
	}
}
