package cmd

import (
	"context"
	"fmt"
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
		for _, item := range args {
			pkg, err := CreatePackage(item)
			if err != nil {
				return err
			}
			pkg.Filter = installFilter
			// Select the release based on version
			release, err := selectRelease(pkg)
			if err != nil {
				return err
			}
			fmt.Printf("Found release %s\n", release.GetTagName())

			err = installRelease(release, pkg)
			return err
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
	for _, item := range assets {
		fmt.Printf("  %s (%s)\n", item.GetName(), item.GetContentType())
		if strings.Contains(item.GetName(), filter) {
			switch item.GetContentType() {
			case "application/octet-stream", "application/zip", "application/gzip":
				return item, nil
			}
		}
	}
	return nil, fmt.Errorf("Supported asset not found")
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

	fmt.Printf("Found asset %s\n", asset.GetName())

	// Install package
	var installedFile string
	switch asset.GetContentType() {
	case "application/octet-stream", "application/zip", "application/gzip":
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
