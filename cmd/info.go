package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v30/github"
	"github.com/spf13/cobra"
)

var infoAll bool
var infoShort bool

const infoAllPattern = "%-15s%-15s%-15s%s"
const infoPattern = "%-20s %s\n"

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info <package>",
	Short: "Show information about a package and a release",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a pakage name, e.g. jsnjack/kazy-go")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		pkg, err := CreatePackage(args[0])
		if err != nil {
			return err
		}
		client := github.NewClient(nil)
		if infoAll {
			opt := &github.ListOptions{}
			releases, _, err := client.Repositories.ListReleases(context.Background(), pkg.Owner, pkg.Repo, opt)
			if err != nil {
				return err
			}
			printReleaseInfoHeader()
			for _, item := range releases {
				printReleaseInfo(item)
			}
		} else {
			// Show just latest release
			release, err := selectRelease(pkg)
			if err != nil {
				return err
			}
			if infoShort {
				fmt.Println(release.GetTagName())
				for _, item := range release.Assets {
					fmt.Printf("  %s\n", item.GetName())
				}
			} else {
				fmt.Printf(fmt.Sprintf(infoPattern, "Version", release.GetTagName()))
				fmt.Printf(fmt.Sprintf(infoPattern, "Published", release.GetPublishedAt().Format("2006-01-02")))
				fmt.Printf(fmt.Sprintf(infoPattern, "URL", release.GetHTMLURL()))
				fmt.Println("Assets:")
				for _, item := range release.Assets {
					fmt.Printf("  %s\n", item.GetName())
					fmt.Printf("    " + fmt.Sprintf(infoPattern, "Type", item.GetContentType()))
					fmt.Printf("    " + fmt.Sprintf(infoPattern, "Downloads", strconv.Itoa(item.GetDownloadCount())))
					fmt.Printf("    " + fmt.Sprintf(infoPattern, "Download URL", item.GetBrowserDownloadURL()))
					fmt.Printf("    " + fmt.Sprintf(infoPattern, "Size", strconv.Itoa(item.GetSize()/1024/1024)+"MB"))
					fmt.Println()
				}
			}
		}
		return nil
	},
}

func printReleaseInfoHeader() {
	fmt.Println(fmt.Sprintf(infoAllPattern, "Version", "Published", "Downloads", "URL"))
}

func printReleaseInfo(release *github.RepositoryRelease) {
	var downloads int
	for _, item := range release.Assets {
		downloads += item.GetDownloadCount()
	}
	fmt.Println(fmt.Sprintf(infoAllPattern, release.GetTagName(), release.GetPublishedAt().Format("2006-01-02"), strconv.Itoa(downloads), release.GetHTMLURL()))
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	infoCmd.Flags().BoolVarP(&infoAll, "all", "a", false, "Display all releases")
	infoCmd.Flags().BoolVarP(&infoShort, "short", "s", false, "Display in compact format")
}
