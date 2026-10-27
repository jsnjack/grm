package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path"
	"runtime"
	"sort"
	"strings"

	"github.com/google/go-github/v93/github"
	"github.com/spf13/cobra"
)

var installFilter []string
var installRefresh bool
var installLock bool
var installRename string
var installAssets []string
var installBinaries []string

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:     "install <package> [<package>...]",
	Aliases: []string{"i"},
	Short:   "Install a package from GitHub releases",
	Long: `Install a package from GitHub releases.

Version specifiers:
  <owner>/<repo>               install latest release
  <owner>/<repo>==<tag>        install exact version  (e.g. ==v1.2.0)
  <owner>/<repo>@<tag>         install exact version  (e.g. @v1.2.0)
  <owner>/<repo>~=<filter>     install most recent release matching filter

The ~= filter matches tag names by prefix. If the filter contains glob
metacharacters (* ? [), standard glob matching is used instead:
  jsnjack/grm~=v0.50           any tag starting with v0.50 (e.g. v0.50.1)
  jsnjack/grm~=v0.5*           any tag starting with v0.5 (glob)
  jsnjack/grm~=v[0-9]*-stable  any versioned stable tag (glob)

Examples:
  grm install jsnjack/grm
  grm install jsnjack/grm==v0.50.0
  grm install jsnjack/grm~=v0.50
  grm install jsnjack/grm~=v0.*-stable`,
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		if len(args) == 0 {
			return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go)")
		}

		// Only one package can be renamed
		if len(args) > 1 && installRename != "" {
			return fmt.Errorf("cannot rename multiple packages")
		}

		for _, item := range args {
			_, err := CreatePackage(item)
			if err != nil {
				return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go), got %s", item)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
	argsLoop:
		for _, item := range args {
			pkg, err := CreatePackage(item)
			if err != nil {
				return err
			}
			requested := pkg
			if previous, ok := config.Packages[pkg.GetFullName()]; ok {
				previous.Components = append([]InstalledComponent(nil), previous.Components...)
				pkg = &previous
			}
			if cmd.Flags().Changed("filter") {
				pkg.Filter = installFilter
				pkg.Selections = nil
			}
			if requested.VersionFilter != "" {
				pkg.VersionFilter = requested.VersionFilter
			} else if requested.Version != "" {
				pkg.VersionFilter = ""
			} else {
				requested.VersionFilter = pkg.VersionFilter
			}
			binaries, err := parseBinaryRequests(installBinaries)
			if err != nil {
				return err
			}
			options := installOptions{assets: installAssets, binaries: binaries, rename: installRename, lock: installLock, reselect: cmd.Flags().Changed("filter")}

			// Check that package is not locked
			for _, installedItem := range config.Packages {
				if installedItem.GetFullName() == pkg.GetFullName() {
					if installedItem.Locked {
						msgLocked("%s is %s", bold(pkg.GetFullName()), yellow("locked"))
						continue argsLoop
					}
				}
			}

			// Select the release based on version
			release, err := selectRelease(requested)
			if err != nil {
				return err
			}
			msgOK("Found release %s", boldCyan(release.GetTagName()))

			if !installRefresh && !installLock && requested.VersionFilter == "" && !cmd.Flags().Changed("asset") && !cmd.Flags().Changed("binary") && !cmd.Flags().Changed("rename") && !cmd.Flags().Changed("filter") {
				// Check if package of selected release has already been installed
				for _, installedItem := range config.Packages {
					if installedItem.GetFullName() == pkg.GetFullName() {
						if installedItem.VerifyVersion(release.GetTagName()) == nil {
							msgOK("%s already at %s", bold(installedItem.GetFullName()), cyan(installedItem.Version))
							continue argsLoop
						}
					}
				}
			}

			if err := installRelease(release, pkg, options); err != nil {
				return err
			}
			config, err = ReadConfig(ConfigFile)
			if err != nil {
				return err
			}

		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringSliceVarP(
		&installFilter, "filter", "f", installFilter,
		`Asset's name should contain provided strings,
e.g. 'linux'. Filtering is case insensitive
and not strict, meaning if none of the assets
contain provided filter, all of them are
considered suitable`,
	)
	installCmd.Flags().BoolVarP(&installRefresh, "refresh", "r", false, "Reinstall package")
	installCmd.Flags().BoolVarP(&installLock, "lock", "l", false, "Lock package version")
	installCmd.Flags().StringVarP(&installRename, "rename", "n", "", "Rename binary file during the installation")
	installCmd.Flags().StringArrayVar(&installAssets, "asset", nil, "Select an asset by exact name or glob (repeat for multiple assets)")
	installCmd.Flags().StringArrayVar(&installBinaries, "binary", nil, "Select an archive binary: [asset-glob::]path-glob[=name] (repeatable)")
}

func selectAssets(assets []*github.ReleaseAsset, filter []string) ([]*github.ReleaseAsset, error) {
	names := make([]string, 0, len(assets))
	for _, asset := range assets {
		names = append(names, asset.GetName())
	}
	filtered := filterSuitableAssets(names, filter)
	if _, err := exec.LookPath("rpm"); err == nil {
		sort.SliceStable(filtered, func(i, j int) bool {
			return strings.HasSuffix(strings.ToLower(filtered[i]), ".rpm") && !strings.HasSuffix(strings.ToLower(filtered[j]), ".rpm")
		})
	}
	msgStep("Found %d suitable assets", len(filtered))
	if len(filtered) == 0 {
		return nil, fmt.Errorf("supported asset not found")
	}
	for i, name := range filtered {
		fmt.Printf("    %d) %s\n", i+1, name)
	}
	indices, err := askForNumbers("Select assets (comma-separated numbers or ranges):", len(filtered), false)
	if err != nil {
		return nil, err
	}
	var selected []*github.ReleaseAsset
	for _, index := range indices {
		for _, asset := range assets {
			if asset.GetName() == filtered[index-1] {
				selected = append(selected, asset)
				break
			}
		}
	}
	return selected, nil
}

func filterSuitableAssets(input []string, filters []string) []string {
	return filterSuitableAssetsForPlatform(input, filters, runtime.GOOS, runtime.GOARCH)
}

func filterSuitableAssetsForPlatform(input []string, filters []string, goos, goarch string) []string {
	filtered := input
	if len(filters) != 0 {
		for _, item := range filters {
			filtered = preferToContain(filtered, item)
		}
	}
	// Filter by operating system
	platformAssets := preferToContain(filtered, goos)
	if goos == "linux" {
		for _, name := range filtered {
			lower := strings.ToLower(name)
			if (strings.HasSuffix(lower, ".rpm") || strings.HasSuffix(lower, ".deb")) && !containsAsset(platformAssets, name) {
				platformAssets = append(platformAssets, name)
			}
		}
	}
	filtered = platformAssets
	// Filter by architecture. For well-known arch pairs (amd64/x86_64,
	// arm64/aarch64), match any equivalent name at once so that e.g. an
	// "amd64" .deb does not shadow an "x86_64" .rpm.
	switch goarch {
	case "amd64":
		filtered = preferToContainAny(filtered, "amd64", "x86_64", "x64", goos+"64", "-64", "_64")
	case "arm64":
		filtered = preferToContainAny(filtered, "arm64", "aarch64")
	case "386":
		filtered = preferToContainAny(filtered, "386", "32", goos+"32")
	default:
		filtered = preferToContain(filtered, goarch)
	}
	// Extra OS aliases
	switch goos {
	case "darwin":
		filtered = preferToContainAny(filtered, "mac", "macos", "darwin")
		if goarch == "arm64" {
			// Universal (fat) binaries work on both Intel and Apple Silicon;
			// prefer them when no arm64/aarch64-specific asset exists.
			filtered = preferToContain(filtered, "universal")
		}
	}
	// Exclude system packages for package managers not available on this system
	if _, err := exec.LookPath("dpkg"); err != nil || goos != "linux" {
		slog.Debug("dpkg not found, excluding .deb assets")
		filtered = hardExcludeExtension(filtered, ".deb")
	} else {
		slog.Debug("dpkg found, keeping .deb assets")
	}
	if _, err := exec.LookPath("rpm"); err != nil || goos != "linux" {
		slog.Debug("rpm not found, excluding .rpm assets")
		filtered = hardExcludeExtension(filtered, ".rpm")
	} else {
		slog.Debug("rpm found, keeping .rpm assets")
	}
	filtered = hardExcludeExtension(filtered, ".src.rpm")
	filtered = hardExcludeExtension(filtered, ".nosrc.rpm")
	// AppImage is not supported
	filtered = hardExcludeExtension(filtered, ".AppImage")
	filtered = hardExcludeExtension(filtered, ".zsync")
	// asc files contain a PGP key (mozilla/geckodriver)
	filtered = exludeExtensions(filtered, ".asc")
	// checksums
	filtered = exludeExtensions(filtered, ".sha256")
	filtered = exludeExtensions(filtered, ".sha256sum")
	filtered = exludeExtensions(filtered, ".md5")
	filtered = exludeExtensions(filtered, ".sha")
	return filtered
}

// preferToContain returns list which contains `filter`. If the result is empty
// list, returns the original list
func preferToContain(list []string, filter string) []string {
	filtered := []string{}
	if filter == "" {
		filtered = list
	} else {
		for _, item := range list {
			litem := strings.ToLower(item)
			if strings.Contains(litem, filter) {
				filtered = append(filtered, item)
			}
		}
	}

	// Return full list if everything was filtered out
	if len(filtered) == 0 {
		filtered = list
	}
	return filtered
}

// preferToContainAny returns items that contain any of the given filters.
// If no items match any filter, returns the original list.
func preferToContainAny(list []string, filters ...string) []string {
	filtered := []string{}
	for _, item := range list {
		litem := strings.ToLower(item)
		for _, filter := range filters {
			if filter != "" && strings.Contains(litem, filter) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	if len(filtered) == 0 {
		return list
	}
	return filtered
}

// hardExcludeExtension removes records which end with `ext` from list unconditionally
func hardExcludeExtension(list []string, ext string) []string {
	filtered := []string{}
	for _, item := range list {
		if !strings.HasSuffix(strings.ToLower(item), strings.ToLower(ext)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// exludeExtensions removes records which end with `ext` from list. If the result
// is empty list, returns the original list
func exludeExtensions(list []string, ext string) []string {
	filtered := []string{}
	if ext == "" {
		filtered = list
	} else {
		for _, item := range list {
			litem := strings.ToLower(item)
			if !strings.HasSuffix(litem, ext) {
				filtered = append(filtered, item)
			}
		}
	}

	// Return full list if everything was filtered out
	if len(filtered) == 0 {
		filtered = list
	}
	return filtered
}

// matchVersionFilter reports whether tagName satisfies filter.
// If filter contains glob metacharacters (* ? [) path.Match is used;
// otherwise tagName must have filter as a prefix.
func matchVersionFilter(filter, tagName string) (bool, error) {
	if strings.ContainsAny(filter, "*?[") {
		return path.Match(filter, tagName)
	}
	return strings.HasPrefix(tagName, filter), nil
}

func selectRelease(pkg *Package) (*github.RepositoryRelease, error) {
	client, err := CreateClient()
	if err != nil {
		return nil, err
	}
	if pkg.Version == "" && pkg.VersionFilter == "" {
		// Get latest release
		release, _, err := client.Repositories.GetLatestRelease(context.Background(), pkg.Owner, pkg.Repo)
		if err != nil {
			return nil, fmt.Errorf("get latest release for %s: %w", pkg.GetFullName(), err)
		}
		return release, nil
	}
	if pkg.Version != "" {
		// Get specific release by exact tag
		release, _, err := client.Repositories.GetReleaseByTag(context.Background(), pkg.Owner, pkg.Repo, pkg.Version)
		if err != nil {
			return nil, fmt.Errorf("get release %s for %s: %w", pkg.Version, pkg.GetFullName(), err)
		}
		return release, nil
	}
	// Find the most recent release whose tag matches the version filter.
	// ListReleases returns releases in reverse-chronological order.
	opt := &github.ListOptions{PerPage: 30}
	for {
		releases, resp, err := client.Repositories.ListReleases(context.Background(), pkg.Owner, pkg.Repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list releases for %s: %w", pkg.GetFullName(), err)
		}
		for _, release := range releases {
			matched, err := matchVersionFilter(pkg.VersionFilter, release.GetTagName())
			if err != nil {
				return nil, fmt.Errorf("invalid version filter %q: %w", pkg.VersionFilter, err)
			}
			if matched {
				return release, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil, fmt.Errorf("no release matching ~=%s found for %s/%s", pkg.VersionFilter, pkg.Owner, pkg.Repo)
}

func containsAsset(names []string, name string) bool {
	for _, item := range names {
		if item == name {
			return true
		}
	}
	return false
}
