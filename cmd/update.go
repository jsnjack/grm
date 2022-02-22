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
		default:
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return err
		}
		for _, p := range config.Packages {
			if len(args) == 1 {
				if args[0] != p.Owner+"/"+p.Repo {
					continue
				}
			}
			fmt.Printf("Checking %s/%s...\n", p.Owner, p.Repo)
			if p.Locked {
				fmt.Println("  locked")
				continue
			}
			release, err := selectRelease(&Package{Owner: p.Owner, Repo: p.Repo})
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = p.VerifyVersion(release.GetTagName())
			if err == nil {
				fmt.Println("  latest")
			} else {
				fmt.Println(" ", err)
				if ok := askForConfirmation("Confirm to update:"); !ok {
					continue
				}

				// p.Version doesn't matter
				err = installRelease(release, &p)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
