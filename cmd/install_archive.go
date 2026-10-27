package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

func unpackArchive(filename string) (string, error) {
	slog.Debug("installing from an archive")
	tmpDir := filepath.Join(filepath.Dir(filename), "unpacked")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("create extraction directory: %w", err)
	}
	msgStep("Unpacking archive %s", bold(strings.TrimPrefix(filename, tmpDir)))
	err := archiver.Unarchive(filename, tmpDir)
	if err != nil {
		// Check if maybe it is a compressed file
		dest := filepath.Join(tmpDir, strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		if err := archiver.DecompressFile(filename, dest); err != nil {
			return "", fmt.Errorf("unpack %s: %w", filename, err)
		}
		slog.Debug("decompressed", "path", dest)
	} else {
		slog.Debug("unpacked", "path", tmpDir)
	}

	return tmpDir, nil
}

type fileEntry struct {
	relPath  string
	fullPath string
	ct       string
	isBinary bool
}

func findBinaryFiles(tmpDir string) ([]fileEntry, error) {
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
			return fmt.Errorf("open %s: %w", path, err)
		}
		defer logClose(f)
		ct, err := getFileType(f)
		if err != nil {
			ct = "unknown"
		}
		relPath, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", path, err)
		}
		relPath = filepath.ToSlash(relPath)
		filename := filepath.Base(path)
		// Ignore files starting with "._" (macOS), they are not executable
		// https://github.com/jsnjack/grm/issues/12
		isBinary := isExecutableFileType(ct) && !strings.HasPrefix(filename, "._") && !isLibrary(filename)
		files = append(files, fileEntry{relPath, path, ct, isBinary})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", tmpDir, err)
	}

	// Find max filename width for alignment
	maxW := 0
	for _, f := range files {
		if len(f.relPath) > maxW {
			maxW = len(f.relPath)
		}
	}

	// Print and select
	var binaries []fileEntry
	for _, f := range files {
		if f.isBinary {
			fmt.Printf("    %s %s %s\n", green(sCheck), padCol(f.relPath, maxW, nil), green(f.ct))
		} else {
			fmt.Printf("    %s %s %s\n", dim(sTriang), padCol(f.relPath, maxW, dim), dim(f.ct))
		}
		if f.isBinary {
			binaries = append(binaries, f)
		}
	}

	if len(binaries) == 0 {
		return nil, fmt.Errorf("unable to find a binary file in archive")
	}
	return binaries, nil
}

func isLibrary(filename string) bool {
	name := strings.ToLower(filename)
	return strings.HasSuffix(name, ".so") || strings.Contains(name, ".so.") ||
		strings.HasSuffix(name, ".dylib") || strings.HasSuffix(name, ".dll") || strings.HasSuffix(name, ".node")
}
