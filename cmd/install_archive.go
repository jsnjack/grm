package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

func installArchive(filename string) (string, error) {
	tmpDir := getTmpDir(filename)
	fmt.Println("Unpacking archive...", strings.TrimPrefix(filename, tmpDir))
	err := archiver.Unarchive(filename, tmpDir)
	if err != nil {
		return "", err
	}
	logf("Unpacked to %s\n", tmpDir)

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
			fmt.Printf("  %-40s %s\n", strings.TrimPrefix(path, tmpDir), ct)
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
