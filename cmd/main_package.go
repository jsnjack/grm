package cmd

import (
	"fmt"
	"strings"
)

// Package represents github package
type Package struct {
	Repo    string
	Owner   string
	Version string
	Filter  string
	Hold    string
}

// GetFullName returns full package name, e.g. jsnjack/kazy-go
func (p *Package) GetFullName() string {
	return p.Owner + "/" + p.Repo
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
