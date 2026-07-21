package cmd

import (
	"fmt"

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
				msgLocked("%s is %s", bold(pkg.GetFullName()), yellow("locked"))
				continue
			}
			ok, err := askForConfirmation(fmt.Sprintf("Are you sure you want to remove %s?", bold(item)))
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			if pkg.IsSystemPackage() {
				err = removeSystemPackage(&pkg)
			} else {
				err = removeBinary(pkg.Filename)
			}
			if err != nil {
				return err
			}
			// Clean db
			delete(config.Packages, item)
		}
		return config.save()
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
