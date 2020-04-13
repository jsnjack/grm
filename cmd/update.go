package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [<package>]",
	Short: "Update installed packages",
	Args: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true
		switch len(args) {
		case 0:
			break
		case 1:
			for _, item := range args {
				_, err := CreatePackage(item)
				if err != nil {
					return fmt.Errorf("requires a package name (e.g. jsnjack/kazy-go), got %s", item)
				}
			}
			break
		default:
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgs, err := loadAllInstalledFromDB()
		if err != nil {
			return err
		}
		for _, p := range pkgs {
			if len(args) == 1 {
				if args[0] != p.Owner+"/"+p.Repo {
					continue
				}
			}
			fmt.Printf("Checking %s/%s...\n", p.Owner, p.Repo)
			if p.Locked == "true" {
				fmt.Println("  held back")
				continue
			}
			release, err := selectRelease(&Package{Owner: p.Owner, Repo: p.Repo})
			if err != nil {
				return err
			}
			if release.GetTagName() == p.Version {
				fmt.Println("  latest")
			} else {
				fmt.Printf("  new version %s\n", release.GetTagName())
				if ok := askForConfirmation("Confirm to update:"); !ok {
					return nil
				}

				// p.Version doesn't matter
				installRelease(release, p)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
