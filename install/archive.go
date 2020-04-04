package install

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

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
	fmt.Println("Looking for a binary file...")
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
		return fmt.Errorf("Unable to find a binary file in archive")
	}
	fmt.Printf("Extracting file %s...\n", filenameA)

	tmpDir := filepath.Dir(filename)
	err = archiver.Extract(filename, filenameA, tmpDir)
	if err == nil {
		fmt.Println("done")
	}

	err = installBinary(tmpDir + "/" + filenameA)
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
