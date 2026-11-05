package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "remove <package> [<package>...]",
	Aliases: []string{"rm"},
	Short:   "Remove a package",
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
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		for _, item := range args {
			pkg, ok := config.Packages[item]
			if !ok {
				msgWarn("Package %s is not installed", bold(item))
				continue
			}
			if pkg.Locked {
				msgLocked("%s is locked at %s %s", pkgLabel(pkg.GetFullName()), cyan(pkg.Version), dim(sDot+" grm unlock "+item+" allows removal"))
				continue
			}
			started := time.Now()
			fmt.Println(pkgLabel(pkg.GetFullName()) + "  " + cyan(pkg.Version))
			fmt.Println()
			var rows [][]string
			binaries, packages := 0, 0
			for _, c := range pkg.installedComponents() {
				rows = append(rows, []string{red(sRemove) + " " + red(componentName(c)) + componentKind(c), dim(componentTarget(c))})
				if c.SystemPkgName != "" {
					packages++
				} else {
					binaries++
				}
			}
			printTable("    ", nil, rows)
			fmt.Println()
			var what []string
			if binaries > 0 {
				what = append(what, plural(binaries, "file", "files"))
			}
			if packages > 0 {
				what = append(what, plural(packages, "system package", "system packages"))
			}
			question := "Remove " + strings.Join(what, " and ") + "?"
			if len(what) == 0 {
				question = "Forget " + item + "?"
			}
			ok, err := askForConfirmation(question, false)
			if err != nil {
				return err
			}
			if !ok {
				msgResult(true, "Nothing removed")
				return nil
			}
			if err := removeInstalledPackage(config, &pkg, removeComponent); err != nil {
				return err
			}
			msgResult(true, "Removed %s %s", pkgLabel(pkg.GetFullName()), dim(sDot+" "+joinDetails(plural(binaries+packages, "component", "components"), formatDuration(time.Since(started)))))
		}
		return config.save()
	},
}

func removeInstalledPackage(config *GrmConfig, pkg *Package, remove func(InstalledComponent) error) error {
	pkg.Components = append([]InstalledComponent(nil), pkg.installedComponents()...)
	pkg.Filename, pkg.MD5, pkg.SystemPkgName, pkg.SystemPkgType = "", "", "", ""
	for len(pkg.Components) > 0 {
		component := pkg.Components[0]
		if err := remove(component); err != nil {
			return fmt.Errorf("remove %s from %s: %w", component.identity(), pkg.GetFullName(), err)
		}
		pkg.Components = pkg.Components[1:]
		pkg.Version = ""
		pkg.Incomplete = true
		if len(pkg.Components) > 0 {
			if err := config.PutPackage(pkg); err != nil {
				return err
			}
		}
	}
	delete(config.Packages, pkg.GetFullName())
	return config.save()
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
