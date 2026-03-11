package cmd

import (
	"fmt"
	"strings"
)

// Package represents github package
type Package struct {
	Repo           string
	Owner          string
	Version        string
	VersionFilter  string // ~= prefix/glob filter; resolved to Version after install
	MD5            string
	Filter         []string
	Locked         bool
	Filename       string
	RenameBinaryTo string
	SystemPkgName  string
	SystemPkgType  string // "deb" or "rpm"
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

// IsSystemPackage returns true if this package was installed via a system package manager
func (p *Package) IsSystemPackage() bool {
	return p.SystemPkgName != ""
}

// VerifyVersion verifies that correct package version is installed
func (p *Package) VerifyVersion(version string) error {
	if version != p.Version {
		return fmt.Errorf("installed version %s, want %s", p.Version, version)
	}
	if p.Filename != "" {
		hash, _ := tomd5(p.Filename)
		if p.MD5 != hash {
			return fmt.Errorf("installed file hash %s, want %s", p.MD5, hash)
		}
	}
	return nil
}

// CreatePackage creates new Package instance from a string
// jsnjack/kazy-go==v1.1.0  (exact version)
// jsnjack/kazy-go~=v146    (version filter: prefix or glob)
func CreatePackage(text string) (*Package, error) {
	p := Package{}

	// Extract version specifier.
	// ~= is checked first because it contains = which is a subset of ==.
	packageName := text
	if splitFilter := strings.SplitN(text, "~=", 2); len(splitFilter) == 2 {
		p.VersionFilter = splitFilter[1]
		packageName = splitFilter[0]
		// Detect accidental use of both operators, e.g. owner/repo==v1~=v2
		if strings.Contains(packageName, "==") {
			return nil, fmt.Errorf("cannot use both == and ~= operators")
		}
	} else if splitVersion := strings.SplitN(text, "==", 2); len(splitVersion) == 2 {
		p.Version = splitVersion[1]
		packageName = splitVersion[0]
	}

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
		return nil, fmt.Errorf("invalid package: expected <owner>/<repo>[==<version>|~=<filter>], got %s", packageName)
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
