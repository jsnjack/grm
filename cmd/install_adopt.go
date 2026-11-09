package cmd

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v93/github"
)

// releaseOnDiskNote explains why grm recorded a release without installing it.
const releaseOnDiskNote = "installed binaries match the release " + sDot + " record updated"

// releaseOnDisk reports whether the binaries recorded for pkg already contain
// release, because the program replaced its own binary (wakeci updates itself
// from its settings page). It returns pkg recorded at that release, so grm
// neither downloads the release again nor reports an update.
//
// It compares each file's SHA-256 with the digest GitHub lists for its asset.
// Only raw-binary assets qualify, because an archive's digest covers the
// archive, not the binaries inside it. Every component must match, at the
// path a new install would use, so a pending move to another directory or a
// changed name still reinstalls. A symlink or a file that isn't executable
// also reinstalls, because grm would replace it with an executable file.
func releaseOnDisk(pkg Package, release *github.RepositoryRelease, binDir string) (Package, bool) {
	tag := release.GetTagName()
	if pkg.Incomplete || len(pkg.Components) == 0 || len(pkg.Components) != len(pkg.Selections) {
		return pkg, false
	}
	selections := generalizeSelections(pkg.Selections, pkg.Version)
	used := make([]bool, len(selections))
	components := make([]InstalledComponent, 0, len(pkg.Components))
	for _, component := range pkg.Components {
		if component.Filename == "" || component.Asset == "" || component.Source != component.Asset {
			return pkg, false
		}
		index := -1
		for i, selection := range selections {
			if !used[i] && matchesPattern(selection.Pattern, component.Asset) {
				index = i
				break
			}
		}
		if index < 0 || len(selections[index].Binaries) > 1 {
			return pkg, false
		}
		used[index] = true
		asset := singleMatchingAsset(release.Assets, selections[index].Pattern)
		if asset == nil {
			return pkg, false
		}
		name := asset.GetName()
		if binaries := selections[index].Binaries; len(binaries) == 1 && binaries[0].Rename != "" {
			name = binaries[0].Rename
		}
		if filepath.Join(binDir, name) != component.Filename {
			return pkg, false
		}
		info, err := os.Lstat(component.Filename)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return pkg, false
		}
		digest, ok := strings.CutPrefix(asset.GetDigest(), "sha256:")
		if !ok {
			return pkg, false
		}
		hash, err := tosha256(component.Filename)
		if err != nil {
			slog.Debug("compare installed binary with release", "path", component.Filename, "err", err)
			return pkg, false
		}
		if !strings.EqualFold(hash, digest) {
			return pkg, false
		}
		components = append(components, InstalledComponent{Asset: asset.GetName(), Source: asset.GetName(), Version: tag, Filename: component.Filename})
	}
	pkg.Components = components
	// Saved as an install of tag saves them
	pkg.Selections = generalizeSelections(selections, tag)
	pkg.Version = tag
	return pkg, true
}

func singleMatchingAsset(assets []*github.ReleaseAsset, pattern string) *github.ReleaseAsset {
	var found *github.ReleaseAsset
	for _, asset := range assets {
		if matchesPattern(pattern, asset.GetName()) {
			if found != nil {
				return nil
			}
			found = asset
		}
	}
	return found
}
