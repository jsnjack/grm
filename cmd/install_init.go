package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/go-github/v32/github"
)

// Install installs binary from an asset
func Install(asset *github.ReleaseAsset, pkg *Package) (string, error) {
	filename, err := downloadFile(asset, pkg)
	if err != nil {
		return "", err
	}
	slog.Debug("installing", "path", filename)

	// Route system packages by extension before MIME detection
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".deb" || ext == ".rpm" {
		return installSystemPackage(filename, pkg)
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filename, err)
	}
	defer logClose(file)

	ct, err := getFileType(file)
	if err != nil {
		return "", fmt.Errorf("detect file type of %s: %w", filename, err)
	}
	slog.Debug("detected file type", "type", ct)

	if isExecutableFileType(ct) {
		return installBinary(filename, pkg.RenameBinaryTo)
	}
	return installArchive(filename, pkg.RenameBinaryTo)
}

func getFileType(out io.Reader) (string, error) {
	kind, err := mimetype.DetectReader(out)
	if err != nil {
		return "", fmt.Errorf("detect mime type: %w", err)
	}
	return kind.String(), nil
}

func isExecutableFileType(ct string) bool {
	switch ct {
	case "application/octet-stream", "application/x-executable", "application/x-elf", "application/x-sharedlib", "application/x-mach-binary":
		return true
	}
	return false
}
