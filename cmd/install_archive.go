package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

func installArchive(filename string, renameBinaryTo string) (string, error) {
	logln("Installing from an archive")
	tmpDir := getTmpDir(filename)
	fmt.Println("Unpacking archive...", strings.TrimPrefix(filename, tmpDir))
	err := archiver.Unarchive(filename, tmpDir)
	if err != nil {
		return "", err
	}
	logf("Unpacked to %s\n", tmpDir)

	fmt.Println("Looking for a binary file...")
	filenameA, err := findBinaryFile(tmpDir)
	if err != nil {
		return "", err
	}
	return installBinary(filenameA, renameBinaryTo)
}

// findBinaryFile finds a binary file in the given directory
func findBinaryFile(tmpDir string) (string, error) {
	var binaryFilepath string
	err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			ct, err := getFileType(f)
			if err != nil {
				ct = "unknown"
			}
			fmt.Printf("  %-50s %s\n", strings.TrimPrefix(path, tmpDir), ct)
			filename := filepath.Base(path)
			// Ignore files starting with "._" (macOS), they are not executable
			// https://github.com/jsnjack/grm/issues/12
			if binaryFilepath == "" && isExecutableFileType(ct) && !strings.HasPrefix(filename, "._") {
				binaryFilepath = path
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if binaryFilepath == "" {
		return "", fmt.Errorf("unable to find a binary file in archive")
	}
	return binaryFilepath, nil
}
