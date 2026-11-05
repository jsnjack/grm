package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v93/github"
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
		client, err := CreateClient()
		if err != nil {
			return err
		}
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		installed, isInstalled := config.Packages[pkg.GetFullName()]
		if infoAll {
			releases, _, err := client.Repositories.ListReleases(context.Background(), pkg.Owner, pkg.Repo, &github.ListOptions{})
			if err != nil {
				return fmt.Errorf("list releases for %s: %w", pkg.GetFullName(), err)
			}
			fmt.Println(pkgLabel(pkg.GetFullName()) + "  " + dim(plural(len(releases), "release", "releases")))
			fmt.Println()
			var rows [][]string
			for i, release := range releases {
				var downloads int
				for _, item := range release.Assets {
					downloads += item.GetDownloadCount()
				}
				version, note := cyan(release.GetTagName()), ""
				if i == 0 {
					version, note = boldCyan(release.GetTagName()), dim("latest")
				}
				if isInstalled && installed.Version == release.GetTagName() {
					note = joinDetails(note, green(sLocked+" installed"))
				}
				rows = append(rows, []string{version, release.GetPublishedAt().Format("2006-01-02"), fmt.Sprintf("%8d", downloads), note})
			}
			printTable("  ", []string{"VERSION", "RELEASED", "DOWNLOADS", ""}, rows)
			return nil
		}
		release, err := selectRelease(pkg)
		if err != nil {
			return err
		}
		fmt.Println(pkgLabel(pkg.GetFullName()) + "  " + boldCyan(release.GetTagName()) + "  " + dim("released "+release.GetPublishedAt().Format("2006-01-02")))
		fmt.Println(dim(release.GetHTMLURL()))
		fmt.Println()
		var selections []AssetSelection
		var facts [][]string
		if isInstalled {
			state := green("up to date")
			if installed.VerifyVersion(release.GetTagName()) != nil {
				state = yellow("update available") + dim(" "+sDot+" grm update "+pkg.GetFullName())
			}
			if installed.Locked {
				state = yellow(sLocked + " locked")
			}
			facts = append(facts, []string{dim("Installed"), cyan(installed.Version) + "  " + state})
			selections = generalizeSelections(installed.Selections, installed.Version)
			var patterns []string
			for _, selection := range selections {
				patterns = append(patterns, selection.Pattern)
			}
			if len(patterns) > 0 {
				facts = append(facts, []string{dim("Selection"), strings.Join(patterns, ", ")})
			}
			if len(installed.Filter) > 0 {
				facts = append(facts, []string{dim("Filter"), strings.Join(installed.Filter, ", ")})
			}
		} else {
			facts = append(facts, []string{dim("Installed"), dim("no " + sDot + " grm install " + pkg.GetFullName())})
		}
		printTable("  ", nil, facts)
		fmt.Println()
		var rows [][]string
		for _, item := range release.Assets {
			marker, note := "  ", ""
			for _, selection := range selections {
				if matchesPattern(selection.Pattern, item.GetName()) {
					marker, note = cyan(sSelect)+" ", cyan("selected")
				}
			}
			rows = append(rows, []string{marker + item.GetName(), dim(assetKind(item.GetName())), formatSize(item.GetSize()), fmt.Sprintf("%9d", item.GetDownloadCount()), note})
		}
		printTable("  ", []string{"  ASSET", "TYPE", "SIZE", "DOWNLOADS", ""}, rows)
		if infoLong {
			msgSection("Download URLs", "")
			for _, item := range release.Assets {
				fmt.Printf("    %s\n", item.GetBrowserDownloadURL())
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVarP(&infoAll, "all", "a", false, "Display all latest releases")
	infoCmd.Flags().BoolVarP(&infoLong, "long", "l", false, "Display in long format")
}
