package install

import (
	"fmt"
	"io"
	"os"

	"github.com/google/go-github/v30/github"
)

// Application ...
func Application(asset *github.ReleaseAsset) error {
	filename, err := downloadFile(asset.GetBrowserDownloadURL(), asset.GetName())
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := fmt.Sprintf(home + "/.grm/bin/")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	from, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(path+asset.GetName(), os.O_RDWR|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return err
}
