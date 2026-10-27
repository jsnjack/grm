package cmd

import (
	"fmt"
	"log/slog"
	"strings"
)

// Package represents github package
type Package struct {
	Repo           string
	Owner          string
	Version        string
	VersionFilter  string // ~= prefix/glob filter; resolved to Version after install
	MD5            string `yaml:"md5,omitempty"`
	Filter         []string
	Locked         bool
	Incomplete     bool                 `yaml:"incomplete,omitempty"`
	Filename       string               `yaml:"filename,omitempty"`
	RenameBinaryTo string               `yaml:"renamebinaryto,omitempty"`
	SystemPkgName  string               `yaml:"systempkgname,omitempty"`
	SystemPkgType  string               `yaml:"systempkgtype,omitempty"` // "deb" or "rpm"
	Selections     []AssetSelection     `yaml:"selections,omitempty"`
	Components     []InstalledComponent `yaml:"components,omitempty"`
}

// AssetSelection remembers an asset pattern and the binaries selected within it.
type AssetSelection struct {
	Pattern  string            `yaml:"pattern"`
	Binaries []BinarySelection `yaml:"binaries,omitempty"`
}

// BinarySelection identifies an archive member and its optional installed name.
type BinarySelection struct {
	Path   string `yaml:"path"`
	Rename string `yaml:"rename,omitempty"`
}

// InstalledComponent records one installed binary or system package.
type InstalledComponent struct {
	Asset         string `yaml:"asset,omitempty"`
	Source        string `yaml:"source,omitempty"`
	Version       string `yaml:"version"`
	Filename      string `yaml:"filename,omitempty"`
	MD5           string `yaml:"md5,omitempty"`
	SystemPkgName string `yaml:"systempkgname,omitempty"`
	SystemPkgType string `yaml:"systempkgtype,omitempty"`
}

func (p *Package) installedComponents() []InstalledComponent {
	if len(p.Components) != 0 {
		return p.Components
	}
	if p.Filename == "" && p.SystemPkgName == "" {
		return nil
	}
	var components []InstalledComponent
	if p.Filename != "" {
		components = append(components, InstalledComponent{Version: p.Version, Filename: p.Filename, MD5: p.MD5})
	}
	if p.SystemPkgName != "" {
		components = append(components, InstalledComponent{Version: p.Version, SystemPkgName: p.SystemPkgName, SystemPkgType: p.SystemPkgType})
	}
	return components
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
	for _, component := range p.installedComponents() {
		if component.SystemPkgName != "" {
			return true
		}
	}
	return false
}

// VerifyVersion verifies that correct package version is installed
func (p *Package) VerifyVersion(version string) error {
	if p.Incomplete {
		return fmt.Errorf("installation of %s is incomplete", p.GetFullName())
	}
	if version != p.Version {
		return fmt.Errorf("installed version %s, want %s", p.Version, version)
	}
	for _, component := range p.installedComponents() {
		if component.Version != version {
			return fmt.Errorf("component %s is at %s, want %s", component.identity(), component.Version, version)
		}
		if component.Filename == "" {
			continue
		}
		hash, err := tomd5(component.Filename)
		if err != nil {
			return fmt.Errorf("verify %s: %w", component.Filename, err)
		}
		if component.MD5 != hash {
			return fmt.Errorf("verify %s: installed file hash %s, want %s", component.Filename, hash, component.MD5)
		}
	}
	return nil
}

func (c InstalledComponent) identity() string {
	if c.Filename != "" {
		return c.Filename
	}
	return c.SystemPkgType + ":" + c.SystemPkgName
}

// CreatePackage creates new Package instance from a string
// jsnjack/kazy-go==v1.1.0  (exact version)
// jsnjack/kazy-go~=v146    (version filter: prefix or glob)
func CreatePackage(text string) (*Package, error) {
	p := Package{}

	// Extract version specifier.
	// ~= is checked first because it contains = which is a subset of ==.
	// @ is supported as an alias for == (e.g. owner/repo@v1.0.0).
	packageName := text
	if splitFilter := strings.SplitN(text, "~=", 2); len(splitFilter) == 2 {
		p.VersionFilter = splitFilter[1]
		packageName = splitFilter[0]
		// Detect accidental use of both operators, e.g. owner/repo==v1~=v2
		if strings.Contains(packageName, "==") || strings.Contains(packageName, "@") {
			return nil, fmt.Errorf("cannot combine version specifiers")
		}
	} else if splitVersion := strings.SplitN(text, "==", 2); len(splitVersion) == 2 {
		p.Version = splitVersion[1]
		packageName = splitVersion[0]
		if strings.Contains(packageName, "@") || strings.Contains(p.Version, "@") {
			return nil, fmt.Errorf("cannot combine version specifiers")
		}
	} else if splitAt := strings.SplitN(text, "@", 2); len(splitAt) == 2 {
		p.Version = splitAt[1]
		packageName = splitAt[0]
	}

	// Check if it is one of the known aliases
	alias, ok := KnownAliases[packageName]
	if ok {
		slog.Debug("resolved alias", "input", text, "alias", alias)
		packageName = alias
	}

	// Extract owner
	split := strings.Split(packageName, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("invalid package: expected <owner>/<repo>[==<version>|@<version>|~=<filter>], got %s", packageName)
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
