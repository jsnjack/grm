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
	msgStep("Unpacking archive %s", bold(strings.TrimPrefix(filename, tmpDir)))
	err := archiver.Unarchive(filename, tmpDir)
	if err != nil {
		// Check if maybe it is a compressed file
		dest := filepath.Join(tmpDir, strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		err = archiver.DecompressFile(filename, dest)
		if err != nil {
			return "", err
		}
		logf("Decompressed to %s\n", dest)
	} else {
		logf("Unpacked to %s\n", tmpDir)
	}

	msgStep("Looking for a binary file...")
	filenameA, err := findBinaryFile(tmpDir)
	if err != nil {
		return "", err
	}
	return installBinary(filenameA, renameBinaryTo)
}

type fileEntry struct {
	relPath  string
	fullPath string
	ct       string
	isBinary bool
}

// findBinaryFile finds a binary file in the given directory
func findBinaryFile(tmpDir string) (string, error) {
	var files []fileEntry
	err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		ct, err := getFileType(f)
		if err != nil {
			ct = "unknown"
		}
		relPath := strings.TrimPrefix(path, tmpDir)
		filename := filepath.Base(path)
		// Ignore files starting with "._" (macOS), they are not executable
		// https://github.com/jsnjack/grm/issues/12
		isBinary := isExecutableFileType(ct) && !strings.HasPrefix(filename, "._")
		files = append(files, fileEntry{relPath, path, ct, isBinary})
		return nil
	})
	if err != nil {
		return "", err
	}

	// Find max filename width for alignment
	maxW := 0
	for _, f := range files {
		if len(f.relPath) > maxW {
			maxW = len(f.relPath)
		}
	}

	// Print and select
	var binaryFilepath string
	for _, f := range files {
		if f.isBinary {
			fmt.Printf("    %s %s %s\n", green(sCheck), padCol(f.relPath, maxW, nil), green(f.ct))
		} else {
			fmt.Printf("    %s %s %s\n", dim(sTriang), padCol(f.relPath, maxW, dim), dim(f.ct))
		}
		if binaryFilepath == "" && f.isBinary {
			binaryFilepath = f.fullPath
		}
	}

	if binaryFilepath == "" {
		return "", fmt.Errorf("unable to find a binary file in archive")
	}
	return binaryFilepath, nil
}
