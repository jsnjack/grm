package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	"token": "GitHub API token",
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
	if err := os.WriteFile(g.path, data, 0644); err != nil {
		return fmt.Errorf("write config %s: %w", g.path, err)
	}
	return nil
}

// PutPackage saves package to config file
func (g *GrmConfig) PutPackage(pkg *Package) error {
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
	g.Settings[key] = value
	return g.save()
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
	return &config, nil
}
