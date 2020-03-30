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
	Use:   "install <repository>",
	Short: "Install a package from github releases",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a pakage name, e.g. jsnjack/kazy-go")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		owner, repo, err := cleanPackage(args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		client := github.NewClient(nil)
		release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Found release %s\n", release.GetTagName())
		asset, err := selectAsset(release.Assets)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = install.Application(asset)
		if err != nil {
			fmt.Println(err)
			return
		}
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
