package cmd

import (
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v30/github"
)

// Install installs binary from an asset
func Install(asset *github.ReleaseAsset) (string, error) {
	filename, err := downloadFile(asset.GetBrowserDownloadURL(), asset.GetName())
	if err != nil {
		return "", err
	}
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return "", err
	}
	ct, err := getFileContentType(file)
	if err != nil {
		return "", err
	}
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
