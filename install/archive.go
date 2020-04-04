package install

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mholt/archiver/v3"

	"github.com/google/go-github/v30/github"
)

// Archive handles compressed assets
func Archive(asset *github.ReleaseAsset) error {
	filename, err := downloadFile(asset.GetBrowserDownloadURL(), asset.GetName())
	if err != nil {
		return err
	}

	// Walk the archive to find binary
	fmt.Println("Looking for file...")
	filenameA := ""
	err = archiver.Walk(filename, func(f archiver.File) error {
		ct, err := getFileContentType(f)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		fmt.Printf("  %-40s %s\n", f.Name(), ct)
		if filenameA == "" && ct == "application/octet-stream" {
			filenameA = f.Name()
		}
		return nil
	})
	if filenameA == "" {
		return fmt.Errorf("Unable to find binary file in archive")
	}
	fmt.Printf("Extracting file %s...\n", filenameA)

	// Remove file if it is already exists
	_, err = os.Stat(DefaultBinDir + filenameA)
	if err == nil {
		err = os.Remove(DefaultBinDir + filenameA)
		if err != nil {
			return err
		}
	}

	err = archiver.Extract(filename, filenameA, DefaultBinDir)
	if err == nil {
		fmt.Println("done")
	}
	return err
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
