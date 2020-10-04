package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jsnjack/archiver/v3"
)

func installArchive(filename string) (string, error) {
	// Walk the archive to find binary
	fmt.Println("Looking for a binary file...")
	filenameA := ""
	err := archiver.Walk(filename, func(f archiver.File) error {
		if f.IsDir() {
			fmt.Printf("  %-40s %s\n", f.Path, "dir")
			return nil
		}
		ct, err := getFileContentType(f)
		if err != nil {
			return err
		}
		fmt.Printf("  %-40s %s\n", f.Path, ct)
		if filenameA == "" && ct == "application/octet-stream" {
			// Strange special case fo zip files (chromedriver)
			if f.Path == "" {
				filenameA = f.Name()
			} else {
				filenameA = f.Path
			}
		}
		return nil
	})
	if filenameA == "" {
		return "", fmt.Errorf("Unable to find a binary file in archive")
	}
	fmt.Printf("Extracting file %s...\n", filenameA)

	tmpDir := filepath.Dir(filename)
	err = archiver.Extract(filename, filenameA, tmpDir)
	if err != nil {
		return "", err
	}
	fmt.Println("done")

	return installBinary(tmpDir + "/" + filenameA)
}
