package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v93/github"
)

type installOptions struct {
	assets   []string
	binaries []binaryRequest
	rename   string
	lock     bool
	reselect bool
}

type binaryRequest struct {
	asset   string
	pattern string
	rename  string
	matched bool
}

func parseBinaryRequests(values []string) ([]binaryRequest, error) {
	var requests []binaryRequest
	for _, value := range values {
		request := binaryRequest{}
		pattern := value
		if asset, member, ok := strings.Cut(pattern, "::"); ok {
			request.asset, pattern = asset, member
			if err := validatePattern(asset); err != nil {
				return nil, err
			}
		}
		if member, rename, ok := strings.Cut(pattern, "="); ok {
			pattern, request.rename = member, rename
			if err := validateBinaryName(rename); err != nil {
				return nil, err
			}
		}
		if err := validatePattern(pattern); err != nil {
			return nil, err
		}
		request.pattern = pattern
		requests = append(requests, request)
	}
	return requests, nil
}

func validatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("selection pattern must not be empty")
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return fmt.Errorf("invalid selection pattern %q: %w", pattern, err)
	}
	return nil
}

func matchesPattern(pattern, name string) bool {
	if pattern == name {
		return true
	}
	matched, err := path.Match(pattern, name)
	return err == nil && matched
}

func rememberPattern(name, tag string) string {
	if tag != "" {
		name = strings.ReplaceAll(name, tag, "*")
		version := strings.TrimPrefix(tag, "v")
		if version != "" {
			name = strings.ReplaceAll(name, version, "*")
		}
	}
	return name
}

func legacyAssetCandidates(assets []*github.ReleaseAsset, pkg *Package) []*github.ReleaseAsset {
	var candidates []*github.ReleaseAsset
	for _, asset := range assets {
		ext := strings.ToLower(path.Ext(asset.GetName()))
		if pkg.SystemPkgName != "" {
			if ext == "."+pkg.SystemPkgType {
				candidates = append(candidates, asset)
			}
		} else if ext != ".deb" && ext != ".rpm" {
			candidates = append(candidates, asset)
		}
	}
	return candidates
}

func resolveAssetSelections(assets []*github.ReleaseAsset, saved []AssetSelection, requested, filter []string, tag string) ([]*github.ReleaseAsset, []AssetSelection, error) {
	selections := append([]AssetSelection(nil), saved...)
	if len(requested) != 0 {
		selections = nil
		for _, pattern := range requested {
			if err := validatePattern(pattern); err != nil {
				return nil, nil, err
			}
			selections = append(selections, AssetSelection{Pattern: pattern})
		}
	}
	if len(selections) == 0 {
		selected, err := selectAssets(assets, filter)
		if err != nil {
			return nil, nil, err
		}
		for _, asset := range selected {
			selections = append(selections, AssetSelection{Pattern: rememberPattern(asset.GetName(), tag)})
		}
		return selected, selections, nil
	}
	var selected []*github.ReleaseAsset
	var resolved []AssetSelection
	seen := make(map[string]bool)
	for _, selection := range selections {
		var matches []*github.ReleaseAsset
		for _, asset := range assets {
			if matchesPattern(selection.Pattern, asset.GetName()) {
				matches = append(matches, asset)
			}
		}
		if len(matches) != 1 {
			return nil, nil, fmt.Errorf("asset pattern %q matches %d assets; use --asset with a more specific choice", selection.Pattern, len(matches))
		}
		asset := matches[0]
		if seen[asset.GetName()] {
			return nil, nil, fmt.Errorf("asset %s selected more than once", asset.GetName())
		}
		seen[asset.GetName()] = true
		selection.Pattern = rememberPattern(selection.Pattern, tag)
		if len(requested) != 0 {
			for _, previous := range saved {
				if matchesPattern(previous.Pattern, asset.GetName()) {
					selection.Binaries = previous.Binaries
				}
			}
		}
		selected = append(selected, asset)
		resolved = append(resolved, selection)
	}
	return selected, resolved, nil
}

func selectBinaryFiles(files []fileEntry, assetName string, saved []BinarySelection, requests []binaryRequest, tag string) ([]fileEntry, []BinarySelection, error) {
	var choices []BinarySelection
	for i := range requests {
		request := &requests[i]
		if request.asset != "" && !matchesPattern(request.asset, assetName) {
			continue
		}
		found := false
		for _, file := range files {
			if matchesPattern(request.pattern, file.relPath) || matchesPattern(request.pattern, path.Base(file.relPath)) {
				choices = append(choices, BinarySelection{Path: file.relPath, Rename: request.rename})
				found, request.matched = true, true
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("binary pattern %q matches no executable in %s", request.pattern, assetName)
		}
	}
	if len(choices) == 0 {
		choices = saved
	}
	if len(choices) == 0 {
		for i, file := range files {
			fmt.Printf("    %d) %s\n", i+1, file.relPath)
		}
		indices, err := askForNumbers("Select binaries (comma-separated numbers, ranges, or all):", len(files), true)
		if err != nil {
			return nil, nil, err
		}
		for _, index := range indices {
			choices = append(choices, BinarySelection{Path: files[index-1].relPath})
		}
	}
	var selected []fileEntry
	var remembered []BinarySelection
	seen := make(map[string]bool)
	for _, choice := range choices {
		var matches []fileEntry
		for _, file := range files {
			if matchesPattern(choice.Path, file.relPath) {
				matches = append(matches, file)
			}
		}
		if len(matches) != 1 {
			return nil, nil, fmt.Errorf("remembered binary %q matches %d files in %s; use --binary to choose again", choice.Path, len(matches), assetName)
		}
		file := matches[0]
		if seen[file.relPath] {
			return nil, nil, fmt.Errorf("binary %s selected more than once in %s", file.relPath, assetName)
		}
		seen[file.relPath] = true
		selected = append(selected, file)
		remembered = append(remembered, BinarySelection{Path: rememberPattern(file.relPath, tag), Rename: choice.Rename})
	}
	return selected, remembered, nil
}

func validateBinaryName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "\\\x00") {
		return fmt.Errorf("invalid binary name %q: expected a filename without directories", name)
	}
	return nil
}

func parseNumbers(response string, count int) ([]int, error) {
	selected := make(map[int]bool)
	if response == "all" {
		for i := 1; i <= count; i++ {
			selected[i] = true
		}
	} else {
		for _, item := range strings.Split(response, ",") {
			first, last, ranged := strings.Cut(item, "-")
			start, err := strconv.Atoi(first)
			if err != nil {
				return nil, fmt.Errorf("invalid selection %q", item)
			}
			end := start
			if ranged {
				end, err = strconv.Atoi(last)
			}
			if err != nil || start < 1 || end > count || end < start {
				return nil, fmt.Errorf("selection %q must be within 1-%d", item, count)
			}
			for i := start; i <= end; i++ {
				selected[i] = true
			}
		}
	}
	var indices []int
	for index := range selected {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	if len(indices) == 0 {
		return nil, fmt.Errorf("select at least one item")
	}
	return indices, nil
}

func askForNumbers(message string, count int, allWithYes bool) ([]int, error) {
	if count == 1 || (rootYes && !allWithYes) {
		return []int{1}, nil
	}
	if rootYes {
		return parseNumbers("all", count)
	}
	for {
		fmt.Printf("%s [1-%d] ", message, count)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return nil, fmt.Errorf("read selection: %w", err)
		}
		indices, err := parseNumbers(response, count)
		if err == nil {
			return indices, nil
		}
		fmt.Println(err)
	}
}
