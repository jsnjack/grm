package cmd

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/spf13/cobra"
)

var installFilter string

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <package> [<package>...]",
	Short: "Install a package from github releases",
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		if len(args) == 0 {
			return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go)")
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
		installedPkgs, err := loadAllInstalledFromDB()
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

			// Check that package is not locked
			for _, installedItem := range installedPkgs {
				if installedItem.GetFullName() == pkg.GetFullName() {
					if installedItem.Locked == "true" {
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

			err = installRelease(release, pkg)
			if err != nil {
				return err
			}
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
	installCmd.Flags().StringVarP(&installFilter, "filter", "f", "", "Asset's name must contain provided string, e.g. 'Linux-64bit'")
}

func selectAsset(assets []*github.ReleaseAsset, filter string) (*github.ReleaseAsset, error) {
	// Get all available assets
	assetNames := []string{}
	for _, item := range assets {
		assetNames = append(assetNames, item.GetName())
	}

	filtered := []string{}
	if filter != "" {
		filtered = filterList(assetNames, filter, true)
	} else {
		// Filter by operating system
		filtered = filterList(assetNames, runtime.GOOS, false)
		// Filter by architecture
		filtered = filterList(filtered, runtime.GOARCH, false)
		// Extra filters
		if runtime.GOARCH == "amd64" {
			filtered = filterList(filtered, "64", false)
		}
		if runtime.GOARCH == "386" {
			filtered = filterList(filtered, "32", false)
		}
	}

	// Print suitable assets
	fmt.Printf("Found %d suitable assets\n", len(filtered))
	for id, item := range filtered {
		fmt.Printf("  %d) %s\n", id, item)
	}

	// Select the asset
	var selected string
	switch len(filtered) {
	case 0:
		return nil, fmt.Errorf("Supported asset not found")
	case 1:
		selected = filtered[0]
		break
	default:
		selected = filtered[askForNumber("Select suitable asset:", len(filtered)-1)]
	}

	fmt.Printf("Selected asset: %s\n", selected)
	for _, item := range assets {
		if item.GetName() == selected {
			return item, nil
		}
	}

	return nil, fmt.Errorf("Unexpected error when selecting the asset")
}

// filterList filters list by `filter`. If `strict` is false returns original
// list in case if it doesn't contain `filter`
func filterList(list []string, filter string, strict bool) []string {
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
	if strict {
		return filtered
	}
	if len(filtered) == 0 {
		filtered = list
	}
	return filtered
}

func selectRelease(pkg *Package) (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)
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
	var installedFile string
	switch asset.GetContentType() {
	case "application/octet-stream", "application/zip", "application/gzip", "application/x-gzip":
		installedFile, err = Install(asset)
		break
	default:
		err = fmt.Errorf("Unsupported type: %s", asset.GetContentType())
	}
	if err != nil {
		return err
	}
	err = saveToDB(pkg, pkg.Filter, installedFile, release.GetTagName())
	if err != nil {
		return err
	}
	fmt.Println("done")
	return nil
}
