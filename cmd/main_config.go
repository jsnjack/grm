package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

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
	data, err := yaml.Marshal(g)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(g.path, []byte(data), 0644)
	return err
}

// PutPackage saves package to config file
func (g *GrmConfig) PutPackage(pkg *Package) error {
	hash, err := tomd5(pkg.Filename)
	if err != nil {
		return err
	}
	pkg.MD5 = hash
	g.Packages[pkg.GetFullName()] = *pkg
	return g.save()
}

// PutSetting saves a setting in config
func (g *GrmConfig) PutSetting(key string, value string) error {
	_, ok := Settings[key]
	if !ok {
		return fmt.Errorf("Unknown key: %s", key)
	}
	g.Settings[key] = value
	return g.save()
}

// ReadConfig reads config file
func ReadConfig(path string) (*GrmConfig, error) {
	config := GrmConfig{}
	config.path = path

	// Verify that config file exists
	if _, err := os.Stat(path); err == nil {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return nil, err
		}
	} else {
		// Config is empty
		fmt.Printf("Initializing config in %s...\n", path)
		err = config.save()
		if err != nil {
			return nil, err
		}
	}
	return &config, nil
}
