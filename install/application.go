package install

import (
	"github.com/google/go-github/v30/github"
)

// Application handles binary assets
func Application(asset *github.ReleaseAsset) error {
	filename, err := downloadFile(asset.GetBrowserDownloadURL(), asset.GetName())
	if err != nil {
		return err
	}
	err = installBinary(filename)
	return err
}
