package cmd

import (
	"context"
	"fmt"

	"github.com/google/go-github/v30/github"
	"github.com/spf13/cobra"
)

var exploreAll bool

const exploreInfoPattern = "%-20s%-40s%s"

// exploreCmd represents the explore command
var exploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Explore releases on github",
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
		if exploreAll {
			opt := &github.ListOptions{}
			releases, _, err := client.Repositories.ListReleases(context.Background(), owner, repo, opt)
			if err != nil {
				fmt.Println(err)
				return
			}
			printReleaseInfoHeader()
			for _, item := range releases {
				printReleaseInfo(item)
			}
		} else {
			// Show just latest release
			release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
			if err != nil {
				fmt.Println(err)
				return
			}
			printReleaseInfoHeader()
			printReleaseInfo(release)
		}
	},
}

func printReleaseInfoHeader() {
	fmt.Println(fmt.Sprintf(exploreInfoPattern, "Version", "Published", "Info"))
}

func printReleaseInfo(release *github.RepositoryRelease) {
	fmt.Println(fmt.Sprintf(exploreInfoPattern, release.GetTagName(), release.GetPublishedAt(), release.GetHTMLURL()))
}

func init() {
	rootCmd.AddCommand(exploreCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// exploreCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	exploreCmd.Flags().BoolVarP(&exploreAll, "all", "a", false, "Display all releases")
}
