package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

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
	g.Settings[key] = value
	return g.save()
}

// ReadConfig reads config file
func ReadConfig(path string) (*GrmConfig, error) {
	config := GrmConfig{}

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
	}
	config.path = path
	return &config, nil
}
