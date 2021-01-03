package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove <package> [<package>...]",
	Short: "Remove a package",
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
				fmt.Printf("Package %s is not installed\n", item)
				continue
			}
			if pkg.Locked {
				fmt.Printf("Package %s is locked\n", pkg.GetFullName())
				continue
			}
			if ok := askForConfirmation(fmt.Sprintf("Are you sure you want to remove %s?", item)); !ok {
				return nil
			}
			// Remove binary
			err = removeBinary(pkg.Filename)
			if err != nil {
				return err
			}
			// Clean db
			delete(config.Packages, item)
		}
		config.save()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
