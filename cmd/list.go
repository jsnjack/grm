package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var listRepoDescription bool
var listFlat bool
var listLong bool

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		packages := sortedPackages(config)
		if len(packages) == 0 {
			fmt.Println("No installed packages " + dim(sDot+" grm install owner/repo adds one"))
			return nil
		}
		switch {
		case listRepoDescription:
			return printDescriptions(packages)
		case listFlat:
			names := make([]string, 0, len(packages))
			for _, p := range packages {
				names = append(names, p.GetFullName())
			}
			fmt.Println(strings.Join(names, " "))
		case listLong:
			printLongList(packages)
		default:
			printList(packages)
		}
		return nil
	},
}

func sortedPackages(config *GrmConfig) []Package {
	packages := make([]Package, 0, len(config.Packages))
	for _, p := range config.Packages {
		packages = append(packages, p)
	}
	sort.Slice(packages, func(i, j int) bool {
		return strings.ToLower(packages[i].GetFullName()) < strings.ToLower(packages[j].GetFullName())
	})
	return packages
}

// provides lists the commands and system packages a package installed.
func provides(p Package) string {
	var parts []string
	for _, c := range p.installedComponents() {
		parts = append(parts, componentLabel(c))
	}
	return strings.Join(parts, dim(", "))
}

func packageStatus(p Package) string {
	switch {
	case p.Incomplete:
		return red(sWarn + " incomplete")
	case p.Locked:
		return yellow(sLocked + " locked")
	}
	return ""
}

func printList(packages []Package) {
	var rows [][]string
	systemPackages, locked, incomplete := 0, 0, 0
	binaries := make(map[string]int)
	for _, p := range packages {
		rows = append(rows, []string{pkgLabel(p.GetFullName()), cyan(p.Version), provides(p), packageStatus(p)})
		for _, c := range p.installedComponents() {
			if c.SystemPkgName != "" {
				systemPackages++
			} else {
				binaries[filepath.Dir(c.Filename)]++
			}
		}
		if p.Locked {
			locked++
		}
		if p.Incomplete {
			incomplete++
		}
	}
	printTable("", []string{"PACKAGE", "VERSION", "PROVIDES", ""}, rows)
	summary := []string{plural(len(packages), "package", "packages")}
	summary = append(summary, binaryCounts(binaries)...)
	if systemPackages > 0 {
		summary = append(summary, plural(systemPackages, "system package", "system packages"))
	}
	line := dim(strings.Join(summary, " "+sDot+" "))
	if locked > 0 {
		line += dim(" "+sDot+" ") + yellow(fmt.Sprintf("%d locked", locked))
	}
	if incomplete > 0 {
		line += dim(" "+sDot+" ") + red(fmt.Sprintf("%d incomplete", incomplete)) + dim(" (grm update finishes them)")
	}
	fmt.Println()
	fmt.Println(line)
}

// binaryCounts describes how many binaries each directory holds, the
// directory with the most first.
func binaryCounts(binaries map[string]int) []string {
	dirs := make([]string, 0, len(binaries))
	for dir := range binaries {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool {
		if binaries[dirs[i]] != binaries[dirs[j]] {
			return binaries[dirs[i]] > binaries[dirs[j]]
		}
		return dirs[i] < dirs[j]
	})
	parts := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		parts = append(parts, plural(binaries[dir], "binary", "binaries")+" in "+dir)
	}
	return parts
}

func printLongList(packages []Package) {
	legacy := 0
	for i, p := range packages {
		if i > 0 {
			fmt.Println()
		}
		var meta []string
		if len(p.Filter) > 0 {
			meta = append(meta, "filter "+strings.Join(p.Filter, ", "))
		}
		if p.VersionFilter != "" {
			meta = append(meta, "versions ~="+p.VersionFilter)
		}
		if p.BinDir != "" {
			meta = append(meta, "bin dir "+p.BinDir)
		}
		old := len(p.Components) == 0 && (p.Filename != "" || p.SystemPkgName != "")
		if old {
			meta = append(meta, "old config record")
			legacy++
		}
		line := pkgLabel(p.GetFullName()) + "  " + cyan(p.Version)
		if status := packageStatus(p); status != "" {
			line += "  " + status
		}
		if len(meta) > 0 {
			line += "  " + dim(strings.Join(meta, " "+sDot+" "))
		}
		fmt.Println(line)
		components := p.installedComponents()
		var rows [][]string
		for j, c := range components {
			tree := sTreeMid
			if j == len(components)-1 {
				tree = sTreeEnd
			}
			where := c.Filename
			if c.SystemPkgName != "" {
				where = "system package"
			}
			source := "source not recorded"
			if c.Asset != "" {
				source = "from " + c.Asset
				if c.Source != "" && c.Source != filepath.Base(c.Filename) {
					source += " " + sArrow + " " + c.Source
				}
			}
			version := ""
			if c.Version != p.Version {
				version = yellow(c.Version)
			}
			rows = append(rows, []string{dim(tree) + " " + componentLabel(c), dim(where), dim(source), version})
		}
		printTable("  ", nil, rows)
	}
	if legacy > 0 {
		fmt.Println()
		fmt.Println(dim(fmt.Sprintf("%s the old config record. The next update records their sources.", plural(legacy, "package uses", "packages use"))))
	}
}

func printDescriptions(packages []Package) error {
	client, err := CreateClient()
	if err != nil {
		return err
	}
	var rows [][]string
	for _, p := range packages {
		description := ""
		repo, _, err := client.Repositories.Get(context.Background(), p.Owner, p.Repo)
		if err != nil {
			description = red(err.Error())
		} else {
			description = repo.GetDescription()
		}
		rows = append(rows, []string{pkgLabel(p.GetFullName()), description})
	}
	printTable("", []string{"PACKAGE", "DESCRIPTION"}, rows)
	return nil
}

func init() {
	listCmd.Flags().BoolVar(&listLong, "long", false, "Show every installed binary and system package")
	listCmd.Flags().BoolVar(&listRepoDescription, "description", false, "Print description of the repositories")
	listCmd.Flags().BoolVarP(&listFlat, "flat", "f", false, "Print installed packages in flat form")
	rootCmd.AddCommand(listCmd)
}
