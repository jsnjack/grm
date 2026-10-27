package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
)

func unpackArchive(filename string) (string, error) {
	slog.Debug("installing from an archive")
	tmpDir := filepath.Join(filepath.Dir(filename), "unpacked")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("create extraction directory: %w", err)
	}
	msgStep("Unpacking archive %s", bold(strings.TrimPrefix(filename, tmpDir)))
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filename, err)
	}
	defer logClose(file)

	ctx := context.Background()
	format, stream, err := archives.Identify(ctx, filename, file)
	if errors.Is(err, archives.NoMatch) {
		return "", fmt.Errorf("unpack %s: unsupported archive format", filename)
	}
	if err != nil {
		return "", fmt.Errorf("identify %s: %w", filename, err)
	}
	switch format := format.(type) {
	case archives.Extractor:
		if err := format.Extract(ctx, stream, extractEntry(tmpDir)); err != nil {
			return "", fmt.Errorf("unpack %s: %w", filename, err)
		}
		slog.Debug("unpacked", "path", tmpDir)
	case archives.Decompressor:
		// A single compressed file, not an archive
		dest := filepath.Join(tmpDir, strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		if err := decompressFile(format, stream, dest); err != nil {
			return "", fmt.Errorf("decompress %s: %w", filename, err)
		}
		slog.Debug("decompressed", "path", dest)
	default:
		return "", fmt.Errorf("unpack %s: unsupported archive format", filename)
	}

	return tmpDir, nil
}

// extractEntry writes regular files and directories below dir. Links and
// other special entries are skipped, and entries outside dir are rejected.
func extractEntry(dir string) archives.FileHandler {
	return func(ctx context.Context, entry archives.FileInfo) error {
		name := filepath.FromSlash(path.Clean(entry.NameInArchive))
		if !filepath.IsLocal(name) {
			return fmt.Errorf("entry %q is outside the extraction directory", entry.NameInArchive)
		}
		dest := filepath.Join(dir, name)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if !entry.Mode().IsRegular() || entry.LinkTarget != "" {
			slog.Debug("skipping archive entry", "name", entry.NameInArchive, "mode", entry.Mode())
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		perm := os.FileMode(0644)
		if entry.Mode()&0111 != 0 {
			perm = 0755
		}
		src, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open %s: %w", entry.NameInArchive, err)
		}
		defer logClose(src)
		return writeFile(dest, src, perm)
	}
}

func decompressFile(format archives.Decompressor, stream io.Reader, dest string) error {
	reader, err := format.OpenReader(stream)
	if err != nil {
		return err
	}
	defer logClose(reader)
	return writeFile(dest, reader, 0644)
}

func writeFile(dest string, src io.Reader, perm os.FileMode) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, src); err != nil {
		logClose(out)
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return out.Close()
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
