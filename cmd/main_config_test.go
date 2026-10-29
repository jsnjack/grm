package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig_missing_creates_default(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grm.yaml")

	config, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if config.Packages == nil || config.Settings == nil {
		t.Errorf("expected initialized maps, got %+v", config)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected config file to be created: %s", err)
	}
}

func TestReadConfig_existing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grm.yaml")
	content := "packages:\n  jsnjack/grm:\n    owner: jsnjack\n    repo: grm\n    version: v1.0.0\nsettings:\n  token: abc\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("setup: %s", err)
	}

	config, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if config.Settings["token"] != "abc" {
		t.Errorf("expected token abc, got %s", config.Settings["token"])
	}
	pkg, ok := config.Packages["jsnjack/grm"]
	if !ok {
		t.Fatalf("expected package jsnjack/grm to be present")
	}
	if pkg.Version != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", pkg.Version)
	}
}

func TestGrmConfig_PutPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grm.yaml")
	config, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("setup: %s", err)
	}

	pkg := &Package{Owner: "jsnjack", Repo: "grm", Version: "v1.0.0"}
	if err := config.PutPackage(pkg); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	reloaded, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("reload: %s", err)
	}
	if _, ok := reloaded.Packages["jsnjack/grm"]; !ok {
		t.Errorf("expected package to be persisted")
	}
}

func TestGrmConfig_PutPackage_hashesFile(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "bin")
	if err := os.WriteFile(binPath, []byte("binary content"), 0644); err != nil {
		t.Fatalf("setup: %s", err)
	}

	config, err := ReadConfig(filepath.Join(dir, "grm.yaml"))
	if err != nil {
		t.Fatalf("setup: %s", err)
	}
	pkg := &Package{Owner: "jsnjack", Repo: "grm", Filename: binPath}
	if err := config.PutPackage(pkg); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want, err := tomd5(binPath)
	if err != nil {
		t.Fatalf("compute expected hash: %s", err)
	}
	if pkg.MD5 != want {
		t.Errorf("expected MD5 %s, got %s", want, pkg.MD5)
	}
}

func TestGrmConfig_PutSetting(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{"known key", "token", "value", false},
		{"unknown key", "bogus", "value", true},
		{"absolute bin_dir", "bin_dir", "/opt/bin", false},
		{"empty bin_dir", "bin_dir", "", false},
		{"relative bin_dir", "bin_dir", "bin", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
			if err != nil {
				t.Fatalf("setup: %s", err)
			}
			err = config.PutSetting(c.key, c.value)
			if c.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestGrmConfig_BinDir(t *testing.T) {
	cases := []struct {
		name     string
		settings map[string]string
		want     string
	}{
		{"unset", map[string]string{}, DefaultBinDir},
		{"empty", map[string]string{"bin_dir": ""}, DefaultBinDir},
		{"set", map[string]string{"bin_dir": "/home/user/.local/bin"}, "/home/user/.local/bin"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config := &GrmConfig{Settings: c.settings}
			if got := config.BinDir(); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestGrmConfigSavePreservesFile(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(map[bool]string{false: "private file", true: "symlink to private file"}[symlink], func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "settings.yaml")
			if err := os.WriteFile(target, []byte("settings: {}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			path := target
			if symlink {
				path = filepath.Join(dir, "grm.yaml")
				if err := os.Symlink(filepath.Base(target), path); err != nil {
					t.Fatal(err)
				}
			}
			config, err := ReadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := config.PutSetting("token", "test-token"); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(target)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("config permissions = %o, want 600", info.Mode().Perm())
			}
			if symlink {
				if _, err := os.Readlink(path); err != nil {
					t.Errorf("config symlink was replaced: %v", err)
				}
			}
			reloaded, err := ReadConfig(target)
			if err != nil {
				t.Fatal(err)
			}
			if reloaded.Settings["token"] != "test-token" {
				t.Error("setting was not saved in the original config file")
			}
		})
	}
}

func TestResolveConfigFile_override(t *testing.T) {
	got, err := resolveConfigFile("/custom/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != "/custom/path.yaml" {
		t.Errorf("expected override path, got %s", got)
	}
}

func TestResolveConfigFile_createsStandardDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := resolveConfigFile("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := filepath.Join(home, ".config", "grm", "grm.yaml")
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Errorf("expected config directory to be created: %s", err)
	}
}

func TestResolveConfigFile_fallsBackToLegacyPath(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdgconfig")
	if err := os.MkdirAll(xdg, os.ModePerm); err != nil {
		t.Fatalf("setup: %s", err)
	}
	legacyDir := filepath.Join(home, ".config", "grm")
	if err := os.MkdirAll(legacyDir, os.ModePerm); err != nil {
		t.Fatalf("setup: %s", err)
	}
	legacyPath := filepath.Join(legacyDir, "grm.yaml")
	if err := os.WriteFile(legacyPath, []byte("packages: {}"), 0644); err != nil {
		t.Fatalf("setup: %s", err)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := resolveConfigFile("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != legacyPath {
		t.Errorf("expected legacy path %s, got %s", legacyPath, got)
	}
}

// TestGrmConfigSaveFormat guards existing config files against formatting
// changes, such as a YAML library upgrade rewriting every user's config.
func TestGrmConfigSaveFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grm.yaml")
	config := &GrmConfig{path: path, Settings: map[string]string{"token": "abc"}, Packages: map[string]Package{
		"jsnjack/grm": {Owner: "jsnjack", Repo: "grm", Version: "v1.0.2", Filter: []string{"linux", "~=v1"}, Locked: true,
			Filename: "/usr/local/bin/grm", MD5: "d41d8cd98f00b204e9800998ecf8427e"},
		"sxyazi/yazi": {Owner: "sxyazi", Repo: "yazi", Version: "1.10", VersionFilter: "v1.*", Incomplete: true,
			Selections: []AssetSelection{{Pattern: "yazi-*-x86_64-unknown-linux-gnu.zip", Binaries: []BinarySelection{
				{Path: "yazi-*/ya"}, {Path: "yazi-*/yazi", Rename: "yes"}}}},
			Components: []InstalledComponent{
				{Asset: "yazi-1.10-x86_64-unknown-linux-gnu.zip", Source: "yazi-1.10/ya", Version: "1.10", Filename: "/usr/local/bin/ya", MD5: "0123"},
				{Asset: "yazi-gui-1.10.x86_64.rpm", Version: "1.10", SystemPkgName: "yazi-gui", SystemPkgType: "rpm"}}},
	}}
	if err := config.save(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != configGolden {
		t.Fatalf("saved config changed format:\n%s\nwant:\n%s", got, configGolden)
	}
}

const configGolden = `packages:
    jsnjack/grm:
        repo: grm
        owner: jsnjack
        version: v1.0.2
        versionfilter: ""
        md5: d41d8cd98f00b204e9800998ecf8427e
        filter:
            - linux
            - ~=v1
        locked: true
        filename: /usr/local/bin/grm
    sxyazi/yazi:
        repo: yazi
        owner: sxyazi
        version: "1.10"
        versionfilter: v1.*
        filter: []
        locked: false
        incomplete: true
        selections:
            - pattern: yazi-*-x86_64-unknown-linux-gnu.zip
              binaries:
                - path: yazi-*/ya
                - path: yazi-*/yazi
                  rename: "yes"
        components:
            - asset: yazi-1.10-x86_64-unknown-linux-gnu.zip
              source: yazi-1.10/ya
              version: "1.10"
              filename: /usr/local/bin/ya
              md5: "0123"
            - asset: yazi-gui-1.10.x86_64.rpm
              version: "1.10"
              systempkgname: yazi-gui
              systempkgtype: rpm
settings:
    token: abc
`
