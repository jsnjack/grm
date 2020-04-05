package cmd

import (
	"github.com/google/go-github/v30/github"
)

// Application handles binary assets
func Application(asset *github.ReleaseAsset) (string, error) {
	filename, err := downloadFile(asset.GetBrowserDownloadURL(), asset.GetName())
	if err != nil {
		return "", err
	}
	return installBinary(filename)
}
