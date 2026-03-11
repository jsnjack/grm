package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/spf13/cobra"
)

var installFilter []string
var installRefresh bool
var installLock bool
var installRename string

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <package> [<package>...]",
	Short: "Install a package from GitHub releases",
	Long: `Install a package from GitHub releases.

Version specifiers:
  <owner>/<repo>               install latest release
  <owner>/<repo>==<tag>        install exact version  (e.g. ==v1.2.0)
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
			pkg.Filter = installFilter
			if installRename != "" {
				pkg.RenameBinaryTo = installRename
			}

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
			release, err := selectRelease(pkg)
			if err != nil {
				return err
			}
			msgOK("Found release %s", boldCyan(release.GetTagName()))

			if !installRefresh {
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

			err = installRelease(release, pkg)
			if err != nil {
				return err
			}

			if installLock {
				pkg.Locked = true
			}
			config.PutPackage(pkg)
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
}

func selectAsset(assets []*github.ReleaseAsset, filter []string) (*github.ReleaseAsset, error) {
	// Get all available assets
	assetNames := []string{}
	for _, item := range assets {
		assetNames = append(assetNames, item.GetName())
	}

	filtered := filterSuitableAssets(assetNames, filter)

	// Print suitable assets
	msgStep("Found %d suitable assets", len(filtered))
	for id, item := range filtered {
		fmt.Printf("    %s %s\n", dim(fmt.Sprintf("%d)", id+1)), item)
	}

	// Select the asset
	var selected string
	switch len(filtered) {
	case 0:
		return nil, fmt.Errorf("supported asset not found")
	case 1:
		selected = filtered[0]
	default:
		selected = filtered[askForNumber("Select suitable asset:", len(filtered))-1]
	}

	msgOK("Selected asset %s", bold(selected))
	for _, item := range assets {
		if item.GetName() == selected {
			return item, nil
		}
	}

	return nil, fmt.Errorf("unexpected error when selecting the asset")
}

func filterSuitableAssets(input []string, filters []string) []string {
	filtered := input
	if len(filters) != 0 {
		for _, item := range filters {
			filtered = preferToContain(filtered, item)
		}
	}
	// Filter by operating system
	filtered = preferToContain(filtered, runtime.GOOS)
	// Filter by architecture
	filtered = preferToContain(filtered, runtime.GOARCH)
	// Extra filters
	if runtime.GOARCH == "amd64" {
		filtered = preferToContain(filtered, "64")
		filtered = preferToContain(filtered, runtime.GOOS+"64")
		filtered = preferToContain(filtered, "x86_64")
		filtered = preferToContain(filtered, "x64")
	}
	if runtime.GOARCH == "386" {
		filtered = preferToContain(filtered, "32")
		filtered = preferToContain(filtered, runtime.GOOS+"32")
	}
	if runtime.GOOS == "darwin" {
		filtered = preferToContain(filtered, "mac")
		filtered = preferToContain(filtered, "macos")
		filtered = preferToContain(filtered, "darwin")
	}
	// Exclude system packages for package managers not available on this system
	if _, err := exec.LookPath("dpkg"); err != nil {
		logln("dpkg not found, excluding .deb assets")
		filtered = exludeExtensions(filtered, ".deb")
	} else {
		logln("dpkg found, keeping .deb assets")
	}
	if _, err := exec.LookPath("rpm"); err != nil {
		logln("rpm not found, excluding .rpm assets")
		filtered = exludeExtensions(filtered, ".rpm")
	} else {
		logln("rpm found, keeping .rpm assets")
	}
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
	client := CreateClient()
	if pkg.Version == "" && pkg.VersionFilter == "" {
		// Get latest release
		release, _, err := client.Repositories.GetLatestRelease(context.Background(), pkg.Owner, pkg.Repo)
		return release, err
	}
	if pkg.Version != "" {
		// Get specific release by exact tag
		release, _, err := client.Repositories.GetReleaseByTag(context.Background(), pkg.Owner, pkg.Repo, pkg.Version)
		return release, err
	}
	// Find the most recent release whose tag matches the version filter.
	// ListReleases returns releases in reverse-chronological order.
	opt := &github.ListOptions{PerPage: 30}
	for {
		releases, resp, err := client.Repositories.ListReleases(context.Background(), pkg.Owner, pkg.Repo, opt)
		if err != nil {
			return nil, err
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

func installRelease(release *github.RepositoryRelease, pkg *Package) error {
	msgStep("Inspecting assets...")
	// Select best mached asset
	asset, err := selectAsset(release.Assets, pkg.Filter)
	if err != nil {
		return err
	}

	// Install package
	installedFile, err := Install(asset, pkg)
	if err != nil {
		return err
	}

	// Write changes to config file
	pkg.Filename = installedFile
	pkg.Version = release.GetTagName()
	config, err := ReadConfig(ConfigFile)
	if err != nil {
		return err
	}
	err = config.PutPackage(pkg)
	if err != nil {
		return err
	}
	msgDone()
	return nil
}
