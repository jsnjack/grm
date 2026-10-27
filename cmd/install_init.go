package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

type plannedComponent struct {
	component   InstalledComponent
	source      string
	systemFiles []string
}

func prepareAsset(filename, assetName, tag, binDir string, selection AssetSelection, requests []binaryRequest, legacy *Package) ([]plannedComponent, AssetSelection, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".deb" || ext == ".rpm" {
		if err := checkSystemPackageSupport(ext); err != nil {
			return nil, selection, err
		}
		name, err := getSystemPackageName(filename)
		if err != nil {
			return nil, selection, err
		}
		files, err := getSystemPackageFiles(filename)
		if err != nil {
			return nil, selection, err
		}
		component := InstalledComponent{Asset: assetName, Version: tag, SystemPkgName: name, SystemPkgType: strings.TrimPrefix(ext, ".")}
		return []plannedComponent{{component: component, source: filename, systemFiles: files}}, selection, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, selection, fmt.Errorf("open %s: %w", filename, err)
	}
	ct, err := getFileType(file)
	logClose(file)
	if err != nil {
		return nil, selection, fmt.Errorf("detect file type of %s: %w", filename, err)
	}
	slog.Debug("detected file type", "type", ct)
	var files []fileEntry
	if isExecutableFileType(ct) && !isLibrary(filepath.Base(filename)) {
		files = []fileEntry{{relPath: filepath.Base(filename), fullPath: filename, ct: ct, isBinary: true}}
	} else {
		dir, err := unpackArchive(filename)
		if err != nil {
			return nil, selection, err
		}
		files, err = findBinaryFiles(dir)
		if err != nil {
			return nil, selection, err
		}
	}
	if legacy != nil && legacy.Filename != "" {
		file := files[0]
		name := filepath.Base(legacy.Filename)
		if legacy.RenameBinaryTo == "" {
			found := false
			for _, candidate := range files {
				if matchesPattern(rememberPattern(name, legacy.Version), filepath.Base(candidate.relPath)) {
					file, found = candidate, true
					break
				}
			}
			if !found {
				return nil, selection, fmt.Errorf("previously installed binary %s was not found in %s; use --binary to select again", name, assetName)
			}
		}
		selection.Binaries = []BinarySelection{{Path: file.relPath, Rename: name}}
	}
	selected, remembered, err := selectBinaryFiles(files, assetName, selection.Binaries, requests, tag)
	if err != nil {
		return nil, selection, err
	}
	selection.Binaries = remembered
	var plan []plannedComponent
	for i, file := range selected {
		name := remembered[i].Rename
		if name == "" {
			name = filepath.Base(file.fullPath)
		}
		if err := validateBinaryName(name); err != nil {
			return nil, selection, err
		}
		hash, err := tomd5(file.fullPath)
		if err != nil {
			return nil, selection, err
		}
		component := InstalledComponent{Asset: assetName, Source: file.relPath, Version: tag, Filename: filepath.Join(binDir, name), MD5: hash}
		plan = append(plan, plannedComponent{component: component, source: file.fullPath})
	}
	return plan, selection, nil
}

func getFileType(out io.Reader) (string, error) {
	kind, err := mimetype.DetectReader(out)
	if err != nil {
		return "", fmt.Errorf("detect mime type: %w", err)
	}
	return kind.String(), nil
}

func isExecutableFileType(ct string) bool {
	switch ct {
	case "application/octet-stream", "application/x-executable", "application/x-elf", "application/x-sharedlib", "application/x-mach-binary":
		return true
	}
	return false
}
