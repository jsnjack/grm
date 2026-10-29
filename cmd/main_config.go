package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// resolveConfigFile returns the path to the config file. If override is
// non-empty (from --config), it is used as-is. Otherwise it resolves the
// OS-standard config directory, falling back to the legacy
// ~/.config/grm/grm.yaml location when the standard path has no config
// file but the legacy one does — this keeps pre-XDG installs working,
// notably on macOS where the standard directory differs.
func resolveConfigFile(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine user config directory: %w", err)
	}
	path := filepath.Join(configDir, "grm", "grm.yaml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if home, herr := os.UserHomeDir(); herr == nil {
			legacy := filepath.Join(home, ".config", "grm", "grm.yaml")
			if _, lerr := os.Stat(legacy); lerr == nil {
				return legacy, nil
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	return path, nil
}

// Settings contains map of all available settings setting/description
var Settings = map[string]string{
	"token":   "GitHub API token",
	"bin_dir": "Absolute path of the directory for installed binaries (default " + DefaultBinDir + ")",
}

// GrmConfig represents grm configuration
type GrmConfig struct {
	Packages map[string]Package `yaml:"packages"`
	Settings map[string]string  `yaml:"settings"`
	path     string
}

func (g *GrmConfig) save() error {
	slog.Debug("saving config", "path", g.path)
	data, err := yaml.Marshal(g)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := g.path
	mode := os.FileMode(0644)
	if _, err := os.Lstat(path); err == nil {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve config %s: %w", g.path, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("inspect config %s: %w", path, err)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect config %s: %w", path, err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".grm-*.yaml")
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
			slog.Log(context.Background(), LevelTrace, "remove temporary config", "err", err)
		}
	}()
	if err := file.Chmod(mode); err != nil {
		logClose(file)
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		logClose(file)
		return fmt.Errorf("write config %s: %w", g.path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close config %s: %w", g.path, err)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("replace config %s: %w", g.path, err)
	}
	return nil
}

// PutPackage saves package to config file
func (g *GrmConfig) PutPackage(pkg *Package) error {
	for i := range pkg.Components {
		component := &pkg.Components[i]
		if component.Filename != "" && component.MD5 == "" {
			hash, err := tomd5(component.Filename)
			if err != nil {
				return fmt.Errorf("hash %s: %w", component.Filename, err)
			}
			component.MD5 = hash
		}
	}
	if pkg.Filename != "" {
		hash, err := tomd5(pkg.Filename)
		if err != nil {
			return fmt.Errorf("hash %s: %w", pkg.Filename, err)
		}
		pkg.MD5 = hash
	}
	g.Packages[pkg.GetFullName()] = *pkg
	return g.save()
}

// PutSetting saves a setting in config
func (g *GrmConfig) PutSetting(key string, value string) error {
	_, ok := Settings[key]
	if !ok {
		return fmt.Errorf("unknown key: %s", key)
	}
	if key == "bin_dir" && value != "" && !filepath.IsAbs(value) {
		return fmt.Errorf("bin_dir must be an absolute path, got %s", value)
	}
	g.Settings[key] = value
	return g.save()
}

// BinDir returns the directory for installed binaries: the bin_dir setting
// when it is set, otherwise DefaultBinDir.
func (g *GrmConfig) BinDir() string {
	if dir := g.Settings["bin_dir"]; dir != "" {
		return dir
	}
	return DefaultBinDir
}

// ReadConfig reads config file
func ReadConfig(path string) (*GrmConfig, error) {
	slog.Debug("loading config", "path", path)
	config := GrmConfig{}
	config.path = path

	// Verify that config file exists
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	} else {
		// Config is empty
		fmt.Printf("Initializing config in %s...\n", path)
		config.Settings = make(map[string]string)
		config.Packages = make(map[string]Package)
		if err := config.save(); err != nil {
			return nil, err
		}
	}
	if config.Packages == nil {
		config.Packages = make(map[string]Package)
	}
	if config.Settings == nil {
		config.Settings = make(map[string]string)
	}
	return &config, nil
}
