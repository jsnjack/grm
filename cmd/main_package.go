package cmd

import (
	"fmt"
	"strings"
)

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

	// Extract owner
	split := strings.Split(text, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("Invalid package: expected <owner>/<repo>==<version>, got %s", text)
	}
	p.Owner = split[0]

	// Extract version and repo
	split2 := strings.SplitN(split[1], "==", 2)
	p.Repo = split2[0]
	if len(split2) == 2 {
		p.Version = split2[1]
	}

	// Verify
	if p.Owner == "" {
		return nil, fmt.Errorf("Got empty <owner> from %s", text)
	}
	if p.Repo == "" {
		return nil, fmt.Errorf("Got empty <repo> from %s", text)
	}

	return &p, nil
}
