package cmd

import (
	"fmt"
	"strings"
)

// KnownAliases is a list of well-known repositories to simplify binary
// installation from a release
var KnownAliases = map[string]string{
	"grm":          "jsnjack/grm",
	"kazy":         "jsnjack/kazy-go",
	"chromedriver": "jsnjack/chromedriver",
	"geckodriver":  "mozilla/geckodriver",
	"gotop":        "xxxserxxx/gotop",
}

// Package represents github package
type Package struct {
	Repo     string
	Owner    string
	Version  string
	MD5      string
	Filter   []string
	Locked   bool
	Filename string
}

// GetFullName returns full package name, e.g. jsnjack/kazy-go
func (p *Package) GetFullName() string {
	return p.Owner + "/" + p.Repo
}

// GetVerboseLocked returns print-friendly value
func (p *Package) GetVerboseLocked() string {
	if p.Locked {
		return "yes"
	}
	return ""
}

// VerifyVersion verifies that correct package version is installed
func (p *Package) VerifyVersion(version string) error {
	if version != p.Version {
		return fmt.Errorf("installed version %s, want %s", p.Version, version)
	}
	hash, _ := tomd5(p.Filename)
	if p.MD5 != hash {
		return fmt.Errorf("installed file hash %s, want %s", p.MD5, hash)
	}
	return nil
}

// CreatePackage creates new Package instance from a string
// jsnjack/kazy-go==v1.1.0
func CreatePackage(text string) (*Package, error) {
	p := Package{}

	// Extract version
	splitVersion := strings.SplitN(text, "==", 2)
	if len(splitVersion) == 2 {
		p.Version = splitVersion[1]
	}

	packageName := splitVersion[0]

	// Check if it is one of the known aliases
	alias, ok := KnownAliases[packageName]
	if ok {
		if rootVerbose {
			fmt.Printf("Found alias for '%s': %s\n", text, alias)
		}
		packageName = alias
	}

	// Extract owner
	split := strings.Split(packageName, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("invalid package: expected <owner>/<repo>==<version>, got %s", packageName)
	}
	p.Owner = split[0]
	p.Repo = split[1]

	// Verify
	if p.Owner == "" {
		return nil, fmt.Errorf("got empty <owner> from %s", packageName)
	}
	if p.Repo == "" {
		return nil, fmt.Errorf("got empty <repo> from %s", packageName)
	}

	return &p, nil
}
