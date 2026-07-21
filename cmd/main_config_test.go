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
		wantErr bool
	}{
		{"known key", "token", false},
		{"unknown key", "bogus", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config, err := ReadConfig(filepath.Join(t.TempDir(), "grm.yaml"))
			if err != nil {
				t.Fatalf("setup: %s", err)
			}
			err = config.PutSetting(c.key, "value")
			if c.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %s", err)
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
