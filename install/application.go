package install

import (
	"fmt"
	"os/exec"

	"github.com/google/go-github/v30/github"
)

// DefaultBinDir is the default location for binary files
const DefaultBinDir = "/usr/local/bin/"

// Application handles binary assets
func Application(asset *github.ReleaseAsset) error {
	filename, err := downloadFile(asset.GetBrowserDownloadURL(), asset.GetName())
	if err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo cp %s %s", filename, DefaultBinDir))
	err = cmd.Run()
	return err
}
