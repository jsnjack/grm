package cmd

import (
	"io"
	"net/http"
	"os"

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
	defer file.Close()
	if err != nil {
		return "", err
	}

	ct, err := getFileContentType(file)
	if err != nil {
		return "", err
	}
	logf("Content type %s\n", ct)

	switch ct {
	case "application/octet-stream":
		return installBinary(filename)
	}
	return installArchive(filename)
}

func getFileContentType(out io.Reader) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}
