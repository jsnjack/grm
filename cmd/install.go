package cmd

import (
	"context"
	"fmt"

	"github.com/google/go-github/v30/github"
	"github.com/jsnjack/grm/install"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <repository...>",
	Short: "Install a package from github releases",
	Args: func(cmd *cobra.Command, args []string) error {
		for _, item := range args {
			_, err := CreatePackage(item)
			if err != nil {
				return fmt.Errorf("requires a package name(e.g. jsnjack/kazy-go), got %s", item)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		for _, item := range args {
			pkg, err := CreatePackage(item)
			if err != nil {
				return err
			}
			client := github.NewClient(nil)
			release, _, err := client.Repositories.GetLatestRelease(context.Background(), pkg.Owner, pkg.Repo)
			if err != nil {
				return err
			}
			fmt.Printf("Found release %s\n", release.GetTagName())
			asset, err := selectAsset(release.Assets)
			if err != nil {
				return err
			}
			err = install.Application(asset)
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
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func selectAsset(assets []*github.ReleaseAsset) (*github.ReleaseAsset, error) {
	for _, item := range assets {
		fmt.Printf("  %s (%s)\n", item.GetName(), item.GetContentType())
		if item.GetContentType() == "application/octet-stream" {
			return item, nil
		}
	}
	return nil, fmt.Errorf("Supported asset not found")
}
