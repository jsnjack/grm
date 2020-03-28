package cmd

import (
	"fmt"
	"strings"
)

// cleanPackage splits package into owner and repo
// should be in format jsnjack/kazy-go
func cleanPackage(pkg string) (string, string, error) {
	split := strings.Split(pkg, "/")
	if len(split) != 2 {
		return "", "", fmt.Errorf("Invalid package: expected <owner>/<repo>, got %s", pkg)
	}
	return split[0], split[1], nil
}
