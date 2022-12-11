package cmd

import (
	"io"
	"os"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/go-github/v32/github"
)

// Install installs binary from an asset
func Install(asset *github.ReleaseAsset, pkg *Package) (string, error) {
	filename, err := downloadFile(asset, pkg)
	if err != nil {
		return "", err
	}
	logf("Installing %s...\n", filename)
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	ct, err := getFileType(file)
	if err != nil {
		return "", err
	}
	logf("File type %s\n", ct)

	if isExecutableFileType(ct) {
		return installBinary(filename, pkg.RenameBinaryTo)
	}
	return installArchive(filename, pkg.RenameBinaryTo)
}

func getFileType(out io.Reader) (string, error) {
	kind, err := mimetype.DetectReader(out)
	if err != nil {
		return "", err
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
