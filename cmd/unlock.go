package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// unlockCmd represents the unlock command
var unlockCmd = &cobra.Command{
	Use:   "unlock <package> [<package>...]",
	Short: "Unlock a package",
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
			if ok {
				pkg.Locked = false
			} else {
				fmt.Printf("Package %s is not installed\n", item)
				continue
			}
		}
		return config.save()
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}
