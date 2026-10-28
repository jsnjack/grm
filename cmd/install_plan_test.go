package cmd

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var fixtureBinary = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

func releaseArchive(t *testing.T, files map[string][]byte) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "tools_linux_amd64.zip")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, data := range files {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}

func TestPrepareAssetMultipleBinaries(t *testing.T) {
	withYes(t, true)
	archive := releaseArchive(t, map[string][]byte{
		"yazi-v1.2.0/ya":           fixtureBinary,
		"yazi-v1.2.0/yazi":         fixtureBinary,
		"yazi-v1.2.0/._ya":         fixtureBinary,
		"yazi-v1.2.0/helper.so":    fixtureBinary,
		"yazi-v1.2.0/runtime.node": fixtureBinary,
		"yazi-v1.2.0/README.md":    []byte("Readme text\n"),
	})
	destination := t.TempDir()
	plan, selection, err := prepareAsset(archive, filepath.Base(archive), "v1.2.0", destination, AssetSelection{Pattern: "tools_linux_amd64.zip"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 || len(selection.Binaries) != 2 {
		t.Fatalf("got %d components, %+v", len(plan), selection)
	}
	dir := t.TempDir()
	fakeProgram(t, dir, "sudo", "exec \"$@\"")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, item := range plan {
		if err := installBinaryTo(item.source, item.component.Filename); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"ya", "yazi"} {
		data, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(data, fixtureBinary) {
			t.Fatalf("wrong %s content", name)
		}
	}
	for _, item := range plan {
		if _, err := os.Stat(item.source); err != nil {
			t.Fatalf("source was removed before batch finished: %v", err)
		}
	}
}

func TestPrepareMixedBinaryAndRPM(t *testing.T) {
	withYes(t, true)
	dir := t.TempDir()
	fakeProgram(t, dir, "rpm", `case "$1" in
 -qp) echo bunk-gui;;
 -qpl) echo /usr/bin/bunk-gui; echo /usr/share/applications/io.github.jsnjack.BunkGui.desktop;;
 *) exit 1;;
esac`)
	t.Setenv("PATH", dir)
	archive := releaseArchive(t, map[string][]byte{"bunk": fixtureBinary})
	binaryPlan, _, err := prepareAsset(archive, "bunk_linux_amd64.zip", "v1.2.0", DefaultBinDir, AssetSelection{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rpm := filepath.Join(t.TempDir(), "bunk-gui-1.2.0.x86_64.rpm")
	systemPlan, _, err := prepareAsset(rpm, filepath.Base(rpm), "v1.2.0", DefaultBinDir, AssetSelection{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan := append(binaryPlan, systemPlan...)
	pkg := &Package{Owner: "jsnjack", Repo: "bunk"}
	config := &GrmConfig{Packages: map[string]Package{}}
	if err := validateInstallPlan(plan, pkg, config); err != nil {
		t.Fatal(err)
	}
	if systemPlan[0].component.SystemPkgName != "bunk-gui" || systemPlan[0].component.SystemPkgType != "rpm" {
		t.Fatalf("wrong RPM: %+v", systemPlan)
	}
}

func TestValidateInstallPlanConflicts(t *testing.T) {
	cli := plannedComponent{component: InstalledComponent{Asset: "cli.zip", Filename: "/usr/local/bin/tool"}}
	rpm := plannedComponent{component: InstalledComponent{Asset: "gui.rpm", SystemPkgName: "gui", SystemPkgType: "rpm"}}
	cases := []struct {
		name      string
		plan      []plannedComponent
		installed map[string]Package
		bad       bool
	}{
		{"distinct outputs", []plannedComponent{cli, rpm}, nil, false},
		{"duplicate binary destination", []plannedComponent{cli, cli}, nil, true},
		{"duplicate package identity", []plannedComponent{rpm, rpm}, nil, true},
		{"another repository owns binary", []plannedComponent{cli}, map[string]Package{"other/tool": {Owner: "other", Repo: "tool", Filename: cli.component.Filename}}, true},
		{"another repository owns RPM", []plannedComponent{rpm}, map[string]Package{"other/gui": {Owner: "other", Repo: "gui", Components: []InstalledComponent{rpm.component}}}, true},
	}
	overlapping := rpm
	overlapping.systemFiles = []string{cli.component.Filename}
	cases = append(cases, struct {
		name      string
		plan      []plannedComponent
		installed map[string]Package
		bad       bool
	}{"RPM supplies selected binary", []plannedComponent{cli, overlapping}, nil, true})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInstallPlan(tc.plan, &Package{Owner: "owner", Repo: "tool"}, &GrmConfig{Packages: tc.installed})
			if (err != nil) != tc.bad {
				t.Fatalf("got %v, want error=%v", err, tc.bad)
			}
		})
	}
}

func TestApplyInstallPlanPersistsProgress(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "RPM fails after binary"}[fail], func(t *testing.T) {
			config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(t.TempDir(), "bunk")
			if err := os.WriteFile(destination, []byte("old"), 0755); err != nil {
				t.Fatal(err)
			}
			pkg := &Package{Owner: "jsnjack", Repo: "bunk", Version: "v1", Filename: destination, SystemPkgName: "", Selections: []AssetSelection{{Pattern: "bunk.zip"}, {Pattern: "bunk-gui*.rpm"}}}
			if err := config.PutPackage(pkg); err != nil {
				t.Fatal(err)
			}
			plan := []plannedComponent{
				{source: "binary", component: InstalledComponent{Asset: "bunk.zip", Filename: destination, Version: "v2"}},
				{source: "rpm", component: InstalledComponent{Asset: "gui.rpm", SystemPkgName: "bunk-gui", SystemPkgType: "rpm", Version: "v2"}},
			}
			operations := installOperations{
				binary: func(source, target string) error { return os.WriteFile(target, []byte("new"), 0755) },
				system: func(source string) error {
					if fail {
						return errors.New("RPM failed")
					}
					return nil
				},
				remove: func(component InstalledComponent) error { t.Fatalf("unexpected removal %+v", component); return nil },
			}
			err = applyInstallPlan(config, pkg, "v2", plan, operations)
			if (err != nil) != fail {
				t.Fatalf("got %v, want error=%v", err, fail)
			}
			reloaded, err := ReadConfig(config.path)
			if err != nil {
				t.Fatal(err)
			}
			saved := reloaded.Packages[pkg.GetFullName()]
			if saved.Filename != "" || saved.SystemPkgName != "" {
				t.Fatalf("legacy fields were not migrated: %+v", saved)
			}
			if len(saved.Components) == 0 || saved.Components[0].Version != "v2" {
				t.Fatalf("successful binary was lost: %+v", saved)
			}
			if fail {
				if saved.Version != "v1" || len(saved.Components) != 1 {
					t.Fatalf("partial run marked complete: %+v", saved)
				}
				if saved.VerifyVersion("v2") == nil {
					t.Fatal("partial run verified as complete")
				}
			} else {
				if len(saved.Components) != 2 || saved.VerifyVersion("v2") != nil {
					t.Fatalf("complete install not tracked: %+v", saved)
				}
				for _, component := range saved.Components {
					if component.Version != "v2" {
						t.Fatal("component versions diverged")
					}
				}
			}
		})
	}
}

func TestRemoveInstalledPackagePersistsProgress(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "all removed", true: "second removal fails"}[fail], func(t *testing.T) {
			config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			pkg := &Package{Owner: "owner", Repo: "repo", Version: "v1", Components: []InstalledComponent{
				{Filename: "/test/bunk", MD5: "known", Version: "v1"},
				{SystemPkgName: "bunk-gui", SystemPkgType: "rpm", Version: "v1"},
			}}
			if err := config.PutPackage(pkg); err != nil {
				t.Fatal(err)
			}
			var removed []string
			err = removeInstalledPackage(config, pkg, func(component InstalledComponent) error {
				if fail && component.SystemPkgName != "" {
					return errors.New("remove failed")
				}
				removed = append(removed, component.identity())
				return nil
			})
			if (err != nil) != fail {
				t.Fatalf("got %v", err)
			}
			reloaded, err := ReadConfig(config.path)
			if err != nil {
				t.Fatal(err)
			}
			saved, exists := reloaded.Packages[pkg.GetFullName()]
			if fail {
				if !exists || len(saved.Components) != 1 || saved.Components[0].SystemPkgName != "bunk-gui" || saved.Version != "" {
					t.Fatalf("lost remaining component: %+v", saved)
				}
			} else if exists || len(removed) != 2 {
				t.Fatalf("not fully removed: %+v, %v", saved, removed)
			}
		})
	}
}

func TestCheckBinaryOwnership(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "bunk")
	if err := os.WriteFile(filename, fixtureBinary, 0755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fakeProgram(t, dir, "rpm", "echo another-package")
	t.Setenv("PATH", dir)
	if err := checkBinaryOwnership(filename); err == nil || !strings.Contains(err.Error(), "another-package") {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
}

func TestSameReleaseFailureRemainsIncomplete(t *testing.T) {
	config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := &Package{Owner: "owner", Repo: "repo", Version: "v1", Components: []InstalledComponent{{SystemPkgName: "cli", SystemPkgType: "rpm", Version: "v1"}}}
	if err := config.PutPackage(pkg); err != nil {
		t.Fatal(err)
	}
	plan := []plannedComponent{
		{component: InstalledComponent{SystemPkgName: "cli", SystemPkgType: "rpm", Version: "v1"}},
		{component: InstalledComponent{SystemPkgName: "gui", SystemPkgType: "rpm", Version: "v1"}},
	}
	calls := 0
	err = applyInstallPlan(config, pkg, "v1", plan, installOperations{system: func(string) error {
		calls++
		if calls == 2 {
			return errors.New("second package failed")
		}
		return nil
	}})
	if err == nil {
		t.Fatal("expected failure")
	}
	config, err = ReadConfig(config.path)
	if err != nil {
		t.Fatal(err)
	}
	saved := config.Packages[pkg.GetFullName()]
	if !saved.Incomplete || saved.VerifyVersion("v1") == nil {
		t.Fatalf("same-release partial install was considered complete: %+v", saved)
	}
}

func TestApplyInstallPlanRemovesObsoleteOutputsAfterSuccess(t *testing.T) {
	config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := &Package{Owner: "owner", Repo: "repo", Version: "v1", Components: []InstalledComponent{{SystemPkgName: "old", SystemPkgType: "rpm", Version: "v1"}}}
	if err := config.PutPackage(pkg); err != nil {
		t.Fatal(err)
	}
	var events []string
	plan := []plannedComponent{{component: InstalledComponent{SystemPkgName: "new", SystemPkgType: "rpm", Version: "v2"}}}
	err = applyInstallPlan(config, pkg, "v2", plan, installOperations{
		system: func(string) error { events = append(events, "install"); return nil },
		remove: func(component InstalledComponent) error {
			events = append(events, "remove "+component.SystemPkgName)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, []string{"install", "remove old"}) {
		t.Fatalf("wrong order: %v", events)
	}
	if len(pkg.Components) != 1 || pkg.Components[0].SystemPkgName != "new" || pkg.Incomplete {
		t.Fatalf("wrong final state: %+v", pkg)
	}
}

func TestRPMFallbackCanRetryAndPreservesArguments(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(t.TempDir(), "args")
	t.Setenv("GRM_TEST_ARGS", capture)
	fakeProgram(t, dir, "rpm", `printf '%s\n' "$@" > "$GRM_TEST_ARGS"`)
	fakeProgram(t, dir, "sudo", `exec "$@"`)
	t.Setenv("PATH", dir)
	filename := "/tmp/a package with spaces.rpm"
	if err := installSystemPackage(filename); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "-U\n--replacepkgs\n"+filename+"\n" {
		t.Fatalf("wrong RPM arguments: %q", got)
	}
}

func TestFailedFirstOperationRemembersIntent(t *testing.T) {
	config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := &Package{Owner: "owner", Repo: "repo", Version: "v1", Components: []InstalledComponent{{SystemPkgName: "cli", SystemPkgType: "rpm", Version: "v1"}}, Selections: []AssetSelection{{Pattern: "gui*.rpm"}}}
	if err := config.PutPackage(pkg); err != nil {
		t.Fatal(err)
	}
	plan := []plannedComponent{{component: InstalledComponent{SystemPkgName: "gui", SystemPkgType: "rpm", Version: "v1"}}}
	err = applyInstallPlan(config, pkg, "v1", plan, installOperations{system: func(string) error { return errors.New("install failed") }})
	if err == nil {
		t.Fatal("expected failure")
	}
	config, err = ReadConfig(config.path)
	if err != nil {
		t.Fatal(err)
	}
	saved := config.Packages[pkg.GetFullName()]
	if !saved.Incomplete || len(saved.Components) != 1 || saved.Components[0].SystemPkgName != "cli" || saved.Selections[0].Pattern != "gui*.rpm" {
		t.Fatalf("retry intent lost: %+v", saved)
	}
}

func TestConfirmInstallPlanPromptsOnlyForRemovals(t *testing.T) {
	withYes(t, false)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	logClose(writer)
	old := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = old; logClose(reader) })

	plan := []plannedComponent{{component: InstalledComponent{Asset: "grm_linux_amd64.tar.gz", Filename: "/usr/local/bin/grm"}}}
	pkg := &Package{Owner: "jsnjack", Repo: "grm", Components: []InstalledComponent{{Filename: "/usr/local/bin/grm"}}}
	confirmed, err := confirmInstallPlan(pkg, "v2", plan)
	if err != nil || !confirmed {
		t.Fatalf("plan without removals got %v,%v; want true without prompting", confirmed, err)
	}

	pkg.Components = append(pkg.Components, InstalledComponent{Filename: "/usr/local/bin/old"})
	if _, err := confirmInstallPlan(pkg, "v2", plan); !errors.Is(err, io.EOF) {
		t.Fatalf("plan with removals got %v; want a prompt that reads EOF", err)
	}
}
