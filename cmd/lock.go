package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// lockCmd represents the lock command
var lockCmd = &cobra.Command{
	Use:   "lock <package> [<package>...]",
	Short: "Lock a package",
	Long:  "Locked packages cannot be installed, removed, or updated unless the lock mark is removed",
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
				pkg.Locked = true
				err = config.PutPackage(&pkg)
				if err != nil {
					return err
				}
				msgLocked("Locked %s at %s %s", pkgLabel(item), cyan(pkg.Version), dim(sDot+" updates skip it until grm unlock "+item))
			} else {
				msgWarn("%s is not installed %s", pkgLabel(item), dim(sDot+" grm list shows installed packages"))
				continue
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
