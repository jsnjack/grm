package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v32/github"
	"github.com/spf13/cobra"
)

var infoAll bool
var infoLong bool

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info <package>",
	Short: "Show information about a package",
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		if len(args) < 1 {
			return fmt.Errorf("requires a pakage name, e.g. jsnjack/kazy-go")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		pkg, err := CreatePackage(args[0])
		if err != nil {
			return err
		}
		client := CreateClient()
		if infoAll {
			opt := &github.ListOptions{}
			releases, _, err := client.Repositories.ListReleases(context.Background(), pkg.Owner, pkg.Repo, opt)
			if err != nil {
				return err
			}
			tableHeader([]int{15, 15, 15}, []string{"Version", "Published", "Downloads", "URL"})
			for _, item := range releases {
				printReleaseInfo(item)
			}
		} else {
			release, err := selectRelease(pkg)
			if err != nil {
				return err
			}
			if !infoLong {
				fmt.Println(boldCyan(release.GetTagName()))
				for _, item := range release.Assets {
					fmt.Printf("  %s %s\n", dim(sTriang), item.GetName())
				}
			} else {
				tableRow(padCol("Version", 20, bold), cyan(release.GetTagName()))
				tableRow(padCol("Published", 20, bold), release.GetPublishedAt().Format("2006-01-02"))
				tableRow(padCol("URL", 20, bold), release.GetHTMLURL())
				fmt.Printf("  %s\n", bold("Assets:"))
				for _, item := range release.Assets {
					fmt.Printf("    %s %s\n", cyan(sTriang), bold(item.GetName()))
					fmt.Printf("      %s %s\n", padCol("Type", 20, dim), item.GetContentType())
					fmt.Printf("      %s %s\n", padCol("Downloads", 20, dim), strconv.Itoa(item.GetDownloadCount()))
					fmt.Printf("      %s %s\n", padCol("Download URL", 20, dim), item.GetBrowserDownloadURL())
					fmt.Printf("      %s %s\n", padCol("Size", 20, dim), strconv.Itoa(item.GetSize()/1024/1024)+"MB")
					fmt.Println()
				}
			}
		}
		return nil
	},
}

func printReleaseInfo(release *github.RepositoryRelease) {
	var downloads int
	for _, item := range release.Assets {
		downloads += item.GetDownloadCount()
	}
	tableRow(
		padCol(release.GetTagName(), 15, cyan),
		padCol(release.GetPublishedAt().Format("2006-01-02"), 15, dim),
		padCol(strconv.Itoa(downloads), 15, nil),
		dim(release.GetHTMLURL()),
	)
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVarP(&infoAll, "all", "a", false, "Display all latest releases")
	infoCmd.Flags().BoolVarP(&infoLong, "long", "l", false, "Display in long format")
}
