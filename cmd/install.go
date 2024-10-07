package cmd

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/spf13/cobra"
)

var installFilter []string
var installRefresh bool
var installLock bool
var installRename string
var installSudo bool

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <package> [<package>...]",
	Short: "Install a package from GitHub releases",
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
			if installSudo {
				pkg.Sudo = "sudo "
			}

			// Check that package is not locked
			for _, installedItem := range config.Packages {
				if installedItem.GetFullName() == pkg.GetFullName() {
					if installedItem.Locked {
						fmt.Printf("Package %s is locked\n", pkg.GetFullName())
						continue argsLoop
					}
				}
			}

			// Select the release based on version
			release, err := selectRelease(pkg)
			if err != nil {
				return err
			}
			fmt.Printf("Found release %s\n", release.GetTagName())

			if !installRefresh {
				// Check if package of selected release has already been installed
				for _, installedItem := range config.Packages {
					if installedItem.GetFullName() == pkg.GetFullName() {
						if installedItem.VerifyVersion(release.GetTagName()) == nil {
							fmt.Printf("Package %s already at %s\n", installedItem.GetFullName(), installedItem.Version)
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
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
	installCmd.Flags().BoolVarP(&installSudo, "sudo", "s", true, "Use sudo to install the package")
}

func selectAsset(assets []*github.ReleaseAsset, filter []string) (*github.ReleaseAsset, error) {
	// Get all available assets
	assetNames := []string{}
	for _, item := range assets {
		assetNames = append(assetNames, item.GetName())
	}

	filtered := filterSuitableAssets(assetNames, filter)

	// Print suitable assets
	fmt.Printf("Found %d suitable assets\n", len(filtered))
	for id, item := range filtered {
		fmt.Printf("  %d) %s\n", id+1, item)
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

	fmt.Printf("Selected asset: %s\n", selected)
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
	// Exclude well-known system packages and other extensions
	filtered = exludeExtensions(filtered, ".deb")
	filtered = exludeExtensions(filtered, ".rpm")
	// asc files contain a PGP key (mozilla/geckodriver)
	filtered = exludeExtensions(filtered, ".asc")
	// checksums
	filtered = exludeExtensions(filtered, ".sha256")
	filtered = exludeExtensions(filtered, ".sha256sum")
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

func selectRelease(pkg *Package) (*github.RepositoryRelease, error) {
	client := CreateClient()
	if pkg.Version == "" {
		// Get latest release
		release, _, err := client.Repositories.GetLatestRelease(context.Background(), pkg.Owner, pkg.Repo)
		return release, err
	}
	// Get specific release
	release, _, err := client.Repositories.GetReleaseByTag(context.Background(), pkg.Owner, pkg.Repo, pkg.Version)
	return release, err
}

func installRelease(release *github.RepositoryRelease, pkg *Package) error {
	fmt.Println("Inspecting assets...")
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
	fmt.Println("done")
	return nil
}
