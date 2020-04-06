package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgs, err := loadInstalledFromDB()
		if err != nil {
			return err
		}
		for _, p := range pkgs {
			fmt.Printf("Checking %s/%s...\n", p.Owner, p.Repo)
			release, err := selectRelease(&Package{Owner: p.Owner, Repo: p.Repo})
			if err != nil {
				return err
			}
			if release.GetTagName() == p.Version {
				fmt.Println("  latest")
			} else {
				fmt.Printf("  new version %s\n", release.GetTagName())
				// p.Version doesn't matter
				installRelease(release, p)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
