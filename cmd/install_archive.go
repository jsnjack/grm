package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mholt/archiver/v3"
)

func installArchive(filename string) (string, error) {
	// Walk the archive to find binary
	fmt.Println("Looking for a binary file...")
	filenameA := ""
	err := archiver.Walk(filename, func(f archiver.File) error {
		if f.IsDir() {
			fmt.Printf("  %-40s %s\n", f.Name(), "dir")
			return nil
		}
		ct, err := getFileContentType(f)
		if err != nil {
			return err
		}
		fmt.Printf("  %-40s %s\n", f.Name(), ct)
		if filenameA == "" && ct == "application/octet-stream" {
			filenameA = f.Name()
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
