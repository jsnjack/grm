package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jsnjack/archiver/v3"
)

func installArchive(filename string) (string, error) {
	fmt.Println("Unpacking archive...", filename)
	tmpDir := filepath.Dir(filename)
	err := archiver.Unarchive(filename, tmpDir)
	if err != nil {
		return "", err
	}

	fmt.Println("Looking for a binary file...")
	filenameA := ""
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			ct, err := getFileContentType(f)
			if err != nil {
				return err
			}
			fmt.Printf("  %-40s %s\n", path, ct)
			if filenameA == "" && ct == "application/octet-stream" {
				filenameA = path
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if filenameA == "" {
		return "", fmt.Errorf("Unable to find a binary file in archive")
	}
	return installBinary(filenameA)
}
